// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

// LogosByOrgKeys maps Ledger org keys to logos. Ledger keys an org by its
// domain, or by its legacy CF org id when the rows carry no domain
// (lfx-self-serve-ops#1647):
//   - a domain gets its matched b2b_org's logo (LogosByDomains), else the logo
//     of the legacy org mapped to that domain by hand (legacy_org_logos.org_domain);
//   - a legacy id gets the logo uploaded to the old org.
//
// Results are keyed by the keys as given.
func (r *OrgLogoRepository) LogosByOrgKeys(ctx context.Context, keys []string) (map[string]string, error) {
	var domains []string
	legacyIDs := map[string][]string{} // canonical uuid -> keys as given
	for _, k := range keys {
		if id, err := uuid.Parse(k); err == nil {
			c := id.String()
			legacyIDs[c] = append(legacyIDs[c], k)
		} else {
			domains = append(domains, k)
		}
	}
	// Look domains up as given and normalized, so a key's case can't hide its
	// b2b_org logo and let the legacy fallback below win instead.
	lookup := make([]string, 0, 2*len(domains))
	for _, d := range domains {
		lookup = append(lookup, d)
		if n := normalizeOrgDomain(d); n != "" && n != d {
			lookup = append(lookup, n)
		}
	}
	b2b, err := r.LogosByDomains(ctx, lookup)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	// Domains without a b2b_org logo may belong to a legacy org.
	legacyDomains := map[string][]string{} // normalized domain -> keys as given
	for _, d := range domains {
		n := normalizeOrgDomain(d)
		if logo, ok := b2b[d]; ok {
			out[d] = logo
		} else if logo, ok := b2b[n]; ok && n != "" {
			out[d] = logo
		} else if n != "" {
			legacyDomains[n] = append(legacyDomains[n], d)
		}
	}
	if len(legacyIDs) == 0 && len(legacyDomains) == 0 {
		return out, nil
	}

	ctx, span := orgLogoTracer.Start(ctx, "db.legacy_org_logos.LogosByOrgKeys")
	defer span.End()
	span.SetAttributes(attribute.Int("legacy_ids.count", len(legacyIDs)), attribute.Int("legacy_domains.count", len(legacyDomains)))

	ids := make([]string, 0, len(legacyIDs))
	for c := range legacyIDs {
		ids = append(ids, c)
	}
	lds := make([]string, 0, len(legacyDomains))
	for d := range legacyDomains {
		lds = append(lds, d)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT legacy_org_id::text, COALESCE(org_domain, ''), logo_url FROM legacy_org_logos
		WHERE legacy_org_id = ANY($1::uuid[]) OR org_domain = ANY($2)`, ids, lds)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("query legacy org logos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, domain, logoURL string
		if err := rows.Scan(&id, &domain, &logoURL); err != nil {
			return nil, fmt.Errorf("scan legacy org logo: %w", err)
		}
		for _, k := range legacyIDs[id] {
			out[k] = logoURL
		}
		if domain != "" {
			for _, k := range legacyDomains[domain] {
				out[k] = logoURL
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy org logos: %w", err)
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

// normalizeOrgDomain lowercases and trims an org domain key; legacy_org_logos
// stores org_domain in this form.
func normalizeOrgDomain(d string) string {
	return strings.ToLower(strings.TrimSpace(d))
}
