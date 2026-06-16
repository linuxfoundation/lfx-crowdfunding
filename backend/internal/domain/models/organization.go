// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Organization maps to the crowdfunding.organizations table.
type Organization struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

// User maps to the crowdfunding.users table.
// users.id (UUID) is the FK target throughout the schema.
// username is the LF SSO username used as the application-level identity input.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`                 // LF SSO username, e.g. zpiatt
	LegacyUserID string    `json:"legacy_user_id,omitempty"` // Auth0 subject, e.g. auth0|abc123
	Email        string    `json:"email,omitempty"`
	GivenName    string    `json:"given_name,omitempty"`
	FamilyName   string    `json:"family_name,omitempty"`
	Name         string    `json:"name,omitempty"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	CreatedOn    time.Time `json:"created_on"`
	UpdatedOn    time.Time `json:"updated_on"`

	// Stripe fields — internal, never serialised.
	StripeCustomerID           string `json:"-"` // cus_xxx
	StripeDefaultPaymentMethod string `json:"-"` // pm_xxx
}

// OwnerInfo holds PII fields returned by the M2M owner-info endpoint.
type OwnerInfo struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// InitiativeSummary holds the minimal fields returned by the M2M published-list endpoint.
type InitiativeSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
