// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

import "time"

// Goal maps to the crowdfunding.initiative_goals table.
type Goal struct {
	ID            string    `json:"id"`
	InitiativeID  string    `json:"initiative_id"`
	Name          string    `json:"name"`
	AmountInCents int64     `json:"amount_in_cents"`
	Allocation    string    `json:"allocation,omitempty"`
	RepoLink      string    `json:"repo_link,omitempty"`
	Description   string    `json:"description,omitempty"`
	Color         string    `json:"color,omitempty"`
	Icon          string    `json:"icon,omitempty"`
	SortOrder     int       `json:"sort_order"`
	CreatedOn     time.Time `json:"created_on"`
	UpdatedOn     time.Time `json:"updated_on"`
}
