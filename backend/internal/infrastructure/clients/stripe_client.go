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
	stripe "github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
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
	// CreateProduct creates a new Stripe Product for an initiative and returns the prod_xxx ID.
	// initiativeID is the pre-generated UUID — stored as Stripe metadata so the product
	// can always be traced back to the local initiative row.
	CreateProduct(ctx context.Context, initiativeID, name string) (string, error)
	// DeleteProduct permanently deletes a Stripe Product.
	// Only safe to call when no Prices have been attached (e.g. as a rollback after
	// CreateProduct succeeds but the subsequent DB INSERT fails).
	DeleteProduct(ctx context.Context, productID string) error

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
	// GetOrCreatePrice creates a new recurring Stripe Price under the given
	// Stripe Product (productID = initiative.StripeProductID), amount, and
	// interval. A fresh Price is always created — Stripe recommends this for
	// variable-amount subscriptions rather than reusing prices. Using an
	// existing Product keeps the Stripe catalog manageable (one Product per
	// initiative, many Prices).
	// idempotencyKey is forwarded to Stripe so retries of the same request
	// return the cached Price rather than creating a duplicate.
	GetOrCreatePrice(ctx context.Context, productID, initiativeID string, amountCents int64, interval string, idempotencyKey string) (string, error)
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
	SecretKey string
	Timeout   time.Duration
	// ReturnURL is the frontend URL Stripe redirects to after a 3DS challenge.
	// Required when Confirm=true on a PaymentIntent.
	ReturnURL string
}

type stripeClientImpl struct {
	api       *stripe.Client
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
	api := stripe.NewClient(cfg.SecretKey, stripe.WithBackends(backends))
	return &stripeClientImpl{api: api, returnURL: cfg.ReturnURL}
}

// CreateProduct creates a Stripe Product for an initiative.
// initiativeID is stored in metadata so the Stripe Dashboard product can be traced
// back to the local initiative UUID.
func (c *stripeClientImpl) CreateProduct(ctx context.Context, initiativeID, name string) (string, error) {
	_, span := stripeTracer.Start(ctx, "stripe.CreateProduct")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.initiative_id", initiativeID))

	p, err := c.api.V1Products.Create(ctx, &stripe.ProductCreateParams{
		Name:     stripe.String(name),
		Metadata: map[string]string{"initiative_id": initiativeID},
		Params: stripe.Params{
			// Idempotency key scoped to the pre-generated initiative UUID so that a
			// network-timeout retry returns the cached Product instead of creating a duplicate.
			IdempotencyKey: stripe.String(fmt.Sprintf("create-product:%s", initiativeID)),
		},
	})
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("stripe create product: %w", err)
	}
	return p.ID, nil
}

// DeleteProduct permanently deletes a Stripe Product.
func (c *stripeClientImpl) DeleteProduct(ctx context.Context, productID string) error {
	_, span := stripeTracer.Start(ctx, "stripe.DeleteProduct")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.product_id", productID))

	_, err := c.api.V1Products.Delete(ctx, productID, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("stripe delete product: %w", err)
	}
	return nil
}

// GetProduct retrieves a Stripe product by ID.
func (c *stripeClientImpl) GetProduct(ctx context.Context, productID string) (*models.StripeProduct, error) {
	_, span := stripeTracer.Start(ctx, "stripe.GetProduct")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.product_id", productID))

	p, err := c.api.V1Products.Retrieve(ctx, productID, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe get product: %w", err)
	}
	return &models.StripeProduct{ID: p.ID, Name: p.Name, Active: p.Active}, nil
}

// ── Customer ─────────────────────────────────────────────────────────────────

// CreateCustomer creates a new Stripe Customer tagged with the service user UUID.
func (c *stripeClientImpl) CreateCustomer(ctx context.Context, userID, email string) (string, error) {
	_, span := stripeTracer.Start(ctx, "stripe.CreateCustomer")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.user_id", userID))

	cust, err := c.api.V1Customers.Create(ctx, &stripe.CustomerCreateParams{
		Email:    stripe.String(email),
		Metadata: map[string]string{"user_id": userID},
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

	si, err := c.api.V1SetupIntents.Create(ctx, &stripe.SetupIntentCreateParams{
		Customer:           stripe.String(customerID),
		PaymentMethodTypes: []*string{stripe.String("card")},
		Usage:              stripe.String("off_session"),
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

	pm, err := c.api.V1PaymentMethods.Attach(ctx, paymentMethodID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customerID),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe attach payment method: %w", err)
	}

	// Set as the customer's default invoice payment method so subscriptions and
	// off-session PaymentIntents use it automatically.
	_, err = c.api.V1Customers.Update(ctx, customerID, &stripe.CustomerUpdateParams{
		InvoiceSettings: &stripe.CustomerUpdateInvoiceSettingsParams{
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

	pm, err := c.api.V1PaymentMethods.Retrieve(ctx, paymentMethodID, nil)
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

	_, err := c.api.V1PaymentMethods.Detach(ctx, paymentMethodID, nil)
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

// buildChargeMetadata assembles the Stripe metadata map for charges and
// subscriptions. initiative_id, user_id, and version are always written;
// all other fields are only included when non-empty to avoid storing empty
// strings in Stripe (and downstream in Ledger).
// version=v2 marks charges originated by this service so the Ledger-service
// webhook handler can skip processing them (they are recorded directly via
// the Ledger /transactions API instead).
func buildChargeMetadata(initiativeID, initiativeSlug, initiativeName, userID, donorName, donorEmail, ownerEmail, category, orgID string) map[string]string {
	m := map[string]string{
		"initiative_id": initiativeID,
		"user_id":       userID,
		"version":       "v2",
	}
	if initiativeSlug != "" {
		m["initiative_slug"] = initiativeSlug
	}
	if initiativeName != "" {
		m["initiative_name"] = initiativeName
	}
	if donorName != "" {
		m["donor_name"] = donorName
	}
	if donorEmail != "" {
		m["donor_email"] = donorEmail
	}
	if ownerEmail != "" {
		m["owner_email"] = ownerEmail
	}
	if category != "" {
		m["category"] = category
	}
	if orgID != "" {
		m["org_id"] = orgID
	}
	return m
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

	params := &stripe.PaymentIntentCreateParams{
		Amount:       stripe.Int64(req.AmountCents),
		Currency:     stripe.String(currency),
		Confirm:      stripe.Bool(true),
		ReceiptEmail: stripe.String(req.DonorEmail),
		// ReturnURL is required by Stripe when Confirm=true and 3DS may redirect.
		ReturnURL: stripe.String(c.returnURL),
		// "automatic" lets Stripe decide when to trigger the 3DS challenge.
		// EU/UK cards under PSD2/SCA will be challenged when the bank requires it.
		PaymentMethodOptions: &stripe.PaymentIntentCreatePaymentMethodOptionsParams{
			Card: &stripe.PaymentIntentCreatePaymentMethodOptionsCardParams{
				RequestThreeDSecure: stripe.String("automatic"),
			},
		},
		Metadata: buildChargeMetadata(req.InitiativeID, req.InitiativeSlug, req.InitiativeName, req.UserID, req.DonorName, req.DonorEmail, req.OwnerEmail, req.Category, req.OrganizationID),
		Params: stripe.Params{
			// Idempotency key is a per-request UUID generated by DonationService so
			// that retries of the same timed-out request are de-duped, while
			// separate donations with identical amounts are not.
			IdempotencyKey: stripe.String(req.IdempotencyKey),
		},
	}
	if req.CustomerID != "" {
		params.Customer = stripe.String(req.CustomerID)
	}
	// PaymentMethodID is guaranteed non-empty by the guard above.
	params.PaymentMethod = stripe.String(req.PaymentMethodID)
	// Expand latest_charge to capture ch_xxx on immediate success.
	params.AddExpand("latest_charge")

	pi, err := c.api.V1PaymentIntents.Create(ctx, params)
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

// GetOrCreatePrice creates a recurring Stripe Price under an existing Stripe
// Product (productID = initiative.StripeProductID). A new Price is created on
// each call — Stripe recommends this pattern for variable-amount subscriptions.
// Using an existing Product avoids spamming Stripe with one Product per price.
// idempotencyKey is passed to Stripe so that a client retry of the same request
// returns the cached Price rather than creating a duplicate.
func (c *stripeClientImpl) GetOrCreatePrice(ctx context.Context, productID, initiativeID string, amountCents int64, frequency string, idempotencyKey string) (string, error) {
	_, span := stripeTracer.Start(ctx, "stripe.GetOrCreatePrice")
	defer span.End()
	span.SetAttributes(
		attribute.String("stripe.product_id", productID),
		attribute.Int64("stripe.amount_cents", amountCents),
		attribute.String("stripe.frequency", frequency),
	)

	interval, err := stripeInterval(frequency)
	if err != nil {
		return "", err
	}

	p, err := c.api.V1Prices.Create(ctx, &stripe.PriceCreateParams{
		Currency:   stripe.String("usd"),
		UnitAmount: stripe.Int64(amountCents),
		Recurring: &stripe.PriceCreateRecurringParams{
			Interval: stripe.String(interval),
		},
		// Attach to the existing initiative Product rather than creating a new
		// Product per Price — keeps the Stripe catalog clean.
		Product: stripe.String(productID),
		Metadata: map[string]string{
			"product_id":    productID,
			"initiative_id": initiativeID,
		},
		Params: stripe.Params{
			// Prefix the key so price and subscription calls never share a key.
			IdempotencyKey: stripe.String(fmt.Sprintf("sub-price:%s", idempotencyKey)),
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

	// Build subscription item metadata so the invoice line item carries initiative_id,
	// org_id and category — the Ledger hook reads Lines.Data[0].Metadata for all
	// three fields as its final fallback (covering new Stripe API versions where
	// Plan is nil and Plan.Metadata is unavailable).
	subItemMeta := map[string]string{
		"initiative_id": req.InitiativeID,
	}
	if req.OrganizationID != "" {
		subItemMeta["org_id"] = req.OrganizationID
	}
	if req.Category != "" {
		subItemMeta["category"] = req.Category
	}
	subItem := &stripe.SubscriptionCreateItemParams{
		Price:    stripe.String(req.StripePriceID),
		Metadata: subItemMeta,
	}
	params := &stripe.SubscriptionCreateParams{
		Customer:        stripe.String(req.StripeCustomerID),
		Items:           []*stripe.SubscriptionCreateItemParams{subItem},
		PaymentBehavior: stripe.String("default_incomplete"),
		Metadata:        buildChargeMetadata(req.InitiativeID, req.InitiativeSlug, req.InitiativeName, req.UserID, req.DonorName, req.DonorEmail, req.OwnerEmail, req.Category, req.OrganizationID),
		Params: stripe.Params{
			// Idempotency key so retries return the cached Subscription rather than
			// creating a duplicate. The price key (sub-price:) uses the same base key.
			IdempotencyKey: stripe.String(fmt.Sprintf("sub:%s", req.IdempotencyKey)),
		},
	}
	// Expand latest_invoice so we can read ConfirmationSecret.ClientSecret.
	params.AddExpand("latest_invoice")
	if req.PaymentMethodID != "" {
		params.DefaultPaymentMethod = stripe.String(req.PaymentMethodID)
	}

	sub, err := c.api.V1Subscriptions.Create(ctx, params)
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
			result.Status = models.SubscriptionStatusIncomplete
		}
	}
	return result, nil
}

// CancelSubscription cancels a Stripe subscription immediately.
func (c *stripeClientImpl) CancelSubscription(ctx context.Context, subscriptionID string) error {
	_, span := stripeTracer.Start(ctx, "stripe.CancelSubscription")
	defer span.End()
	span.SetAttributes(attribute.String("stripe.subscription_id", subscriptionID))

	_, err := c.api.V1Subscriptions.Cancel(ctx, subscriptionID, nil)
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
