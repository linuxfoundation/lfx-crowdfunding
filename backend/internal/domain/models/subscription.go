// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Subscription status values stored in crowdfunding.subscriptions.status.
const (
	SubscriptionStatusIncomplete = "incomplete"
	SubscriptionStatusActive     = "active"
	SubscriptionStatusPastDue    = "past_due"
	SubscriptionStatusCanceled   = "canceled"
)

// Subscription maps to the crowdfunding.subscriptions table.
type Subscription struct {
	ID                 string `json:"id"`
	UserID             string `json:"-"`
	InitiativeID       string `json:"initiative_id"`
	OrganizationID     string `json:"-"`
	Category           string `json:"category,omitempty"`
	CurrentAmountCents int64  `json:"amount_cents"`
	Frequency          string `json:"frequency,omitempty"`
	Status             string `json:"status,omitempty"`
	// Stripe IDs are internal operational fields used by webhook reconciliation.
	// They are never serialised to API consumers.
	StripeSubscriptionID     string    `json:"-"`
	StripeSubscriptionItemID string    `json:"-"`
	StripePriceID            string    `json:"-"`
	CreatedOn                time.Time `json:"created_on"`
	UpdatedOn                time.Time `json:"updated_on"`

	// ClientSecret is transient — set by the service when 3DS is required, never stored.
	ClientSecret string `json:"client_secret,omitempty"`
}

// SubscriptionCreateInput is the request body for creating a subscription.
type SubscriptionCreateInput struct {
	AmountCents           int64  `json:"amount_cents"`
	Frequency             string `json:"frequency"`
	Category              string `json:"category,omitempty"`
	OrganizationID        string `json:"organization_id,omitempty"`
	StripePaymentMethodID string `json:"stripe_payment_method_id"`
	// IdempotencyKey is set by the handler from the Idempotency-Key HTTP header.
	// Not decoded from the JSON body (json:"-").
	IdempotencyKey string `json:"-"`
}
