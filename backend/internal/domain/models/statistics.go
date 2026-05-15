// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

// PlatformStatistics holds platform-wide aggregates for the landing page hero section.
// Values are derived from initiative_ledger_stats which is refreshed by ledger-stats-sync.
type PlatformStatistics struct {
	TotalRaisedCents int64 `json:"total_raised_cents"`
	TotalSupporters  int   `json:"total_supporters"`
	TotalInitiatives int   `json:"total_initiatives"`
}
