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

// subscriptionRepo is a configurable SubscriptionRepository stub for subscription handler tests.
type subscriptionRepo struct {
	getByIDResult           *models.Subscription
	getByIDErr              error
	getActiveByUserAndInit  *models.Subscription
	getActiveErr            error
	listByInitiative        []models.Subscription
	listByInitiativeErr     error
	listByUserResult        []models.Subscription
	listByUserErr           error
	createResult            *models.Subscription
	createErr               error
	lastCreated             *models.Subscription
	updateErr               error
	lastUpdated             *models.Subscription
}

func (r *subscriptionRepo) GetByID(_ context.Context, _ string) (*models.Subscription, error) {
	return r.getByIDResult, r.getByIDErr
}
func (r *subscriptionRepo) GetActiveByUserAndInitiative(_ context.Context, _, _ string) (*models.Subscription, error) {
	if r.getActiveErr != nil {
		return nil, r.getActiveErr
	}
	return r.getActiveByUserAndInit, nil
}
func (r *subscriptionRepo) ListByInitiative(_ context.Context, _ string, _ models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	if r.listByInitiativeErr != nil {
		return nil, nil, r.listByInitiativeErr
	}
	meta := &models.PaginationMeta{Total: len(r.listByInitiative), Limit: 20, Offset: 0}
	return r.listByInitiative, meta, nil
}
func (r *subscriptionRepo) ListByUser(_ context.Context, _ string, _ models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	if r.listByUserErr != nil {
		return nil, nil, r.listByUserErr
	}
	meta := &models.PaginationMeta{Total: len(r.listByUserResult), Limit: 20, Offset: 0}
	return r.listByUserResult, meta, nil
}
func (r *subscriptionRepo) Create(_ context.Context, s *models.Subscription) (*models.Subscription, error) {
	r.lastCreated = s
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.createResult != nil {
		return r.createResult, nil
	}
	return s, nil
}
func (r *subscriptionRepo) Update(_ context.Context, s *models.Subscription) (*models.Subscription, error) {
	r.lastUpdated = s
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	return s, nil
}
func (r *subscriptionRepo) UpdateByStripeSubscriptionID(_ context.Context, _, _ string) error {
	return r.updateErr
}

// subscriptionInitiativeRepo is a minimal InitiativeRepository stub for subscription tests.
type subscriptionInitiativeRepo struct {
	initiative      *models.Initiative
	getErr          error
	usersByIDs      map[string]models.User
	orgsByIDs       map[string]models.Organization
}

func (r *subscriptionInitiativeRepo) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return r.initiative, r.getErr
}
func (r *subscriptionInitiativeRepo) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, domain.ErrInitiativeNotFound
}
func (r *subscriptionInitiativeRepo) GetIDBySlug(_ context.Context, _ string) (string, error) {
	return "", domain.ErrInitiativeNotFound
}
func (r *subscriptionInitiativeRepo) ResolveSlug(_ context.Context, _ string) (string, error) {
	return "", domain.ErrInitiativeNotFound
}
func (r *subscriptionInitiativeRepo) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *subscriptionInitiativeRepo) Create(_ context.Context, i *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	return i, nil
}
func (r *subscriptionInitiativeRepo) Update(_ context.Context, i *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	return i, nil
}
func (r *subscriptionInitiativeRepo) Delete(_ context.Context, _ string) error { return nil }
func (r *subscriptionInitiativeRepo) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	if r.usersByIDs != nil {
		return r.usersByIDs, nil
	}
	return make(map[string]models.User), nil
}
func (r *subscriptionInitiativeRepo) UpdateStripeProductID(_ context.Context, _, _ string) error {
	return nil
}
func (r *subscriptionInitiativeRepo) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	if r.orgsByIDs != nil {
		return r.orgsByIDs, nil
	}
	return make(map[string]models.Organization), nil
}

// subscriptionUserRepo is a configurable UserRepository stub for subscription tests.
type subscriptionUserRepo struct {
	user *models.User
	err  error
}

func (r *subscriptionUserRepo) GetByUsername(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}
func (r *subscriptionUserRepo) GetByID(_ context.Context, _ string) (*models.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *subscriptionUserRepo) Upsert(_ context.Context, u *models.User) (*models.User, error) {
	return u, nil
}
func (r *subscriptionUserRepo) UpdateStripeInfo(_ context.Context, _, _, _ string) error {
	return nil
}
func (r *subscriptionUserRepo) ClearStripePaymentMethod(_ context.Context, _ string) error {
	return nil
}

// subscriptionStripeClient is a configurable StripeClient stub for subscription tests.
type subscriptionStripeClient struct {
	onGetOrCreatePrice    func(ctx context.Context, productID, initiativeID string, amount int64, frequency, idempotencyKey string) (string, error)
	onCreateSubscription  func(ctx context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error)
	onCancelSubscription  func(ctx context.Context, subscriptionID string) error
}

func (c *subscriptionStripeClient) GetProduct(_ context.Context, _ string) (*models.StripeProduct, error) {
	return nil, nil
}
func (c *subscriptionStripeClient) CreateProduct(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *subscriptionStripeClient) DeleteProduct(_ context.Context, _ string) error {
	return nil
}
func (c *subscriptionStripeClient) CreatePaymentIntent(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	return nil, nil
}
func (c *subscriptionStripeClient) CreateSubscription(ctx context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	if c.onCreateSubscription != nil {
		return c.onCreateSubscription(ctx, req)
	}
	return nil, nil
}
func (c *subscriptionStripeClient) CancelSubscription(ctx context.Context, subscriptionID string) error {
	if c.onCancelSubscription != nil {
		return c.onCancelSubscription(ctx, subscriptionID)
	}
	return nil
}
func (c *subscriptionStripeClient) UpdatePaymentIntentMetadata(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
func (c *subscriptionStripeClient) ConstructWebhookEvent(_ []byte, _, _ string) (stripe.Event, error) {
	return stripe.Event{}, nil
}
func (c *subscriptionStripeClient) CreateCustomer(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *subscriptionStripeClient) CreateSetupIntent(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (c *subscriptionStripeClient) AttachPaymentMethod(_ context.Context, _, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *subscriptionStripeClient) GetPaymentMethod(_ context.Context, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *subscriptionStripeClient) DetachPaymentMethod(_ context.Context, _ string) error {
	return nil
}
func (c *subscriptionStripeClient) GetOrCreatePrice(ctx context.Context, productID, initiativeID string, amount int64, frequency, idempotencyKey string) (string, error) {
	if c.onGetOrCreatePrice != nil {
		return c.onGetOrCreatePrice(ctx, productID, initiativeID, amount, frequency, idempotencyKey)
	}
	return "", nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// newSubscriptionHandler builds a SubscriptionHandler wired to the given repos and stripe client.
func newSubscriptionHandler(
	subRepo *subscriptionRepo,
	initRepo *subscriptionInitiativeRepo,
	userRepo *subscriptionUserRepo,
	stripeClient *subscriptionStripeClient,
) *SubscriptionHandler {
	svc := service.NewSubscriptionService(subRepo, initRepo, userRepo, stripeClient)
	return NewSubscriptionHandler(svc)
}

// subscriptionListReq builds a GET request to /v1/initiatives/{id}/subscriptions.
func subscriptionListReq(initiativeID string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID+"/subscriptions", nil)
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// subscriptionListForUserReq builds a GET request to /v1/me/subscriptions.
func subscriptionListForUserReq(principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/me/subscriptions", nil)
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// subscriptionCreateReq builds a POST request to /v1/initiatives/{id}/subscriptions.
func subscriptionCreateReq(initiativeID string, idempotencyKey string, body string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/v1/initiatives/"+initiativeID+"/subscriptions",
		strings.NewReader(body))
	r.Header.Set("Idempotency-Key", idempotencyKey)
	r.Header.Set("Content-Type", "application/json")
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// subscriptionCancelReq builds a DELETE request to /v1/subscriptions/{id}.
func subscriptionCancelReq(subscriptionID string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodDelete, "/v1/subscriptions/"+subscriptionID, nil)
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestSubscriptionList_ReturnsOK(t *testing.T) {
	initiativeID := "init-123"
	subRepo := &subscriptionRepo{
		listByInitiative: []models.Subscription{
			{ID: "sub-1", CurrentAmountCents: 1000},
		},
	}
	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, &subscriptionUserRepo{}, &subscriptionStripeClient{})

	req := subscriptionListReq(initiativeID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", initiativeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body struct {
		Data []models.Subscription `json:"data"`
		Meta *models.PaginationMeta `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(body.Data))
	}
}

func TestSubscriptionList_ServiceError_Returns500(t *testing.T) {
	initiativeID := "init-123"
	subRepo := &subscriptionRepo{
		listByInitiativeErr: errors.New("db error"),
	}
	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, &subscriptionUserRepo{}, &subscriptionStripeClient{})

	req := subscriptionListReq(initiativeID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", initiativeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestSubscriptionCreate_NoPrincipal_Returns401(t *testing.T) {
	initiativeID := "init-123"
	subRepo := &subscriptionRepo{}
	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, &subscriptionUserRepo{}, &subscriptionStripeClient{})

	req := subscriptionCreateReq(initiativeID, "key-1", `{"amount_cents":1000,"frequency":"month","stripe_payment_method_id":"pm_xxx"}`, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", initiativeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSubscriptionCreate_MissingIdempotencyKey_Returns400(t *testing.T) {
	initiativeID := "init-123"
	principal := &models.Principal{Username: "testuser"}
	subRepo := &subscriptionRepo{}
	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, &subscriptionUserRepo{}, &subscriptionStripeClient{})

	r := httptest.NewRequest(http.MethodPost, "/v1/initiatives/"+initiativeID+"/subscriptions",
		strings.NewReader(`{"amount_cents":1000,"frequency":"month","stripe_payment_method_id":"pm_xxx"}`))
	r.Header.Set("Content-Type", "application/json")
	// Deliberately omit Idempotency-Key header
	r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", initiativeID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubscriptionCreate_InvalidBody_Returns400(t *testing.T) {
	initiativeID := "init-123"
	principal := &models.Principal{Username: "testuser"}
	subRepo := &subscriptionRepo{}
	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, &subscriptionUserRepo{}, &subscriptionStripeClient{})

	req := subscriptionCreateReq(initiativeID, "key-1", `invalid json`, principal)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", initiativeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubscriptionCreate_Success_Returns201(t *testing.T) {
	initiativeID := "init-123"
	userID := "user-123"
	username := "testuser"
	stripeCustomerID := "cus_xxx"
	stripeProductID := "prod_yyy"
	priceID := "price_zzz"
	subscriptionID := "sub_xxx"

	principal := &models.Principal{Username: username}

	userRepo := &subscriptionUserRepo{
		user: &models.User{
			ID:               userID,
			Username:         username,
			Email:            "test@example.com",
			StripeCustomerID: stripeCustomerID,
		},
	}
	initiativeRepo := &subscriptionInitiativeRepo{
		initiative: &models.Initiative{
			ID:              initiativeID,
			StripeProductID: stripeProductID,
			Status:          "published",
			AcceptFunding:   true,
		},
	}
	subRepo := &subscriptionRepo{
		getActiveErr: domain.ErrSubscriptionNotFound,
		createResult: &models.Subscription{
			ID:                   "sub-123",
			UserID:               userID,
			InitiativeID:         initiativeID,
			CurrentAmountCents:   1000,
			Frequency:            "month",
			Status:               "incomplete",
			StripeSubscriptionID: subscriptionID,
		},
	}
	stripeClient := &subscriptionStripeClient{
		onGetOrCreatePrice: func(ctx context.Context, productID, initiativeID string, amount int64, frequency, idempotencyKey string) (string, error) {
			return priceID, nil
		},
		onCreateSubscription: func(ctx context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
			return &models.StripeSubscriptionResult{
				SubscriptionID: subscriptionID,
				ClientSecret:   "pi_xxx_secret",
				Status:         "incomplete",
			}, nil
		},
	}

	h := newSubscriptionHandler(subRepo, initiativeRepo, userRepo, stripeClient)

	req := subscriptionCreateReq(initiativeID, "key-1",
		`{"amount_cents":1000,"frequency":"month","stripe_payment_method_id":"pm_xxx"}`, principal)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", initiativeID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var body models.Subscription
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID != "sub-123" {
		t.Errorf("expected subscription ID sub-123, got %s", body.ID)
	}
}

func TestSubscriptionListForUser_NoPrincipal_Returns401(t *testing.T) {
	subRepo := &subscriptionRepo{}
	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, &subscriptionUserRepo{}, &subscriptionStripeClient{})

	req := subscriptionListForUserReq(nil)
	w := httptest.NewRecorder()

	h.ListForUser(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSubscriptionListForUser_ReturnsOwnSubs(t *testing.T) {
	username := "testuser"
	userID := "user-123"
	subRepo := &subscriptionRepo{
		listByUserResult: []models.Subscription{},
	}
	userRepo := &subscriptionUserRepo{
		user: &models.User{
			ID:       userID,
			Username: username,
		},
	}
	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, userRepo, &subscriptionStripeClient{})

	req := subscriptionListForUserReq(&models.Principal{Username: username})
	w := httptest.NewRecorder()

	h.ListForUser(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body struct {
		Data []models.Subscription  `json:"data"`
		Meta *models.PaginationMeta `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 0 {
		t.Errorf("expected empty list, got %d items", len(body.Data))
	}
}

func TestSubscriptionCancel_NoPrincipal_Returns401(t *testing.T) {
	subscriptionID := "sub-123"
	subRepo := &subscriptionRepo{}
	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, &subscriptionUserRepo{}, &subscriptionStripeClient{})

	req := subscriptionCancelReq(subscriptionID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", subscriptionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Cancel(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSubscriptionCancel_Success_Returns204(t *testing.T) {
	subscriptionID := "sub-123"
	userID := "user-123"
	username := "testuser"
	stripeSubID := "stripe_sub_xxx"

	principal := &models.Principal{Username: username}

	userRepo := &subscriptionUserRepo{
		user: &models.User{
			ID:       userID,
			Username: username,
		},
	}
	subRepo := &subscriptionRepo{
		getByIDResult: &models.Subscription{
			ID:                   subscriptionID,
			UserID:               userID,
			StripeSubscriptionID: stripeSubID,
			Status:               "active",
		},
	}
	stripeClient := &subscriptionStripeClient{
		onCancelSubscription: func(ctx context.Context, subID string) error {
			return nil
		},
	}

	h := newSubscriptionHandler(subRepo, &subscriptionInitiativeRepo{}, userRepo, stripeClient)

	req := subscriptionCancelReq(subscriptionID, principal)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", subscriptionID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.Cancel(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
