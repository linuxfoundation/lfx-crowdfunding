// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var orgLogoTracer = otel.Tracer("org-logo-db")

// OrgLogoRepository implements domain.OrgLogoRepository against PostgreSQL.
type OrgLogoRepository struct {
	pool *pgxpool.Pool
}

// NewOrgLogoRepository creates a new OrgLogoRepository.
func NewOrgLogoRepository(pool *pgxpool.Pool) *OrgLogoRepository {
	return &OrgLogoRepository{pool: pool}
}

// SetOrgLogo stores the logo of the b2b_org sfid; an empty logoURL removes it.
func (r *OrgLogoRepository) SetOrgLogo(ctx context.Context, sfid, logoURL string) error {
	ctx, span := orgLogoTracer.Start(ctx, "db.b2b_org_logos.SetOrgLogo")
	defer span.End()
	span.SetAttributes(attribute.String("b2b_org.sfid", sfid))

	var err error
	if logoURL == "" {
		_, err = r.pool.Exec(ctx, `DELETE FROM b2b_org_logos WHERE b2b_org_sfid = $1`, sfid)
	} else {
		_, err = r.pool.Exec(ctx, `
			INSERT INTO b2b_org_logos (b2b_org_sfid, logo_url, refreshed_on)
			VALUES ($1, $2, NOW())
			ON CONFLICT (b2b_org_sfid) DO UPDATE
			SET logo_url = EXCLUDED.logo_url, refreshed_on = NOW()`, sfid, logoURL)
	}
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("set b2b_org logo: %w", err)
	}
	return nil
}

// LogosByDomains maps org domains to the logo of the b2b_org their donations
// and subscriptions are matched to. A domain matched to more than one SFID is
// left out rather than shown with either org's logo.
func (r *OrgLogoRepository) LogosByDomains(ctx context.Context, domains []string) (map[string]string, error) {
	out := map[string]string{}
	if len(domains) == 0 {
		return out, nil
	}
	ctx, span := orgLogoTracer.Start(ctx, "db.b2b_org_logos.LogosByDomains")
	defer span.End()
	span.SetAttributes(attribute.Int("domains.count", len(domains)))

	rows, err := r.pool.Query(ctx, `
		WITH matched AS (
			SELECT org_domain, b2b_org_sfid FROM donations
			WHERE org_domain = ANY($1) AND b2b_org_sfid IS NOT NULL
			UNION
			SELECT org_domain, b2b_org_sfid FROM subscriptions
			WHERE org_domain = ANY($1) AND b2b_org_sfid IS NOT NULL
		), unique_match AS (
			SELECT org_domain, MIN(b2b_org_sfid) AS b2b_org_sfid
			FROM matched
			GROUP BY org_domain
			HAVING COUNT(DISTINCT b2b_org_sfid) = 1
		)
		SELECT u.org_domain, l.logo_url
		FROM unique_match u
		JOIN b2b_org_logos l ON l.b2b_org_sfid = u.b2b_org_sfid`, domains)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("query org logos by domain: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var domain, logoURL string
		if err := rows.Scan(&domain, &logoURL); err != nil {
			return nil, fmt.Errorf("scan org logo: %w", err)
		}
		out[domain] = logoURL
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate org logos: %w", err)
	}
	return out, nil
}

// MatchedSFIDs returns every b2b_org SFID on a donation or subscription.
func (r *OrgLogoRepository) MatchedSFIDs(ctx context.Context) ([]string, error) {
	ctx, span := orgLogoTracer.Start(ctx, "db.b2b_org_logos.MatchedSFIDs")
	defer span.End()

	rows, err := r.pool.Query(ctx, `
		SELECT b2b_org_sfid FROM donations WHERE b2b_org_sfid IS NOT NULL
		UNION
		SELECT b2b_org_sfid FROM subscriptions WHERE b2b_org_sfid IS NOT NULL
		ORDER BY 1`)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list matched b2b_org sfids: %w", err)
	}
	defer rows.Close()
	var sfids []string
	for rows.Next() {
		var sfid string
		if err := rows.Scan(&sfid); err != nil {
			return nil, fmt.Errorf("scan matched b2b_org sfid: %w", err)
		}
		sfids = append(sfids, sfid)
	}
	return sfids, rows.Err()
}
