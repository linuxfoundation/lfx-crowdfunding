// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Transaction represents a single donation or disbursement returned by the Ledger service.
type Transaction struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`         // "donation" | "reimbursement"
	AmountCents int64     `json:"amount_cents"`
	Date        time.Time `json:"date"`
	Category    string    `json:"category,omitempty"`

	DonorName     string `json:"donor_name,omitempty"`
	DonorType     string `json:"donor_type,omitempty"` // "organization" | "individual"
	DonorLogoURL  string `json:"donor_logo_url,omitempty"`
	DonorUsername string `json:"donor_username,omitempty"` // reserved; not yet populated

	// Internal: used by the service to look up CF DB records. Not serialised.
	LedgerUserID string `json:"-"`
	LedgerOrgID  string `json:"-"`
}

// TransactionList wraps a paginated list of transactions.
type TransactionList struct {
	Data       []Transaction `json:"data"`
	TotalCount int           `json:"total_count"`
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
}
