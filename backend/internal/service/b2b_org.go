// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"log/slog"
)

// B2BOrgResolver matches an org domain to the b2b_org (Salesforce Account SFID)
// that owns it. "" with a nil error means no single match.
type B2BOrgResolver interface {
	LookupB2BOrgByWebsite(ctx context.Context, domain string) (string, error)
}

// resolveB2BOrg looks up domain in the background, after the row is saved, and
// hands a match to store. Checkout never waits on it: a failed lookup or store
// only logs and leaves the SFID empty for #412 to retry.
func resolveB2BOrg(ctx context.Context, r B2BOrgResolver, kind, id, domain string, store func(context.Context, string) error) {
	if r == nil || domain == "" {
		return
	}
	// Detached so the lookup isn't cancelled when the request that saved the row completes.
	go lookupAndStoreB2BOrg(context.WithoutCancel(ctx), r, kind, id, domain, store)
}

func lookupAndStoreB2BOrg(ctx context.Context, r B2BOrgResolver, kind, id, domain string, store func(context.Context, string) error) {
	sfid, err := r.LookupB2BOrgByWebsite(ctx, domain)
	if err != nil {
		slog.WarnContext(ctx, "b2b_org lookup failed", "kind", kind, "id", id, "org_domain", domain, "error", err)
		return
	}
	if sfid == "" {
		slog.InfoContext(ctx, "b2b_org lookup: no match", "kind", kind, "id", id, "org_domain", domain)
		return
	}
	if err := store(ctx, sfid); err != nil {
		slog.WarnContext(ctx, "b2b_org store failed", "kind", kind, "id", id, "b2b_org_sfid", sfid, "error", err)
	}
}
