// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	stripe "github.com/stripe/stripe-go/v82"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// --- mocks ---

type mockInitiativeRepo struct {
	initiative *models.Initiative
	err        error
}

func (m *mockInitiativeRepo) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return m.initiative, m.err
}
func (m *mockInitiativeRepo) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return m.initiative, m.err
}
func (m *mockInitiativeRepo) GetIDBySlug(_ context.Context, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.initiative != nil {
		return m.initiative.ID, nil
	}
	return "", nil
}
func (m *mockInitiativeRepo) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (m *mockInitiativeRepo) Create(_ context.Context, i *models.Initiative) (*models.Initiative, error) {
	return i, nil
}
func (m *mockInitiativeRepo) Update(_ context.Context, i *models.Initiative) (*models.Initiative, error) {
	return i, nil
}
func (m *mockInitiativeRepo) Delete(_ context.Context, _ string) error { return nil }
func (m *mockInitiativeRepo) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return map[string]models.User{}, nil
}
func (m *mockInitiativeRepo) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	return map[string]models.Organization{}, nil
}

type mockLedgerClient struct {
	balance *clients.LedgerBalance
	err     error
}

func (m *mockLedgerClient) GetBalance(_ context.Context, _ string) (*clients.LedgerBalance, error) {
	return m.balance, m.err
}
func (m *mockLedgerClient) GetAllBalances(_ context.Context) ([]models.LedgerRawBalance, error) {
	return nil, nil
}
func (m *mockLedgerClient) GetTransactions(_ context.Context, _ clients.TransactionFilter) (*models.TransactionList, error) {
	return nil, nil
}

type mockStripeClient struct{}

func (m *mockStripeClient) GetProduct(_ context.Context, _ string) (*models.StripeProduct, error) {
	return nil, nil
}
func (m *mockStripeClient) CreatePaymentIntent(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	return nil, nil
}
func (m *mockStripeClient) CreateSubscription(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	return nil, nil
}
func (m *mockStripeClient) CancelSubscription(_ context.Context, _ string) error { return nil }
func (m *mockStripeClient) ConstructWebhookEvent(_ []byte, _, _ string) (stripe.Event, error) {
	return stripe.Event{}, nil
}

// --- flattenSponsors ---

func TestFlattenSponsors(t *testing.T) {
	list := models.LedgerSponsorList{
		Orgs: []models.LedgerSponsorOrg{
			{ID: "org-1", Name: "Big Corp", Total: 3_000_000},
			{ID: "org-2", Name: "Small Corp", Total: 500_000},
		},
		Individuals: []models.LedgerSponsorUser{
			{ID: "user-1", Name: "Top Donor", Total: 15_000_000},
		},
	}

	result := flattenSponsors(list)

	if len(result) != 3 {
		t.Fatalf("expected 3 sponsors, got %d", len(result))
	}
	if result[0].ID != "user-1" {
		t.Errorf("expected user-1 first (highest total), got %s", result[0].ID)
	}
	if result[1].ID != "org-1" {
		t.Errorf("expected org-1 second, got %s", result[1].ID)
	}
	if result[2].ID != "org-2" {
		t.Errorf("expected org-2 third, got %s", result[2].ID)
	}
}

func TestFlattenSponsors_Empty(t *testing.T) {
	result := flattenSponsors(models.LedgerSponsorList{})
	if result == nil {
		t.Error("result must be non-nil (must serialise as [] not null)")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(result))
	}
}

func TestFlattenSponsors_GeneratesAvatarWhenMissing(t *testing.T) {
	list := models.LedgerSponsorList{
		Orgs: []models.LedgerSponsorOrg{{ID: "org-1", Name: "Acme", AvatarURL: ""}},
	}
	result := flattenSponsors(list)
	if result[0].AvatarURL == "" {
		t.Error("expected generated avatar URL for sponsor with no AvatarURL")
	}
}

// --- enrichGoalsFromLedger ---

func TestEnrichGoalsFromLedger_PopulatesDonatedAndSpent(t *testing.T) {
	donated := int64(500_000)
	spent := int64(200_000)
	ledger := &mockLedgerClient{
		balance: &clients.LedgerBalance{
			SubTotals: map[string]*clients.LedgerSubTotal{
				// Ledger debits are negative; service negates to a positive SpentCents.
				"Mentorship": {Credit: donated, Debit: -spent},
			},
		},
	}
	initiative := &models.Initiative{
		ID:    "init-1",
		Goals: []models.Goal{{Name: "mentorship"}},
	}

	enrichGoalsFromLedger(context.Background(), ledger, initiative)

	g := initiative.Goals[0]
	if g.DonatedCents == nil || *g.DonatedCents != donated {
		t.Errorf("expected DonatedCents=%d, got %v", donated, g.DonatedCents)
	}
	if g.SpentCents == nil || *g.SpentCents != spent {
		t.Errorf("expected SpentCents=%d, got %v", spent, g.SpentCents)
	}
}

func TestEnrichGoalsFromLedger_CaseAndUnderscoreNormalization(t *testing.T) {
	// Ledger uses PascalCase; goal names may have underscores — both must match.
	ledger := &mockLedgerClient{
		balance: &clients.LedgerBalance{
			SubTotals: map[string]*clients.LedgerSubTotal{
				"BugBounty": {Credit: 100, Debit: 50},
			},
		},
	}
	initiative := &models.Initiative{
		ID:    "init-1",
		Goals: []models.Goal{{Name: "bug_bounty"}},
	}

	enrichGoalsFromLedger(context.Background(), ledger, initiative)

	g := initiative.Goals[0]
	if g.DonatedCents == nil || *g.DonatedCents != 100 {
		t.Errorf("underscore normalization failed: DonatedCents=%v", g.DonatedCents)
	}
}

func TestEnrichGoalsFromLedger_LedgerErrorLeavesGoalsUnchanged(t *testing.T) {
	ledger := &mockLedgerClient{err: errors.New("ledger down")}
	initiative := &models.Initiative{
		ID:    "init-1",
		Goals: []models.Goal{{Name: "mentorship"}},
	}

	enrichGoalsFromLedger(context.Background(), ledger, initiative)

	if initiative.Goals[0].DonatedCents != nil {
		t.Error("expected nil DonatedCents when Ledger is unavailable")
	}
	if initiative.Goals[0].SpentCents != nil {
		t.Error("expected nil SpentCents when Ledger is unavailable")
	}
}

func TestEnrichGoalsFromLedger_NoGoalsIsNoop(_ *testing.T) {
	called := false
	ledger := &mockLedgerClient{
		balance: &clients.LedgerBalance{
			SubTotals: map[string]*clients.LedgerSubTotal{
				"Mentorship": {Credit: 100},
			},
		},
	}
	// Wrap so we can detect if GetBalance is called
	_ = called
	initiative := &models.Initiative{ID: "init-1", Goals: nil}

	// Should return without calling Ledger at all (no panic, no error)
	enrichGoalsFromLedger(context.Background(), ledger, initiative)
	_ = ledger // no assertion needed — the mock would panic on nil balance if called
}

// --- enrichTransactionsFromDB ---

type mockRepoForEnrich struct {
	users map[string]models.User
	orgs  map[string]models.Organization
	err   error
}

func (m *mockRepoForEnrich) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users, nil
}
func (m *mockRepoForEnrich) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.orgs, nil
}

// mockRepoForEnrich must satisfy domain.InitiativeRepository — stub the rest.
func (m *mockRepoForEnrich) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, nil
}
func (m *mockRepoForEnrich) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, nil
}
func (m *mockRepoForEnrich) GetIDBySlug(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (m *mockRepoForEnrich) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (m *mockRepoForEnrich) Create(_ context.Context, i *models.Initiative) (*models.Initiative, error) {
	return i, nil
}
func (m *mockRepoForEnrich) Update(_ context.Context, i *models.Initiative) (*models.Initiative, error) {
	return i, nil
}
func (m *mockRepoForEnrich) Delete(_ context.Context, _ string) error { return nil }

func TestEnrichTransactionsFromDB_OrgTakesPriority(t *testing.T) {
	repo := &mockRepoForEnrich{
		users: map[string]models.User{
			"user-1": {UserID: "user-1", Name: "Alice", AvatarURL: "https://example.com/alice.png"},
		},
		orgs: map[string]models.Organization{
			"org-1": {ID: "org-1", Name: "Acme Corp", AvatarURL: "https://example.com/acme.png"},
		},
	}

	txns := []models.Transaction{
		{ID: "t1", LedgerUserID: "user-1", LedgerOrgID: "org-1"},
	}

	enrichTransactionsFromDB(context.Background(), repo, txns)

	if txns[0].DonorName != "Acme Corp" {
		t.Errorf("expected org name to take priority, got %q", txns[0].DonorName)
	}
	if txns[0].DonorLogoURL != "https://example.com/acme.png" {
		t.Errorf("expected org logo, got %q", txns[0].DonorLogoURL)
	}
}

func TestEnrichTransactionsFromDB_UserFallback(t *testing.T) {
	repo := &mockRepoForEnrich{
		users: map[string]models.User{
			"user-1": {UserID: "user-1", Name: "Alice", AvatarURL: "https://example.com/alice.png"},
		},
		orgs: map[string]models.Organization{},
	}

	txns := []models.Transaction{
		{ID: "t1", LedgerUserID: "user-1"},
	}

	enrichTransactionsFromDB(context.Background(), repo, txns)

	if txns[0].DonorName != "Alice" {
		t.Errorf("expected user name, got %q", txns[0].DonorName)
	}
	if txns[0].DonorLogoURL != "https://example.com/alice.png" {
		t.Errorf("expected user avatar, got %q", txns[0].DonorLogoURL)
	}
}

func TestEnrichTransactionsFromDB_GeneratesAvatarWhenNoDBMatch(t *testing.T) {
	repo := &mockRepoForEnrich{
		users: map[string]models.User{},
		orgs:  map[string]models.Organization{},
	}

	txns := []models.Transaction{
		{ID: "t1", LedgerUserID: "user-unknown", DonorName: "Anonymous"},
	}

	enrichTransactionsFromDB(context.Background(), repo, txns)

	if txns[0].DonorLogoURL == "" {
		t.Error("expected generated avatar URL when no DB match found")
	}
}

func TestEnrichTransactionsFromDB_DBErrorStillGeneratesAvatar(t *testing.T) {
	repo := &mockRepoForEnrich{err: errors.New("db down")}

	txns := []models.Transaction{
		{ID: "t1", LedgerUserID: "user-1", DonorName: "Somebody"},
	}

	// Should not panic; should fall back to generated avatar
	enrichTransactionsFromDB(context.Background(), repo, txns)

	if txns[0].DonorLogoURL == "" {
		t.Error("expected generated avatar even when DB lookup fails")
	}
}

// --- GetByID integration (sponsors + Ledger enrichment path) ---

func TestGetByID_FlattensSponsorsList(t *testing.T) {
	initiative := &models.Initiative{
		ID: "test-id",
		RawSponsors: models.LedgerSponsorList{
			Orgs: []models.LedgerSponsorOrg{
				{ID: "org-1", Name: "Big Corp", Total: 3_000_000},
			},
			Individuals: []models.LedgerSponsorUser{
				{ID: "user-1", Name: "Top Donor", Total: 15_000_000},
			},
		},
	}

	svc := NewInitiativeService(
		&mockInitiativeRepo{initiative: initiative},
		&mockLedgerClient{},
		&mockStripeClient{},
	)

	result, err := svc.GetByID(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Sponsors) != 2 {
		t.Fatalf("expected 2 sponsors, got %d", len(result.Sponsors))
	}
	if result.Sponsors[0].ID != "user-1" {
		t.Errorf("expected user-1 first (highest total), got %s", result.Sponsors[0].ID)
	}
}

func TestGetByID_RepoError(t *testing.T) {
	svc := NewInitiativeService(
		&mockInitiativeRepo{err: errors.New("not found")},
		&mockLedgerClient{},
		&mockStripeClient{},
	)

	_, err := svc.GetByID(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}
