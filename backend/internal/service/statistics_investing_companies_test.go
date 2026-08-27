// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/config"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// TestGetInvestingCompanies_FeaturedIDsPresent verifies that when the database
// has rows for the curated featured org IDs, they are returned in the order
// featured_orgs.json defines rather than the amount-descending order the
// repository returns.
func TestGetInvestingCompanies_FeaturedIDsPresent(t *testing.T) {
	featured := config.FeaturedOrgIDs()
	if len(featured) < 2 {
		t.Fatalf("expected at least 2 featured org IDs, got %d", len(featured))
	}
	// Intentionally give the lower-priority featured org the larger amount, so
	// a naive amount-descending sort would put it first.
	first, second := featured[0], featured[1]
	repo := &testStatisticsRepo{contributions: []models.OrgContribution{
		{OrgID: second, Name: "Second", AvatarURL: "https://example.com/second.png", AmountCents: 900_00},
		{OrgID: first, Name: "First", AvatarURL: "https://example.com/first.png", AmountCents: 100_00},
	}}
	svc := newStatsSvc(repo, &testLedgerClient{})

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 companies, got %d", len(got))
	}
	if got[0].OrgID != first || got[1].OrgID != second {
		t.Errorf("expected featured-list order [%s, %s], got [%s, %s]", first, second, got[0].OrgID, got[1].OrgID)
	}
}

// TestGetInvestingCompanies_FallsBackWhenFeaturedIDsMissing verifies that when
// none of the curated featured org IDs exist in this database (local, dev,
// staging), the service falls back to the top fallbackFeaturedOrgLimit
// organizations by contribution amount — no environment flag required.
func TestGetInvestingCompanies_FallsBackWhenFeaturedIDsMissing(t *testing.T) {
	var contributions []models.OrgContribution
	for i := 0; i < fallbackFeaturedOrgLimit+5; i++ {
		contributions = append(contributions, models.OrgContribution{
			OrgID:       string(rune('a' + i)),
			Name:        "Org",
			AvatarURL:   "https://example.com/logo.png",
			AmountCents: int64(fallbackFeaturedOrgLimit+5-i) * 100,
		})
	}
	repo := &testStatisticsRepo{contributions: contributions}
	svc := newStatsSvc(repo, &testLedgerClient{})

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != fallbackFeaturedOrgLimit {
		t.Fatalf("expected exactly %d fallback companies, got %d", fallbackFeaturedOrgLimit, len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].AmountCents > got[i-1].AmountCents {
			t.Errorf("expected amount-descending order, got %d after %d", got[i].AmountCents, got[i-1].AmountCents)
		}
	}
}

// TestGetInvestingCompanies_GeneratesAvatarWhenMissing verifies that an org
// with no avatar_url in the database gets a deterministic generated avatar
// instead of rendering with a blank logo.
func TestGetInvestingCompanies_GeneratesAvatarWhenMissing(t *testing.T) {
	featured := config.FeaturedOrgIDs()
	if len(featured) == 0 {
		t.Fatal("expected at least 1 featured org ID")
	}
	repo := &testStatisticsRepo{contributions: []models.OrgContribution{
		{OrgID: featured[0], Name: "No Logo Org", AvatarURL: "", AmountCents: 500_00},
	}}
	svc := newStatsSvc(repo, &testLedgerClient{})

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 company, got %d", len(got))
	}
	if got[0].AvatarURL == "" {
		t.Error("expected a generated avatar URL, got empty string")
	}
}
