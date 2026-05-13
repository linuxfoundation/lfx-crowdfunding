// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

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
	Status         string
	Search         string
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
	UserID string // Auth0 subject — matches users.user_id
	Email  string
	Name   string
}
