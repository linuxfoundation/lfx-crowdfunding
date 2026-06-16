// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// ─── Mock implementations ────────────────────────────────────────────────────

type mockLedgerSource struct {
	balances []models.LedgerRawBalance
	err      error
}

func (m *mockLedgerSource) GetAllBalances(_ context.Context) ([]models.LedgerRawBalance, error) {
	return m.balances, m.err
}

type mockLedgerStatsRepo struct {
	initiativeIDs []string
	listErr       error
	orgMap        map[string]models.Organization
	orgErr        error
	userMap       map[string]models.User
	userErr       error
	upsertErr     error
	// captured for assertion
	capturedUpserts []models.LedgerStats
}

func (m *mockLedgerStatsRepo) ListActiveSyncIDs(_ context.Context) ([]string, error) {
	return m.initiativeIDs, m.listErr
}

func (m *mockLedgerStatsRepo) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	if m.orgMap == nil {
		return map[string]models.Organization{}, m.orgErr
	}
	return m.orgMap, m.orgErr
}

func (m *mockLedgerStatsRepo) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	if m.userMap == nil {
		return map[string]models.User{}, m.userErr
	}
	return m.userMap, m.userErr
}

func (m *mockLedgerStatsRepo) BulkUpsertLedgerStats(_ context.Context, stats []models.LedgerStats) (int, error) {
	m.capturedUpserts = stats
	if m.upsertErr != nil {
		return 0, m.upsertErr
	}
	return len(stats), nil
}

// ─── buildBalanceIndex ───────────────────────────────────────────────────────

func TestBuildBalanceIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []models.LedgerRawBalance
		wantKeys []string
	}{
		{
			name:     "empty slice returns empty map",
			input:    nil,
			wantKeys: nil,
		},
		{
			name: "single entry indexed by projectID",
			input: []models.LedgerRawBalance{
				{ProjectID: "proj-1", TotalCredit: 100},
			},
			wantKeys: []string{"proj-1"},
		},
		{
			name: "empty projectID is dropped",
			input: []models.LedgerRawBalance{
				{ProjectID: "", TotalCredit: 999},
				{ProjectID: "proj-2", TotalCredit: 50},
			},
			wantKeys: []string{"proj-2"},
		},
		{
			name: "duplicate projectID — last writer wins",
			input: []models.LedgerRawBalance{
				{ProjectID: "proj-1", TotalCredit: 100},
				{ProjectID: "proj-1", TotalCredit: 200},
			},
			wantKeys: []string{"proj-1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			idx := buildBalanceIndex(tc.input)

			if len(idx) != len(tc.wantKeys) {
				t.Fatalf("got %d keys, want %d", len(idx), len(tc.wantKeys))
			}
			for _, k := range tc.wantKeys {
				if _, ok := idx[k]; !ok {
					t.Errorf("expected key %q not found in index", k)
				}
			}
		})
	}
}

// ─── collectSponsorIDs ───────────────────────────────────────────────────────

func TestCollectSponsorIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		balances      []models.LedgerRawBalance
		wantOrgCount  int
		wantUserCount int
	}{
		{
			name:          "empty balances returns empty slices",
			balances:      nil,
			wantOrgCount:  0,
			wantUserCount: 0,
		},
		{
			name: "single balance with sponsors",
			balances: []models.LedgerRawBalance{
				{Sponsors: models.LedgerRawSponsors{
					Orgs:        []models.LedgerRawSponsor{{ID: "org-1"}, {ID: "org-2"}},
					Individuals: []models.LedgerRawSponsor{{ID: "auth0|u1"}},
				}},
			},
			wantOrgCount:  2,
			wantUserCount: 1,
		},
		{
			name: "empty strings are excluded",
			balances: []models.LedgerRawBalance{
				{Sponsors: models.LedgerRawSponsors{
					Orgs:        []models.LedgerRawSponsor{{ID: ""}, {ID: "org-1"}},
					Individuals: []models.LedgerRawSponsor{{ID: ""}},
				}},
			},
			wantOrgCount:  1,
			wantUserCount: 0,
		},
		{
			name: "duplicates across balances are deduplicated",
			balances: []models.LedgerRawBalance{
				{Sponsors: models.LedgerRawSponsors{
					Orgs:        []models.LedgerRawSponsor{{ID: "org-1"}},
					Individuals: []models.LedgerRawSponsor{{ID: "auth0|u1"}},
				}},
				{Sponsors: models.LedgerRawSponsors{
					Orgs:        []models.LedgerRawSponsor{{ID: "org-1"}},
					Individuals: []models.LedgerRawSponsor{{ID: "auth0|u1"}},
				}},
			},
			wantOrgCount:  1,
			wantUserCount: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orgIDs, userIDs := collectSponsorIDs(tc.balances)

			if len(orgIDs) != tc.wantOrgCount {
				t.Errorf("org count: got %d, want %d", len(orgIDs), tc.wantOrgCount)
			}
			if len(userIDs) != tc.wantUserCount {
				t.Errorf("user count: got %d, want %d", len(userIDs), tc.wantUserCount)
			}
		})
	}
}

// ─── mapBalance ──────────────────────────────────────────────────────────────

func TestMapBalance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		raw              models.LedgerRawBalance
		wantTotalRaised  int64
		wantTotalDebited int64
		wantTotalBalance int64
		wantAvailable    int64
		wantFeeBalance   int64
		wantSupporters   int
	}{
		{
			name: "negative debit and fee are made positive",
			raw: models.LedgerRawBalance{
				ProjectID:        "p1",
				TotalCredit:      1000,
				TotalDebit:       -300,
				TotalBalance:     700,
				AvailableBalance: 650,
				FeeBalance:       -50,
				Backers:          5,
			},
			wantTotalRaised:  1000,
			wantTotalDebited: 300,
			wantTotalBalance: 700,
			wantAvailable:    650,
			wantFeeBalance:   50,
			wantSupporters:   5,
		},
		{
			name: "already positive debit and fee are unchanged",
			raw: models.LedgerRawBalance{
				TotalDebit: 100,
				FeeBalance: 20,
			},
			wantTotalDebited: 100,
			wantFeeBalance:   20,
		},
		{
			name: "zero fields remain zero",
			raw:  models.LedgerRawBalance{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapBalance(tc.raw, nil, nil)

			if got.TotalRaisedCents != tc.wantTotalRaised {
				t.Errorf("TotalRaisedCents: got %d, want %d", got.TotalRaisedCents, tc.wantTotalRaised)
			}
			if got.TotalDebitedCents != tc.wantTotalDebited {
				t.Errorf("TotalDebitedCents: got %d, want %d", got.TotalDebitedCents, tc.wantTotalDebited)
			}
			if got.TotalBalanceCents != tc.wantTotalBalance {
				t.Errorf("TotalBalanceCents: got %d, want %d", got.TotalBalanceCents, tc.wantTotalBalance)
			}
			if got.AvailableBalanceCents != tc.wantAvailable {
				t.Errorf("AvailableBalanceCents: got %d, want %d", got.AvailableBalanceCents, tc.wantAvailable)
			}
			if got.FeeBalanceCents != tc.wantFeeBalance {
				t.Errorf("FeeBalanceCents: got %d, want %d", got.FeeBalanceCents, tc.wantFeeBalance)
			}
			if got.Supporters != tc.wantSupporters {
				t.Errorf("Supporters: got %d, want %d", got.Supporters, tc.wantSupporters)
			}
		})
	}
}

// ─── enrichSponsors ──────────────────────────────────────────────────────────

func TestEnrichSponsors(t *testing.T) {
	t.Parallel()

	orgMap := map[string]models.Organization{
		"org-1": {ID: "org-1", Name: "Linux Foundation", AvatarURL: "http://lf.org/logo.png"},
	}
	userMap := map[string]models.User{
		"auth0|u1": {ID: "auth0|u1", Name: "Jane Doe", AvatarURL: "http://avatar.io/jane.png"},
	}

	tests := []struct {
		name              string
		raw               models.LedgerRawSponsors
		wantOrgNames      []string
		wantOrgAvatarURLs []string
		wantUserNames     []string
	}{
		{
			name: "empty sponsors returns empty lists",
			raw:  models.LedgerRawSponsors{},
		},
		{
			name: "known org and user are enriched",
			raw: models.LedgerRawSponsors{
				Orgs:        []models.LedgerRawSponsor{{ID: "org-1", Total: 5000}},
				Individuals: []models.LedgerRawSponsor{{ID: "auth0|u1", Total: 200}},
			},
			wantOrgNames:      []string{"Linux Foundation"},
			wantOrgAvatarURLs: []string{"http://lf.org/logo.png"},
			wantUserNames:     []string{"Jane Doe"},
		},
		{
			name: "unknown IDs are included with empty name",
			raw: models.LedgerRawSponsors{
				Orgs:        []models.LedgerRawSponsor{{ID: "org-unknown"}},
				Individuals: []models.LedgerRawSponsor{{ID: "auth0|unknown"}},
			},
			wantOrgNames:  []string{""},
			wantUserNames: []string{""},
		},
		{
			name: "empty ID strings are dropped",
			raw: models.LedgerRawSponsors{
				Orgs:        []models.LedgerRawSponsor{{ID: ""}, {ID: "org-1"}},
				Individuals: []models.LedgerRawSponsor{{ID: ""}},
			},
			wantOrgNames: []string{"Linux Foundation"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := enrichSponsors(tc.raw, orgMap, userMap)

			if len(got.Orgs) != len(tc.wantOrgNames) {
				t.Fatalf("orgs count: got %d, want %d", len(got.Orgs), len(tc.wantOrgNames))
			}
			for i, name := range tc.wantOrgNames {
				if got.Orgs[i].Name != name {
					t.Errorf("org[%d].Name: got %q, want %q", i, got.Orgs[i].Name, name)
				}
			}
			if len(tc.wantOrgAvatarURLs) > 0 {
				for i, url := range tc.wantOrgAvatarURLs {
					if got.Orgs[i].AvatarURL != url {
						t.Errorf("org[%d].AvatarURL: got %q, want %q", i, got.Orgs[i].AvatarURL, url)
					}
				}
			}
			if len(got.Individuals) != len(tc.wantUserNames) {
				t.Fatalf("individuals count: got %d, want %d", len(got.Individuals), len(tc.wantUserNames))
			}
			for i, name := range tc.wantUserNames {
				if got.Individuals[i].Name != name {
					t.Errorf("individual[%d].Name: got %q, want %q", i, got.Individuals[i].Name, name)
				}
			}
		})
	}
}

// ─── Syncer.Run ──────────────────────────────────────────────────────────────

func TestSyncer_Run_noActiveInitiatives(t *testing.T) {
	t.Parallel()

	repo := &mockLedgerStatsRepo{initiativeIDs: []string{}}
	ledger := &mockLedgerSource{}

	s := newSyncer(repo, ledger, discardLogger())
	result, err := s.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.total != 0 || result.matched != 0 || result.upserted != 0 || result.skipped != 0 {
		t.Errorf("expected all-zero result, got %+v", result)
	}
	if len(repo.capturedUpserts) != 0 {
		t.Errorf("expected no upserts, got %d", len(repo.capturedUpserts))
	}
}

func TestSyncer_Run_skipsInitiativeWithNoLedgerEntry(t *testing.T) {
	t.Parallel()

	repo := &mockLedgerStatsRepo{
		initiativeIDs: []string{"init-1", "init-2"},
	}
	ledger := &mockLedgerSource{
		balances: []models.LedgerRawBalance{
			{ProjectID: "init-1", TotalCredit: 500},
			// init-2 is absent from Ledger
		},
	}

	s := newSyncer(repo, ledger, discardLogger())
	result, err := s.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.total != 2 {
		t.Errorf("total: got %d, want 2", result.total)
	}
	if result.matched != 1 {
		t.Errorf("matched: got %d, want 1", result.matched)
	}
	if result.upserted != 1 {
		t.Errorf("upserted: got %d, want 1", result.upserted)
	}
	if result.skipped != 1 {
		t.Errorf("skipped: got %d, want 1", result.skipped)
	}
}

func TestSyncer_Run_upsertsWithEnrichedSponsors(t *testing.T) {
	t.Parallel()

	repo := &mockLedgerStatsRepo{
		initiativeIDs: []string{"init-1"},
		orgMap: map[string]models.Organization{
			"org-1": {ID: "org-1", Name: "CNCF", AvatarURL: "http://cncf.io/logo.png"},
		},
		userMap: map[string]models.User{
			"auth0|user1": {ID: "auth0|user1", Name: "Alice", AvatarURL: "http://img.io/alice.png"},
		},
	}
	ledger := &mockLedgerSource{
		balances: []models.LedgerRawBalance{
			{
				ProjectID:        "init-1",
				TotalCredit:      1000,
				TotalDebit:       -200,
				TotalBalance:     800,
				AvailableBalance: 780,
				FeeBalance:       -20,
				Backers:          3,
				Sponsors: models.LedgerRawSponsors{
					Orgs:        []models.LedgerRawSponsor{{ID: "org-1", Total: 600000}},
					Individuals: []models.LedgerRawSponsor{{ID: "auth0|user1", Total: 780}},
				},
			},
		},
	}

	s := newSyncer(repo, ledger, discardLogger())
	result, err := s.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.upserted != 1 {
		t.Fatalf("upserted: got %d, want 1", result.upserted)
	}

	got := repo.capturedUpserts[0]

	if got.InitiativeID != "init-1" {
		t.Errorf("InitiativeID: got %q, want init-1", got.InitiativeID)
	}
	if got.TotalRaisedCents != 1000 {
		t.Errorf("TotalRaisedCents: got %d, want 1000", got.TotalRaisedCents)
	}
	if got.TotalDebitedCents != 200 {
		t.Errorf("TotalDebitedCents: got %d, want 200 (ABS)", got.TotalDebitedCents)
	}
	if got.FeeBalanceCents != 20 {
		t.Errorf("FeeBalanceCents: got %d, want 20 (ABS)", got.FeeBalanceCents)
	}
	if got.Supporters != 3 {
		t.Errorf("Supporters: got %d, want 3", got.Supporters)
	}

	if len(got.Sponsors.Orgs) != 1 {
		t.Fatalf("sponsors.orgs: got %d, want 1", len(got.Sponsors.Orgs))
	}
	if got.Sponsors.Orgs[0].Name != "CNCF" {
		t.Errorf("sponsors.orgs[0].Name: got %q, want CNCF", got.Sponsors.Orgs[0].Name)
	}
	if got.Sponsors.Orgs[0].AvatarURL != "http://cncf.io/logo.png" {
		t.Errorf("sponsors.orgs[0].AvatarURL: got %q", got.Sponsors.Orgs[0].AvatarURL)
	}

	if len(got.Sponsors.Individuals) != 1 {
		t.Fatalf("sponsors.individuals: got %d, want 1", len(got.Sponsors.Individuals))
	}
	if got.Sponsors.Individuals[0].Name != "Alice" {
		t.Errorf("sponsors.individuals[0].Name: got %q, want Alice", got.Sponsors.Individuals[0].Name)
	}
}

func TestSyncer_Run_propagatesLedgerError(t *testing.T) {
	t.Parallel()

	repo := &mockLedgerStatsRepo{
		initiativeIDs: []string{"init-1"},
	}
	ledger := &mockLedgerSource{err: fmt.Errorf("ledger service unavailable")}

	s := newSyncer(repo, ledger, discardLogger())
	_, err := s.Run(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSyncer_Run_propagatesListError(t *testing.T) {
	t.Parallel()

	repo := &mockLedgerStatsRepo{listErr: errors.New("db connection lost")}
	ledger := &mockLedgerSource{}

	s := newSyncer(repo, ledger, discardLogger())
	_, err := s.Run(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSyncer_Run_propagatesUpsertError(t *testing.T) {
	t.Parallel()

	repo := &mockLedgerStatsRepo{
		initiativeIDs: []string{"init-1"},
		upsertErr:     errors.New("constraint violation"),
	}
	ledger := &mockLedgerSource{
		balances: []models.LedgerRawBalance{{ProjectID: "init-1"}},
	}

	s := newSyncer(repo, ledger, discardLogger())
	_, err := s.Run(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestSyncer_Run_multipleInitiativesAllMatched verifies all three counters
// (total, matched, upserted) when every initiative has a Ledger entry.
func TestSyncer_Run_multipleInitiativesAllMatched(t *testing.T) {
	t.Parallel()

	ids := []string{"a", "b", "c"}
	balances := make([]models.LedgerRawBalance, len(ids))
	for i, id := range ids {
		balances[i] = models.LedgerRawBalance{ProjectID: id, TotalCredit: int64(i+1) * 100}
	}

	repo := &mockLedgerStatsRepo{initiativeIDs: ids}
	ledger := &mockLedgerSource{balances: balances}

	s := newSyncer(repo, ledger, discardLogger())
	result, err := s.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.total != 3 {
		t.Errorf("total: got %d, want 3", result.total)
	}
	if result.matched != 3 {
		t.Errorf("matched: got %d, want 3", result.matched)
	}
	if result.upserted != 3 {
		t.Errorf("upserted: got %d, want 3", result.upserted)
	}
	if result.skipped != 0 {
		t.Errorf("skipped: got %d, want 0", result.skipped)
	}

	// Verify mapped TotalRaisedCents for each captured upsert.
	captured := make(map[string]int64, len(repo.capturedUpserts))
	for _, u := range repo.capturedUpserts {
		captured[u.InitiativeID] = u.TotalRaisedCents
	}
	for i, id := range ids {
		want := int64(i+1) * 100
		if captured[id] != want {
			t.Errorf("initiative %q TotalRaisedCents: got %d, want %d", id, captured[id], want)
		}
	}
}

// TestSyncer_Run_sponsorIDsCollectedForBulkLookup verifies that all unique
// sponsor IDs across multiple balances are passed to the repo lookups.
func TestSyncer_Run_sponsorIDsCollectedForBulkLookup(t *testing.T) {
	t.Parallel()

	// Track which org/user IDs were requested.
	var gotOrgIDs, gotUserIDs []string

	repo := &mockCapturingRepo{
		base: &mockLedgerStatsRepo{
			initiativeIDs: []string{"i1", "i2"},
		},
		captureOrgIDs:  &gotOrgIDs,
		captureUserIDs: &gotUserIDs,
	}
	ledger := &mockLedgerSource{
		balances: []models.LedgerRawBalance{
			{ProjectID: "i1", Sponsors: models.LedgerRawSponsors{
				Orgs: []models.LedgerRawSponsor{{ID: "org-a"}}, Individuals: []models.LedgerRawSponsor{{ID: "u1"}},
			}},
			{ProjectID: "i2", Sponsors: models.LedgerRawSponsors{
				Orgs: []models.LedgerRawSponsor{{ID: "org-b"}, {ID: "org-a"}}, Individuals: []models.LedgerRawSponsor{{ID: "u2"}, {ID: "u1"}},
			}},
		},
	}

	s := newSyncer(repo, ledger, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(gotOrgIDs)
	sort.Strings(gotUserIDs)

	wantOrgs := []string{"org-a", "org-b"}
	wantUsers := []string{"u1", "u2"}

	if fmt.Sprint(gotOrgIDs) != fmt.Sprint(wantOrgs) {
		t.Errorf("org IDs: got %v, want %v", gotOrgIDs, wantOrgs)
	}
	if fmt.Sprint(gotUserIDs) != fmt.Sprint(wantUsers) {
		t.Errorf("user IDs: got %v, want %v", gotUserIDs, wantUsers)
	}
}

// mockCapturingRepo wraps mockLedgerStatsRepo and captures the IDs passed to
// the org/user lookup calls for assertion in tests.
type mockCapturingRepo struct {
	base           *mockLedgerStatsRepo
	captureOrgIDs  *[]string
	captureUserIDs *[]string
}

func (r *mockCapturingRepo) ListActiveSyncIDs(ctx context.Context) ([]string, error) {
	return r.base.ListActiveSyncIDs(ctx)
}

func (r *mockCapturingRepo) GetOwnerInfoBySlug(_ context.Context, _ string) (models.OwnerInfo, error) {
	return models.OwnerInfo{}, nil
}
func (r *mockCapturingRepo) ListPublished(_ context.Context) ([]models.InitiativeSummary, error) {
	return nil, nil
}
func (r *mockCapturingRepo) GetOrganizationsByIDs(ctx context.Context, ids []string) (map[string]models.Organization, error) {
	*r.captureOrgIDs = append(*r.captureOrgIDs, ids...)
	sort.Strings(*r.captureOrgIDs)
	// deduplicate
	deduped := make([]string, 0, len(*r.captureOrgIDs))
	seen := map[string]struct{}{}
	for _, id := range *r.captureOrgIDs {
		if _, ok := seen[id]; !ok {
			deduped = append(deduped, id)
			seen[id] = struct{}{}
		}
	}
	*r.captureOrgIDs = deduped
	return r.base.GetOrganizationsByIDs(ctx, ids)
}

func (r *mockCapturingRepo) GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]models.User, error) {
	*r.captureUserIDs = append(*r.captureUserIDs, userIDs...)
	sort.Strings(*r.captureUserIDs)
	// deduplicate
	deduped := make([]string, 0, len(*r.captureUserIDs))
	seen := map[string]struct{}{}
	for _, id := range *r.captureUserIDs {
		if _, ok := seen[id]; !ok {
			deduped = append(deduped, id)
			seen[id] = struct{}{}
		}
	}
	*r.captureUserIDs = deduped
	return r.base.GetUsersByIDs(ctx, userIDs)
}

func (r *mockCapturingRepo) BulkUpsertLedgerStats(ctx context.Context, stats []models.LedgerStats) (int, error) {
	return r.base.BulkUpsertLedgerStats(ctx, stats)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
