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
	ID                       string    `json:"id"`
	UserID                   string    `json:"user_id"`
	InitiativeID             string    `json:"initiative_id"`
	OrganizationID           string    `json:"organization_id,omitempty"`
	Category                 string    `json:"category,omitempty"`
	CurrentAmountCents       int64     `json:"current_amount_in_cents"`
	Frequency                string    `json:"frequency,omitempty"`
	Status                   string    `json:"status,omitempty"`
	StripeSubscriptionID     string    `json:"stripe_subscription_id,omitempty"`
	StripeSubscriptionItemID string    `json:"stripe_subscription_item_id,omitempty"`
	StripePriceID            string    `json:"stripe_price_id,omitempty"`
	CreatedOn                time.Time `json:"created_on"`
	UpdatedOn                time.Time `json:"updated_on"`

	// ClientSecret is transient — set by the service when 3DS is required, never stored.
	ClientSecret string `json:"client_secret,omitempty"`
}

// SubscriptionCreateInput is the request body for creating a subscription.
type SubscriptionCreateInput struct {
	AmountCents           int64  `json:"amount_in_cents"`
	Frequency             string `json:"frequency"`
	Category              string `json:"category,omitempty"`
	OrganizationID        string `json:"organization_id,omitempty"`
	StripePaymentMethodID string `json:"stripe_payment_method_id"`
	// IdempotencyKey is set by the handler from the Idempotency-Key HTTP header.
	// Not decoded from the JSON body (json:"-").
	IdempotencyKey string `json:"-"`
}
