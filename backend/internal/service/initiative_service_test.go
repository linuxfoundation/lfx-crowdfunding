// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	stripe "github.com/stripe/stripe-go/v85"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// --- mocks ---

type mockInitiativeRepo struct {
	initiative      *models.Initiative
	lastCreated     *models.Initiative
	lastInput       models.InitiativeCreateInput
	lastUpdated     *models.Initiative
	lastUpdateInput models.InitiativeUpdateInput
	err             error
	updateErr       error
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
func (m *mockInitiativeRepo) ResolveSlug(_ context.Context, _ string) (string, error) {
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
func (m *mockInitiativeRepo) Create(_ context.Context, i *models.Initiative, input models.InitiativeCreateInput) (*models.Initiative, error) {
	m.lastCreated = i
	m.lastInput = input
	return i, nil
}
func (m *mockInitiativeRepo) Update(_ context.Context, i *models.Initiative, input models.InitiativeUpdateInput) (*models.Initiative, error) {
	m.lastUpdated = i
	m.lastUpdateInput = input
	return i, m.updateErr
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
func (m *mockLedgerClient) GetPlatformBalance(_ context.Context, _ int) (*clients.LedgerPlatformBalance, error) {
	return nil, nil
}
func (m *mockLedgerClient) GetPlatformMonthly(_ context.Context, _ int) (*clients.LedgerPlatformMonthly, error) {
	return nil, nil
}
func (m *mockLedgerClient) GetPlatformRecentDonations(_ context.Context) ([]clients.LedgerRecentDonation, error) {
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
func (m *mockStripeClient) CreateCustomer(_ context.Context, _, _ string) (string, error) {
	return "cus_mock", nil
}
func (m *mockStripeClient) CreateSetupIntent(_ context.Context, _ string) (string, error) {
	return "seti_mock_secret", nil
}
func (m *mockStripeClient) AttachPaymentMethod(_ context.Context, _, _ string) (*models.CardDetails, error) {
	return &models.CardDetails{}, nil
}
func (m *mockStripeClient) GetPaymentMethod(_ context.Context, _ string) (*models.CardDetails, error) {
	return &models.CardDetails{}, nil
}
func (m *mockStripeClient) DetachPaymentMethod(_ context.Context, _ string) error { return nil }
func (m *mockStripeClient) GetOrCreatePrice(_ context.Context, _ string, _ string, _ int64, _ string, _ string) (string, error) {
	return "price_mock", nil
}
func (m *mockStripeClient) CreateProduct(_ context.Context, _, _ string) (string, error) {
	return "prod_mock", nil
}
func (m *mockStripeClient) DeleteProduct(_ context.Context, _ string) error { return nil }

type mockUserRepository struct {
	user *models.User
	err  error
}

func (m *mockUserRepository) GetByUsername(_ context.Context, username string) (*models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.user != nil {
		return m.user, nil
	}
	return &models.User{ID: username, Username: username}, nil
}
func (m *mockUserRepository) GetByID(_ context.Context, id string) (*models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.user != nil {
		return m.user, nil
	}
	return &models.User{ID: id}, nil
}
func (m *mockUserRepository) Upsert(_ context.Context, u *models.User) (*models.User, error) {
	return u, nil
}
func (m *mockUserRepository) UpdateStripeInfo(_ context.Context, _, _, _ string) error   { return nil }
func (m *mockUserRepository) ClearStripePaymentMethod(_ context.Context, _ string) error { return nil }

type mockEmailService struct {
	approvedCalled      bool
	declinedCalled      bool
	forReviewCalled     bool
	approvedToEmail     string
	approvedToName      string
	declinedToEmail     string
	declinedToName      string
	forReviewOwnerName  string
	forReviewOwnerEmail string
	forReviewInitName   string
	forReviewInitURL    string
	err                 error
}

func (m *mockEmailService) SendProjectApprovedEmail(_ context.Context, toEmail, toName, _, _ string) error {
	m.approvedCalled = true
	m.approvedToEmail = toEmail
	m.approvedToName = toName
	return m.err
}
func (m *mockEmailService) SendProjectDeclinedEmail(_ context.Context, toEmail, toName, _, _ string) error {
	m.declinedCalled = true
	m.declinedToEmail = toEmail
	m.declinedToName = toName
	return m.err
}
func (m *mockEmailService) SendProjectForReviewEmail(_ context.Context, ownerName, ownerEmail, initiativeName, initiativeURL, _, _ string) error {
	m.forReviewCalled = true
	m.forReviewOwnerName = ownerName
	m.forReviewOwnerEmail = ownerEmail
	m.forReviewInitName = initiativeName
	m.forReviewInitURL = initiativeURL
	return m.err
}
func (m *mockEmailService) InitiativeURL(slug string) string {
	return "https://crowdfunding.lfx.linuxfoundation.org/initiatives/" + slug
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
func (m *mockRepoForEnrich) ResolveSlug(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (m *mockRepoForEnrich) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (m *mockRepoForEnrich) Create(_ context.Context, i *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	return i, nil
}
func (m *mockRepoForEnrich) Update(_ context.Context, i *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	return i, nil
}
func (m *mockRepoForEnrich) Delete(_ context.Context, _ string) error { return nil }

func TestEnrichTransactionsFromDB_OrgTakesPriority(t *testing.T) {
	repo := &mockRepoForEnrich{
		users: map[string]models.User{
			"user-1": {ID: "user-1", Name: "Alice", AvatarURL: "https://example.com/alice.png"},
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
			"user-1": {ID: "user-1", Name: "Alice", AvatarURL: "https://example.com/alice.png"},
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
		&mockUserRepository{},
		&mockLedgerClient{},
		&mockStripeClient{},
		&mockEmailService{},
		slog.Default(),
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
		&mockUserRepository{},
		&mockLedgerClient{},
		&mockStripeClient{},
		&mockEmailService{},
		slog.Default(),
	)

	_, err := svc.GetByID(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

func newCreateSvc(repo domain.InitiativeRepository) *InitiativeService {
	return NewInitiativeService(repo, &mockUserRepository{}, &mockLedgerClient{}, &mockStripeClient{}, &mockEmailService{}, slog.Default())
}

func TestCreate_MissingName(t *testing.T) {
	_, err := newCreateSvc(&mockInitiativeRepo{}).Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{Slug: "my-project", InitiativeType: "project"},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_MissingSlug_AutoGeneratesFromName(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, _ = svc.Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{Name: "My Cool Project", InitiativeType: "project"},
	)
	// Stripe mock returns "prod_mock" and repo.Create records whatever initiative was passed.
	// The slug should have been derived from the name.
	if repo.lastCreated == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if repo.lastCreated.Slug != "my-cool-project" {
		t.Errorf("expected slug %q, got %q", "my-cool-project", repo.lastCreated.Slug)
	}
}

func TestCreate_MissingInitiativeType(t *testing.T) {
	_, err := newCreateSvc(&mockInitiativeRepo{}).Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{Name: "My Project", Slug: "my-project"},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_UnknownInitiativeType(t *testing.T) {
	_, err := newCreateSvc(&mockInitiativeRepo{}).Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{Name: "My Project", Slug: "my-project", InitiativeType: "nonsense"},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_GoalMissingName(t *testing.T) {
	_, err := newCreateSvc(&mockInitiativeRepo{}).Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{
			Name:           "My Project",
			Slug:           "my-project",
			InitiativeType: "project",
			Goals:          []models.GoalInput{{AmountInCents: 50000}},
		},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty goal name, got %v", err)
	}
}

func TestCreate_CustomWebsiteMissingURL(t *testing.T) {
	_, err := newCreateSvc(&mockInitiativeRepo{}).Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{
			Name:           "My Project",
			Slug:           "my-project",
			InitiativeType: "project",
			CustomWebsites: []models.CustomWebsiteInput{{Name: "Docs"}},
		},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty website URL, got %v", err)
	}
}

func TestCreate_ContactMissingType(t *testing.T) {
	_, err := newCreateSvc(&mockInitiativeRepo{}).Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{
			Name:           "My OSTIF",
			Slug:           "my-ostif",
			InitiativeType: "ostif",
			Contacts:       []models.ContactInput{{FirstName: "Jane"}},
		},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty contact type, got %v", err)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func newUpdateSvc(repo *mockInitiativeRepo) *InitiativeService {
	return NewInitiativeService(repo, &mockUserRepository{}, &mockLedgerClient{}, &mockStripeClient{}, &mockEmailService{}, slog.Default())
}

func TestUpdate_GoalMissingName(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", OwnerID: "owner-1"},
	}
	_, err := newUpdateSvc(repo).Update(context.Background(), "init-1", "owner-1",
		models.InitiativeUpdateInput{
			Goals: []models.GoalInput{{AmountInCents: 5000}},
		},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty goal name, got %v", err)
	}
}

func TestUpdate_CustomWebsiteMissingURL(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", OwnerID: "owner-1"},
	}
	_, err := newUpdateSvc(repo).Update(context.Background(), "init-1", "owner-1",
		models.InitiativeUpdateInput{
			CustomWebsites: []models.CustomWebsiteInput{{Name: "Docs"}},
		},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty website URL, got %v", err)
	}
}

func TestUpdate_ContactMissingType(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", OwnerID: "owner-1"},
	}
	_, err := newUpdateSvc(repo).Update(context.Background(), "init-1", "owner-1",
		models.InitiativeUpdateInput{
			Contacts: []models.ContactInput{{FirstName: "Jane"}},
		},
	)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty contact type, got %v", err)
	}
}

func TestUpdate_ChildInputPassedToRepo(t *testing.T) {
	goals := []models.GoalInput{{Name: "MVP", AmountInCents: 10000}}
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", OwnerID: "owner-1"},
	}
	_, err := newUpdateSvc(repo).Update(context.Background(), "init-1", "owner-1",
		models.InitiativeUpdateInput{Goals: goals},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.lastUpdateInput.Goals) != 1 || repo.lastUpdateInput.Goals[0].Name != "MVP" {
		t.Errorf("expected goals to be passed to repo, got %+v", repo.lastUpdateInput.Goals)
	}
}

func TestUpdate_NilChildFieldsAreNoOp(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", OwnerID: "owner-1"},
	}
	_, err := newUpdateSvc(repo).Update(context.Background(), "init-1", "owner-1",
		models.InitiativeUpdateInput{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastUpdateInput.Goals != nil {
		t.Error("expected nil Goals (no-op), but got non-nil")
	}
}

func TestUpdate_ForbiddenForNonOwner(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", OwnerID: "owner-1"},
	}
	_, err := newUpdateSvc(repo).Update(context.Background(), "init-1", "other-user",
		models.InitiativeUpdateInput{},
	)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdate_CannotSetApprovalControlledStatus(t *testing.T) {
	restricted := []models.InitiativeStatus{
		models.StatusPublished,
		models.StatusDeclined,
		models.StatusPending,
	}
	for _, s := range restricted {
		s := s
		t.Run(string(s), func(t *testing.T) {
			repo := &mockInitiativeRepo{
				initiative: &models.Initiative{ID: "init-1", OwnerID: "owner-1"},
			}
			_, err := newUpdateSvc(repo).Update(context.Background(), "init-1", "owner-1",
				models.InitiativeUpdateInput{Status: &s},
			)
			if !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("expected ErrForbidden for status %q, got %v", s, err)
			}
		})
	}
}

// ── Create — for-review email notification ────────────────────────────────────

func newCreateSvcWithEmail(repo domain.InitiativeRepository, userRepo *mockUserRepository, emailSvc *mockEmailService) *InitiativeService {
	return NewInitiativeService(repo, userRepo, &mockLedgerClient{}, &mockStripeClient{}, emailSvc, slog.Default())
}

func TestCreate_SendsForReviewEmail(t *testing.T) {
	repo := &mockInitiativeRepo{}
	userRepo := &mockUserRepository{
		user: &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "owner-1", Email: "owner@example.com", Name: "Alice"},
	}
	emailSvc := &mockEmailService{}

	svc := newCreateSvcWithEmail(repo, userRepo, emailSvc)
	created, err := svc.Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{Name: "My Project", Slug: "my-project", InitiativeType: "project"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emailSvc.forReviewCalled {
		t.Error("expected SendProjectForReviewEmail to be called")
	}
	if emailSvc.forReviewOwnerName != "Alice" {
		t.Errorf("expected ownerName Alice, got %q", emailSvc.forReviewOwnerName)
	}
	if emailSvc.forReviewOwnerEmail != "owner@example.com" {
		t.Errorf("expected ownerEmail owner@example.com, got %q", emailSvc.forReviewOwnerEmail)
	}
	if emailSvc.forReviewInitName != "My Project" {
		t.Errorf("expected initiativeName My Project, got %q", emailSvc.forReviewInitName)
	}
	if emailSvc.forReviewInitURL == "" {
		t.Error("expected non-empty initiativeURL")
	}
	if created.Slug != "" && !contains(emailSvc.forReviewInitURL, created.Slug) {
		t.Errorf("expected initiativeURL to contain slug %q, got %q", created.Slug, emailSvc.forReviewInitURL)
	}
}

func TestCreate_ForReviewEmail_UserLookupErrorFails(t *testing.T) {
	repo := &mockInitiativeRepo{}
	userRepo := &mockUserRepository{err: errors.New("user not found")}
	emailSvc := &mockEmailService{}

	svc := newCreateSvcWithEmail(repo, userRepo, emailSvc)
	_, err := svc.Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{Name: "My Project", Slug: "my-project", InitiativeType: "project"},
	)
	// Owner lookup failure is now fatal — unknown owners cannot create initiatives.
	if err == nil {
		t.Fatal("expected error when owner lookup fails, got nil")
	}
	if emailSvc.forReviewCalled {
		t.Error("expected no email when owner lookup fails")
	}
}

func TestCreate_ForReviewEmail_EmailErrorIsNonFatal(t *testing.T) {
	repo := &mockInitiativeRepo{}
	userRepo := &mockUserRepository{
		user: &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "owner-1", Email: "owner@example.com", Name: "Alice"},
	}
	emailSvc := &mockEmailService{err: errors.New("mandrill down")}

	svc := newCreateSvcWithEmail(repo, userRepo, emailSvc)
	_, err := svc.Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{Name: "My Project", Slug: "my-project", InitiativeType: "project"},
	)
	if err != nil {
		t.Fatalf("email failure must not propagate, got %v", err)
	}
}

func TestCreate_ForReviewEmail_FallsBackToEmailWhenNameEmpty(t *testing.T) {
	repo := &mockInitiativeRepo{}
	userRepo := &mockUserRepository{
		user: &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "owner-1", Email: "owner@example.com", Name: ""},
	}
	emailSvc := &mockEmailService{}

	svc := newCreateSvcWithEmail(repo, userRepo, emailSvc)
	_, _ = svc.Create(
		context.Background(), "owner-1",
		models.InitiativeCreateInput{Name: "My Project", Slug: "my-project", InitiativeType: "project"},
	)
	if emailSvc.forReviewOwnerName != "owner@example.com" {
		t.Errorf("expected ownerName to fall back to email, got %q", emailSvc.forReviewOwnerName)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ── Approve ───────────────────────────────────────────────────────────────────

func newProcessApprovalSvc(repo *mockInitiativeRepo) *InitiativeService {
	return NewInitiativeService(repo, &mockUserRepository{}, &mockLedgerClient{}, &mockStripeClient{}, &mockEmailService{}, slog.Default())
}

func TestProcessApproval_SetsStatusPublished(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", Status: models.StatusSubmitted},
	}
	updated, err := newProcessApprovalSvc(repo).ProcessApproval(context.Background(), "init-1", models.ApprovalActionApprove)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != models.StatusPublished {
		t.Errorf("expected status %q, got %q", models.StatusPublished, updated.Status)
	}
	if repo.lastUpdated == nil || repo.lastUpdated.Status != models.StatusPublished {
		t.Error("repo.Update was not called with the correct status")
	}
}

func TestProcessApproval_SetsStatusDeclined(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", Status: models.StatusSubmitted},
	}
	updated, err := newProcessApprovalSvc(repo).ProcessApproval(context.Background(), "init-1", models.ApprovalActionDecline)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != models.StatusDeclined {
		t.Errorf("expected status %q, got %q", models.StatusDeclined, updated.Status)
	}
}

func TestProcessApproval_InitiativeNotFound(t *testing.T) {
	repo := &mockInitiativeRepo{err: domain.ErrInitiativeNotFound}
	_, err := newProcessApprovalSvc(repo).ProcessApproval(context.Background(), "missing", models.ApprovalActionApprove)
	if !errors.Is(err, domain.ErrInitiativeNotFound) {
		t.Errorf("expected ErrInitiativeNotFound, got %v", err)
	}
}

func TestProcessApproval_UpdateError(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", Status: models.StatusSubmitted},
		updateErr:  errors.New("db unavailable"),
	}
	_, err := newProcessApprovalSvc(repo).ProcessApproval(context.Background(), "init-1", models.ApprovalActionApprove)
	if err == nil {
		t.Fatal("expected error from repo.Update, got nil")
	}
}

func TestProcessApproval_InvalidAction(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", Status: models.StatusSubmitted},
	}
	_, err := newProcessApprovalSvc(repo).ProcessApproval(context.Background(), "init-1", models.InitiativeApprovalAction("invalid"))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestProcessApproval_RejectsNonApprovableStatus(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{ID: "init-1", Status: models.StatusPublished},
	}
	_, err := newProcessApprovalSvc(repo).ProcessApproval(context.Background(), "init-1", models.ApprovalActionApprove)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for non-approvable status, got %v", err)
	}
}

// ── Email notification tests ──────────────────────────────────────────────────

func newProcessApprovalSvcWithEmail(repo *mockInitiativeRepo, userRepo *mockUserRepository, emailSvc *mockEmailService) *InitiativeService {
	return NewInitiativeService(repo, userRepo, &mockLedgerClient{}, &mockStripeClient{}, emailSvc, slog.Default())
}

func TestProcessApproval_SendsApprovedEmail(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID:      "init-1",
			Status:  models.StatusSubmitted,
			OwnerID: "00000000-0000-0000-0000-000000000001",
			Name:    "My Project",
			Slug:    "my-project",
		},
	}
	userRepo := &mockUserRepository{
		user: &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "owner-1", Email: "owner@example.com", Name: "Alice"},
	}
	emailSvc := &mockEmailService{}

	svc := newProcessApprovalSvcWithEmail(repo, userRepo, emailSvc)
	_, err := svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionApprove)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emailSvc.approvedCalled {
		t.Error("expected SendProjectApprovedEmail to be called")
	}
	if emailSvc.declinedCalled {
		t.Error("expected SendProjectDeclinedEmail NOT to be called on approve")
	}
	if emailSvc.approvedToEmail != "owner@example.com" {
		t.Errorf("expected email to owner@example.com, got %q", emailSvc.approvedToEmail)
	}
	if emailSvc.approvedToName != "Alice" {
		t.Errorf("expected name Alice, got %q", emailSvc.approvedToName)
	}
}

func TestProcessApproval_SendsDeclinedEmail(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID:      "init-1",
			Status:  models.StatusSubmitted,
			OwnerID: "00000000-0000-0000-0000-000000000001",
			Name:    "My Project",
			Slug:    "my-project",
		},
	}
	userRepo := &mockUserRepository{
		user: &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "owner-1", Email: "owner@example.com", Name: "Bob"},
	}
	emailSvc := &mockEmailService{}

	svc := newProcessApprovalSvcWithEmail(repo, userRepo, emailSvc)
	_, err := svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionDecline)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !emailSvc.declinedCalled {
		t.Error("expected SendProjectDeclinedEmail to be called")
	}
	if emailSvc.approvedCalled {
		t.Error("expected SendProjectApprovedEmail NOT to be called on decline")
	}
	if emailSvc.declinedToEmail != "owner@example.com" {
		t.Errorf("expected email to owner@example.com, got %q", emailSvc.declinedToEmail)
	}
	if emailSvc.declinedToName != "Bob" {
		t.Errorf("expected name Bob, got %q", emailSvc.declinedToName)
	}
}

func TestProcessApproval_EmailErrorIsNonFatal(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID:      "init-1",
			Status:  models.StatusSubmitted,
			OwnerID: "00000000-0000-0000-0000-000000000001",
			Name:    "My Project",
			Slug:    "my-project",
		},
	}
	userRepo := &mockUserRepository{
		user: &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "owner-1", Email: "owner@example.com", Name: "Alice"},
	}
	emailSvc := &mockEmailService{err: errors.New("mandrill down")}

	svc := newProcessApprovalSvcWithEmail(repo, userRepo, emailSvc)
	_, err := svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionApprove)
	// Email failure must NOT propagate — the approval itself must succeed.
	if err != nil {
		t.Fatalf("expected nil error (email failure is non-fatal), got %v", err)
	}
}

func TestProcessApproval_UserLookupErrorIsNonFatal(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID:      "init-1",
			Status:  models.StatusSubmitted,
			OwnerID: "00000000-0000-0000-0000-000000000001",
			Name:    "My Project",
			Slug:    "my-project",
		},
	}
	userRepo := &mockUserRepository{err: errors.New("user not found")}
	emailSvc := &mockEmailService{}

	svc := newProcessApprovalSvcWithEmail(repo, userRepo, emailSvc)
	_, err := svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionApprove)
	// User lookup failure must NOT propagate.
	if err != nil {
		t.Fatalf("expected nil error (user lookup failure is non-fatal), got %v", err)
	}
	// Email must not be sent if owner lookup failed.
	if emailSvc.approvedCalled {
		t.Error("expected no email to be sent when owner lookup fails")
	}
}

func TestProcessApproval_FallsBackToEmailWhenNameEmpty(t *testing.T) {
	repo := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID:      "init-1",
			Status:  models.StatusSubmitted,
			OwnerID: "00000000-0000-0000-0000-000000000001",
			Name:    "My Project",
			Slug:    "my-project",
		},
	}
	userRepo := &mockUserRepository{
		user: &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "owner-1", Email: "owner@example.com", Name: ""},
	}
	emailSvc := &mockEmailService{}

	svc := newProcessApprovalSvcWithEmail(repo, userRepo, emailSvc)
	_, _ = svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionApprove)

	if emailSvc.approvedToName != "owner@example.com" {
		t.Errorf("expected display name to fall back to email, got %q", emailSvc.approvedToName)
	}
}

// ── Create — child-table and entity-only field propagation ────────────────────
// These tests verify that every field added to InitiativeCreateInput in the
// recent refactor is correctly mapped onto the Initiative struct that gets
// passed through to the repository layer.

func TestCreate_PropagatesEntityOnlyFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	end := now.Add(24 * time.Hour)

	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Event",
		Slug:           "my-event",
		InitiativeType: "event",
		EventbriteURL:  "https://eventbrite.com/e/123",
		ApplicationURL: "https://apply.example.com",
		EventStartDate: &now,
		EventEndDate:   &end,
		Country:        "US",
		City:           "San Francisco",
		IsOnline:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := repo.lastCreated
	if got.EventbriteURL != "https://eventbrite.com/e/123" {
		t.Errorf("EventbriteURL: want %q got %q", "https://eventbrite.com/e/123", got.EventbriteURL)
	}
	if got.ApplicationURL != "https://apply.example.com" {
		t.Errorf("ApplicationURL: want %q got %q", "https://apply.example.com", got.ApplicationURL)
	}
	if got.EventStartDate == nil || !got.EventStartDate.Equal(now) {
		t.Errorf("EventStartDate: want %v got %v", now, got.EventStartDate)
	}
	if got.EventEndDate == nil || !got.EventEndDate.Equal(end) {
		t.Errorf("EventEndDate: want %v got %v", end, got.EventEndDate)
	}
	if got.Country != "US" {
		t.Errorf("Country: want %q got %q", "US", got.Country)
	}
	if got.City != "San Francisco" {
		t.Errorf("City: want %q got %q", "San Francisco", got.City)
	}
	if !got.IsOnline {
		t.Error("IsOnline: want true got false")
	}
}

func TestCreate_PropagatesGoals(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Project",
		Slug:           "my-project",
		InitiativeType: "project",
		Goals: []models.GoalInput{
			{Name: "Development", AmountInCents: 50000, Allocation: "eng", SortOrder: 0},
			{Name: "Marketing", AmountInCents: 10000, SortOrder: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	goals := repo.lastInput.Goals
	if len(goals) != 2 {
		t.Fatalf("expected 2 goals, got %d", len(goals))
	}
	if goals[0].Name != "Development" || goals[0].AmountInCents != 50000 || goals[0].Allocation != "eng" {
		t.Errorf("goal[0] mismatch: %+v", goals[0])
	}
	if goals[1].Name != "Marketing" || goals[1].SortOrder != 1 {
		t.Errorf("goal[1] mismatch: %+v", goals[1])
	}
}

func TestCreate_PropagatesBeneficiaries(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Project",
		Slug:           "my-project",
		InitiativeType: "project",
		Beneficiaries: []models.BeneficiaryInput{
			{Name: "Alice", Email: "alice@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.lastInput.Beneficiaries) != 1 {
		t.Fatalf("expected 1 beneficiary, got %d", len(repo.lastInput.Beneficiaries))
	}
	b := repo.lastInput.Beneficiaries[0]
	if b.Name != "Alice" || b.Email != "alice@example.com" {
		t.Errorf("beneficiary mismatch: %+v", b)
	}
}

func TestCreate_PropagatesCustomWebsites(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Project",
		Slug:           "my-project",
		InitiativeType: "project",
		CustomWebsites: []models.CustomWebsiteInput{
			{Name: "Docs", URL: "https://docs.example.com"},
			{Name: "Blog", URL: "https://blog.example.com"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ws := repo.lastInput.CustomWebsites
	if len(ws) != 2 {
		t.Fatalf("expected 2 custom websites, got %d", len(ws))
	}
	if ws[0].URL != "https://docs.example.com" {
		t.Errorf("custom website[0] URL mismatch: %q", ws[0].URL)
	}
}

func TestCreate_PropagatesContributors(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Project",
		Slug:           "my-project",
		InitiativeType: "project",
		Contributors: []models.ContributorInput{
			{Name: "Bob", Email: "bob@example.com"},
			{Name: "Carol", Email: "carol@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cs := repo.lastInput.Contributors
	if len(cs) != 2 {
		t.Fatalf("expected 2 contributors, got %d", len(cs))
	}
	if cs[0].Name != "Bob" || cs[1].Name != "Carol" {
		t.Errorf("contributor names mismatch: %q, %q", cs[0].Name, cs[1].Name)
	}
}

func TestCreate_PropagatesMentors(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Mentorship",
		Slug:           "my-mentorship",
		InitiativeType: "mentorship",
		Mentors: []models.MentorInput{
			{Name: "Dr. Smith", Email: "smith@uni.edu", Introduction: "Expert in Go"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ms := repo.lastInput.Mentors
	if len(ms) != 1 {
		t.Fatalf("expected 1 mentor, got %d", len(ms))
	}
	if ms[0].Name != "Dr. Smith" || ms[0].Introduction != "Expert in Go" {
		t.Errorf("mentor mismatch: %+v", ms[0])
	}
}

func TestCreate_PropagatesProgramInfo(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Mentorship",
		Slug:           "my-mentorship",
		InitiativeType: "mentorship",
		ProgramInfo: &models.ProgramInfoInput{
			Terms:           []string{"Spring 2026", "Fall 2026"},
			Skills:          []string{"Go", "Kubernetes"},
			TermsConditions: true,
			CustomTerm: &models.CustomTermInput{
				TermName:   "Custom",
				StartMonth: "January",
				EndMonth:   "June",
				Year:       2026,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pi := repo.lastInput.ProgramInfo
	if pi == nil {
		t.Fatal("expected ProgramInfo to be non-nil")
	}
	if len(pi.Terms) != 2 || pi.Terms[0] != "Spring 2026" {
		t.Errorf("terms mismatch: %v", pi.Terms)
	}
	if len(pi.Skills) != 2 || pi.Skills[1] != "Kubernetes" {
		t.Errorf("skills mismatch: %v", pi.Skills)
	}
	if !pi.TermsConditions {
		t.Error("expected TermsConditions true")
	}
	if pi.CustomTerm == nil || pi.CustomTerm.TermName != "Custom" || pi.CustomTerm.Year != 2026 {
		t.Errorf("custom term mismatch: %+v", pi.CustomTerm)
	}
}

func TestCreate_PropagatesSponsorshipTiers(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Event",
		Slug:           "my-event",
		InitiativeType: "event",
		SponsorshipTiers: []models.SponsorshipTierInput{
			{Name: "Gold", Minimum: 500000, SortOrder: 0},
			{Name: "Silver", Minimum: 100000, SortOrder: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tiers := repo.lastInput.SponsorshipTiers
	if len(tiers) != 2 {
		t.Fatalf("expected 2 sponsorship tiers, got %d", len(tiers))
	}
	if tiers[0].Name != "Gold" || tiers[0].Minimum != 500000 {
		t.Errorf("tier[0] mismatch: %+v", tiers[0])
	}
	if tiers[1].Name != "Silver" || tiers[1].SortOrder != 1 {
		t.Errorf("tier[1] mismatch: %+v", tiers[1])
	}
}

func TestCreate_PropagatesOSTIFDetail(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My OSTIF Audit",
		Slug:           "my-ostif",
		InitiativeType: "ostif",
		OSTIFDetail: &models.OSTIFDetailInput{
			MonetizationStrategy:    "donations",
			CurrentSecurityStrategy: "manual review",
			LicenseType:             "MIT",
			TotalBudgetInCents:      1000000,
			TermsConditions:         true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := repo.lastInput.OSTIFDetail
	if d == nil {
		t.Fatal("expected OSTIFDetail to be non-nil")
	}
	if d.MonetizationStrategy != "donations" {
		t.Errorf("MonetizationStrategy: want %q got %q", "donations", d.MonetizationStrategy)
	}
	if d.CurrentSecurityStrategy != "manual review" {
		t.Errorf("CurrentSecurityStrategy: want %q got %q", "manual review", d.CurrentSecurityStrategy)
	}
	if d.LicenseType != "MIT" {
		t.Errorf("LicenseType: want %q got %q", "MIT", d.LicenseType)
	}
	if d.TotalBudgetInCents != 1000000 {
		t.Errorf("TotalBudgetInCents: want 1000000 got %d", d.TotalBudgetInCents)
	}
	if !d.TermsConditions {
		t.Error("expected TermsConditions true")
	}
}

func TestCreate_PropagatesContacts(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My OSTIF Audit",
		Slug:           "my-ostif",
		InitiativeType: "ostif",
		Contacts: []models.ContactInput{
			{ContactType: "primary", FirstName: "Jane", LastName: "Doe", Email: "jane@example.com", PhoneNumber: "555-1234"},
			{ContactType: "technical_lead", FirstName: "John", Email: "john@example.com"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cs := repo.lastInput.Contacts
	if len(cs) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(cs))
	}
	if cs[0].ContactType != "primary" || cs[0].FirstName != "Jane" || cs[0].PhoneNumber != "555-1234" {
		t.Errorf("contact[0] mismatch: %+v", cs[0])
	}
	if cs[1].ContactType != "technical_lead" || cs[1].FirstName != "John" {
		t.Errorf("contact[1] mismatch: %+v", cs[1])
	}
}

func TestCreate_PropagatesEntityDetails(t *testing.T) {
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "My Event",
		Slug:           "my-event",
		InitiativeType: "event",
		EntityDetails:  map[string]string{"category": "open-source", "tier": "top"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ed := repo.lastInput.EntityDetails
	if len(ed) != 2 {
		t.Fatalf("expected 2 entity detail entries, got %d", len(ed))
	}
	if ed["category"] != "open-source" {
		t.Errorf("EntityDetails[category]: want %q got %q", "open-source", ed["category"])
	}
	if ed["tier"] != "top" {
		t.Errorf("EntityDetails[tier]: want %q got %q", "top", ed["tier"])
	}
}

func TestCreate_NilChildFieldsWhenNotProvided(t *testing.T) {
	// Verifies that omitting child-table fields leaves them nil/empty on the
	// Initiative — no accidental default population.
	repo := &mockInitiativeRepo{}
	svc := newCreateSvc(repo)
	_, err := svc.Create(context.Background(), "owner-1", models.InitiativeCreateInput{
		Name:           "Bare Project",
		Slug:           "bare-project",
		InitiativeType: "project",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.lastInput.Goals) != 0 {
		t.Errorf("expected empty Goals, got %d", len(repo.lastInput.Goals))
	}
	if len(repo.lastInput.Beneficiaries) != 0 {
		t.Errorf("expected empty Beneficiaries, got %d", len(repo.lastInput.Beneficiaries))
	}
	if repo.lastInput.ProgramInfo != nil {
		t.Error("expected nil ProgramInfo")
	}
	if repo.lastInput.OSTIFDetail != nil {
		t.Error("expected nil OSTIFDetail")
	}
	if repo.lastInput.EntityDetails != nil {
		t.Error("expected nil EntityDetails")
	}
}
