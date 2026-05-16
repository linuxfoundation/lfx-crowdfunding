// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

// Sponsor is a single sponsor entry in the initiative overview response.
// Name and AvatarURL are sourced from initiative_ledger_stats.sponsors (synced by CronJob).
type Sponsor struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	TotalCents int64  `json:"total_cents"`
}

// LedgerBalanceSummary is the live balance fetched from Ledger at request time.
// Nil in the Initiative response when the Ledger Service is unavailable.
type LedgerBalanceSummary struct {
	TotalRaisedCents    int64 `json:"total_raised_cents"`
	TotalDisbursedCents int64 `json:"total_disbursed_cents"`
	AvailableCents      int64 `json:"available_cents"`
}
