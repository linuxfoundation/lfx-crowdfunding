// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

// StripeProduct is a read-only view of a Stripe Product, used when enriching initiative data.
type StripeProduct struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// PaymentIntentRequest is the input for creating a one-time Stripe payment.
type PaymentIntentRequest struct {
	InitiativeID    string
	UserID          string
	AmountCents     int64
	PaymentMethodID string
}

// PaymentIntent holds the Stripe PaymentIntent result.
type PaymentIntent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// StripeSubscriptionRequest is the input for creating a Stripe subscription.
type StripeSubscriptionRequest struct {
	InitiativeID     string
	UserID           string
	StripeCustomerID string
	StripePriceID    string
	PaymentMethodID  string
}

// StripeSubscriptionResult holds the IDs needed to record the subscription in Postgres.
type StripeSubscriptionResult struct {
	SubscriptionID     string `json:"subscription_id"`
	SubscriptionItemID string `json:"subscription_item_id"`
	Status             string `json:"status"`
}
