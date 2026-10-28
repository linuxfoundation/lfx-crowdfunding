// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/config"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

func orgTotalsLedger(totals map[string]int64) *testLedgerClient {
	return &testLedgerClient{orgTotals: totals}
}

// TestGetInvestingCompanies_RequestsEveryFeaturedKey verifies Ledger is asked
// for each featured org's domain and legacy UUID by key, so no featured org
// can fall outside a top-N cutoff.
func TestGetInvestingCompanies_RequestsEveryFeaturedKey(t *testing.T) {
	ledger := orgTotalsLedger(nil)
	if _, err := newStatsSvc(&testStatisticsRepo{}, ledger).GetInvestingCompanies(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range config.FeaturedOrgs() {
		for _, k := range []string{f.Domain, f.LegacyID} {
			if k != "" && !slices.Contains(ledger.gotOrgKeys, k) {
				t.Errorf("key %q for %s not requested; got %v", k, f.Name, ledger.gotOrgKeys)
			}
		}
	}
}

// TestGetInvestingCompanies_PreservesConfigOrder verifies the result follows
// featured_orgs.json's order, not the ledger's amount-descending order.
func TestGetInvestingCompanies_PreservesConfigOrder(t *testing.T) {
	featured := config.FeaturedOrgs()
	if len(featured) < 2 {
		t.Fatalf("expected at least 2 featured orgs, got %d", len(featured))
	}
	// Give the lower-priority featured org the larger amount, so a naive
	// amount-descending sort would put it first.
	first, second := featured[0], featured[1]
	svc := newStatsSvc(&testStatisticsRepo{}, orgTotalsLedger(map[string]int64{
		second.Domain: 900_00,
		first.Domain:  100_00,
	}))

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 companies, got %d", len(got))
	}
	if got[0].OrgID != first.Domain || got[1].OrgID != second.Domain {
		t.Errorf("expected featured-list order [%s, %s], got [%s, %s]", first.Domain, second.Domain, got[0].OrgID, got[1].OrgID)
	}
}

// TestGetInvestingCompanies_SumsDomainAndLegacyBuckets verifies a featured org
// totals its domain bucket plus its legacy CF UUID bucket (rows not yet
// backfilled), and takes name and logo from config.
func TestGetInvestingCompanies_SumsDomainAndLegacyBuckets(t *testing.T) {
	f := config.FeaturedOrgs()[0]
	svc := newStatsSvc(&testStatisticsRepo{}, orgTotalsLedger(map[string]int64{
		f.Domain:        300_00,
		f.LegacyID:      200_00,
		"unrelated.com": 999_00,
	}))

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 company, got %d: %+v", len(got), got)
	}
	if got[0].AmountCents != 500_00 {
		t.Errorf("expected domain + legacy total 50000, got %d", got[0].AmountCents)
	}
	if got[0].Name != f.Name || got[0].AvatarURL != f.LogoURL {
		t.Errorf("expected config name/logo, got %+v", got[0])
	}
}

// TestGetInvestingCompanies_SkipsFeaturedOrgMissingFromLedger verifies that a
// config entry with no Ledger bucket is skipped rather than rendered at zero.
func TestGetInvestingCompanies_SkipsFeaturedOrgMissingFromLedger(t *testing.T) {
	svc := newStatsSvc(&testStatisticsRepo{}, orgTotalsLedger(nil))

	got, err := svc.GetInvestingCompanies(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 companies (ledger has none), got %d: %+v", len(got), got)
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
