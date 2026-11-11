// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

const (
	TransactionTypeDonation      = "donation"
	TransactionTypeReimbursement = "reimbursement"

	TransactionQueryTypeDonations = "donations"
	TransactionQueryTypeExpenses  = "expenses"

	TransactionResponseTypeList        = "list"
	TransactionResponseTypeCategorized = "categorized"
)

// Transaction represents a single donation or disbursement returned by the Ledger service.
type Transaction struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"` // models.TransactionTypeDonation | models.TransactionTypeReimbursement
	AmountCents    int64     `json:"amount_cents"`
	Date           time.Time `json:"date"`
	Category       string    `json:"category,omitempty"`
	Recurring      bool      `json:"recurring"`
	InitiativeName string    `json:"initiative_name,omitempty"`
	InitiativeID   string    `json:"initiative_id,omitempty"` // set on the caller's own transactions (GetAllMyTransactions)

	DonorName     string `json:"donor_name,omitempty"`
	DonorType     string `json:"donor_type,omitempty"` // "organization" | "individual"
	DonorLogoURL  string `json:"donor_logo_url,omitempty"`
	DonorUsername string `json:"donor_username,omitempty"` // reserved; not yet populated

	// Anonymous is set when the donor hid their name on this row (#466). On
	// public lists the person then shows as "Anonymous"; the donor's own lists
	// keep their name and also carry the CF record to toggle it with.
	Anonymous      bool   `json:"anonymous,omitempty"`
	DonationID     string `json:"donation_id,omitempty"`     // caller's own rows only
	SubscriptionID string `json:"subscription_id,omitempty"` // caller's own rows only

	// Internal: used by the service to look up CF DB records. Not serialised.
	LedgerUserID    string `json:"-"`
	LedgerOrgID     string `json:"-"`
	LedgerOrgName   string `json:"-"`
	LedgerOrgDomain string `json:"-"`
	LedgerProjectID string `json:"-"`
	// LedgerSourceTxnID is the Stripe charge (or PaymentIntent) ID and
	// LedgerSubscriptionID the Stripe subscription ID; they match the row to its
	// CF donation or subscription.
	LedgerSourceTxnID    string `json:"-"`
	LedgerSubscriptionID string `json:"-"`
}

// TransactionList wraps a paginated list of transactions.
type TransactionList struct {
	ResponseType string        `json:"response_type,omitempty"`
	Data         []Transaction `json:"data"`
	TotalCount   int           `json:"total_count"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
}

// CategorizedTransactions groups positive credit transactions by donor type.
type CategorizedTransactions struct {
	ResponseType             string        `json:"response_type,omitempty"`
	IndividualTransactions   []Transaction `json:"individual_transactions"`
	OrganizationTransactions []Transaction `json:"organization_transactions"`
	TotalCount               int           `json:"total_count"`
	Limit                    int           `json:"limit"`
	Offset                   int           `json:"offset"`
}

// DonationStats is the caller's donation aggregate returned by GET /crowdfunding/me/donations/stats.
type DonationStats struct {
	TotalDonatedCents    int64 `json:"total_donated_cents"`
	InitiativesSupported int64 `json:"initiatives_supported"`
}

// LedgerRef is the CF donation or subscription a ledger row was matched to.
type LedgerRef struct {
	ID        string
	Anonymous bool
}

// LedgerRefMatches holds the CF records matched to a page of ledger rows,
// keyed by the ledger row's Stripe charge/PaymentIntent ID (Donations) or
// Stripe subscription ID (Subscriptions).
type LedgerRefMatches struct {
	Donations     map[string]LedgerRef
	Subscriptions map[string]LedgerRef
}
