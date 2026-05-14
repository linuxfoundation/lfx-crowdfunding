// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Initiative maps to the crowdfunding.initiatives table.
// Balance and FundingStatus are NOT stored here — computed at read time from Ledger API.
type Initiative struct {
	ID                string `json:"id"`
	InitiativeType    string `json:"initiative_type"`
	SourceDynamoTable string `json:"source_dynamo_table,omitempty"`
	OwnerID           string `json:"owner_id"`
	Name              string `json:"name"`
	Slug              string `json:"slug,omitempty"`
	Status            string `json:"status,omitempty"`
	Industry          string `json:"industry,omitempty"`
	Description       string `json:"description,omitempty"`
	Color             string `json:"color,omitempty"`
	LogoURL           string `json:"logo_url,omitempty"`
	WebsiteURL        string `json:"website_url,omitempty"`
	CocURL            string `json:"coc_url,omitempty"`
	StripePlanID      string `json:"stripe_plan_id,omitempty"`
	StripeProductID   string `json:"stripe_product_id,omitempty"`
	AmountRaisedCents int64  `json:"amount_raised_in_cents"`
	AcceptFunding     bool   `json:"accept_funding"`
	// Project-only fields
	CiiProjectID       string `json:"cii_project_id,omitempty"`
	JobspringProjectID string `json:"jobspring_project_id,omitempty"`
	StacksIdentifier   string `json:"stacks_identifier,omitempty"`
	// Entity-only fields
	EventbriteURL  string     `json:"eventbrite_url,omitempty"`
	ApplicationURL string     `json:"application_url,omitempty"`
	EventStartDate *time.Time `json:"event_start_date,omitempty"`
	EventEndDate   *time.Time `json:"event_end_date,omitempty"`
	Country        string     `json:"country,omitempty"`
	City           string     `json:"city,omitempty"`
	IsOnline       bool       `json:"is_online"`
	CreatedOn      time.Time  `json:"created_on"`
	UpdatedOn      time.Time  `json:"updated_on"`
	// Computed from initiative_financial_stats materialized view (nil on Create/Update responses).
	Stats         *InitiativeStats `json:"initiative_stats,omitempty"`
	FundingStatus *FundingStatus   `json:"funding_status,omitempty"`
}

// InitiativeStats holds engagement metrics derived from donations and subscriptions.
// Populated from the initiative_financial_stats materialized view.
type InitiativeStats struct {
	Backers  int `json:"backers"`
	Sponsors int `json:"sponsors"`
}

// FundingStatus holds subscription and goal aggregates from the initiative_financial_stats view.
// AmountRaisedCents is NOT included here — it comes from the Ledger API (see Balance).
type FundingStatus struct {
	TotalAnnualGoalInCents             int64  `json:"total_annual_goal_in_cents"`
	TotalDonationCount                 int    `json:"total_donation_count"`
	TotalSubscriptionCount             int    `json:"total_subscription_count"`
	AnnualSubscriptionAmountInCents    int64  `json:"annual_subscription_amount_in_cents"`
	AnnualSubscriptionRemainingInCents *int64 `json:"annual_subscription_remaining_in_cents,omitempty"`
}

// InitiativeDetail enriches Initiative with live data computed at read time.
type InitiativeDetail struct {
	Initiative
	Goals        []Goal   `json:"goals,omitempty"`
	Balance      *Balance `json:"balance,omitempty"`
	CacheControl string   `json:"-"`
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

// Balance is fetched from the Ledger API at read time — never stored in PostgreSQL.
type Balance struct {
	InitiativeID        string `json:"initiative_id"`
	TotalRaisedCents    int64  `json:"total_raised_cents"`
	TotalDisbursedCents int64  `json:"total_disbursed_cents"`
	AvailableCents      int64  `json:"available_cents"`
}
