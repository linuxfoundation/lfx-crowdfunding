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

	DonorName     string `json:"donor_name,omitempty"`
	DonorType     string `json:"donor_type,omitempty"` // "organization" | "individual"
	DonorLogoURL  string `json:"donor_logo_url,omitempty"`
	DonorUsername string `json:"donor_username,omitempty"` // reserved; not yet populated

	// Internal: used by the service to look up CF DB records. Not serialised.
	LedgerUserID    string `json:"-"`
	LedgerOrgID     string `json:"-"`
	LedgerProjectID string `json:"-"`
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
