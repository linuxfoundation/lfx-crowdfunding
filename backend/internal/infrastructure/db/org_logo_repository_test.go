// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"
)

// matchRow sets a donation's or subscription's org domain and b2b_org SFID.
func matchRow(t *testing.T, ctx context.Context, table, id, domain, sfid string) { //nolint:revive // t first is Go test convention
	t.Helper()
	if _, err := testPool.Exec(ctx, "UPDATE crowdfunding."+table+" SET org_name = $2, org_domain = $2, b2b_org_sfid = NULLIF($3, '') WHERE id = $1", id, domain, sfid); err != nil { //nolint:gosec // table is a test-internal constant
		t.Fatalf("match %s %s: %v", table, id, err)
	}
}

func TestOrgLogoRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.legacy_org_logos", "crowdfunding.b2b_org_logos", "crowdfunding.donations", "crowdfunding.subscriptions", "crowdfunding.initiatives", "crowdfunding.users")
	repo := NewOrgLogoRepository(testPool)

	user := seedUser(t, ctx, "logo-donor")
	init := seedInitiative(t, ctx, user.ID, "Logo Initiative", "logo-initiative")

	// acme.com: matched on a donation and a subscription (same SFID).
	matchRow(t, ctx, "donations", seedDonation(t, ctx, user.ID, init.ID).ID, "acme.com", "001ACME")
	matchRow(t, ctx, "subscriptions", seedSubscription(t, ctx, user.ID, init.ID, "active", "sub_acme").ID, "acme.com", "001ACME")
	// split.org: rows matched to two different orgs, so neither logo is shown.
	matchRow(t, ctx, "donations", seedDonation(t, ctx, user.ID, init.ID).ID, "split.org", "001SPLITA")
	matchRow(t, ctx, "donations", seedDonation(t, ctx, user.ID, init.ID).ID, "split.org", "001SPLITB")
	// nologo.org: matched, but its org has no logo row.
	matchRow(t, ctx, "subscriptions", seedSubscription(t, ctx, user.ID, init.ID, "active", "sub_nologo").ID, "nologo.org", "001NOLOGO")
	// unmatched.org: no SFID at all.
	matchRow(t, ctx, "donations", seedDonation(t, ctx, user.ID, init.ID).ID, "unmatched.org", "")

	for sfid, logo := range map[string]string{
		"001ACME":   "https://cdn.example/acme-old.png",
		"001SPLITA": "https://cdn.example/a.png",
		"001SPLITB": "https://cdn.example/b.png",
		"001GONE":   "https://cdn.example/gone.png",
	} {
		if err := repo.SetOrgLogo(ctx, sfid, logo); err != nil {
			t.Fatalf("SetOrgLogo %s: %v", sfid, err)
		}
	}
	// An upsert replaces the logo; an empty one removes the row.
	if err := repo.SetOrgLogo(ctx, "001ACME", "https://cdn.example/acme.png"); err != nil {
		t.Fatalf("SetOrgLogo update: %v", err)
	}
	if err := repo.SetOrgLogo(ctx, "001GONE", ""); err != nil {
		t.Fatalf("SetOrgLogo remove: %v", err)
	}
	var gone int
	if err := testPool.QueryRow(ctx, "SELECT COUNT(*) FROM crowdfunding.b2b_org_logos WHERE b2b_org_sfid = '001GONE'").Scan(&gone); err != nil || gone != 0 {
		t.Errorf("removed logo rows = %d (err %v), want 0", gone, err)
	}

	logos, err := repo.LogosByDomains(ctx, []string{"acme.com", "split.org", "nologo.org", "unmatched.org", "unknown.io"})
	if err != nil {
		t.Fatalf("LogosByDomains: %v", err)
	}
	if len(logos) != 1 || logos["acme.com"] != "https://cdn.example/acme.png" {
		t.Errorf("logos = %v, want only acme.com with the updated logo", logos)
	}

	if empty, err := repo.LogosByDomains(ctx, nil); err != nil || len(empty) != 0 {
		t.Errorf("no domains = %v (err %v), want empty", empty, err)
	}

	// Ledger keys domainless legacy orgs by their old CF org UUID
	// (lfx-self-serve-ops#1647); those logos come from legacy_org_logos.
	const legacyID = "5213987d-928d-4f75-831c-ee41a062832b"
	if _, err := testPool.Exec(ctx, `INSERT INTO crowdfunding.legacy_org_logos (legacy_org_id, name, logo_url)
		VALUES ($1, 'Legacy BV', 'https://uploads.example/legacy')`, legacyID); err != nil {
		t.Fatalf("seed legacy logo: %v", err)
	}
	upperID := strings.ToUpper(legacyID)
	keyed, err := repo.LogosByOrgKeys(ctx, []string{"acme.com", "unmatched.org", legacyID, upperID, "00000000-0000-4000-8000-000000000000"})
	if err != nil {
		t.Fatalf("LogosByOrgKeys: %v", err)
	}
	want := map[string]string{
		"acme.com": "https://cdn.example/acme.png",
		legacyID:   "https://uploads.example/legacy",
		upperID:    "https://uploads.example/legacy",
	}
	if !maps.Equal(keyed, want) {
		t.Errorf("logos by org key = %v, want %v", keyed, want)
	}
	if empty, err := repo.LogosByOrgKeys(ctx, nil); err != nil || len(empty) != 0 {
		t.Errorf("no keys = %v (err %v), want empty", empty, err)
	}

	sfids, err := repo.MatchedSFIDs(ctx)
	if err != nil {
		t.Fatalf("MatchedSFIDs: %v", err)
	}
	if want := []string{"001ACME", "001NOLOGO", "001SPLITA", "001SPLITB"}; !slices.Equal(sfids, want) {
		t.Errorf("MatchedSFIDs = %v, want %v", sfids, want)
	}
}
