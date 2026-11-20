// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

import "time"

// Ledger adjustment kinds (#489): money an approver takes out of an initiative.
const (
	// AdjustmentKindReturn sends unused funds back to the donor org, e.g. a
	// mentee's unpaid stipend. Posted as a negative credit with the org.
	AdjustmentKindReturn = "return"
	// AdjustmentKindPayout pays a person out of a fund, e.g. a reward payout.
	// Posted as a negative debit with the payee.
	AdjustmentKindPayout = "payout"
)

// LedgerAdjustmentInput is the body of POST /crowdfunding/ledger-adjustments.
type LedgerAdjustmentInput struct {
	Kind         string `json:"kind"`
	InitiativeID string `json:"initiative_id"`
	// Organization is the donor whose money it is: required for a return,
	// optional for a payout.
	Organization *OrgClaim `json:"organization"`
	// PaidTo is the payee of a payout; the public Financials tab shows it.
	PaidTo string `json:"paid_to"`
	// Amount is the positive dollar amount taken out ("1500" or "1500.50");
	// the Ledger row gets it negated.
	Amount string `json:"amount"`
	// AdjustedOn is the date of the return or payout, YYYY-MM-DD.
	AdjustedOn  string `json:"adjusted_on"`
	Category    string `json:"category"`
	Reason      string `json:"reason"`
	SourceTxnID string `json:"source_txn_id"`
}

// LedgerAdjustment is a recorded return or payout, as returned to approvers.
type LedgerAdjustment struct {
	ID                 string    `json:"id"`
	Kind               string    `json:"kind"`
	InitiativeID       string    `json:"initiative_id"`
	InitiativeName     string    `json:"initiative_name,omitempty"`
	InitiativeSlug     string    `json:"initiative_slug,omitempty"`
	OrganizationName   string    `json:"organization_name"`
	OrganizationDomain string    `json:"organization_domain"`
	PaidTo             string    `json:"paid_to"`
	AmountCents        int64     `json:"amount_cents"`
	Category           string    `json:"category"`
	Reason             string    `json:"reason"`
	AdjustedOn         string    `json:"adjusted_on"`
	SourceTxnID        string    `json:"source_txn_id"`
	Status             string    `json:"status"`
	RecordedBy         string    `json:"recorded_by,omitempty"`
	CreatedOn          time.Time `json:"created_on"`

	// Internal: the approver's row and the matched b2b_org.
	RecordedByID string `json:"-"`
	B2BOrgSFID   string `json:"-"`
}
