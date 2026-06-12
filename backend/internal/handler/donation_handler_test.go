// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
	stripe "github.com/stripe/stripe-go/v85"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

// donationRepo is a configurable DonationRepository stub for donation handler tests.
type donationRepo struct {
	getByIDResult       *models.Donation
	getByIDErr          error
	listByInitiative    []models.Donation
	listByInitiativeErr error
	listByUserResult    []models.Donation
	listByUserErr       error
	createResult        *models.Donation
	createErr           error
	lastCreated         *models.Donation
	updateByPIIDErr     error
}

func (r *donationRepo) GetByID(_ context.Context, _ string) (*models.Donation, error) {
	return r.getByIDResult, r.getByIDErr
}
func (r *donationRepo) ListByInitiative(_ context.Context, _ string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	if r.listByInitiativeErr != nil {
		return nil, nil, r.listByInitiativeErr
	}
	meta := &models.PaginationMeta{Total: len(r.listByInitiative), Limit: 20, Offset: 0}
	return r.listByInitiative, meta, nil
}
func (r *donationRepo) ListByUser(_ context.Context, _ string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	if r.listByUserErr != nil {
		return nil, nil, r.listByUserErr
	}
	meta := &models.PaginationMeta{Total: len(r.listByUserResult), Limit: 20, Offset: 0}
	return r.listByUserResult, meta, nil
}
func (r *donationRepo) Create(_ context.Context, d *models.Donation) (*models.Donation, error) {
	r.lastCreated = d
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.createResult != nil {
		return r.createResult, nil
	}
	return d, nil
}
func (r *donationRepo) UpdateByPaymentIntentID(_ context.Context, _, _, _ string) error {
	return r.updateByPIIDErr
}

// donationInitiativeRepo is a minimal InitiativeRepository stub for donation tests.
type donationInitiativeRepo struct {
	initiative *models.Initiative
	getErr     error
	usersByIDs map[string]models.User
	orgsByIDs  map[string]models.Organization
}

func (r *donationInitiativeRepo) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return r.initiative, r.getErr
}
func (r *donationInitiativeRepo) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, domain.ErrInitiativeNotFound
}
func (r *donationInitiativeRepo) GetIDBySlug(_ context.Context, _ string) (string, error) {
	return "", domain.ErrInitiativeNotFound
}
func (r *donationInitiativeRepo) ResolveSlug(_ context.Context, _ string) (string, error) {
	return "", domain.ErrInitiativeNotFound
}
func (r *donationInitiativeRepo) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *donationInitiativeRepo) Create(_ context.Context, i *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	return i, nil
}
func (r *donationInitiativeRepo) Update(_ context.Context, i *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	return i, nil
}
func (r *donationInitiativeRepo) Delete(_ context.Context, _ string) error { return nil }
func (r *donationInitiativeRepo) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	if r.usersByIDs != nil {
		return r.usersByIDs, nil
	}
	return make(map[string]models.User), nil
}
func (r *donationInitiativeRepo) GetUsersByLegacyIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return make(map[string]models.User), nil
}
func (r *donationInitiativeRepo) UpdateStripeProductID(_ context.Context, _, _ string) error {
	return nil
}
func (r *donationInitiativeRepo) GetOwnerInfoBySlug(_ context.Context, _ string) (models.OwnerInfo, error) {
	return models.OwnerInfo{}, nil
}
func (r *donationInitiativeRepo) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	if r.orgsByIDs != nil {
		return r.orgsByIDs, nil
	}
	return make(map[string]models.Organization), nil
}

// donationUserRepo is a configurable UserRepository stub for donation tests.
type donationUserRepo struct {
	user *models.User
	err  error
}

func (r *donationUserRepo) GetByUsername(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}
func (r *donationUserRepo) GetByID(_ context.Context, _ string) (*models.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *donationUserRepo) Upsert(_ context.Context, u *models.User) (*models.User, error) {
	return u, nil
}
func (r *donationUserRepo) UpdateStripeInfo(_ context.Context, _, _, _ string) error {
	return nil
}
func (r *donationUserRepo) ClearStripePaymentMethod(_ context.Context, _ string) error {
	return nil
}

// donationStripeClient is a configurable StripeClient stub for donation tests.
type donationStripeClient struct {
	onCreatePaymentIntent func(ctx context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error)
}

func (c *donationStripeClient) GetProduct(_ context.Context, _ string) (*models.StripeProduct, error) {
	return nil, nil
}
func (c *donationStripeClient) CreateProduct(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *donationStripeClient) DeleteProduct(_ context.Context, _ string) error {
	return nil
}
func (c *donationStripeClient) CreatePaymentIntent(ctx context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	if c.onCreatePaymentIntent != nil {
		return c.onCreatePaymentIntent(ctx, req)
	}
	return nil, nil
}
func (c *donationStripeClient) CreateSubscription(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	return nil, nil
}
func (c *donationStripeClient) CancelSubscription(_ context.Context, _ string) error {
	return nil
}
func (c *donationStripeClient) UpdatePaymentIntentMetadata(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
func (c *donationStripeClient) ConstructWebhookEvent(_ []byte, _, _ string) (stripe.Event, error) {
	return stripe.Event{}, nil
}
func (c *donationStripeClient) CreateCustomer(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *donationStripeClient) CreateSetupIntent(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (c *donationStripeClient) AttachPaymentMethod(_ context.Context, _, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *donationStripeClient) GetPaymentMethod(_ context.Context, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *donationStripeClient) DetachPaymentMethod(_ context.Context, _ string) error {
	return nil
}
func (c *donationStripeClient) GetOrCreatePrice(_ context.Context, _, _ string, _ int64, _, _ string) (string, error) {
	return "", nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// newDonationHandler builds a DonationHandler wired to the given repos and stripe client.
func newDonationHandler(
	donRepo *donationRepo,
	initRepo *donationInitiativeRepo,
	userRepo *donationUserRepo,
	stripeClient *donationStripeClient,
) *DonationHandler {
	svc := service.NewDonationService(donRepo, initRepo, userRepo, stripeClient)
	return NewDonationHandler(svc)
}

// donationListReq builds a GET request to /v1/initiatives/{id}/donations with optional principal.
func donationListReq(initiativeID string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID+"/donations", nil)
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// donationListForUserReq builds a GET request to /v1/me/donations with optional principal.
func donationListForUserReq(principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/me/donations", nil)
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// donationCreateReq builds a POST request to /v1/initiatives/{id}/donations.
func donationCreateReq(initiativeID string, idempotencyKey string, body string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/v1/initiatives/"+initiativeID+"/donations", strings.NewReader(body))
	r.Header.Set("Idempotency-Key", idempotencyKey)
	r.Header.Set("Content-Type", "application/json")
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// setChiParam adds a URL parameter to the request via Chi route context.
func setChiParam(r *http.Request, paramName string, paramValue string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(paramName, paramValue)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestDonationList_ReturnsOK(t *testing.T) {
	initiativeID := "init-123"
	donRepo := &donationRepo{
		listByInitiative: []models.Donation{
			{ID: "don-1", CurrentAmountCents: 1000},
		},
	}
	h := newDonationHandler(donRepo, &donationInitiativeRepo{}, &donationUserRepo{}, &donationStripeClient{})

	req := donationListReq(initiativeID, nil)
	req = setChiParam(req, "id", initiativeID)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body struct {
		Data []models.DonationSummary `json:"data"`
		Meta *models.PaginationMeta   `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Errorf("expected 1 donation, got %d", len(body.Data))
	}
}

func TestDonationList_ServiceError_Returns500(t *testing.T) {
	initiativeID := "init-123"
	donRepo := &donationRepo{
		listByInitiativeErr: errors.New("db error"),
	}
	h := newDonationHandler(donRepo, &donationInitiativeRepo{}, &donationUserRepo{}, &donationStripeClient{})

	req := donationListReq(initiativeID, nil)
	req = setChiParam(req, "id", initiativeID)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestDonationListForUser_NoPrincipal_Returns401(t *testing.T) {
	donRepo := &donationRepo{}
	h := newDonationHandler(donRepo, &donationInitiativeRepo{}, &donationUserRepo{}, &donationStripeClient{})

	req := donationListForUserReq(nil)
	w := httptest.NewRecorder()

	h.ListForUser(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDonationListForUser_ReturnsOwnDonations(t *testing.T) {
	username := "testuser"
	userID := "user-123"
	donRepo := &donationRepo{
		listByUserResult: []models.Donation{},
	}
	userRepo := &donationUserRepo{
		user: &models.User{
			ID:       userID,
			Username: username,
		},
	}
	h := newDonationHandler(donRepo, &donationInitiativeRepo{}, userRepo, &donationStripeClient{})

	req := donationListForUserReq(&models.Principal{Username: username})
	w := httptest.NewRecorder()

	h.ListForUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body struct {
		Data []models.Donation      `json:"data"`
		Meta *models.PaginationMeta `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 0 {
		t.Errorf("expected empty list, got %d items", len(body.Data))
	}
}

func TestDonationCreate_NoPrincipal_Returns401(t *testing.T) {
	initiativeID := "init-123"
	donRepo := &donationRepo{}
	h := newDonationHandler(donRepo, &donationInitiativeRepo{}, &donationUserRepo{}, &donationStripeClient{})

	req := donationCreateReq(initiativeID, "key-1", `{"amount_cents":1000,"stripe_payment_method_id":"pm_xxx"}`, nil)
	req = setChiParam(req, "id", initiativeID)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDonationCreate_MissingIdempotencyKey_Returns400(t *testing.T) {
	initiativeID := "init-123"
	principal := &models.Principal{Username: "testuser"}
	donRepo := &donationRepo{}
	h := newDonationHandler(donRepo, &donationInitiativeRepo{}, &donationUserRepo{}, &donationStripeClient{})

	r := httptest.NewRequest(http.MethodPost, "/v1/initiatives/"+initiativeID+"/donations",
		strings.NewReader(`{"amount_cents":1000,"stripe_payment_method_id":"pm_xxx"}`))
	r.Header.Set("Content-Type", "application/json")
	// Deliberately omit Idempotency-Key header
	r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	r = setChiParam(r, "id", initiativeID)
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDonationCreate_InvalidBody_Returns400(t *testing.T) {
	initiativeID := "init-123"
	principal := &models.Principal{Username: "testuser"}
	donRepo := &donationRepo{}
	h := newDonationHandler(donRepo, &donationInitiativeRepo{}, &donationUserRepo{}, &donationStripeClient{})

	req := donationCreateReq(initiativeID, "key-1", `invalid json`, principal)
	req = setChiParam(req, "id", initiativeID)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDonationCreate_ServiceError_Returns500(t *testing.T) {
	initiativeID := "init-123"
	principal := &models.Principal{Username: "testuser"}

	// Initiative repo returns an error to simulate database failure
	initiativeRepo := &donationInitiativeRepo{
		getErr: errors.New("database error"),
	}
	h := newDonationHandler(&donationRepo{}, initiativeRepo, &donationUserRepo{}, &donationStripeClient{})

	req := donationCreateReq(initiativeID, "key-1", `{"amount_cents":1000,"stripe_payment_method_id":"pm_xxx"}`, principal)
	req = setChiParam(req, "id", initiativeID)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestDonationCreate_Success_Returns201(t *testing.T) {
	initiativeID := "init-123"
	userID := "user-123"
	username := "testuser"
	stripeCustomerID := "cus_xxx"
	stripeProductID := "prod_yyy"

	principal := &models.Principal{Username: username}

	userRepo := &donationUserRepo{
		user: &models.User{
			ID:               userID,
			Username:         username,
			Email:            "test@example.com",
			StripeCustomerID: stripeCustomerID,
		},
	}
	initiativeRepo := &donationInitiativeRepo{
		initiative: &models.Initiative{
			ID:              initiativeID,
			StripeProductID: stripeProductID,
			Status:          "published",
			AcceptFunding:   true,
		},
	}
	donRepo := &donationRepo{
		createResult: &models.Donation{
			ID:                 "don-123",
			UserID:             userID,
			InitiativeID:       initiativeID,
			CurrentAmountCents: 1000,
			Status:             "pending",
		},
	}
	stripeClient := &donationStripeClient{
		onCreatePaymentIntent: func(ctx context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			return &models.PaymentIntent{
				ID:           "pi_xxx",
				ClientSecret: "pi_xxx_secret",
				Status:       "requires_payment_method",
			}, nil
		},
	}

	h := newDonationHandler(donRepo, initiativeRepo, userRepo, stripeClient)

	req := donationCreateReq(initiativeID, "key-1", `{"amount_cents":1000,"stripe_payment_method_id":"pm_xxx"}`, principal)
	req = setChiParam(req, "id", initiativeID)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var body models.Donation
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID != "don-123" {
		t.Errorf("expected donation ID don-123, got %s", body.ID)
	}
}
