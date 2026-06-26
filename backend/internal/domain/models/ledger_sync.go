// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

// LedgerRawSponsor is one sponsor entry in the Ledger GET /balance response,
// containing the entity ID and their total contribution in cents.
type LedgerRawSponsor struct {
	ID    string `json:"id"`
	Total int64  `json:"total"`
}

// LedgerRawSponsors contains the raw sponsor entries as returned by the
// Ledger GET /balance endpoint.  The CronJob enriches these with names and
// avatar/logo URLs before writing them to initiative_ledger_stats.sponsors.
type LedgerRawSponsors struct {
	Orgs        []LedgerRawSponsor `json:"orgs"`
	Individuals []LedgerRawSponsor `json:"individuals"`
}

// LedgerRawBalance is one entry in the Ledger GET /balance bulk response.
// Field names match the Ledger service JSON contract.
//
//	totalDebit  and feeBalance are stored by Ledger as negative integers.
//	The CronJob converts both to their absolute values before persisting.
type LedgerRawBalance struct {
	ProjectID        string            `json:"projectID"`
	TotalCredit      int64             `json:"totalCredit"`
	TotalDebit       int64             `json:"totalDebit"` // negative; ABS before storing
	TotalBalance     int64             `json:"totalBalance"`
	AvailableBalance int64             `json:"availableBalance"`
	FeeBalance       int64             `json:"feeBalance"` // negative; ABS before storing
	Backers          int               `json:"backers"`    // distinct individual-user count (not used; supporters derived from Sponsors)
	Sponsors         LedgerRawSponsors `json:"sponsors"`
}

// LedgerAllBalances is the top-level response from the Ledger GET /balance endpoint.
type LedgerAllBalances struct {
	Balances []LedgerRawBalance `json:"balances"`
}

// LedgerSponsorOrg is an enriched organisation entry persisted inside the
// sponsors JSONB column of initiative_ledger_stats.
type LedgerSponsorOrg struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Total     int64  `json:"total"`
}

// LedgerSponsorUser is an enriched user entry persisted inside the sponsors
// JSONB column of initiative_ledger_stats.
type LedgerSponsorUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Total     int64  `json:"total"`
}

// LedgerSponsorList is the enriched value stored as JSONB in the
// initiative_ledger_stats.sponsors column.
type LedgerSponsorList struct {
	Orgs        []LedgerSponsorOrg  `json:"orgs"`
	Individuals []LedgerSponsorUser `json:"individuals"`
}

// LedgerStats holds one row destined for initiative_ledger_stats.
// All *Cents values are non-negative; negative Ledger values are ABS'd on write.
type LedgerStats struct {
	InitiativeID          string
	TotalRaisedCents      int64
	TotalDebitedCents     int64
	TotalBalanceCents     int64
	AvailableBalanceCents int64
	FeeBalanceCents       int64
	Supporters            int // count of unique org + individual donors from sponsors lists
	Sponsors              LedgerSponsorList
}
