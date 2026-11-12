// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import (
	"strings"
	"time"
)

// OrgClaim is the organization a donor gives on behalf of, as the donor stated it.
// It is stored on the donation or subscription itself. Domain is normalized by
// the service before it is stored.
type OrgClaim struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// B2BOrgMatch is the member-service b2b_org an org domain resolves to (#411):
// its Salesforce Account SFID and logo URL (#472). A zero value means no match.
type B2BOrgMatch struct {
	SFID    string
	LogoURL string
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

// DisplayName returns the user's name, falling back to given + family name.
// It never falls back to the email.
func (u User) DisplayName() string {
	if name := strings.TrimSpace(u.Name); name != "" {
		return name
	}
	return strings.TrimSpace(strings.TrimSpace(u.GivenName) + " " + strings.TrimSpace(u.FamilyName))
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
