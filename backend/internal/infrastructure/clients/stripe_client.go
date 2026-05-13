// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

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
type StripeClient interface {
	GetProduct(ctx context.Context, productID string) (*models.StripeProduct, error)
	CreatePaymentIntent(ctx context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error)
	CreateSubscription(ctx context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	ConstructWebhookEvent(payload []byte, sigHeader, secret string) (stripe.Event, error)
}

// StripeConfig holds Stripe API connection settings.
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	Timeout       time.Duration
}

type stripeClientImpl struct {
	api           *client.API
	webhookSecret string
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
	return &stripeClientImpl{api: api, webhookSecret: cfg.WebhookSecret}
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

// CreatePaymentIntent creates a Stripe PaymentIntent for a one-time donation.
func (c *stripeClientImpl) CreatePaymentIntent(ctx context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	_, span := stripeTracer.Start(ctx, "stripe.CreatePaymentIntent")
	defer span.End()

	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(req.AmountCents),
		Currency:      stripe.String(string(stripe.CurrencyUSD)),
		PaymentMethod: stripe.String(req.PaymentMethodID),
		Confirm:       stripe.Bool(true),
		Params: stripe.Params{
			Metadata: map[string]string{
				"initiative_id": req.InitiativeID,
				"user_id":       req.UserID,
			},
		},
	}
	pi, err := c.api.PaymentIntents.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe create payment intent: %w", err)
	}
	return &models.PaymentIntent{
		ID:     pi.ID,
		Status: string(pi.Status),
	}, nil
}

// CreateSubscription creates a Stripe subscription for recurring donations.
func (c *stripeClientImpl) CreateSubscription(ctx context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	_, span := stripeTracer.Start(ctx, "stripe.CreateSubscription")
	defer span.End()

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(req.StripeCustomerID),
		Items: []*stripe.SubscriptionItemsParams{
			{Price: stripe.String(req.StripePriceID)},
		},
		DefaultPaymentMethod: stripe.String(req.PaymentMethodID),
		Params: stripe.Params{
			Metadata: map[string]string{
				"initiative_id": req.InitiativeID,
				"user_id":       req.UserID,
			},
		},
	}
	sub, err := c.api.Subscriptions.New(params)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe create subscription: %w", err)
	}
	var itemID string
	if len(sub.Items.Data) > 0 {
		itemID = sub.Items.Data[0].ID
	}
	return &models.StripeSubscriptionResult{
		SubscriptionID:     sub.ID,
		SubscriptionItemID: itemID,
		Status:             string(sub.Status),
	}, nil
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

// ConstructWebhookEvent validates a Stripe webhook signature and returns the event.
// Always validate the Stripe-Signature header — never process unverified events.
func (c *stripeClientImpl) ConstructWebhookEvent(payload []byte, sigHeader, secret string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, sigHeader, secret)
}
