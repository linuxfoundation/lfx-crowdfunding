// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

// MentorshipProgram holds the Mentorship program fields that mentorship-sync
// reads from Snowflake and writes into CF Postgres.
// Field names mirror the ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS columns.
type MentorshipProgram struct {
	JobspringProjectID string // upsert key — PROGRAM_ID from Snowflake
	Name               string // PROGRAM_NAME
	Status             string // PROGRAM_STATUS; normalised to lowercase in syncer
	OwnerLFUsername    string // OWNER_LF_USERNAME — LF SSO username of the program owner
	// Beneficiaries is nil when the source did not provide beneficiary data
	// (e.g. the Snowflake query does not yet fetch SELECTED_MENTEES).
	// A nil slice means "do not touch beneficiaries"; an empty non-nil slice
	// means "source returned zero beneficiaries — delete all existing rows".
	Beneficiaries []MentorshipBeneficiary
}

// MentorshipBeneficiary is one approved beneficiary on a program.
type MentorshipBeneficiary struct {
	Name  string
	Email string
}
