// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

import "time"

// PaymentMethodOffline marks a donation LF finance received by invoice or wire
// and an approver recorded (#464). It has no Stripe IDs.
const PaymentMethodOffline = "offline"

// OfflineDonationInput is the body of POST /crowdfunding/offline-donations.
type OfflineDonationInput struct {
	InitiativeID string    `json:"initiative_id"`
	Organization *OrgClaim `json:"organization"`
	// Amount is in dollars as a decimal string ("1300" or "1300.50"), so it is
	// converted to cents exactly instead of through a float.
	Amount string `json:"amount"`
	// ReceivedOn is the date LF finance received the money, YYYY-MM-DD.
	ReceivedOn string `json:"received_on"`
	Category   string `json:"category"`
	// SourceTxnID is the Ledger sourceTxnID: the reference approvers already
	// use (e.g. "LFX Mentorship CNCF Fall 2026 Row 52"). The Ledger keeps one
	// row per sourceTxnID, so it must be unique.
	SourceTxnID string `json:"source_txn_id"`
}

// OfflineDonation is a recorded offline org donation, as returned to approvers.
type OfflineDonation struct {
	ID                 string    `json:"id"`
	InitiativeID       string    `json:"initiative_id"`
	InitiativeName     string    `json:"initiative_name,omitempty"`
	InitiativeSlug     string    `json:"initiative_slug,omitempty"`
	OrganizationName   string    `json:"organization_name"`
	OrganizationDomain string    `json:"organization_domain"`
	AmountCents        int64     `json:"amount_cents"`
	Category           string    `json:"category"`
	ReceivedOn         string    `json:"received_on"`
	SourceTxnID        string    `json:"source_txn_id"`
	Status             string    `json:"status"`
	RecordedBy         string    `json:"recorded_by,omitempty"`
	CreatedOn          time.Time `json:"created_on"`

	// Internal: the offline account row, the approver's row and the matched b2b_org.
	UserID       string `json:"-"`
	RecordedByID string `json:"-"`
	B2BOrgSFID   string `json:"-"`
}
