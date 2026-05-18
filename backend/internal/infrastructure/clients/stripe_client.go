// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/client"
	"github.com/stripe/stripe-go/v82/webhook"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var stripeTracer = otel.Tracer("stripe-client")

// StripeClient is the interface consumed by the service layer.
// All mutating operations that reach Stripe are routed through this interface
// so the service layer remains testable without a live Stripe connection.
type StripeClient interface {
	GetProduct(ctx context.Context, productID string) (*models.StripeProduct, error)

	// Customer management
	// CreateCustomer creates a Stripe Customer for a user and returns the cus_xxx ID.
	// The caller (service layer) is responsible for persisting the ID to the DB.
	CreateCustomer(ctx context.Context, userID, email string) (string, error)

	// Card management (SetupIntent flow for 3DS-authenticated card saving)
	// CreateSetupIntent creates a SetupIntent and returns its client_secret for
	// the frontend Payment Element to collect and 3DS-challenge the card.
	CreateSetupIntent(ctx context.Context, customerID string) (string, error)
	// AttachPaymentMethod attaches pm_xxx to the customer and sets it as the
	// invoice default. Returns card details for the API response.
	AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) (*models.CardDetails, error)
	// GetPaymentMethod returns card details for a given pm_xxx.
	GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.CardDetails, error)
	// DetachPaymentMethod removes pm_xxx from the Stripe customer.
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error

	// One-time payments
	// CreatePaymentIntent creates a PaymentIntent with automatic 3DS.
	// When the bank requires a challenge, Status == "requires_action" and
	// ClientSecret is non-empty — the frontend must call stripe.confirmCardPayment.
	CreatePaymentIntent(ctx context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error)

	// Recurring payments
	// GetOrCreatePrice finds or creates a recurring Stripe Price for the given
	// initiative / amount / interval combination.
	GetOrCreatePrice(ctx context.Context, initiativeID string, amountCents int64, interval string) (string, error)
	// CreateSubscription creates a subscription with payment_behavior=default_incomplete
	// so the first invoice's PaymentIntent can require 3DS before the subscription activates.
	CreateSubscription(ctx context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error

	// Webhooks
	// ConstructWebhookEvent validates the Stripe-Signature header and parses the event.
	// Always call this before processing any webhook payload.
	ConstructWebhookEvent(payload []byte, sigHeader, secret string) (stripe.Event, error)
}

// StripeConfig holds Stripe API connection settings.
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	Timeout       time.Duration
	// ReturnURL is the frontend URL Stripe redirects to after a 3DS challenge.
	// Required when Confirm=true on a PaymentIntent.
	ReturnURL string
}

type stripeClientImpl struct {
	api       *client.API
	returnURL string
}

// NewStripeClient creates a Stripe client with an OTel-traced HTTP transport.
func NewStripeClient(cfg StripeConfig) StripeClient {
	httpClient := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	backends := stripe.NewBackendsWithConfig(&stripe.BackendConfig{
		HTTPClient: httpClient,
	})
	api := &client.API{}
	api.Init(cfg.SecretKey, backends)
	return &stripeClientImpl{api: api, returnURL: cfg.ReturnURL}
}

// GetProduct retrieves a Stripe product by ID.
func (c *stripeClientImpl) GetProduct(ctx context.Context, productID string) (*models.StripeProduct, error) {
	_, span := stripeTracer.Start(ctx, "stripe.GetProduct")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.product_id", productID))

	p, err := c.api.Products.Get(productID, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe get product: %w", err)
	}
	return &models.StripeProduct{ID: p.ID, Name: p.Name, Active: p.Active}, nil
}

// ── Customer ─────────────────────────────────────────────────────────────────

// CreateCustomer creates a new Stripe Customer tagged with the Auth0 user_id.
func (c *stripeClientImpl) CreateCustomer(ctx context.Context, userID, email string) (string, error) {
	_, span := stripeTracer.Start(ctx, "stripe.CreateCustomer")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.user_id", userID))

	cust, err := c.api.Customers.New(&stripe.CustomerParams{
		Email: stripe.String(email),
		Params: stripe.Params{
			Metadata: map[string]string{"user_id": userID},
		},
	})
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("stripe create customer: %w", err)
	}
	return cust.ID, nil
}

// ── Card management ───────────────────────────────────────────────────────────

// CreateSetupIntent creates a SetupIntent for off-session card saving with 3DS.
func (c *stripeClientImpl) CreateSetupIntent(ctx context.Context, customerID string) (string, error) {
	_, span := stripeTracer.Start(ctx, "stripe.CreateSetupIntent")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.customer_id", customerID))

	si, err := c.api.SetupIntents.New(&stripe.SetupIntentParams{
		Customer: stripe.String(customerID),
		AutomaticPaymentMethods: &stripe.SetupIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
		Usage: stripe.String("off_session"),
	})
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("stripe create setup intent: %w", err)
	}
	return si.ClientSecret, nil
}

// AttachPaymentMethod attaches pm_xxx to the customer and sets it as the
// customer's default invoice payment method.
func (c *stripeClientImpl) AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) (*models.CardDetails, error) {
	_, span := stripeTracer.Start(ctx, "stripe.AttachPaymentMethod")
	defer span.End()
	span.SetAttributes(
		attribute.String("stripe.customer_id", customerID),
		attribute.String("stripe.payment_method_id", paymentMethodID),
	)

	pm, err := c.api.PaymentMethods.Attach(paymentMethodID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe attach payment method: %w", err)
	}

	// Set as the customer's default invoice payment method so subscriptions and
	// off-session PaymentIntents use it automatically.
	_, err = c.api.Customers.Update(customerID, &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe set default payment method: %w", err)
	}

	return toCardDetails(pm), nil
}

// GetPaymentMethod returns card details for a pm_xxx.
func (c *stripeClientImpl) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*models.CardDetails, error) {
	_, span := stripeTracer.Start(ctx, "stripe.GetPaymentMethod")
	defer span.End()

	pm, err := c.api.PaymentMethods.Get(paymentMethodID, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe get payment method: %w", err)
	}
	return toCardDetails(pm), nil
}

// DetachPaymentMethod removes pm_xxx from its Stripe customer.
func (c *stripeClientImpl) DetachPaymentMethod(ctx context.Context, paymentMethodID string) error {
	_, span := stripeTracer.Start(ctx, "stripe.DetachPaymentMethod")
	defer span.End()

	_, err := c.api.PaymentMethods.Detach(paymentMethodID, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("stripe detach payment method: %w", err)
	}
	return nil
}

func toCardDetails(pm *stripe.PaymentMethod) *models.CardDetails {
	cd := &models.CardDetails{PaymentMethodID: pm.ID}
	if pm.Card != nil {
		cd.LastFour = pm.Card.Last4
		cd.Brand = string(pm.Card.Brand)
		cd.ExpiryMonth = int(pm.Card.ExpMonth)
		cd.ExpiryYear = int(pm.Card.ExpYear)
	}
	return cd
}

// ── One-time payments ─────────────────────────────────────────────────────────

// CreatePaymentIntent creates a PaymentIntent with automatic 3DS support.
// When the bank requires an authentication challenge, Status is "requires_action"
// and ClientSecret is non-empty — the frontend calls stripe.confirmCardPayment.
func (c *stripeClientImpl) CreatePaymentIntent(ctx context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	_, span := stripeTracer.Start(ctx, "stripe.CreatePaymentIntent")
	defer span.End()

	if req.PaymentMethodID == "" {
		return nil, fmt.Errorf("stripe create payment intent: payment_method_id is required when confirming server-side")
	}

	currency := req.Currency
	if currency == "" {
		currency = "usd"
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.AmountCents),
		Currency: stripe.String(currency),
		Confirm:  stripe.Bool(true),
		// ReturnURL is required by Stripe when Confirm=true and 3DS may redirect.
		ReturnURL: stripe.String(c.returnURL),
		// "automatic" lets Stripe decide when to trigger the 3DS challenge.
		// EU/UK cards under PSD2/SCA will be challenged when the bank requires it.
		PaymentMethodOptions: &stripe.PaymentIntentPaymentMethodOptionsParams{
			Card: &stripe.PaymentIntentPaymentMethodOptionsCardParams{
				RequestThreeDSecure: stripe.String("automatic"),
			},
		},
		Params: stripe.Params{
			Metadata: map[string]string{
				"initiative_id": req.InitiativeID,
				"user_id":       req.UserID,
			},
		},
	}
	if req.CustomerID != "" {
		params.Customer = stripe.String(req.CustomerID)
	}
	if req.PaymentMethodID != "" {
		params.PaymentMethod = stripe.String(req.PaymentMethodID)
	}
	// Expand latest_charge to capture ch_xxx on immediate success.
	params.AddExpand("latest_charge")

	pi, err := c.api.PaymentIntents.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe create payment intent: %w", err)
	}

	result := &models.PaymentIntent{
		ID:     pi.ID,
		Status: string(pi.Status),
	}
	// Return client_secret when 3DS challenge is required.
	if pi.Status == stripe.PaymentIntentStatusRequiresAction {
		result.ClientSecret = pi.ClientSecret
	}
	if pi.LatestCharge != nil {
		result.ChargeID = pi.LatestCharge.ID
	}
	return result, nil
}

// ── Subscriptions ─────────────────────────────────────────────────────────────

// stripeInterval maps caller-friendly frequency values to the Stripe API
// interval strings ("month", "year", etc.).
func stripeInterval(frequency string) (string, error) {
	switch frequency {
	case "monthly", "month":
		return "month", nil
	case "yearly", "year", "annual":
		return "year", nil
	case "weekly", "week":
		return "week", nil
	case "daily", "day":
		return "day", nil
	}
	return "", fmt.Errorf("unsupported billing frequency: %q", frequency)
}

// GetOrCreatePrice creates a recurring Stripe Price for the given initiative,
// amount, and billing interval. A new Price is created on each call — Stripe
// recommends this pattern for variable-amount subscriptions.
func (c *stripeClientImpl) GetOrCreatePrice(ctx context.Context, initiativeID string, amountCents int64, frequency string) (string, error) {
	_, span := stripeTracer.Start(ctx, "stripe.GetOrCreatePrice")
	defer span.End()
	span.SetAttributes(
		attribute.String("stripe.initiative_id", initiativeID),
		attribute.Int64("stripe.amount_cents", amountCents),
		attribute.String("stripe.frequency", frequency),
	)

	interval, err := stripeInterval(frequency)
	if err != nil {
		return "", err
	}

	p, err := c.api.Prices.New(&stripe.PriceParams{
		Currency:   stripe.String("usd"),
		UnitAmount: stripe.Int64(amountCents),
		Recurring: &stripe.PriceRecurringParams{
			Interval: stripe.String(interval),
		},
		ProductData: &stripe.PriceProductDataParams{
			Name: stripe.String(fmt.Sprintf("LFX Crowdfunding — Initiative %s", initiativeID)),
		},
		Params: stripe.Params{
			Metadata: map[string]string{
				"initiative_id": initiativeID,
			},
		},
	})
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("stripe create price: %w", err)
	}
	return p.ID, nil
}

// CreateSubscription creates a subscription with payment_behavior=default_incomplete
// so the subscription stays in status "incomplete" until the first invoice's
// PaymentIntent is confirmed. When 3DS is required on that invoice, ClientSecret
// is returned for the frontend to call stripe.confirmPayment.
func (c *stripeClientImpl) CreateSubscription(ctx context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	_, span := stripeTracer.Start(ctx, "stripe.CreateSubscription")
	defer span.End()

	params := &stripe.SubscriptionParams{
		Customer:        stripe.String(req.StripeCustomerID),
		Items:           []*stripe.SubscriptionItemsParams{{Price: stripe.String(req.StripePriceID)}},
		PaymentBehavior: stripe.String("default_incomplete"),
	}
	if req.PaymentMethodID != "" {
		params.DefaultPaymentMethod = stripe.String(req.PaymentMethodID)
	}
	// Expand latest_invoice so we can read ConfirmationSecret.ClientSecret.
	params.Params = stripe.Params{
		Expand: []*string{stripe.String("latest_invoice")},
		Metadata: map[string]string{
			"initiative_id": req.InitiativeID,
			"user_id":       req.UserID,
		},
	}

	sub, err := c.api.Subscriptions.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe create subscription: %w", err)
	}

	result := &models.StripeSubscriptionResult{
		SubscriptionID: sub.ID,
		PriceID:        req.StripePriceID,
		Status:         string(sub.Status),
	}
	if len(sub.Items.Data) > 0 {
		result.SubscriptionItemID = sub.Items.Data[0].ID
	}
	// When the first invoice requires 3DS, return the client_secret so the
	// frontend can call stripe.confirmPayment() to complete the challenge.
	if sub.LatestInvoice != nil && sub.LatestInvoice.ConfirmationSecret != nil {
		cs := sub.LatestInvoice.ConfirmationSecret
		if cs.ClientSecret != "" {
			result.ClientSecret = cs.ClientSecret
			result.Status = "incomplete"
		}
	}
	return result, nil
}

// CancelSubscription cancels a Stripe subscription immediately.
func (c *stripeClientImpl) CancelSubscription(ctx context.Context, subscriptionID string) error {
	_, span := stripeTracer.Start(ctx, "stripe.CancelSubscription")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.subscription_id", subscriptionID))

	_, err := c.api.Subscriptions.Cancel(subscriptionID, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("stripe cancel subscription: %w", err)
	}
	return nil
}

// ── Webhooks ──────────────────────────────────────────────────────────────────

// ConstructWebhookEvent validates a Stripe webhook signature and returns the event.
// Always validate the Stripe-Signature header — never process unverified events.
func (c *stripeClientImpl) ConstructWebhookEvent(payload []byte, sigHeader, secret string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, sigHeader, secret)
}
