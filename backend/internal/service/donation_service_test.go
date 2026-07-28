// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// summaryInitiativeRepo is a minimal InitiativeRepository stub for
// projectDonationSummaries tests. Only the lookup methods are configurable.
type summaryInitiativeRepo struct {
	onGetUsersByIDs         func(context.Context, []string) (map[string]models.User, error)
	onGetOrganizationsByIDs func(context.Context, []string) (map[string]models.Organization, error)
}

func (r *summaryInitiativeRepo) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, nil
}
func (r *summaryInitiativeRepo) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, nil
}
func (r *summaryInitiativeRepo) GetIDBySlug(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (r *summaryInitiativeRepo) ResolveSlug(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (r *summaryInitiativeRepo) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *summaryInitiativeRepo) Create(_ context.Context, i *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	return i, nil
}
func (r *summaryInitiativeRepo) Update(_ context.Context, i *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	return i, nil
}
func (r *summaryInitiativeRepo) Delete(_ context.Context, _ string) error { return nil }
func (r *summaryInitiativeRepo) GetUsersByIDs(ctx context.Context, ids []string) (map[string]models.User, error) {
	if r.onGetUsersByIDs != nil {
		return r.onGetUsersByIDs(ctx, ids)
	}
	return map[string]models.User{}, nil
}
func (r *summaryInitiativeRepo) GetUsersByLegacyIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return map[string]models.User{}, nil
}
func (r *summaryInitiativeRepo) UpdateStripeProductID(_ context.Context, _, _ string) error {
	return nil
}
func (r *summaryInitiativeRepo) GetOwnerInfoBySlug(_ context.Context, _ string) (models.OwnerInfo, error) {
	return models.OwnerInfo{}, nil
}
func (r *summaryInitiativeRepo) ListPublished(_ context.Context) ([]models.InitiativeSummary, error) {
	return nil, nil
}
func (r *summaryInitiativeRepo) GetOrganizationsByIDs(ctx context.Context, ids []string) (map[string]models.Organization, error) {
	if r.onGetOrganizationsByIDs != nil {
		return r.onGetOrganizationsByIDs(ctx, ids)
	}
	return map[string]models.Organization{}, nil
}
func (r *summaryInitiativeRepo) GetInitiativesByIDs(_ context.Context, _ []string) (map[string]*models.Initiative, error) {
	return map[string]*models.Initiative{}, nil
}

// --- projectDonationSummaries ---

func TestProjectDonationSummaries_Empty(t *testing.T) {
	result := projectDonationSummaries(context.Background(), &summaryInitiativeRepo{}, nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(result))
	}
}

func TestProjectDonationSummaries_IndividualDonor(t *testing.T) {
	now := time.Now()
	donations := []models.Donation{
		{ID: "d1", UserID: "u1", CurrentAmountCents: 1000, Status: "succeeded", Category: "general", CreatedOn: now},
	}
	repo := &summaryInitiativeRepo{
		onGetUsersByIDs: func(_ context.Context, ids []string) (map[string]models.User, error) {
			if len(ids) != 1 || ids[0] != "u1" {
				t.Errorf("unexpected userIDs: %v", ids)
			}
			return map[string]models.User{
				"u1": {ID: "u1", Name: "Alice", AvatarURL: "https://example.com/alice.png"},
			}, nil
		},
	}

	result := projectDonationSummaries(context.Background(), repo, donations)

	if len(result) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(result))
	}
	s := result[0]
	if s.ID != "d1" {
		t.Errorf("ID = %q, want d1", s.ID)
	}
	if s.AmountCents != 1000 {
		t.Errorf("AmountCents = %d, want 1000", s.AmountCents)
	}
	if s.DonorType != donorTypeIndividual {
		t.Errorf("DonorType = %q, want %q", s.DonorType, donorTypeIndividual)
	}
	if s.DonorName != "Alice" {
		t.Errorf("DonorName = %q, want Alice", s.DonorName)
	}
	if s.DonorAvatarURL != "https://example.com/alice.png" {
		t.Errorf("DonorAvatar = %q, want alice.png URL", s.DonorAvatarURL)
	}
}

func TestProjectDonationSummaries_OrganizationDonor(t *testing.T) {
	donations := []models.Donation{
		{ID: "d2", OrganizationID: "org1", CurrentAmountCents: 5000, Status: "succeeded"},
	}
	repo := &summaryInitiativeRepo{
		onGetOrganizationsByIDs: func(_ context.Context, ids []string) (map[string]models.Organization, error) {
			return map[string]models.Organization{
				"org1": {ID: "org1", Name: "Acme Corp", AvatarURL: "https://example.com/acme.png"},
			}, nil
		},
	}

	result := projectDonationSummaries(context.Background(), repo, donations)

	if len(result) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(result))
	}
	s := result[0]
	if s.DonorType != donorTypeOrganization {
		t.Errorf("DonorType = %q, want %q", s.DonorType, donorTypeOrganization)
	}
	if s.DonorName != "Acme Corp" {
		t.Errorf("DonorName = %q, want Acme Corp", s.DonorName)
	}
}

func TestProjectDonationSummaries_DeduplicatesUserIDs(t *testing.T) {
	lookupCount := 0
	donations := []models.Donation{
		{ID: "d1", UserID: "u1", CurrentAmountCents: 100},
		{ID: "d2", UserID: "u1", CurrentAmountCents: 200},
	}
	repo := &summaryInitiativeRepo{
		onGetUsersByIDs: func(_ context.Context, ids []string) (map[string]models.User, error) {
			lookupCount++
			if len(ids) != 1 {
				t.Errorf("expected 1 unique userID, got %d: %v", len(ids), ids)
			}
			return map[string]models.User{"u1": {ID: "u1", Name: "Bob"}}, nil
		},
	}

	result := projectDonationSummaries(context.Background(), repo, donations)

	if lookupCount != 1 {
		t.Errorf("GetUsersByIDs called %d times, want 1", lookupCount)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(result))
	}
	for _, s := range result {
		if s.DonorName != "Bob" {
			t.Errorf("DonorName = %q, want Bob", s.DonorName)
		}
	}
}

func TestProjectDonationSummaries_UserLookupError_DegradeGracefully(t *testing.T) {
	donations := []models.Donation{
		{ID: "d1", UserID: "u1", CurrentAmountCents: 500},
	}
	repo := &summaryInitiativeRepo{
		onGetUsersByIDs: func(_ context.Context, _ []string) (map[string]models.User, error) {
			return nil, errors.New("db timeout")
		},
	}

	result := projectDonationSummaries(context.Background(), repo, donations)

	if len(result) != 1 {
		t.Fatalf("expected 1 summary even on lookup error, got %d", len(result))
	}
	s := result[0]
	if s.ID != "d1" {
		t.Errorf("ID = %q, want d1", s.ID)
	}
	if s.DonorName != "" || s.DonorAvatarURL != "" {
		t.Errorf("expected empty donor info on lookup error, got name=%q avatar=%q", s.DonorName, s.DonorAvatarURL)
	}
	if s.DonorType != donorTypeIndividual {
		t.Errorf("DonorType = %q, want %q", s.DonorType, donorTypeIndividual)
	}
}

func TestProjectDonationSummaries_OrgLookupError_DegradeGracefully(t *testing.T) {
	donations := []models.Donation{
		{ID: "d1", OrganizationID: "org1", CurrentAmountCents: 999},
	}
	repo := &summaryInitiativeRepo{
		onGetOrganizationsByIDs: func(_ context.Context, _ []string) (map[string]models.Organization, error) {
			return nil, errors.New("connection reset")
		},
	}

	result := projectDonationSummaries(context.Background(), repo, donations)

	if len(result) != 1 {
		t.Fatalf("expected 1 summary even on lookup error, got %d", len(result))
	}
	s := result[0]
	if s.DonorType != donorTypeOrganization {
		t.Errorf("DonorType = %q, want %q", s.DonorType, donorTypeOrganization)
	}
	if s.DonorName != "" {
		t.Errorf("DonorName should be empty on lookup error, got %q", s.DonorName)
	}
}

func TestProjectDonationSummaries_UnknownUserID_NoName(t *testing.T) {
	donations := []models.Donation{
		{ID: "d1", UserID: "u-unknown", CurrentAmountCents: 100},
	}
	repo := &summaryInitiativeRepo{
		onGetUsersByIDs: func(_ context.Context, _ []string) (map[string]models.User, error) {
			return map[string]models.User{}, nil // user not in DB
		},
	}

	result := projectDonationSummaries(context.Background(), repo, donations)

	if len(result) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(result))
	}
	if result[0].DonorName != "" || result[0].DonorAvatarURL != "" {
		t.Errorf("expected empty name/avatar for unknown user, got name=%q avatar=%q", result[0].DonorName, result[0].DonorAvatarURL)
	}
}

func newDonationSvc(
	donRepo *testDonationRepo,
	initRepo *mockInitiativeRepo,
	userRepo *testUserRepo,
	stripe *configStripeClient,
) *DonationService {
	return NewDonationService(donRepo, initRepo, userRepo, stripe)
}

// --- ListByInitiative ---

func TestDonationService_ListByInitiative(t *testing.T) {
	donRepo := &testDonationRepo{
		onListByInitiative: func(_ context.Context, initiativeID string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
			if initiativeID != "init-1" {
				t.Errorf("ListByInitiative id = %q, want init-1", initiativeID)
			}
			return []models.Donation{
					{ID: "d1", UserID: "u1", CurrentAmountCents: 1000, Status: "succeeded"},
				},
				&models.PaginationMeta{Total: 1, Limit: 20, Offset: 0}, nil
		},
	}
	initRepo := &mockInitiativeRepo{}

	svc := newDonationSvc(donRepo, initRepo, &testUserRepo{}, &configStripeClient{})
	summaries, meta, err := svc.ListByInitiative(context.Background(), "init-1", models.DonationFilter{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != "d1" {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}
	if meta == nil || meta.Total != 1 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestDonationService_ListByInitiative_RepoError(t *testing.T) {
	repoErr := errors.New("query failed")
	donRepo := &testDonationRepo{
		onListByInitiative: func(_ context.Context, _ string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
			return nil, nil, repoErr
		},
	}

	svc := newDonationSvc(donRepo, &mockInitiativeRepo{}, &testUserRepo{}, &configStripeClient{})
	_, _, err := svc.ListByInitiative(context.Background(), "init-1", models.DonationFilter{})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

// --- ListByUser ---

func TestDonationService_ListByUser(t *testing.T) {
	donRepo := &testDonationRepo{
		onListByUser: func(_ context.Context, userID string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
			if userID != "user-uuid-1" {
				t.Errorf("ListByUser userID = %q, want user-uuid-1", userID)
			}
			return []models.Donation{{ID: "d1"}}, &models.PaginationMeta{Total: 1}, nil
		},
	}
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "user-uuid-1"}, nil
		},
	}

	svc := newDonationSvc(donRepo, &mockInitiativeRepo{}, userRepo, &configStripeClient{})
	donations, meta, err := svc.ListByUser(context.Background(), "alice", models.DonationFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(donations) != 1 || donations[0].ID != "d1" {
		t.Fatalf("unexpected donations: %+v", donations)
	}
	if meta == nil || meta.Total != 1 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestDonationService_ListByUser_UserNotFoundReturnsEmpty(t *testing.T) {
	// A user with no DB record has never donated — return an empty list, not an error.
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}

	svc := newDonationSvc(&testDonationRepo{}, &mockInitiativeRepo{}, userRepo, &configStripeClient{})
	donations, meta, err := svc.ListByUser(context.Background(), "ghost", models.DonationFilter{Limit: 10, Offset: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(donations) != 0 {
		t.Errorf("expected empty donations, got %d", len(donations))
	}
	if meta == nil || meta.Limit != 10 || meta.Offset != 5 {
		t.Errorf("expected meta echoing filter, got %+v", meta)
	}
}

func TestDonationService_ListByUser_UserLookupError(t *testing.T) {
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, errors.New("db down")
		},
	}

	svc := newDonationSvc(&testDonationRepo{}, &mockInitiativeRepo{}, userRepo, &configStripeClient{})
	_, _, err := svc.ListByUser(context.Background(), "alice", models.DonationFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDonationService_ListByUser_RepoError(t *testing.T) {
	repoErr := errors.New("query failed")
	donRepo := &testDonationRepo{
		onListByUser: func(_ context.Context, _ string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
			return nil, nil, repoErr
		},
	}
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "user-uuid-1"}, nil
		},
	}

	svc := newDonationSvc(donRepo, &mockInitiativeRepo{}, userRepo, &configStripeClient{})
	_, _, err := svc.ListByUser(context.Background(), "alice", models.DonationFilter{})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

// --- ListOrgDonations ---

func TestDonationService_ListOrgDonations(t *testing.T) {
	donRepo := &testDonationRepo{
		onListOrgDonations: func(_ context.Context) ([]models.OrgDonationRow, error) {
			return []models.OrgDonationRow{{OrganizationName: "Acme"}}, nil
		},
	}

	svc := newDonationSvc(donRepo, &mockInitiativeRepo{}, &testUserRepo{}, &configStripeClient{})
	rows, err := svc.ListOrgDonations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 || rows[0].OrganizationName != "Acme" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestDonationService_ListOrgDonations_RepoError(t *testing.T) {
	repoErr := errors.New("export failed")
	donRepo := &testDonationRepo{
		onListOrgDonations: func(_ context.Context) ([]models.OrgDonationRow, error) {
			return nil, repoErr
		},
	}

	svc := newDonationSvc(donRepo, &mockInitiativeRepo{}, &testUserRepo{}, &configStripeClient{})
	_, err := svc.ListOrgDonations(context.Background())
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

func acceptingInitiative() *mockInitiativeRepo {
	return &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", AcceptFunding: true, StripeProductID: "prod-test"}}
}

// --- input validation ---

func TestDonationService_Create_ZeroAmount(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 0})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDonationService_Create_NegativeAmount(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: -100})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDonationService_Create_MissingPaymentMethod(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 1000})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing PM, got %v", err)
	}
}

func TestDonationService_Create_MissingIdempotencyKey(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing idempotency key, got %v", err)
	}
}

func TestDonationService_Create_InitiativeNotFound(t *testing.T) {
	notFound := errors.New("initiative not found")
	initRepo := &mockInitiativeRepo{err: notFound}
	svc := newDonationSvc(&testDonationRepo{}, initRepo, &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-missing", "u1",
		models.DonationCreateInput{AmountCents: 100, StripePaymentMethodID: "pm_test", IdempotencyKey: "idem-key-1"})
	if !errors.Is(err, notFound) {
		t.Errorf("expected initiative-not-found error, got %v", err)
	}
}

func TestDonationService_Create_InitiativeNotAccepting(t *testing.T) {
	initRepo := &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", AcceptFunding: false}}
	svc := newDonationSvc(&testDonationRepo{}, initRepo, &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 500, StripePaymentMethodID: "pm_test", IdempotencyKey: "idem-key-1"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// --- happy path: new customer, immediate success (no 3DS) ---

func TestDonationService_Create_NewCustomerImmediateSuccess(t *testing.T) {
	customerCreated := false

	donRepo := &testDonationRepo{
		onCreate: func(_ context.Context, d *models.Donation) (*models.Donation, error) {
			// The donation must always be persisted as pending so the webhook
			// can perform the pending→succeeded transition and send emails,
			// even when Stripe confirms synchronously (no 3DS).
			if d.Status != models.DonationStatusPending {
				t.Errorf("repo.Create called with Status=%q, want %q", d.Status, models.DonationStatusPending)
			}
			return d, nil
		},
	}
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "u1", Email: "u1@test.example", StripeCustomerID: ""}, nil
		},
	}
	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, userID, email string) (string, error) {
			customerCreated = true
			return "cus_new", nil
		},
		onCreatePaymentIntent: func(_ context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			if req.CustomerID != "cus_new" {
				t.Errorf("PaymentIntent CustomerID = %q, want cus_new", req.CustomerID)
			}
			if req.AmountCents != 2000 {
				t.Errorf("AmountCents = %d, want 2000", req.AmountCents)
			}
			if req.IdempotencyKey != "idem-key-abc" {
				t.Errorf("IdempotencyKey = %q, want idem-key-abc", req.IdempotencyKey)
			}
			return &models.PaymentIntent{
				ID:     "pi_test",
				Status: "succeeded",
			}, nil
		},
	}

	svc := newDonationSvc(donRepo, acceptingInitiative(), userRepo, stripe)
	don, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 2000, StripePaymentMethodID: "pm_abc", IdempotencyKey: "idem-key-abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if don.Status != "succeeded" {
		t.Errorf("Status = %q, want \"succeeded\" (pi.Status overlaid on response)", don.Status)
	}
	if don.StripePaymentIntentID != "pi_test" {
		t.Errorf("StripePaymentIntentID = %q, want pi_test", don.StripePaymentIntentID)
	}
	if don.ClientSecret != "" {
		t.Errorf("ClientSecret should be empty for succeeded payment, got %q", don.ClientSecret)
	}
	if !customerCreated {
		t.Error("CreateCustomer was not called for new user")
	}
}

// --- 3DS required: existing customer, returns client_secret ---

func TestDonationService_Create_ExistingCustomer3DS(t *testing.T) {
	const existingCustomerID = "cus_existing"
	const wantSecret = "pi_test_secret_3ds"

	customerCreated := false

	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "u1", Email: "u1@test.example", StripeCustomerID: existingCustomerID}, nil
		},
	}
	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
			customerCreated = true
			return "cus_unexpected", nil
		},
		onCreatePaymentIntent: func(_ context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			if req.CustomerID != existingCustomerID {
				t.Errorf("CustomerID = %q, want %q", req.CustomerID, existingCustomerID)
			}
			return &models.PaymentIntent{
				ID:           "pi_3ds",
				Status:       "requires_action",
				ClientSecret: wantSecret,
			}, nil
		},
	}

	donRepo3DS := &testDonationRepo{
		onCreate: func(_ context.Context, d *models.Donation) (*models.Donation, error) {
			// Even for 3DS flows the donation must be persisted as pending;
			// requires_action is returned to the caller but not stored.
			if d.Status != models.DonationStatusPending {
				t.Errorf("repo.Create called with Status=%q, want %q", d.Status, models.DonationStatusPending)
			}
			return d, nil
		},
	}
	svc := newDonationSvc(donRepo3DS, acceptingInitiative(), userRepo, stripe)
	don, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 5000, StripePaymentMethodID: "pm_eu", IdempotencyKey: "idem-key-eu"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if don.Status != "requires_action" {
		t.Errorf("Status = %q, want \"requires_action\" (pi.Status overlaid on response)", don.Status)
	}
	if don.ClientSecret != wantSecret {
		t.Errorf("ClientSecret = %q, want %q", don.ClientSecret, wantSecret)
	}
	if customerCreated {
		t.Error("CreateCustomer must not be called when customer already exists")
	}
}

// --- stripe error propagation ---

func TestDonationService_Create_StripePaymentIntentError(t *testing.T) {
	stripeErr := errors.New("stripe error")

	svc := newDonationSvc(
		&testDonationRepo{},
		acceptingInitiative(),
		&testUserRepo{},
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				return "cus_1", nil
			},
			onCreatePaymentIntent: func(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
				return nil, stripeErr
			},
		},
	)

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test", IdempotencyKey: "idem-key-1"})
	if !errors.Is(err, stripeErr) {
		t.Errorf("error = %v, want to wrap %v", err, stripeErr)
	}
}

// --- DB error propagation ---

func TestDonationService_Create_UserRepoTransientError(t *testing.T) {
	dbErr := errors.New("connection reset")

	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, dbErr
		},
	}
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), userRepo, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test", IdempotencyKey: "idem-key-1"})
	if !errors.Is(err, dbErr) {
		t.Errorf("error = %v, want to wrap %v", err, dbErr)
	}
}

func TestDonationService_Create_UserNotFound_DescriptiveError(t *testing.T) {
	// When the user has not yet synced their profile, GetByUsername returns
	// ErrUserNotFound. The service converts this to ErrProfileNotSynced with
	// a PATCH /v1/me hint so the API response is actionable (maps to 400).
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), userRepo, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test", IdempotencyKey: "key-1"})

	if !errors.Is(err, domain.ErrProfileNotSynced) {
		t.Fatalf("expected ErrProfileNotSynced, got %v", err)
	}
	if !strings.Contains(err.Error(), "PATCH /v1/me") {
		t.Errorf("error should mention PATCH /v1/me, got: %v", err)
	}
}

func TestDonationService_Create_DBError(t *testing.T) {
	dbErr := errors.New("db write failed")

	donRepo := &testDonationRepo{
		onCreate: func(_ context.Context, _ *models.Donation) (*models.Donation, error) {
			return nil, dbErr
		},
	}
	svc := newDonationSvc(
		donRepo,
		acceptingInitiative(),
		&testUserRepo{},
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				return "cus_1", nil
			},
			onCreatePaymentIntent: func(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
				return &models.PaymentIntent{ID: "pi_1", Status: "succeeded"}, nil
			},
		},
	)

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test", IdempotencyKey: "idem-key-1"})
	if !errors.Is(err, dbErr) {
		t.Errorf("error = %v, want to wrap %v", err, dbErr)
	}
}

func TestDonationService_Create_EmptyEmail_RequiresProfileSync(t *testing.T) {
	// A legacy/migrated user row may exist without an email address.
	// Stripe requires a non-empty email, so the service must fail fast and
	// direct the caller to PATCH /v1/me before creating a Stripe customer.
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "u-uuid", Username: "u1", Email: ""}, nil
		},
	}
	customerCreated := false
	svc := newDonationSvc(
		&testDonationRepo{},
		acceptingInitiative(),
		userRepo,
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				customerCreated = true
				return "cus_new", nil
			},
		},
	)

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test", IdempotencyKey: "key-2"})

	if !errors.Is(err, domain.ErrProfileNotSynced) {
		t.Fatalf("expected ErrProfileNotSynced for empty email, got %v", err)
	}
	if !strings.Contains(err.Error(), "PATCH /v1/me") {
		t.Errorf("error should mention PATCH /v1/me, got: %v", err)
	}
	if customerCreated {
		t.Error("CreateCustomer must not be called when user email is empty")
	}
}

// ── donation_tier validation ───────────────────────────────────────────────────

// tieredInitiative returns a repo that serves an initiative in tiers mode with
// gold (min 2500000 cents) and silver (min 1000000 cents) configured.
func tieredInitiative() *mockInitiativeRepo {
	return &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID:            "init-1",
			AcceptFunding: true,
			DonationMode:  models.DonationModeTiers,
			SponsorshipTiers: []models.SponsorshipTier{
				{ID: "t1", Name: "gold", Minimum: 2500000, Enabled: true},
				{ID: "t2", Name: "silver", Minimum: 1000000, Enabled: true},
			},
		},
	}
}

func TestDonationService_Create_InvalidDonationTier(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, tieredInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", models.DonationCreateInput{
		AmountCents:           3000000,
		StripePaymentMethodID: "pm_test",
		IdempotencyKey:        "key-1",
		DonationTier:          "diamond",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for unknown tier name, got %v", err)
	}
}

func TestDonationService_Create_DonationTierOnOpenModeInitiative(t *testing.T) {
	openInit := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID:            "init-1",
			AcceptFunding: true,
			DonationMode:  models.DonationModeOpen,
		},
	}
	svc := newDonationSvc(&testDonationRepo{}, openInit, &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", models.DonationCreateInput{
		AmountCents:           500000,
		StripePaymentMethodID: "pm_test",
		IdempotencyKey:        "key-2",
		DonationTier:          "gold",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for tier on open-mode initiative, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "does not use sponsorship tiers") {
		t.Errorf("error message should mention sponsorship tiers, got: %v", err)
	}
}

func TestDonationService_Create_TierBelowMinimum(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, tieredInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", models.DonationCreateInput{
		AmountCents:           999999, // 1 cent below silver minimum
		StripePaymentMethodID: "pm_test",
		IdempotencyKey:        "key-3",
		DonationTier:          "silver",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput when amount is below tier minimum, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "below the minimum") {
		t.Errorf("error message should mention minimum, got: %v", err)
	}
}

func TestDonationService_Create_TierAtMinimum_Succeeds(t *testing.T) {
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, u string) (*models.User, error) {
			return &models.User{ID: "u1", Username: u, Email: u + "@test.example", StripeCustomerID: "cus_existing"}, nil
		},
	}
	stripe := &configStripeClient{
		onCreatePaymentIntent: func(_ context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			return &models.PaymentIntent{ID: "pi_ok", Status: "succeeded"}, nil
		},
	}
	svc := newDonationSvc(&testDonationRepo{}, tieredInitiative(), userRepo, stripe)

	_, err := svc.Create(context.Background(), "init-1", "u1", models.DonationCreateInput{
		AmountCents:           1000000, // exactly at silver minimum
		StripePaymentMethodID: "pm_test",
		IdempotencyKey:        "key-4",
		DonationTier:          "silver",
	})
	if err != nil {
		t.Fatalf("unexpected error at exactly the tier minimum: %v", err)
	}
}

func TestDonationService_Create_DonationTierStoredOnDonation(t *testing.T) {
	var storedTier string
	donRepo := &testDonationRepo{
		onCreate: func(_ context.Context, d *models.Donation) (*models.Donation, error) {
			storedTier = d.DonationTier
			return d, nil
		},
	}
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, u string) (*models.User, error) {
			return &models.User{ID: "u1", Username: u, Email: u + "@test.example", StripeCustomerID: "cus_existing"}, nil
		},
	}
	stripe := &configStripeClient{
		onCreatePaymentIntent: func(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			return &models.PaymentIntent{ID: "pi_tier", Status: "succeeded"}, nil
		},
	}
	svc := newDonationSvc(donRepo, tieredInitiative(), userRepo, stripe)

	_, err := svc.Create(context.Background(), "init-1", "u1", models.DonationCreateInput{
		AmountCents:           2500000,
		StripePaymentMethodID: "pm_test",
		IdempotencyKey:        "key-5",
		DonationTier:          "gold",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if storedTier != "gold" {
		t.Errorf("DonationTier stored = %q, want \"gold\"", storedTier)
	}
}

func TestDonationService_Create_NoTier_Succeeds(t *testing.T) {
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, u string) (*models.User, error) {
			return &models.User{ID: "u1", Username: u, Email: u + "@test.example", StripeCustomerID: "cus_existing"}, nil
		},
	}
	stripe := &configStripeClient{
		onCreatePaymentIntent: func(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			return &models.PaymentIntent{ID: "pi_notier", Status: "succeeded"}, nil
		},
	}
	// Donation without a tier on a tiers-mode initiative is valid.
	svc := newDonationSvc(&testDonationRepo{}, tieredInitiative(), userRepo, stripe)

	don, err := svc.Create(context.Background(), "init-1", "u1", models.DonationCreateInput{
		AmountCents:           100,
		StripePaymentMethodID: "pm_test",
		IdempotencyKey:        "key-6",
	})
	if err != nil {
		t.Fatalf("unexpected error for no-tier donation: %v", err)
	}
	if don.DonationTier != "" {
		t.Errorf("DonationTier should be empty when not set, got %q", don.DonationTier)
	}
}

func TestDonationService_Create_TierNotConfiguredOnInitiative(t *testing.T) {
	// Initiative only has silver, not gold.
	initRepo := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID:            "init-1",
			AcceptFunding: true,
			DonationMode:  models.DonationModeTiers,
			SponsorshipTiers: []models.SponsorshipTier{
				{ID: "t1", Name: "silver", Minimum: 1000000, Enabled: true},
			},
		},
	}
	svc := newDonationSvc(&testDonationRepo{}, initRepo, &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", models.DonationCreateInput{
		AmountCents:           2000000,
		StripePaymentMethodID: "pm_test",
		IdempotencyKey:        "key-notfound",
		DonationTier:          "gold", // valid name but not configured on this initiative
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for tier not on initiative, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error message should mention 'not configured', got: %v", err)
	}
}

func TestProjectDonationSummaries_PreservesDonationTier(t *testing.T) {
	donations := []models.Donation{
		{ID: "d1", UserID: "u1", CurrentAmountCents: 2500000, DonationTier: "gold"},
		{ID: "d2", UserID: "u2", CurrentAmountCents: 500, DonationTier: ""},
	}

	result := projectDonationSummaries(context.Background(), &summaryInitiativeRepo{}, donations)

	if len(result) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(result))
	}
	if result[0].DonationTier != "gold" {
		t.Errorf("summary[0].DonationTier = %q, want \"gold\"", result[0].DonationTier)
	}
	if result[1].DonationTier != "" {
		t.Errorf("summary[1].DonationTier = %q, want empty", result[1].DonationTier)
	}
}
