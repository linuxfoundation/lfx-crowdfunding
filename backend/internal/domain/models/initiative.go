// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Financials holds funding statistics sourced from initiative_ledger_stats,
// populated by the ledger-stats-sync CronJob. All fields are zero when the
// cron has not yet run for this initiative.
type Financials struct {
	TotalRaisedCents      int64 `json:"total_raised_cents"`
	TotalDisbursedCents   int64 `json:"total_disbursed_cents"`
	AvailableBalanceCents int64 `json:"available_balance_cents"`
	Supporters            int   `json:"supporters"`
	GoalsTotalCents       int64 `json:"goals_total_cents"`
	FundedPercent         int   `json:"funded_percent"`
}

// Initiative is the unified domain model for both list and detail responses.
// Internal fields (Stripe IDs, DynamoDB source, CII/Jobspring/Stacks identifiers)
// are excluded from JSON output — they are operational metadata, not API data.
type Initiative struct {
	ID             string `json:"id"`
	InitiativeType string `json:"initiative_type"`
	OwnerID        string `json:"owner_id"`
	Name           string `json:"name"`
	Slug           string `json:"slug,omitempty"`
	Status         string `json:"status,omitempty"`
	Industry       string `json:"industry,omitempty"`
	Description    string `json:"description,omitempty"`
	Color          string `json:"color,omitempty"`
	LogoURL        string `json:"logo_url,omitempty"`
	WebsiteURL     string `json:"website_url,omitempty"`
	CocURL         string `json:"coc_url,omitempty"`
	AcceptFunding  bool   `json:"accept_funding"`

	// Entity-only display fields
	EventbriteURL  string     `json:"eventbrite_url,omitempty"`
	ApplicationURL string     `json:"application_url,omitempty"`
	EventStartDate *time.Time `json:"event_start_date,omitempty"`
	EventEndDate   *time.Time `json:"event_end_date,omitempty"`
	Country        string     `json:"country,omitempty"`
	City           string     `json:"city,omitempty"`
	IsOnline       bool       `json:"is_online,omitempty"`

	// Always populated — cheap indexed query, needed by every consumer
	Goals []Goal `json:"goals"`

	// Populated from initiative_ledger_stats; zero when cron has not yet run
	Financials Financials `json:"financials"`

	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`

	// Internal fields — never serialised
	SourceDynamoTable  string `json:"-"`
	StripePlanID       string `json:"-"`
	StripeProductID    string `json:"-"`
	CiiProjectID       string `json:"-"`
	JobspringProjectID string `json:"-"`
	StacksIdentifier   string `json:"-"`
}

// InitiativeCreateInput is the request body for creating an initiative.
type InitiativeCreateInput struct {
	InitiativeType string `json:"initiative_type"`
	Name           string `json:"name"`
	Slug           string `json:"slug,omitempty"`
	Description    string `json:"description,omitempty"`
	Industry       string `json:"industry,omitempty"`
	Color          string `json:"color,omitempty"`
	LogoURL        string `json:"logo_url,omitempty"`
	WebsiteURL     string `json:"website_url,omitempty"`
	CocURL         string `json:"coc_url,omitempty"`
	AcceptFunding  bool   `json:"accept_funding"`
}

// InitiativeUpdateInput is the request body for updating an initiative.
type InitiativeUpdateInput struct {
	Name          *string `json:"name,omitempty"`
	Slug          *string `json:"slug,omitempty"`
	Status        *string `json:"status,omitempty"`
	Description   *string `json:"description,omitempty"`
	Industry      *string `json:"industry,omitempty"`
	Color         *string `json:"color,omitempty"`
	LogoURL       *string `json:"logo_url,omitempty"`
	WebsiteURL    *string `json:"website_url,omitempty"`
	CocURL        *string `json:"coc_url,omitempty"`
	AcceptFunding *bool   `json:"accept_funding,omitempty"`
}
