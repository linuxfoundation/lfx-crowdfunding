// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v85"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/handler"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/db"
)

// intStripeClient implements clients.StripeClient for integration tests.
// Signature validation is intentionally bypassed — it is covered by the unit tests in
// webhook_handler_test.go. These integration tests focus on the handler→service→DB path.
type intStripeClient struct {
	onConstruct func(payload []byte, sig, secret string) (stripe.Event, error)
}

func (c *intStripeClient) GetProduct(_ context.Context, _ string) (*models.StripeProduct, error) {
	return nil, nil
}
func (c *intStripeClient) CreateProduct(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *intStripeClient) DeleteProduct(_ context.Context, _ string) error { return nil }
func (c *intStripeClient) CreatePaymentIntent(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	return nil, nil
}
func (c *intStripeClient) CreateSubscription(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	return nil, nil
}
func (c *intStripeClient) CancelSubscription(_ context.Context, _ string) error { return nil }
func (c *intStripeClient) GetSubscriptionCurrentPeriodEnd(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (c *intStripeClient) UpdatePaymentIntentMetadata(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
func (c *intStripeClient) ConstructWebhookEvent(payload []byte, sig, secret string) (stripe.Event, error) {
	if c.onConstruct != nil {
		return c.onConstruct(payload, sig, secret)
	}
	// For integration tests, parse the payload directly without signature validation
	// (signature validation is tested in the unit tests).
	// Explicitly populate Data.Raw from the data.object JSON so the handler's
	// json.Unmarshal(event.Data.Raw, &...) calls receive the correct bytes.
	var envelope struct {
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return stripe.Event{}, err
	}
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return stripe.Event{}, err
	}
	event.Data.Raw = envelope.Data.Object
	return event, nil
}
func (c *intStripeClient) CreateCustomer(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *intStripeClient) CreateSetupIntent(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (c *intStripeClient) AttachPaymentMethod(_ context.Context, _, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *intStripeClient) GetPaymentMethod(_ context.Context, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *intStripeClient) DetachPaymentMethod(_ context.Context, _ string) error { return nil }
func (c *intStripeClient) GetOrCreatePrice(_ context.Context, _, _ string, _ int64, _, _ string) (string, error) {
	return "", nil
}

var _ clients.StripeClient = (*intStripeClient)(nil)

// intLedgerClient is a no-op LedgerClient stub for integration tests.
type intLedgerClient struct {
	onPostTransaction func(ctx context.Context, txn clients.LedgerTransaction) error
}

func (c *intLedgerClient) GetBalance(_ context.Context, _ string) (*clients.LedgerBalance, error) {
	return nil, nil
}
func (c *intLedgerClient) GetAllBalances(_ context.Context) ([]models.LedgerRawBalance, error) {
	return nil, nil
}
func (c *intLedgerClient) GetTransactions(_ context.Context, _ clients.TransactionFilter) (*models.TransactionList, error) {
	return nil, nil
}
func (c *intLedgerClient) GetPlatformBalance(_ context.Context, _ int) (*clients.LedgerPlatformBalance, error) {
	return nil, nil
}
func (c *intLedgerClient) GetPlatformMonthly(_ context.Context, _ int) (*clients.LedgerPlatformMonthly, error) {
	return nil, nil
}
func (c *intLedgerClient) GetPlatformRecentDonations(_ context.Context) ([]clients.LedgerRecentDonation, error) {
	return nil, nil
}
func (c *intLedgerClient) GetOrgDonations(_ context.Context) ([]clients.LedgerOrgDonation, error) {
	return nil, nil
}
func (c *intLedgerClient) PostTransaction(ctx context.Context, txn clients.LedgerTransaction) error {
	if c.onPostTransaction != nil {
		return c.onPostTransaction(ctx, txn)
	}
	return nil
}

var _ clients.LedgerClient = (*intLedgerClient)(nil)

// intEmailService is a no-op EmailService stub for integration tests.
type intEmailService struct {
	onConfirmation func(toEmail, toName, initiativeName, initiativeURL, amount string)
	onAdminNotify  func(ownerEmail, donorName, donorEmail, initiativeName, initiativeURL, amount string)
}

func (e *intEmailService) SendProjectApprovedEmail(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (e *intEmailService) SendProjectDeclinedEmail(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (e *intEmailService) SendProjectForReviewEmail(_ context.Context, _, _, _, _, _, _ string) error {
	return nil
}
func (e *intEmailService) SendDonationConfirmationEmail(_ context.Context, toEmail, toName, initiativeName, initiativeURL, amount, _, _, _, _ string) error {
	if e.onConfirmation != nil {
		e.onConfirmation(toEmail, toName, initiativeName, initiativeURL, amount)
	}
	return nil
}
func (e *intEmailService) SendDonationAdminNotificationEmail(_ context.Context, ownerEmail, _, donorName, donorEmail, initiativeName, initiativeURL, amount, _, _, _, _ string) error {
	if e.onAdminNotify != nil {
		e.onAdminNotify(ownerEmail, donorName, donorEmail, initiativeName, initiativeURL, amount)
	}
	return nil
}
func (e *intEmailService) InitiativeURL(slug string) string {
	return "https://crowdfunding.lfx.linuxfoundation.org/initiatives/" + slug
}

var _ domain.EmailService = (*intEmailService)(nil)

// discardLogger creates a slog.Logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// postSignedWebhook sends a Stripe webhook request with a mock signature header
// (real signature validation is tested in unit tests; this tests handler behavior).
func postSignedWebhook(t *testing.T, h *handler.WebhookHandler, payload []byte, _ string) *httptest.ResponseRecorder {
	t.Helper()
	// For integration tests, we use a dummy signature since we're not testing
	// signature validation (that's done in unit tests with mocks).
	// The webhook will be processed based on the payload content.
	sigHeader := "t=1000000000,v1=test_sig"

	req := httptest.NewRequest(http.MethodPost, "/v1/stripe/webhook", bytes.NewBuffer(payload))
	req.Header.Set("Stripe-Signature", sigHeader)
	rr := httptest.NewRecorder()
	h.Handle(rr, req)
	return rr
}

// skipIfNoTestDB skips the test if the test database is not available.
func skipIfNoTestDB(t *testing.T) {
	if handlerTestPool == nil {
		t.Skip("TEST_DATABASE_URL not set or database unavailable")
	}
}

// --- Integration Tests ---

// TestWebhookIntegration_PaymentIntentSucceeded verifies that a payment_intent.succeeded
// event advances a donation from "pending" to "succeeded" in the DB.
// Signature validation is bypassed in the stub — see webhook_handler_test.go for that coverage.
func TestWebhookIntegration_PaymentIntentSucceeded(t *testing.T) {
	skipIfNoTestDB(t)

	ctx := context.Background()
	const webhookSecret = "whsec_test_integration_001"

	// Clean up before test
	if _, err := handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.donations WHERE stripe_payment_intent_id = 'pi_int_test_001'"); err != nil {
		t.Fatalf("cleanup donations: %v", err)
	}
	if _, err := handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.initiatives WHERE initiative_type = 'test'"); err != nil {
		t.Fatalf("cleanup initiatives: %v", err)
	}
	if _, err := handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.users WHERE email = 'test@int-test.example.com'"); err != nil {
		t.Fatalf("cleanup users: %v", err)
	}

	// Seed a user with UUID
	userID := uuid.New()
	if _, err := handlerTestPool.Exec(ctx, `
		INSERT INTO crowdfunding.users (id, username, email) VALUES ($1, $2, $3)
	`, userID, "test_int_user_001", "test@int-test.example.com"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Seed an initiative
	initiativeID := uuid.New()
	if _, err := handlerTestPool.Exec(ctx, `
		INSERT INTO crowdfunding.initiatives
		(id, initiative_type, owner_id, name, slug, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, initiativeID, "test", userID, "Integration Test Initiative", "int-test-init", "published"); err != nil {
		t.Fatalf("seed initiative: %v", err)
	}

	// Seed a donation with stripe_payment_intent_id, status = pending
	piID := "pi_int_test_001"
	if _, err := handlerTestPool.Exec(ctx, `
		INSERT INTO crowdfunding.donations
		(id, initiative_id, user_id, stripe_payment_intent_id, status, current_amount_in_cents)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
	`, initiativeID, userID, piID, "pending", 5000); err != nil {
		t.Fatalf("seed donation: %v", err)
	}

	// Build payment_intent.succeeded event - the webhook handler expects event.Data.Raw
	// to contain the PaymentIntent object JSON (not wrapped in a "data" envelope)
	piObjectJSON := map[string]any{
		"id":            piID,
		"object":        "payment_intent",
		"amount":        5000,
		"currency":      "usd",
		"status":        "succeeded",
		"latest_charge": map[string]string{"id": "ch_int_test_001"},
	}

	// The event payload itself contains the wrapped structure, but intStripeClient
	// will extract just the object for Data.Raw
	eventPayload := map[string]any{
		"id":   "evt_test_001",
		"type": "payment_intent.succeeded",
		"data": map[string]any{
			"object": piObjectJSON,
		},
	}
	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		t.Fatalf("marshal event payload: %v", err)
	}

	// Create webhook handler and post the signed event
	sc := &intStripeClient{}
	dr := db.NewDonationRepository(handlerTestPool)
	sr := db.NewSubscriptionRepository(handlerTestPool)
	h := handler.NewWebhookHandler(sc, &intLedgerClient{}, dr, sr, &intEmailService{}, webhookSecret, discardLogger(), false)

	rr := postSignedWebhook(t, h, payloadBytes, webhookSecret)

	// Verify webhook returned 200 OK
	if rr.Code != http.StatusOK {
		t.Errorf("webhook status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	// Verify donation status was updated to succeeded in the DB
	var status string
	if err := handlerTestPool.QueryRow(ctx, `
		SELECT status FROM crowdfunding.donations WHERE stripe_payment_intent_id = $1
	`, piID).Scan(&status); err != nil {
		t.Fatalf("query donation status: %v", err)
	}
	if status != "succeeded" {
		t.Errorf("donation status = %q, want succeeded", status)
	}

	// Clean up after test (already deleted in cleanup before)
	_, _ = handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.donations WHERE stripe_payment_intent_id = $1", piID)
	_, _ = handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.initiatives WHERE id = $1", initiativeID)
	_, _ = handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.users WHERE id = $1", userID)
}

// TestWebhookIntegration_SubscriptionActivated tests that a real invoice.payment_succeeded
// event advances a subscription from "incomplete" to "active" in the DB.
func TestWebhookIntegration_SubscriptionActivated(t *testing.T) {
	skipIfNoTestDB(t)

	ctx := context.Background()
	const webhookSecret = "whsec_test_integration_002"

	// Clean up before test
	if _, err := handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.donations WHERE stripe_invoice_id = 'in_int_test_001'"); err != nil {
		t.Fatalf("cleanup donations: %v", err)
	}
	var existingDonationCount int
	if err := handlerTestPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM crowdfunding.donations WHERE stripe_invoice_id = 'in_int_test_001'
	`).Scan(&existingDonationCount); err != nil {
		t.Fatalf("verify donation cleanup: %v", err)
	}
	if existingDonationCount != 0 {
		t.Fatalf("donation cleanup left %d rows, want 0", existingDonationCount)
	}
	if _, err := handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.subscriptions WHERE stripe_subscription_id = 'sub_int_test_001'"); err != nil {
		t.Fatalf("cleanup subscriptions: %v", err)
	}
	if _, err := handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.initiatives WHERE initiative_type = 'test'"); err != nil {
		t.Fatalf("cleanup initiatives: %v", err)
	}
	if _, err := handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.users WHERE email = 'test-sub@int-test.example.com'"); err != nil {
		t.Fatalf("cleanup users: %v", err)
	}

	// Seed a user
	userID := uuid.New()
	const legacyUserID = "auth0|test_int_user_002"
	if _, err := handlerTestPool.Exec(ctx, `
		INSERT INTO crowdfunding.users (id, username, legacy_user_id, email) VALUES ($1, $2, $3, $4)
	`, userID, "test_int_user_002", legacyUserID, "test-sub@int-test.example.com"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Seed an initiative
	initiativeID := uuid.New()
	if _, err := handlerTestPool.Exec(ctx, `
		INSERT INTO crowdfunding.initiatives
		(id, initiative_type, owner_id, name, slug, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, initiativeID, "test", userID, "Integration Test Sub Initiative", "int-test-sub-init", "published"); err != nil {
		t.Fatalf("seed initiative: %v", err)
	}

	// Seed a subscription with status = incomplete
	subID := "sub_int_test_001"
	if _, err := handlerTestPool.Exec(ctx, `
		INSERT INTO crowdfunding.subscriptions
		(id, initiative_id, user_id, stripe_subscription_id, status, frequency, current_amount_in_cents)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
	`, initiativeID, userID, subID, "incomplete", "monthly", 2500); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	// Build invoice.payment_succeeded event. The nested subscription reference is
	// what activates the subscription, while the invoice-level amount_paid and
	// charge plus the subscription metadata are what the donation insert uses.
	invoiceObjectJSON := map[string]any{
		"id":             "in_int_test_001",
		"object":         "invoice",
		"status":         "paid",
		"amount_paid":    2500,
		"charge":         "ch_int_test_001",
		"customer_email": "test-sub@int-test.example.com",
		"parent": map[string]any{
			"subscription_details": map[string]any{
				"subscription": map[string]string{
					"id": subID,
				},
				"metadata": map[string]string{
					"version":         "v2",
					"initiative_id":   initiativeID.String(),
					"initiative_slug": "int-test-sub-init",
					"initiative_name": "Integration Test Sub Initiative",
					"user_id":         legacyUserID,
					"donor_email":     "test-sub@int-test.example.com",
					"owner_email":     "test-sub@int-test.example.com",
					"owner_name":      "Integration Owner",
					"category":        "fund",
					"payment_method":  "stripe",
					"frequency":       "monthly",
				},
			},
		},
	}

	eventPayload := map[string]any{
		"id":   "evt_inv_test_001",
		"type": "invoice.payment_succeeded",
		"data": map[string]any{
			"object": invoiceObjectJSON,
		},
	}
	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		t.Fatalf("marshal event payload: %v", err)
	}

	// Create webhook handler and post the signed event
	sc := &intStripeClient{}
	dr := db.NewDonationRepository(handlerTestPool)
	sr := db.NewSubscriptionRepository(handlerTestPool)
	h := handler.NewWebhookHandler(sc, &intLedgerClient{}, dr, sr, &intEmailService{}, webhookSecret, discardLogger(), false).
		WithLegacyUserLookup(db.NewUserRepository(handlerTestPool))

	rr := postSignedWebhook(t, h, payloadBytes, webhookSecret)

	// Verify webhook returned 200 OK
	if rr.Code != http.StatusOK {
		t.Errorf("webhook status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	// Verify subscription status was updated to active in the DB
	var status string
	if err := handlerTestPool.QueryRow(ctx, `
		SELECT status FROM crowdfunding.subscriptions WHERE stripe_subscription_id = $1
	`, subID).Scan(&status); err != nil {
		t.Fatalf("query subscription status: %v", err)
	}
	if status != "active" {
		t.Errorf("subscription status = %q, want active", status)
	}

	var donationCount int
	if err := handlerTestPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM crowdfunding.donations WHERE stripe_invoice_id = $1
	`, "in_int_test_001").Scan(&donationCount); err != nil {
		t.Fatalf("query donation count: %v", err)
	}
	if donationCount != 1 {
		t.Fatalf("donation count = %d, want 1", donationCount)
	}

	var donationStatus, donationUserID, donationInitiativeID string
	var donationAmount int64
	if err := handlerTestPool.QueryRow(ctx, `
		SELECT status, user_id::text, initiative_id::text, current_amount_in_cents
		FROM crowdfunding.donations
		WHERE stripe_invoice_id = $1
	`, "in_int_test_001").Scan(&donationStatus, &donationUserID, &donationInitiativeID, &donationAmount); err != nil {
		t.Fatalf("query donation row: %v", err)
	}
	if donationStatus != "succeeded" {
		t.Errorf("donation status = %q, want succeeded", donationStatus)
	}
	if donationUserID != userID.String() {
		t.Errorf("donation user_id = %q, want %q", donationUserID, userID.String())
	}
	if donationInitiativeID != initiativeID.String() {
		t.Errorf("donation initiative_id = %q, want %q", donationInitiativeID, initiativeID.String())
	}
	if donationAmount != 2500 {
		t.Errorf("donation amount = %d, want 2500", donationAmount)
	}

	// Duplicate delivery must not create a second donation row.
	rr = postSignedWebhook(t, h, payloadBytes, webhookSecret)
	if rr.Code != http.StatusOK {
		t.Errorf("duplicate webhook status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	if err := handlerTestPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM crowdfunding.donations WHERE stripe_invoice_id = $1
	`, "in_int_test_001").Scan(&donationCount); err != nil {
		t.Fatalf("query duplicate donation count: %v", err)
	}
	if donationCount != 1 {
		t.Errorf("donation count after duplicate = %d, want 1", donationCount)
	}

	// Clean up after test (already deleted in cleanup before)
	_, _ = handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.donations WHERE stripe_invoice_id = $1", "in_int_test_001")
	_, _ = handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.subscriptions WHERE stripe_subscription_id = $1", subID)
	_, _ = handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.initiatives WHERE id = $1", initiativeID)
	_, _ = handlerTestPool.Exec(ctx, "DELETE FROM crowdfunding.users WHERE id = $1", userID)
}
