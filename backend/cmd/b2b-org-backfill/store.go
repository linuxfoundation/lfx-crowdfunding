// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgStore struct {
	pool *pgxpool.Pool
}

func (s *pgStore) UnmatchedDomains(ctx context.Context) ([]string, error) {
	const q = `
		SELECT org_domain FROM crowdfunding.donations
		WHERE org_domain IS NOT NULL AND b2b_org_sfid IS NULL
		UNION
		SELECT org_domain FROM crowdfunding.subscriptions
		WHERE org_domain IS NOT NULL AND b2b_org_sfid IS NULL
		ORDER BY 1`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list unmatched domains: %w", err)
	}
	defer rows.Close()
	var domains []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("scan unmatched domain: %w", err)
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

func (s *pgStore) SetDonationsSFID(ctx context.Context, domain, sfid string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE crowdfunding.donations SET b2b_org_sfid = $2, updated_on = NOW()
		WHERE org_domain = $1 AND b2b_org_sfid IS NULL`, domain, sfid)
	if err != nil {
		return 0, fmt.Errorf("set donations b2b_org sfid for %s: %w", domain, err)
	}
	return tag.RowsAffected(), nil
}

func (s *pgStore) UnmatchedSubscriptions(ctx context.Context, domain string) ([]unmatchedSub, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(status, ''), COALESCE(stripe_subscription_id, ''), COALESCE(stripe_subscription_item_id, '')
		FROM crowdfunding.subscriptions
		WHERE org_domain = $1 AND b2b_org_sfid IS NULL`, domain)
	if err != nil {
		return nil, fmt.Errorf("list unmatched subscriptions for %s: %w", domain, err)
	}
	defer rows.Close()
	var subs []unmatchedSub
	for rows.Next() {
		var sub unmatchedSub
		if err := rows.Scan(&sub.ID, &sub.Status, &sub.StripeSubscriptionID, &sub.StripeSubscriptionItemID); err != nil {
			return nil, fmt.Errorf("scan unmatched subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (s *pgStore) SetSubscriptionSFID(ctx context.Context, id, sfid string) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE crowdfunding.subscriptions SET b2b_org_sfid = $2, updated_on = NOW()
		WHERE id = $1 AND b2b_org_sfid IS NULL`, id, sfid); err != nil {
		return fmt.Errorf("set subscription b2b_org sfid: %w", err)
	}
	return nil
}

// ExportMatches writes domain,sfid for every matched domain, the input for
// ledger-service's backfill-organization-claim --sfid-csv. A domain whose rows
// carry more than one SFID is left out and returned, so it can be checked by hand.
func (s *pgStore) ExportMatches(ctx context.Context, w io.Writer) (exported int, conflicts []string, err error) {
	rows, err := s.pool.Query(ctx, `
		SELECT org_domain, MIN(b2b_org_sfid), COUNT(DISTINCT b2b_org_sfid)
		FROM (
			SELECT org_domain, b2b_org_sfid FROM crowdfunding.donations WHERE b2b_org_sfid IS NOT NULL
			UNION ALL
			SELECT org_domain, b2b_org_sfid FROM crowdfunding.subscriptions WHERE b2b_org_sfid IS NOT NULL
		) m
		WHERE org_domain IS NOT NULL
		GROUP BY org_domain
		ORDER BY org_domain`)
	if err != nil {
		return 0, nil, fmt.Errorf("list matched domains: %w", err)
	}
	defer rows.Close()

	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"domain", "sfid"}); err != nil {
		return 0, nil, err
	}
	for rows.Next() {
		var domain, sfid string
		var n int
		if err := rows.Scan(&domain, &sfid, &n); err != nil {
			return 0, nil, fmt.Errorf("scan matched domain: %w", err)
		}
		if n > 1 {
			conflicts = append(conflicts, domain)
			continue
		}
		if err := cw.Write([]string{domain, sfid}); err != nil {
			return 0, nil, err
		}
		exported++
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	cw.Flush()
	return exported, conflicts, cw.Error()
}
