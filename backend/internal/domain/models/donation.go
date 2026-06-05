// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Donation status values stored in crowdfunding.donations.status.
const (
	DonationStatusPending   = "pending"
	DonationStatusSucceeded = "succeeded"
	DonationStatusFailed    = "failed"
)

// Donation maps to the crowdfunding.donations table.
type Donation struct {
	ID                 string `json:"id"`
	UserID             string `json:"-"`
	InitiativeID       string `json:"initiative_id"`
	OrganizationID     string `json:"-"`
	Category           string `json:"category,omitempty"`
	CurrentAmountCents int64  `json:"amount_cents"`
	PONumber           string `json:"po_number,omitempty"`
	PaymentMethod      string `json:"payment_method,omitempty"`
	Status             string `json:"status,omitempty"`
	// Stripe IDs are internal operational fields used by the webhook reconciliation
	// flow. They are never serialised to API consumers.
	StripePaymentIntentID string    `json:"-"`
	StripeChargeID        string    `json:"-"`
	CreatedOn             time.Time `json:"created_on"`
	UpdatedOn             time.Time `json:"updated_on"`

	// ClientSecret is transient — set by the service when 3DS is required, never stored.
	ClientSecret string `json:"client_secret,omitempty"`
}

// DonationCreateInput is the request body for creating a donation.
type DonationCreateInput struct {
	AmountCents    int64  `json:"amount_cents"`
	Category       string `json:"category,omitempty"`
	PONumber       string `json:"po_number,omitempty"`
	PaymentMethod  string `json:"payment_method,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	// StripePaymentMethodID is used to create a Stripe charge
	StripePaymentMethodID string `json:"stripe_payment_method_id,omitempty"`
	// IdempotencyKey is set by the handler from the Idempotency-Key HTTP header.
	// It is not decoded from the JSON body (json:"-").
	IdempotencyKey string `json:"-"`
}

// DonationSummary is the public-facing projection returned by the initiative
// donation list (GET /v1/initiatives/{id}/donations). It omits internal
// identifiers (user_id, organization_id) and Stripe IDs; donor_name and
// donor_avatar_url are display-only fields intentionally included.
type DonationSummary struct {
	ID             string    `json:"id"`
	AmountCents    int64     `json:"amount_cents"`
	Status         string    `json:"status,omitempty"`
	Category       string    `json:"category,omitempty"`
	DonorName      string    `json:"donor_name,omitempty"`
	DonorType      string    `json:"donor_type,omitempty"` // "organization" | "individual"
	DonorAvatarURL string    `json:"donor_avatar_url,omitempty"`
	CreatedOn      time.Time `json:"created_on"`
}
