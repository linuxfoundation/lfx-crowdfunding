// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/config"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// TestGetInvestingCompanies_PreservesConfigOrder verifies the result follows
// featured_orgs.json's order, not the ledger's amount-descending order.
func TestGetInvestingCompanies_PreservesConfigOrder(t *testing.T) {
	featured := config.FeaturedOrgIDs()
	if len(featured) < 2 {
		t.Fatalf("expected at least 2 featured org IDs, got %d", len(featured))
	}
	// Give the lower-priority featured org the larger amount, so a naive
	// amount-descending sort would put it first.
	first, second := featured[0], featured[1]
	ledger := &testLedgerClient{
		orgDonations: []clients.LedgerOrgDonation{
			{OrgID: second, Name: "Second", AmountInCents: 900_00},
			{OrgID: first, Name: "First", AmountInCents: 100_00},
		},
	}
	svc := newStatsSvc(&testStatisticsRepo{}, ledger)

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

// TestGetInvestingCompanies_SkipsFeaturedOrgMissingFromLedger verifies that a
// config entry the ledger doesn't return an amount for is skipped rather than
// rendered at zero.
func TestGetInvestingCompanies_SkipsFeaturedOrgMissingFromLedger(t *testing.T) {
	featured := config.FeaturedOrgIDs()
	if len(featured) == 0 {
		t.Fatal("expected at least 1 featured org ID")
	}
	svc := newStatsSvc(&testStatisticsRepo{}, &testLedgerClient{})

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 companies (ledger has none), got %d: %+v", len(got), got)
	}
}

// TestGetInvestingCompanies_CFDatabaseOverridesLedgerNameAndAvatar verifies
// that name/avatar from the CF database take priority over the ledger's
// values when the org is known locally, since CF Postgres is fresher.
func TestGetInvestingCompanies_CFDatabaseOverridesLedgerNameAndAvatar(t *testing.T) {
	featured := config.FeaturedOrgIDs()
	if len(featured) == 0 {
		t.Fatal("expected at least 1 featured org ID")
	}
	id := featured[0]
	repo := &testStatisticsRepo{
		orgs: map[string]models.Organization{
			id: {ID: id, Name: "Fresh Name", AvatarURL: "https://example.com/fresh.png"},
		},
	}
	ledger := &testLedgerClient{
		orgDonations: []clients.LedgerOrgDonation{
			{OrgID: id, Name: "Stale Name", AvatarURL: "https://example.com/stale.png", AmountInCents: 500_00},
		},
	}
	svc := newStatsSvc(repo, ledger)

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 company, got %d", len(got))
	}
	if got[0].Name != "Fresh Name" || got[0].AvatarURL != "https://example.com/fresh.png" {
		t.Errorf("expected CF DB values to win, got %+v", got[0])
	}
	if got[0].AmountCents != 500_00 {
		t.Errorf("expected ledger amount to be used, got %d", got[0].AmountCents)
	}
}

// TestGetInvestingCompanies_GeneratesAvatarWhenMissing verifies that an org
// with no avatar_url from either source gets a deterministic generated
// avatar instead of rendering with a blank logo.
func TestGetInvestingCompanies_GeneratesAvatarWhenMissing(t *testing.T) {
	featured := config.FeaturedOrgIDs()
	if len(featured) == 0 {
		t.Fatal("expected at least 1 featured org ID")
	}
	ledger := &testLedgerClient{
		orgDonations: []clients.LedgerOrgDonation{
			{OrgID: featured[0], Name: "No Logo Org", AmountInCents: 500_00},
		},
	}
	svc := newStatsSvc(&testStatisticsRepo{}, ledger)

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

// TestGetInvestingCompanies_UpstreamUnavailableReturnsEmpty verifies the
// endpoint degrades to an empty list (not a 500) when the ledger is down,
// matching GetPlatformDetails' degradation behavior.
func TestGetInvestingCompanies_UpstreamUnavailableReturnsEmpty(t *testing.T) {
	ledger := &testLedgerClient{err: domain.ErrUpstreamUnavailable}
	svc := newStatsSvc(&testStatisticsRepo{}, ledger)

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %d: %+v", len(got), got)
	}
}

// TestGetInvestingCompanies_LedgerErrorPropagates verifies a non-upstream
// ledger error is surfaced rather than swallowed.
func TestGetInvestingCompanies_LedgerErrorPropagates(t *testing.T) {
	ledger := &testLedgerClient{err: errors.New("ledger exploded")}
	svc := newStatsSvc(&testStatisticsRepo{}, ledger)

	_, err := svc.GetInvestingCompanies(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
