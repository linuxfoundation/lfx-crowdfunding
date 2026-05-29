// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

// GoalInput carries the create-time data for a single funding goal.
// It mirrors the writable columns of initiative_goals, excluding id,
// initiative_id, created_on, and updated_on which are assigned by the database.
type GoalInput struct {
	Name          string `json:"name"`
	AmountInCents int64  `json:"amount_cents"`
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
	TotalBudgetInCents      int64  `json:"total_budget_cents"`
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

// ── Read models (populated on detail GET, absent in list responses) ───────────

// Beneficiary is a read row from initiative_beneficiaries.
type Beneficiary struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// CustomWebsite is a read row from initiative_custom_websites.
type CustomWebsite struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	URL  string `json:"url"`
}

// Contributor is a read row from initiative_contributors (project only).
type Contributor struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// Mentor is a read row from initiative_mentors (mentorship only).
type Mentor struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	Email        string `json:"email,omitempty"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	Introduction string `json:"introduction,omitempty"`
}

// ProgramCustomTerm is read from initiative_program_info_custom_term.
type ProgramCustomTerm struct {
	TermName   string `json:"term_name"`
	StartMonth string `json:"start_month,omitempty"`
	EndMonth   string `json:"end_month,omitempty"`
	Year       int    `json:"year"`
}

// ProgramInfo aggregates rows from the four initiative_program_info_* tables
// (mentorship only). Nil when the initiative has no program info.
type ProgramInfo struct {
	Terms           []string           `json:"terms"`
	Skills          []string           `json:"skills"`
	TermsConditions bool               `json:"terms_conditions"`
	CustomTerm      *ProgramCustomTerm `json:"custom_term,omitempty"`
}

// SponsorshipTier is a read row from initiative_sponsorship_tiers (entity only).
type SponsorshipTier struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Minimum     int64  `json:"minimum"`
}

// OSTIFDetail is read from initiative_ostif_detail (ostif only).
type OSTIFDetail struct {
	MonetizationStrategy    string `json:"monetization_strategy,omitempty"`
	CurrentSecurityStrategy string `json:"current_security_strategy,omitempty"`
	LicenseType             string `json:"license_type,omitempty"`
	TotalBudgetInCents      int64  `json:"total_budget_cents"`
	TermsConditions         bool   `json:"terms_conditions"`
}

// Contact is a read row from initiative_contacts (ostif only).
type Contact struct {
	ID                     string `json:"id"`
	ContactType            string `json:"contact_type"`
	FirstName              string `json:"first_name,omitempty"`
	LastName               string `json:"last_name,omitempty"`
	Email                  string `json:"email,omitempty"`
	PhoneNumber            string `json:"phone_number,omitempty"`
	OtherContactOption     string `json:"other_contact_option,omitempty"`
	PreferredContactMethod string `json:"preferred_contact_method,omitempty"`
}

// GitHubStats is read from initiative_github_stats (project only).
type GitHubStats struct {
	Forks      int `json:"forks"`
	Stars      int `json:"stars"`
	OpenIssues int `json:"open_issues"`
}
