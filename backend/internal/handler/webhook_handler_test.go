// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	stripe "github.com/stripe/stripe-go/v82"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// --- webhook-specific mocks ---

// wbStripeClient implements clients.StripeClient for webhook handler tests.
// Only ConstructWebhookEvent matters; all other methods are no-ops.
type wbStripeClient struct {
	onConstruct func(payload []byte, sig, secret string) (stripe.Event, error)
}

func (c *wbStripeClient) GetProduct(_ context.Context, _ string) (*models.StripeProduct, error) {
	return nil, nil
}
func (c *wbStripeClient) CreatePaymentIntent(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	return nil, nil
}
func (c *wbStripeClient) CreateSubscription(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	return nil, nil
}
func (c *wbStripeClient) CancelSubscription(_ context.Context, _ string) error { return nil }
func (c *wbStripeClient) ConstructWebhookEvent(payload []byte, sig, secret string) (stripe.Event, error) {
	if c.onConstruct != nil {
		return c.onConstruct(payload, sig, secret)
	}
	return stripe.Event{}, nil
}
func (c *wbStripeClient) CreateCustomer(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *wbStripeClient) CreateSetupIntent(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (c *wbStripeClient) AttachPaymentMethod(_ context.Context, _, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *wbStripeClient) GetPaymentMethod(_ context.Context, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *wbStripeClient) DetachPaymentMethod(_ context.Context, _ string) error { return nil }
func (c *wbStripeClient) GetOrCreatePrice(_ context.Context, _ string, _ int64, _ string) (string, error) {
	return "", nil
}

// Ensure interface is fully satisfied at compile time.
var _ clients.StripeClient = (*wbStripeClient)(nil)

// wbDonationRepo implements domain.DonationRepository for webhook tests.
type wbDonationRepo struct {
	onUpdateByPaymentIntentID func(ctx context.Context, piID, status, chargeID string) error
}

func (r *wbDonationRepo) GetByID(_ context.Context, _ string) (*models.Donation, error) {
	return nil, nil
}
func (r *wbDonationRepo) ListByInitiative(_ context.Context, _ string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *wbDonationRepo) ListByUser(_ context.Context, _ string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *wbDonationRepo) Create(_ context.Context, d *models.Donation) (*models.Donation, error) {
	return d, nil
}
func (r *wbDonationRepo) UpdateByPaymentIntentID(ctx context.Context, piID, status, chargeID string) error {
	if r.onUpdateByPaymentIntentID != nil {
		return r.onUpdateByPaymentIntentID(ctx, piID, status, chargeID)
	}
	return nil
}

// wbSubscriptionRepo implements domain.SubscriptionRepository for webhook tests.
type wbSubscriptionRepo struct {
	onUpdateByStripeSubscriptionID func(ctx context.Context, subID, status string) error
}

func (r *wbSubscriptionRepo) GetByID(_ context.Context, _ string) (*models.Subscription, error) {
	return nil, nil
}
func (r *wbSubscriptionRepo) ListByInitiative(_ context.Context, _ string, _ models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *wbSubscriptionRepo) ListByUser(_ context.Context, _ string, _ models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *wbSubscriptionRepo) Create(_ context.Context, s *models.Subscription) (*models.Subscription, error) {
	return s, nil
}
func (r *wbSubscriptionRepo) Update(_ context.Context, s *models.Subscription) (*models.Subscription, error) {
	return s, nil
}
func (r *wbSubscriptionRepo) UpdateByStripeSubscriptionID(ctx context.Context, subID, status string) error {
	if r.onUpdateByStripeSubscriptionID != nil {
		return r.onUpdateByStripeSubscriptionID(ctx, subID, status)
	}
	return nil
}

// discardLogger creates a slog.Logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestWebhookHandler wires up a WebhookHandler with the given mocks.
func newTestWebhookHandler(sc *wbStripeClient, dr *wbDonationRepo, sr *wbSubscriptionRepo) *WebhookHandler {
	return NewWebhookHandler(sc, dr, sr, "whsec_test", discardLogger(), false)
}

// postWebhook sends a simulated Stripe webhook POST to the handler.
func postWebhook(t *testing.T, h *WebhookHandler, sigHeader, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/stripe/webhook", bytes.NewBufferString(body))
	if sigHeader != "" {
		req.Header.Set("Stripe-Signature", sigHeader)
	}
	rr := httptest.NewRecorder()
	h.Handle(rr, req)
	return rr
}

// buildEvent builds a test stripe.Event with a JSON raw payload.
func buildEvent(eventType string, rawPayload string) stripe.Event {
	return stripe.Event{
		Type: stripe.EventType(eventType),
		Data: &stripe.EventData{Raw: json.RawMessage(rawPayload)},
	}
}

// --- security: missing / invalid signature ---

func TestWebhookHandler_MissingSignature_Rejects401(t *testing.T) {
	h := newTestWebhookHandler(&wbStripeClient{}, &wbDonationRepo{}, &wbSubscriptionRepo{})
	rr := postWebhook(t, h, "", `{}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestWebhookHandler_InvalidSignature_Rejects401(t *testing.T) {
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) {
			return stripe.Event{}, errors.New("signature mismatch")
		},
	}
	h := newTestWebhookHandler(sc, &wbDonationRepo{}, &wbSubscriptionRepo{})
	rr := postWebhook(t, h, "t=1,v1=bad", `{}`)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

// --- payment_intent.succeeded ---

func TestWebhookHandler_PaymentIntentSucceeded(t *testing.T) {
	var gotPIID, gotStatus, gotChargeID string

	dr := &wbDonationRepo{
		onUpdateByPaymentIntentID: func(_ context.Context, piID, status, chargeID string) error {
			gotPIID = piID
			gotStatus = status
			gotChargeID = chargeID
			return nil
		},
	}
	event := buildEvent("payment_intent.succeeded",
		`{"id":"pi_test_001","latest_charge":{"id":"ch_test_001"}}`)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, dr, &wbSubscriptionRepo{}), "t=1,v1=sig", `{}`)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if gotPIID != "pi_test_001" {
		t.Errorf("piID = %q, want pi_test_001", gotPIID)
	}
	if gotStatus != "succeeded" {
		t.Errorf("status = %q, want succeeded", gotStatus)
	}
	if gotChargeID != "ch_test_001" {
		t.Errorf("chargeID = %q, want ch_test_001", gotChargeID)
	}
}

// --- payment_intent.payment_failed ---

func TestWebhookHandler_PaymentIntentFailed(t *testing.T) {
	var gotPIID, gotStatus string

	dr := &wbDonationRepo{
		onUpdateByPaymentIntentID: func(_ context.Context, piID, status, _ string) error {
			gotPIID = piID
			gotStatus = status
			return nil
		},
	}
	event := buildEvent("payment_intent.payment_failed", `{"id":"pi_fail_001"}`)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, dr, &wbSubscriptionRepo{}), "t=1,v1=sig", `{}`)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if gotPIID != "pi_fail_001" {
		t.Errorf("piID = %q, want pi_fail_001", gotPIID)
	}
	if gotStatus != "failed" {
		t.Errorf("status = %q, want failed", gotStatus)
	}
}

// --- invoice.payment_succeeded ---

func TestWebhookHandler_InvoicePaymentSucceeded_ActivatesSubscription(t *testing.T) {
	var gotSubID, gotStatus string

	sr := &wbSubscriptionRepo{
		onUpdateByStripeSubscriptionID: func(_ context.Context, subID, status string) error {
			gotSubID = subID
			gotStatus = status
			return nil
		},
	}
	// Invoice JSON includes parent.subscription_details.subscription for stripe-go v82.
	invoiceJSON := `{"parent":{"subscription_details":{"subscription":{"id":"sub_test_001"}}}}`
	event := buildEvent("invoice.payment_succeeded", invoiceJSON)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, &wbDonationRepo{}, sr), "t=1,v1=sig", `{}`)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if gotSubID != "sub_test_001" {
		t.Errorf("subID = %q, want sub_test_001", gotSubID)
	}
	if gotStatus != "active" {
		t.Errorf("status = %q, want active", gotStatus)
	}
}

func TestWebhookHandler_InvoicePaymentSucceeded_NoSubscription_IsIgnored(t *testing.T) {
	// An invoice not related to a subscription must be ignored (no DB call).
	dbCalled := false
	sr := &wbSubscriptionRepo{
		onUpdateByStripeSubscriptionID: func(_ context.Context, _, _ string) error {
			dbCalled = true
			return nil
		},
	}
	event := buildEvent("invoice.payment_succeeded", `{"id":"in_standalone"}`)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, &wbDonationRepo{}, sr), "t=1,v1=sig", `{}`)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if dbCalled {
		t.Error("UpdateByStripeSubscriptionID must not be called for non-subscription invoices")
	}
}

// --- invoice.payment_failed ---

func TestWebhookHandler_InvoicePaymentFailed_MarksPastDue(t *testing.T) {
	var gotSubID, gotStatus string

	sr := &wbSubscriptionRepo{
		onUpdateByStripeSubscriptionID: func(_ context.Context, subID, status string) error {
			gotSubID = subID
			gotStatus = status
			return nil
		},
	}
	invoiceJSON := `{"parent":{"subscription_details":{"subscription":{"id":"sub_pastdue_001"}}}}`
	event := buildEvent("invoice.payment_failed", invoiceJSON)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, &wbDonationRepo{}, sr), "t=1,v1=sig", `{}`)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if gotSubID != "sub_pastdue_001" {
		t.Errorf("subID = %q, want sub_pastdue_001", gotSubID)
	}
	if gotStatus != "past_due" {
		t.Errorf("status = %q, want past_due", gotStatus)
	}
}

// --- customer.subscription.deleted ---

func TestWebhookHandler_SubscriptionDeleted_MarkesCanceled(t *testing.T) {
	var gotSubID, gotStatus string

	sr := &wbSubscriptionRepo{
		onUpdateByStripeSubscriptionID: func(_ context.Context, subID, status string) error {
			gotSubID = subID
			gotStatus = status
			return nil
		},
	}
	event := buildEvent("customer.subscription.deleted", `{"id":"sub_deleted_001"}`)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, &wbDonationRepo{}, sr), "t=1,v1=sig", `{}`)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if gotSubID != "sub_deleted_001" {
		t.Errorf("subID = %q, want sub_deleted_001", gotSubID)
	}
	if gotStatus != "canceled" {
		t.Errorf("status = %q, want canceled", gotStatus)
	}
}

// --- unknown event type ---

func TestWebhookHandler_UnknownEvent_Returns200(t *testing.T) {
	event := buildEvent("customer.updated", `{"id":"cus_001"}`)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, &wbDonationRepo{}, &wbSubscriptionRepo{}), "t=1,v1=sig", `{}`)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

// --- DB error returns 500 ---

func TestWebhookHandler_PaymentIntentSucceeded_DBError_Returns500(t *testing.T) {
	dr := &wbDonationRepo{
		onUpdateByPaymentIntentID: func(_ context.Context, _, _, _ string) error {
			return errors.New("db failure")
		},
	}
	event := buildEvent("payment_intent.succeeded",
		`{"id":"pi_err","latest_charge":{"id":"ch_err"}}`)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, dr, &wbSubscriptionRepo{}), "t=1,v1=sig", `{}`)
	if rr.Code != http.StatusInternalServerError {
		// ackUnimplemented=false → handler returns 500 on processing error so Stripe retries
		t.Errorf("status = %d, want 500 on DB error with ackUnimplemented=false", rr.Code)
	}
}

// --- not-found: no local row for Stripe ID → non-200 so Stripe retries ---

func TestWebhookHandler_PaymentIntentSucceeded_DonationNotFound_Returns500(t *testing.T) {
	dr := &wbDonationRepo{
		onUpdateByPaymentIntentID: func(_ context.Context, _, _, _ string) error {
			return domain.ErrDonationNotFound
		},
	}
	event := buildEvent("payment_intent.succeeded",
		`{"id":"pi_orphan","latest_charge":{"id":"ch_orphan"}}`)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, dr, &wbSubscriptionRepo{}), "t=1,v1=sig", `{}`)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 when donation row not found (so Stripe retries)", rr.Code)
	}
}

func TestWebhookHandler_InvoicePaymentSucceeded_SubscriptionNotFound_Returns500(t *testing.T) {
	sr := &wbSubscriptionRepo{
		onUpdateByStripeSubscriptionID: func(_ context.Context, _, _ string) error {
			return domain.ErrSubscriptionNotFound
		},
	}
	body := `{"parent":{"type":"subscription","subscription_details":{"subscription":{"id":"sub_orphan"}}}}`
	event := buildEvent("invoice.payment_succeeded", body)
	sc := &wbStripeClient{
		onConstruct: func(_ []byte, _ string, _ string) (stripe.Event, error) { return event, nil },
	}

	rr := postWebhook(t, newTestWebhookHandler(sc, &wbDonationRepo{}, sr), "t=1,v1=sig", `{}`)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 when subscription row not found (so Stripe retries)", rr.Code)
	}
}
