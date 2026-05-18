// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Donation maps to the crowdfunding.donations table.
type Donation struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	InitiativeID          string    `json:"initiative_id"`
	OrganizationID        string    `json:"organization_id,omitempty"`
	Category              string    `json:"category,omitempty"`
	CurrentAmountCents    int64     `json:"current_amount_in_cents"`
	PONumber              string    `json:"po_number,omitempty"`
	PaymentMethod         string    `json:"payment_method,omitempty"`
	Status                string    `json:"status,omitempty"`
	StripePaymentIntentID string    `json:"stripe_payment_intent_id,omitempty"`
	StripeChargeID        string    `json:"stripe_charge_id,omitempty"`
	CreatedOn             time.Time `json:"created_on"`
	UpdatedOn             time.Time `json:"updated_on"`

	// ClientSecret is transient — set by the service when 3DS is required, never stored.
	ClientSecret string `json:"client_secret,omitempty"`
}

// DonationCreateInput is the request body for creating a donation.
type DonationCreateInput struct {
	AmountCents    int64  `json:"amount_in_cents"`
	Category       string `json:"category,omitempty"`
	PONumber       string `json:"po_number,omitempty"`
	PaymentMethod  string `json:"payment_method,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	// StripePaymentMethodID is used to create a Stripe charge
	StripePaymentMethodID string `json:"stripe_payment_method_id,omitempty"`
}
