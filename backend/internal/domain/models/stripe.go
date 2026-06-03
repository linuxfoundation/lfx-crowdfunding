// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

// StripeProduct is a read-only view of a Stripe Product, used when enriching initiative data.
type StripeProduct struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// CardDetails holds information about a saved Stripe PaymentMethod.
type CardDetails struct {
	PaymentMethodID string `json:"payment_method_id"` // pm_xxx
	LastFour        string `json:"last_four"`
	Brand           string `json:"brand"`
	ExpiryMonth     int    `json:"expiry_month"`
	ExpiryYear      int    `json:"expiry_year"`
}

// SetupIntentResult is returned by POST /v1/me/setup-intent.
// The frontend passes ClientSecret to the Stripe.js Payment Element to collect
// and 3DS-authenticate the card before attaching it to the customer.
type SetupIntentResult struct {
	ClientSecret string `json:"client_secret"`
}

// PaymentIntentRequest is the input for creating a one-time Stripe payment.
type PaymentIntentRequest struct {
	InitiativeID    string
	UserID          string
	CustomerID      string // Stripe cus_xxx — required for 3DS off-session charges
	AmountCents     int64
	Currency        string // defaults to "usd"
	PaymentMethodID string
	Category        string // e.g. "mentorship", "general fund", "event" — stored in Stripe metadata for Ledger
	OrgID           string // organization ID — stored in Stripe metadata for Ledger
	// IdempotencyKey is a per-request unique value (UUID) that prevents Stripe
	// from creating a duplicate PaymentIntent when the client retries a timed-out
	// request. Must be different for each logically distinct charge.
	IdempotencyKey string
}

// PaymentIntent holds the Stripe PaymentIntent result.
// ClientSecret is non-empty when Status == "requires_action"; the frontend
// must call stripe.confirmCardPayment(ClientSecret) to complete 3DS.
type PaymentIntent struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	ClientSecret string `json:"client_secret,omitempty"` // non-empty when 3DS required
	ChargeID     string `json:"charge_id,omitempty"`     // ch_xxx once succeeded
}

// StripeSubscriptionRequest is the input for creating a Stripe subscription.
type StripeSubscriptionRequest struct {
	InitiativeID     string
	UserID           string
	StripeCustomerID string
	StripePriceID    string
	PaymentMethodID  string
	Category         string // e.g. "mentorship", "general fund", "event" — stored in Stripe metadata for Ledger
	OrgID            string // organization ID — stored in Stripe metadata for Ledger
	// IdempotencyKey is the client-supplied key forwarded to Stripe so that
	// retries of the same logical request are de-duped at the Stripe layer.
	IdempotencyKey string
}

// StripeSubscriptionResult holds the IDs needed to record the subscription in Postgres.
// ClientSecret is non-empty when Status == "incomplete" (first invoice needs 3DS).
type StripeSubscriptionResult struct {
	SubscriptionID     string `json:"subscription_id"`
	SubscriptionItemID string `json:"subscription_item_id"`
	PriceID            string `json:"price_id"`
	Status             string `json:"status"`
	ClientSecret       string `json:"client_secret,omitempty"` // non-empty when 3DS required
}
