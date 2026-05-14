// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

import "time"

// CategoryStat holds aggregated funding data for a single initiative category.
type CategoryStat struct {
	Category      string `json:"category"`
	TotalCents    int64  `json:"total_cents"`
	SupporterCount int   `json:"supporter_count"`
}

// MonthlyDonationStat holds aggregated donation data for a calendar month.
type MonthlyDonationStat struct {
	Month         string `json:"month"` // "YYYY-MM"
	TotalCents    int64  `json:"total_cents"`
	NewSupporters int    `json:"new_supporters"`
}

// DonorStat holds a ranked donor entry for top organizations or top individuals.
type DonorStat struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	TotalCents int64  `json:"total_cents"`
	Rank       int    `json:"rank"`
}

// PlatformStatistics aggregates platform-wide funding data from the CF database.
// total_raised_cents is sourced from initiatives.amount_raised_in_cents (refreshed hourly by ledger-stats-sync).
// All other fields are computed live from the donations and subscriptions tables.
type PlatformStatistics struct {
	TotalRaisedCents    int64                 `json:"total_raised_cents"`
	TotalSupporters     int                   `json:"total_supporters"`
	AverageDonationCents int64               `json:"average_donation_cents"`
	FundingByCategory   []CategoryStat        `json:"funding_by_category"`
	MonthlyDonations    []MonthlyDonationStat `json:"monthly_donations"`
	TopOrganizations    []DonorStat           `json:"top_organizations"`
	TopIndividuals      []DonorStat           `json:"top_individuals"`
	ComputedAt          time.Time             `json:"computed_at"`
}
