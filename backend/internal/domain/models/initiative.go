// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

import "time"

// Initiative maps to the crowdfunding.initiatives table.
// Balance and FundingStatus are NOT stored here — computed at read time from Ledger API.
type Initiative struct {
	ID                string    `json:"id"`
	InitiativeType    string    `json:"initiative_type"`
	SourceDynamoTable string    `json:"source_dynamo_table,omitempty"`
	OwnerID           string    `json:"owner_id"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug,omitempty"`
	Status            string    `json:"status,omitempty"`
	Industry          string    `json:"industry,omitempty"`
	Description       string    `json:"description,omitempty"`
	Color             string    `json:"color,omitempty"`
	LogoURL           string    `json:"logo_url,omitempty"`
	WebsiteURL        string    `json:"website_url,omitempty"`
	CocURL            string    `json:"coc_url,omitempty"`
	StripePlanID      string    `json:"stripe_plan_id,omitempty"`
	StripeProductID   string    `json:"stripe_product_id,omitempty"`
	AmountRaisedCents int64     `json:"amount_raised_in_cents"`
	AcceptFunding     bool      `json:"accept_funding"`
	// Project-only fields
	CiiProjectID      string    `json:"cii_project_id,omitempty"`
	JobspringProjectID string   `json:"jobspring_project_id,omitempty"`
	StacksIdentifier  string    `json:"stacks_identifier,omitempty"`
	// Entity-only fields
	EventbriteURL    string     `json:"eventbrite_url,omitempty"`
	ApplicationURL   string     `json:"application_url,omitempty"`
	EventStartDate   *time.Time `json:"event_start_date,omitempty"`
	EventEndDate     *time.Time `json:"event_end_date,omitempty"`
	Country          string     `json:"country,omitempty"`
	City             string     `json:"city,omitempty"`
	IsOnline         bool       `json:"is_online"`
	CreatedOn        time.Time  `json:"created_on"`
	UpdatedOn        time.Time  `json:"updated_on"`
}

// InitiativeDetail enriches Initiative with live data computed at read time.
type InitiativeDetail struct {
	Initiative
	Goals        []Goal        `json:"goals,omitempty"`
	Balance      *Balance      `json:"balance,omitempty"`
	CacheControl string        `json:"-"`
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
