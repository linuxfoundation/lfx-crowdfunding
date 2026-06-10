// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
	stripe "github.com/stripe/stripe-go/v85"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

// paymentUserRepo is a configurable UserRepository stub for payment handler tests.
type paymentUserRepo struct {
	user *models.User
	err  error
}

func (r *paymentUserRepo) GetByUsername(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}
func (r *paymentUserRepo) GetByID(_ context.Context, _ string) (*models.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *paymentUserRepo) Upsert(_ context.Context, u *models.User) (*models.User, error) {
	return u, nil
}
func (r *paymentUserRepo) UpdateStripeInfo(_ context.Context, _, _, _ string) error {
	return nil
}
func (r *paymentUserRepo) ClearStripePaymentMethod(_ context.Context, _ string) error {
	return nil
}

// paymentStripeClient is a configurable StripeClient stub for payment handler tests.
type paymentStripeClient struct {
	onCreateSetupIntent    func(ctx context.Context, customerID string) (string, error)
	onAttachPaymentMethod  func(ctx context.Context, customerID, pmID string) (*models.CardDetails, error)
	onGetPaymentMethod     func(ctx context.Context, pmID string) (*models.CardDetails, error)
	onDetachPaymentMethod  func(ctx context.Context, pmID string) error
}

func (c *paymentStripeClient) GetProduct(_ context.Context, _ string) (*models.StripeProduct, error) {
	return nil, nil
}
func (c *paymentStripeClient) CreateProduct(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *paymentStripeClient) DeleteProduct(_ context.Context, _ string) error {
	return nil
}
func (c *paymentStripeClient) CreatePaymentIntent(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	return nil, nil
}
func (c *paymentStripeClient) CreateSubscription(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	return nil, nil
}
func (c *paymentStripeClient) CancelSubscription(_ context.Context, _ string) error {
	return nil
}
func (c *paymentStripeClient) UpdatePaymentIntentMetadata(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
func (c *paymentStripeClient) ConstructWebhookEvent(_ []byte, _, _ string) (stripe.Event, error) {
	return stripe.Event{}, nil
}
func (c *paymentStripeClient) CreateCustomer(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *paymentStripeClient) CreateSetupIntent(ctx context.Context, customerID string) (string, error) {
	if c.onCreateSetupIntent != nil {
		return c.onCreateSetupIntent(ctx, customerID)
	}
	return "", nil
}
func (c *paymentStripeClient) AttachPaymentMethod(ctx context.Context, customerID, pmID string) (*models.CardDetails, error) {
	if c.onAttachPaymentMethod != nil {
		return c.onAttachPaymentMethod(ctx, customerID, pmID)
	}
	return nil, nil
}
func (c *paymentStripeClient) GetPaymentMethod(ctx context.Context, pmID string) (*models.CardDetails, error) {
	if c.onGetPaymentMethod != nil {
		return c.onGetPaymentMethod(ctx, pmID)
	}
	return nil, nil
}
func (c *paymentStripeClient) DetachPaymentMethod(ctx context.Context, pmID string) error {
	if c.onDetachPaymentMethod != nil {
		return c.onDetachPaymentMethod(ctx, pmID)
	}
	return nil
}
func (c *paymentStripeClient) GetOrCreatePrice(_ context.Context, _, _ string, _ int64, _, _ string) (string, error) {
	return "", nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// newPaymentHandler builds a PaymentHandler wired to the given repos and stripe client.
func newPaymentHandler(userRepo *paymentUserRepo, stripeClient *paymentStripeClient) *PaymentHandler {
	svc := service.NewPaymentService(userRepo, stripeClient)
	return NewPaymentHandler(svc)
}

// paymentReq builds a request for payment endpoints with optional principal and body.
func paymentReq(method string, path string, body string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(method, path, io.NopCloser(strings.NewReader(body)))
	r.Header.Set("Content-Type", "application/json")
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCreateSetupIntent_NoPrincipal_Returns401(t *testing.T) {
	userRepo := &paymentUserRepo{}
	stripeClient := &paymentStripeClient{}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodPost, "/v1/me/setup-intent", "", nil)
	w := httptest.NewRecorder()

	h.CreateSetupIntent(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateSetupIntent_ProfileNotSynced_Returns400(t *testing.T) {
	userRepo := &paymentUserRepo{err: domain.ErrUserNotFound}
	stripeClient := &paymentStripeClient{}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodPost, "/v1/me/setup-intent", "", &models.Principal{Username: "testuser"})
	w := httptest.NewRecorder()

	h.CreateSetupIntent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateSetupIntent_Success_Returns201(t *testing.T) {
	username := "testuser"
	stripeCustomerID := "cus_xxx"
	clientSecret := "seti_secret_xxx"

	userRepo := &paymentUserRepo{
		user: &models.User{
			Username:         username,
			StripeCustomerID: stripeCustomerID,
		},
	}
	stripeClient := &paymentStripeClient{
		onCreateSetupIntent: func(ctx context.Context, customerID string) (string, error) {
			return clientSecret, nil
		},
	}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodPost, "/v1/me/setup-intent", "", &models.Principal{Username: username})
	w := httptest.NewRecorder()

	h.CreateSetupIntent(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["client_secret"] != clientSecret {
		t.Errorf("expected client_secret %q, got %q", clientSecret, body["client_secret"])
	}
}

func TestAttachPaymentMethod_NoPrincipal_Returns401(t *testing.T) {
	userRepo := &paymentUserRepo{}
	stripeClient := &paymentStripeClient{}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodPost, "/v1/me/payment-method", `{"payment_method_id":"pm_xxx"}`, nil)
	w := httptest.NewRecorder()

	h.AttachPaymentMethod(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAttachPaymentMethod_MissingBody_Returns400(t *testing.T) {
	userRepo := &paymentUserRepo{}
	stripeClient := &paymentStripeClient{}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodPost, "/v1/me/payment-method", `{}`, &models.Principal{Username: "testuser"})
	w := httptest.NewRecorder()

	h.AttachPaymentMethod(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAttachPaymentMethod_Success_Returns200(t *testing.T) {
	username := "testuser"
	stripeCustomerID := "cus_xxx"

	userRepo := &paymentUserRepo{
		user: &models.User{
			Username:         username,
			Email:            "test@example.com",
			StripeCustomerID: stripeCustomerID,
		},
	}
	stripeClient := &paymentStripeClient{
		onAttachPaymentMethod: func(ctx context.Context, customerID, pmID string) (*models.CardDetails, error) {
			return &models.CardDetails{
				PaymentMethodID: "pm_xxx",
				Brand:           "visa",
				LastFour:        "4242",
				ExpiryMonth:     12,
				ExpiryYear:      2025,
			}, nil
		},
	}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodPost, "/v1/me/payment-method",
		`{"payment_method_id":"pm_xxx"}`, &models.Principal{Username: username})
	w := httptest.NewRecorder()

	h.AttachPaymentMethod(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body models.CardDetails
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.LastFour != "4242" {
		t.Errorf("expected last_four 4242, got %s", body.LastFour)
	}
}

func TestGetPaymentAccount_NoPrincipal_Returns401(t *testing.T) {
	userRepo := &paymentUserRepo{}
	stripeClient := &paymentStripeClient{}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodGet, "/v1/me/payment-account", "", nil)
	w := httptest.NewRecorder()

	h.GetPaymentAccount(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetPaymentAccount_NotFound_Returns404(t *testing.T) {
	username := "testuser"
	userRepo := &paymentUserRepo{
		user: &models.User{
			Username: username,
			// No StripeDefaultPaymentMethod set
		},
	}
	stripeClient := &paymentStripeClient{
		onGetPaymentMethod: func(ctx context.Context, pmID string) (*models.CardDetails, error) {
			return nil, domain.ErrPaymentMethodNotFound
		},
	}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodGet, "/v1/me/payment-account", "", &models.Principal{Username: username})
	w := httptest.NewRecorder()

	h.GetPaymentAccount(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetPaymentAccount_Success_Returns200(t *testing.T) {
	username := "testuser"
	paymentMethodID := "pm_xxx"
	userRepo := &paymentUserRepo{
		user: &models.User{
			Username:              username,
			Email:                 "test@example.com",
			StripeDefaultPaymentMethod: paymentMethodID,
		},
	}
	stripeClient := &paymentStripeClient{
		onGetPaymentMethod: func(ctx context.Context, pmID string) (*models.CardDetails, error) {
			return &models.CardDetails{
				PaymentMethodID: "pm_xxx",
				Brand:           "visa",
				LastFour:        "4242",
				ExpiryMonth:     12,
				ExpiryYear:      2025,
			}, nil
		},
	}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodGet, "/v1/me/payment-account", "", &models.Principal{Username: username})
	w := httptest.NewRecorder()

	h.GetPaymentAccount(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body models.CardDetails
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.LastFour != "4242" {
		t.Errorf("expected last_four 4242, got %s", body.LastFour)
	}
}

func TestDeletePaymentMethod_NoPrincipal_Returns401(t *testing.T) {
	userRepo := &paymentUserRepo{}
	stripeClient := &paymentStripeClient{}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodDelete, "/v1/me/payment-method", "", nil)
	w := httptest.NewRecorder()

	h.DeletePaymentMethod(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDeletePaymentMethod_Success_Returns204(t *testing.T) {
	username := "testuser"

	userRepo := &paymentUserRepo{
		user: &models.User{
			Username:                   username,
			StripeDefaultPaymentMethod: "pm_xxx",
		},
	}
	stripeClient := &paymentStripeClient{
		onDetachPaymentMethod: func(ctx context.Context, pmID string) error {
			return nil
		},
	}
	h := newPaymentHandler(userRepo, stripeClient)

	req := paymentReq(http.MethodDelete, "/v1/me/payment-method", "", &models.Principal{Username: username})
	w := httptest.NewRecorder()

	h.DeletePaymentMethod(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
