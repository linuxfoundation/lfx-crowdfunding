// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Goal maps to the crowdfunding.initiative_goals table.
type Goal struct {
	ID            string    `json:"id"`
	InitiativeID  string    `json:"-"`
	Name          string    `json:"name"`
	AmountInCents int64     `json:"goal_amount_cents"`
	Allocation    string    `json:"allocation,omitempty"`
	RepoLink      string    `json:"repo_link,omitempty"`
	Description   string    `json:"description,omitempty"`
	Color         string    `json:"color,omitempty"`
	Icon          string    `json:"icon,omitempty"`
	SortOrder     int       `json:"-"`
	CreatedOn     time.Time `json:"-"`
	UpdatedOn     time.Time `json:"-"`

	// Transient — populated from Ledger subTotals at request time; absent when Ledger is unavailable.
	DonatedCents *int64 `json:"donated_cents,omitempty"`
	SpentCents   *int64 `json:"spent_cents,omitempty"`
}
