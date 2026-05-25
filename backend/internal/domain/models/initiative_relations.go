// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

// GoalInput carries the create-time data for a single funding goal.
// It mirrors the writable columns of initiative_goals, excluding id,
// initiative_id, created_on, and updated_on which are assigned by the database.
type GoalInput struct {
	Name          string `json:"name"`
	AmountInCents int64  `json:"amount_in_cents"`
	Allocation    string `json:"allocation,omitempty"`
	RepoLink      string `json:"repo_link,omitempty"`
	Description   string `json:"description,omitempty"`
	Color         string `json:"color,omitempty"`
	Icon          string `json:"icon,omitempty"`
	SortOrder     int    `json:"sort_order"`
}

// BeneficiaryInput represents one row in initiative_beneficiaries.
type BeneficiaryInput struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// CustomWebsiteInput represents one row in initiative_custom_websites.
type CustomWebsiteInput struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url"`
}

// ContributorInput represents one row in initiative_contributors (project only).
type ContributorInput struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// MentorInput represents one row in initiative_mentors (mentorship only).
type MentorInput struct {
	Name         string `json:"name,omitempty"`
	Email        string `json:"email,omitempty"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	Introduction string `json:"introduction,omitempty"`
}

// CustomTermInput describes a custom term window for a mentorship program.
// Only persisted when TermName is non-empty.
type CustomTermInput struct {
	TermName   string `json:"term_name,omitempty"`
	StartMonth string `json:"start_month,omitempty"`
	EndMonth   string `json:"end_month,omitempty"`
	Year       int    `json:"year"`
}

// ProgramInfoInput holds all mentorship program configuration in a single struct.
// It writes to initiative_program_info_terms, _skills, _config, and _custom_term.
type ProgramInfoInput struct {
	Terms           []string         `json:"terms,omitempty"`
	Skills          []string         `json:"skills,omitempty"`
	TermsConditions bool             `json:"terms_conditions"`
	CustomTerm      *CustomTermInput `json:"custom_term,omitempty"`
}

// SponsorshipTierInput represents one row in initiative_sponsorship_tiers (entity only).
type SponsorshipTierInput struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Minimum     int64  `json:"minimum"`
	SortOrder   int    `json:"sort_order"`
}

// OSTIFDetailInput holds OSTIF-specific funding detail for initiative_ostif_detail.
type OSTIFDetailInput struct {
	MonetizationStrategy    string `json:"monetization_strategy,omitempty"`
	CurrentSecurityStrategy string `json:"current_security_strategy,omitempty"`
	LicenseType             string `json:"license_type,omitempty"`
	TotalBudgetInCents      int64  `json:"total_budget_in_cents"`
	TermsConditions         bool   `json:"terms_conditions"`
}

// ContactInput represents one row in initiative_contacts (ostif only).
// ContactType must be one of "primary", "secondary", or "technical_lead".
type ContactInput struct {
	ContactType            string `json:"contact_type"`
	FirstName              string `json:"first_name,omitempty"`
	LastName               string `json:"last_name,omitempty"`
	Email                  string `json:"email,omitempty"`
	PhoneNumber            string `json:"phone_number,omitempty"`
	OtherContactOption     string `json:"other_contact_option,omitempty"`
	PreferredContactMethod string `json:"preferred_contact_method,omitempty"`
}
