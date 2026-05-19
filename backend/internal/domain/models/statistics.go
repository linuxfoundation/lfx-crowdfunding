// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

// PlatformStatistics holds platform-wide aggregates for the landing page hero section.
// Values are derived from initiative_ledger_stats which is refreshed by ledger-stats-sync.
type PlatformStatistics struct {
	TotalRaisedCents int64 `json:"total_raised_cents"`
	TotalSupporters  int64 `json:"total_supporters"`
	TotalInitiatives int64 `json:"total_initiatives"`
}

// PlatformDetails is returned by GET /v1/statistics/platform.
// Aggregates category totals, donor split, and top sponsors from Ledger.
type PlatformDetails struct {
	TotalRaisedCents   int64          `json:"total_raised_cents"`
	TotalSupporters    int64          `json:"total_supporters"`
	OrganizationsCents int64          `json:"organizations_cents"`
	IndividualsCents   int64          `json:"individuals_cents"`
	Categories         []CategoryTotal `json:"categories"`
	TopOrganizations   []SponsorEntry  `json:"top_organizations"`
	TopIndividuals     []SponsorEntry  `json:"top_individuals"`
}

// CategoryTotal holds the aggregate donation total for one Ledger txnCategory.
type CategoryTotal struct {
	Name       string `json:"name"`
	TotalCents int64  `json:"total_cents"`
	Count      int    `json:"count"`
}

// SponsorEntry represents a single top donor (org or individual).
type SponsorEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	TotalCents int64  `json:"total_cents"`
}

// PlatformMonthly is returned by GET /v1/statistics/monthly.
type PlatformMonthly struct {
	Buckets []MonthlyBucket `json:"buckets"`
}

// MonthlyBucket holds aggregate donation data for a single calendar month.
type MonthlyBucket struct {
	Year       int   `json:"year"`
	Month      int   `json:"month"`
	TotalCents int64 `json:"total_cents"`
	Supporters int64 `json:"supporters"`
}

// RecentDonation is one entry in the platform-wide recent donations feed.
type RecentDonation struct {
	TxnID          string `json:"txn_id"`
	ProjectID      string `json:"project_id"`
	DonorName      string `json:"donor_name"`
	DonorAvatarURL string `json:"donor_avatar_url,omitempty"`
	DonorType      string `json:"donor_type"` // "organization" or "individual"
	AmountCents    int64  `json:"amount_cents"`
	TxnDate        int64  `json:"txn_date"`
	Category       string `json:"category,omitempty"`
}

// RecentDonationsResponse wraps the recent donations list.
type RecentDonationsResponse struct {
	Data []RecentDonation `json:"data"`
}
