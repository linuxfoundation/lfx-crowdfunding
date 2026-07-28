// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package models defines the domain model types shared across the application.
package models

import "time"

// Announcement maps to the crowdfunding.initiative_announcements table.
type Announcement struct {
	ID           string    `json:"id"`
	InitiativeID string    `json:"initiative_id"`
	CreatedBy    string    `json:"created_by"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	CreatedOn    time.Time `json:"created_on"`
	UpdatedOn    time.Time `json:"updated_on"`
}

// AnnouncementCreateInput is the request body for creating an announcement.
type AnnouncementCreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// AnnouncementUpdateInput is the request body for updating an announcement.
type AnnouncementUpdateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// AnnouncementFilter constrains list queries for announcements.
type AnnouncementFilter struct {
	Limit  int
	Offset int
}
