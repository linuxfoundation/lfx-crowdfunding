// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

// PaginationMeta carries cursor/page information returned alongside list results.
type PaginationMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// InitiativeFilter constrains list queries for initiatives.
type InitiativeFilter struct {
	OwnerID        string
	InitiativeType string
	Status         InitiativeStatus
	Search         string
	SortBy         string // "supporters" | "total_raised" | "created_on" (default)
	SortDir        string // "asc" | "desc" (default)
	Limit          int
	Offset         int
}

// DonationFilter constrains list queries for donations.
type DonationFilter struct {
	UserID string
	Status string
	Limit  int
	Offset int
}

// SubscriptionFilter constrains list queries for subscriptions.
type SubscriptionFilter struct {
	UserID string
	Status string
	Limit  int
	Offset int
}

// Principal holds the authenticated user's identity extracted from the JWT.
type Principal struct {
	UserID        string // Auth0 subject (e.g. "auth0|elim") — matches users.user_id
	Username      string // LF SSO username from custom claim
	Email         string
	EmailVerified bool
	Name          string // full name
	GivenName     string
	FamilyName    string
	Picture       string
}
