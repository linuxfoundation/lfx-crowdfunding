// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// B2BOrgResolver matches an org domain to the b2b_org (Salesforce Account SFID
// and logo) that owns it. A zero match with a nil error means no single match.
type B2BOrgResolver interface {
	LookupB2BOrgByWebsite(ctx context.Context, domain string) (models.B2BOrgMatch, error)
}

// resolveB2BOrg looks up domain in the background, after the row is saved, and
// hands a match to store. The match's logo goes to logos (#472), so donor lists
// show it without calling member-service. Checkout never waits on it: a failed
// lookup or store only logs and leaves the SFID empty for #412 to retry.
func resolveB2BOrg(ctx context.Context, r B2BOrgResolver, logos domain.OrgLogoRepository, kind, id, domain string, store func(context.Context, string) error) {
	if r == nil || domain == "" {
		return
	}
	// Detached so the lookup isn't cancelled when the request that saved the row completes.
	go lookupAndStoreB2BOrg(context.WithoutCancel(ctx), r, logos, kind, id, domain, store)
}

func lookupAndStoreB2BOrg(ctx context.Context, r B2BOrgResolver, logos domain.OrgLogoRepository, kind, id, domain string, store func(context.Context, string) error) {
	match, err := r.LookupB2BOrgByWebsite(ctx, domain)
	if err != nil {
		slog.WarnContext(ctx, "b2b_org lookup failed", "kind", kind, "id", id, "org_domain", domain, "error", err)
		return
	}
	if match.SFID == "" {
		slog.InfoContext(ctx, "b2b_org lookup: no match", "kind", kind, "id", id, "org_domain", domain)
		return
	}
	if err := store(ctx, match.SFID); err != nil {
		slog.WarnContext(ctx, "b2b_org store failed", "kind", kind, "id", id, "b2b_org_sfid", match.SFID, "error", err)
		return
	}
	if logos == nil {
		return
	}
	// The reply is the org's current logo, so an empty one clears a stale row.
	if err := logos.SetOrgLogo(ctx, match.SFID, match.LogoURL); err != nil {
		slog.WarnContext(ctx, "b2b_org logo store failed", "kind", kind, "id", id, "b2b_org_sfid", match.SFID, "error", err)
	}
}
