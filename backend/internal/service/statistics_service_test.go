// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// --- test doubles ---

type testStatisticsRepo struct {
	stats        *models.PlatformStatistics
	orgs         map[string]models.Organization
	users        map[string]models.User
	projectNames map[string]string
	err          error
}

func (r *testStatisticsRepo) GetPlatformStatistics(_ context.Context) (*models.PlatformStatistics, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.stats != nil {
		return r.stats, nil
	}
	return &models.PlatformStatistics{}, nil
}

func (r *testStatisticsRepo) GetOrganizationsByIDs(_ context.Context, ids []string) (map[string]models.Organization, error) {
	if r.err != nil {
		return nil, r.err
	}
	result := make(map[string]models.Organization)
	for _, id := range ids {
		if o, ok := r.orgs[id]; ok {
			result[id] = o
		}
	}
	return result, nil
}

func (r *testStatisticsRepo) GetUsersByIDs(_ context.Context, ids []string) (map[string]models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	result := make(map[string]models.User)
	for _, id := range ids {
		if u, ok := r.users[id]; ok {
			result[id] = u
		}
	}
	return result, nil
}

func (r *testStatisticsRepo) GetInitiativeNamesByIDs(_ context.Context, ids []string) (map[string]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	result := make(map[string]string)
	for _, id := range ids {
		if name, ok := r.projectNames[id]; ok {
			result[id] = name
		}
	}
	return result, nil
}

type testLedgerClient struct {
	platformBalance *clients.LedgerPlatformBalance
	platformMonthly *clients.LedgerPlatformMonthly
	recentDonations []clients.LedgerRecentDonation
	err             error
}

func (c *testLedgerClient) GetBalance(_ context.Context, _ string) (*clients.LedgerBalance, error) {
	return nil, nil
}
func (c *testLedgerClient) GetAllBalances(_ context.Context) ([]models.LedgerRawBalance, error) {
	return nil, nil
}
func (c *testLedgerClient) GetTransactions(_ context.Context, _ clients.TransactionFilter) (*models.TransactionList, error) {
	return nil, nil
}
func (c *testLedgerClient) GetPlatformBalance(_ context.Context) (*clients.LedgerPlatformBalance, error) {
	return c.platformBalance, c.err
}
func (c *testLedgerClient) GetPlatformMonthly(_ context.Context, _ int) (*clients.LedgerPlatformMonthly, error) {
	return c.platformMonthly, c.err
}
func (c *testLedgerClient) GetPlatformRecentDonations(_ context.Context) ([]clients.LedgerRecentDonation, error) {
	return c.recentDonations, c.err
}

func newStatsSvc(repo *testStatisticsRepo, ledger *testLedgerClient) *StatisticsService {
	return NewStatisticsService(repo, ledger)
}

// --- GetPlatformDetails ---

func TestGetPlatformDetails_EnrichesOrgsAndUsers(t *testing.T) {
	repo := &testStatisticsRepo{
		orgs: map[string]models.Organization{
			"org-1": {ID: "org-1", Name: "Acme Corp", AvatarURL: "https://example.com/acme.png"},
		},
		users: map[string]models.User{
			"user-1": {UserID: "user-1", Name: "Alice Smith", AvatarURL: "https://example.com/alice.png"},
		},
	}
	ledger := &testLedgerClient{
		platformBalance: &clients.LedgerPlatformBalance{
			OrganizationsCents: 800_000,
			IndividualsCents:   200_000,
			TotalSupporters:    10,
			TopOrganizations: []clients.LedgerSponsorRaw{
				{ID: "org-1", Total: 500_000},
			},
			TopIndividuals: []clients.LedgerSponsorRaw{
				{ID: "user-1", Total: 200_000},
			},
		},
	}

	svc := newStatsSvc(repo, ledger)
	details, err := svc.GetPlatformDetails(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(details.TopOrganizations) != 1 {
		t.Fatalf("expected 1 org, got %d", len(details.TopOrganizations))
	}
	if details.TopOrganizations[0].Name != "Acme Corp" {
		t.Errorf("org name: want Acme Corp, got %s", details.TopOrganizations[0].Name)
	}
	if details.TopOrganizations[0].AvatarURL != "https://example.com/acme.png" {
		t.Errorf("org avatar: want https://example.com/acme.png, got %s", details.TopOrganizations[0].AvatarURL)
	}

	if len(details.TopIndividuals) != 1 {
		t.Fatalf("expected 1 individual, got %d", len(details.TopIndividuals))
	}
	if details.TopIndividuals[0].Name != "Alice Smith" {
		t.Errorf("user name: want Alice Smith, got %s", details.TopIndividuals[0].Name)
	}
}

func TestGetPlatformDetails_MissingEnrichmentUsesFallbackName(t *testing.T) {
	// Ledger has an org/user that doesn't exist in CF DB — falls back to placeholder names.
	repo := &testStatisticsRepo{
		orgs:  map[string]models.Organization{},
		users: map[string]models.User{},
	}
	ledger := &testLedgerClient{
		platformBalance: &clients.LedgerPlatformBalance{
			TopOrganizations: []clients.LedgerSponsorRaw{{ID: "org-unknown", Total: 100_000}},
			TopIndividuals:   []clients.LedgerSponsorRaw{{ID: "user-unknown", Total: 50_000}},
		},
	}

	svc := newStatsSvc(repo, ledger)
	details, err := svc.GetPlatformDetails(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if details.TopOrganizations[0].Name != unknownOrgName {
		t.Errorf("org name: want %q, got %q", unknownOrgName, details.TopOrganizations[0].Name)
	}
	if details.TopIndividuals[0].Name != anonymousName {
		t.Errorf("individual name: want %q, got %q", anonymousName, details.TopIndividuals[0].Name)
	}
}

func TestGetPlatformDetails_LedgerError(t *testing.T) {
	repo := &testStatisticsRepo{}
	ledger := &testLedgerClient{err: errors.New("ledger down")}

	svc := newStatsSvc(repo, ledger)
	_, err := svc.GetPlatformDetails(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetPlatformDetails_LedgerUnavailableReturnsEmpty(t *testing.T) {
	repo := &testStatisticsRepo{}
	ledger := &testLedgerClient{err: domain.ErrUpstreamUnavailable}

	svc := newStatsSvc(repo, ledger)
	details, err := svc.GetPlatformDetails(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for upstream unavailable, got %v", err)
	}
	if details == nil {
		t.Fatal("expected non-nil details, got nil")
	}
	if len(details.Categories) != 0 {
		t.Errorf("expected empty categories, got %d", len(details.Categories))
	}
}

func TestGetPlatformDetails_MapsFields(t *testing.T) {
	repo := &testStatisticsRepo{
		orgs:  map[string]models.Organization{},
		users: map[string]models.User{},
	}
	ledger := &testLedgerClient{
		platformBalance: &clients.LedgerPlatformBalance{
			TotalSupporters:    42,
			OrganizationsCents: 3_000_000,
			IndividualsCents:   2_000_000,
			Categories: []clients.LedgerCategoryTotal{
				{Name: "Development", TotalCents: 2_500_000, Count: 10},
			},
		},
	}

	svc := newStatsSvc(repo, ledger)
	details, err := svc.GetPlatformDetails(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if details.TotalRaisedCents != 5_000_000 { // 3_000_000 + 2_000_000
		t.Errorf("TotalRaisedCents: want 5000000, got %d", details.TotalRaisedCents)
	}
	if details.TotalSupporters != 42 {
		t.Errorf("TotalSupporters: want 42, got %d", details.TotalSupporters)
	}
	if details.OrganizationsCents != 3_000_000 {
		t.Errorf("OrganizationsCents: want 3000000, got %d", details.OrganizationsCents)
	}
	if len(details.Categories) != 1 || details.Categories[0].Name != "Development" {
		t.Errorf("Categories: expected [{Development}], got %v", details.Categories)
	}
}

// --- GetRecentDonations ---

func TestGetRecentDonations_OrgDonor(t *testing.T) {
	repo := &testStatisticsRepo{
		orgs: map[string]models.Organization{
			"org-1": {ID: "org-1", Name: "BigCorp", AvatarURL: "https://example.com/logo.png"},
		},
		users: map[string]models.User{},
	}
	ledger := &testLedgerClient{
		recentDonations: []clients.LedgerRecentDonation{
			{TxnID: "txn-1", ProjectID: "proj-1", OrganizationID: "org-1", Amount: 100_000, TxnDate: 1_700_000_000, TxnCategory: "Development"},
		},
	}

	svc := newStatsSvc(repo, ledger)
	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 donation, got %d", len(resp.Data))
	}
	d := resp.Data[0]
	if d.DonorType != donorTypeOrganization {
		t.Errorf("DonorType: want %q, got %s", donorTypeOrganization, d.DonorType)
	}
	if d.DonorName != "BigCorp" {
		t.Errorf("DonorName: want BigCorp, got %s", d.DonorName)
	}
	if d.DonorAvatarURL != "https://example.com/logo.png" {
		t.Errorf("DonorAvatarURL: want https://example.com/logo.png, got %s", d.DonorAvatarURL)
	}
	if d.AmountCents != 100_000 {
		t.Errorf("AmountCents: want 100000, got %d", d.AmountCents)
	}
	if d.Category != "Development" {
		t.Errorf("Category: want Development, got %q", d.Category)
	}
}

func TestGetRecentDonations_EnrichesProjectName(t *testing.T) {
	repo := &testStatisticsRepo{
		orgs:         map[string]models.Organization{"org-1": {ID: "org-1", Name: "BigCorp"}},
		users:        map[string]models.User{},
		projectNames: map[string]string{"proj-1": "Kubernetes"},
	}
	ledger := &testLedgerClient{
		recentDonations: []clients.LedgerRecentDonation{
			{TxnID: "txn-1", ProjectID: "proj-1", OrganizationID: "org-1", Amount: 100_000, TxnDate: 1_700_000_000},
		},
	}

	svc := newStatsSvc(repo, ledger)
	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := resp.Data[0]
	if d.ProjectName != "Kubernetes" {
		t.Errorf("ProjectName: want Kubernetes, got %q", d.ProjectName)
	}
	if d.ProjectID != "proj-1" {
		t.Errorf("ProjectID: want proj-1, got %q", d.ProjectID)
	}
}

func TestGetRecentDonations_IndividualDonor(t *testing.T) {
	repo := &testStatisticsRepo{
		orgs:  map[string]models.Organization{},
		users: map[string]models.User{"user-1": {UserID: "user-1", Name: "Bob"}},
	}
	ledger := &testLedgerClient{
		recentDonations: []clients.LedgerRecentDonation{
			{TxnID: "txn-2", UserID: "user-1", Amount: 5_000, TxnDate: 1_700_000_000},
		},
	}

	svc := newStatsSvc(repo, ledger)
	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := resp.Data[0]
	if d.DonorType != "individual" {
		t.Errorf("DonorType: want individual, got %s", d.DonorType)
	}
	if d.DonorName != "Bob" {
		t.Errorf("DonorName: want Bob, got %s", d.DonorName)
	}
}

func TestGetRecentDonations_FallsBackToSubmitterName(t *testing.T) {
	// No CF DB record — use SubmitterName from Ledger.
	repo := &testStatisticsRepo{
		orgs:  map[string]models.Organization{},
		users: map[string]models.User{},
	}
	ledger := &testLedgerClient{
		recentDonations: []clients.LedgerRecentDonation{
			{TxnID: "txn-3", UserID: "user-unknown", SubmitterName: "Carol", Amount: 2_000},
		},
	}

	svc := newStatsSvc(repo, ledger)
	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data[0].DonorName != "Carol" {
		t.Errorf("DonorName: want Carol, got %s", resp.Data[0].DonorName)
	}
}

func TestGetRecentDonations_AnonymousFallback(t *testing.T) {
	// No CF record and no SubmitterName — falls back to "Anonymous".
	repo := &testStatisticsRepo{
		orgs:  map[string]models.Organization{},
		users: map[string]models.User{},
	}
	ledger := &testLedgerClient{
		recentDonations: []clients.LedgerRecentDonation{
			{TxnID: "txn-4", UserID: "user-unknown", SubmitterName: "", Amount: 1_000},
		},
	}

	svc := newStatsSvc(repo, ledger)
	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data[0].DonorName != anonymousName {
		t.Errorf("DonorName: want %q, got %s", anonymousName, resp.Data[0].DonorName)
	}
}

func TestGetRecentDonations_LedgerError(t *testing.T) {
	repo := &testStatisticsRepo{}
	ledger := &testLedgerClient{err: errors.New("timeout")}

	svc := newStatsSvc(repo, ledger)
	_, err := svc.GetRecentDonations(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetRecentDonations_UpstreamUnavailable(t *testing.T) {
	repo := &testStatisticsRepo{}
	ledger := &testLedgerClient{err: domain.ErrUpstreamUnavailable}

	svc := newStatsSvc(repo, ledger)
	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("expected empty data, got %d entries", len(resp.Data))
	}
}

// --- GetPlatformMonthly ---

func TestGetPlatformMonthly_MapsBuckets(t *testing.T) {
	repo := &testStatisticsRepo{}
	ledger := &testLedgerClient{
		platformMonthly: &clients.LedgerPlatformMonthly{
			Buckets: []clients.LedgerMonthlyBucket{
				{Year: 2025, Month: 1, TotalCents: 1_000_000, Supporters: 50},
				{Year: 2025, Month: 2, TotalCents: 1_200_000, Supporters: 60},
			},
		},
	}

	svc := newStatsSvc(repo, ledger)
	monthly, err := svc.GetPlatformMonthly(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(monthly.Buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(monthly.Buckets))
	}
	if monthly.Buckets[0].Year != 2025 || monthly.Buckets[0].Month != 1 {
		t.Errorf("bucket[0]: want {2025,1}, got {%d,%d}", monthly.Buckets[0].Year, monthly.Buckets[0].Month)
	}
	if monthly.Buckets[1].TotalCents != 1_200_000 {
		t.Errorf("bucket[1].TotalCents: want 1200000, got %d", monthly.Buckets[1].TotalCents)
	}
}

func TestGetPlatformMonthly_LedgerError(t *testing.T) {
	repo := &testStatisticsRepo{}
	ledger := &testLedgerClient{err: errors.New("ledger down")}

	svc := newStatsSvc(repo, ledger)
	_, err := svc.GetPlatformMonthly(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetPlatformMonthly_UpstreamUnavailableReturnsEmpty(t *testing.T) {
	repo := &testStatisticsRepo{}
	ledger := &testLedgerClient{err: domain.ErrUpstreamUnavailable}

	svc := newStatsSvc(repo, ledger)
	monthly, err := svc.GetPlatformMonthly(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for upstream unavailable, got %v", err)
	}
	if monthly == nil {
		t.Fatal("expected non-nil response, got nil")
	}
	if len(monthly.Buckets) != 0 {
		t.Errorf("expected empty buckets, got %d", len(monthly.Buckets))
	}
}

func TestGetPlatformDetails_EnrichmentRepoError(t *testing.T) {
	// Ledger succeeds but CF DB fails during org enrichment.
	repo := &testStatisticsRepo{err: errors.New("db connection lost")}
	ledger := &testLedgerClient{
		platformBalance: &clients.LedgerPlatformBalance{
			TopOrganizations: []clients.LedgerSponsorRaw{{ID: "org-1", Total: 100_000}},
			TopIndividuals:   []clients.LedgerSponsorRaw{},
		},
	}

	svc := newStatsSvc(repo, ledger)
	_, err := svc.GetPlatformDetails(context.Background())
	if err == nil {
		t.Fatal("expected error from repo during enrichment, got nil")
	}
}

func TestGetRecentDonations_EnrichmentRepoError(t *testing.T) {
	// Ledger succeeds but CF DB fails during donor enrichment.
	repo := &testStatisticsRepo{err: errors.New("db connection lost")}
	ledger := &testLedgerClient{
		recentDonations: []clients.LedgerRecentDonation{
			{TxnID: "txn-1", OrganizationID: "org-1", Amount: 50_000},
		},
	}

	svc := newStatsSvc(repo, ledger)
	_, err := svc.GetRecentDonations(context.Background())
	if err == nil {
		t.Fatal("expected error from repo during enrichment, got nil")
	}
}
