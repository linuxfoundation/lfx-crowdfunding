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
	Description        string // PROGRAM_DESCRIPTION
	Slug               string // program_slug
	Industry           string // PROGRAM_TECHNOLOGY (comma-separated list)
	OwnerLFUsername    string // OWNER_LF_USERNAME — LF SSO username of the program owner
	OwnerEmail         string // OWNER_EMAIL
	OwnerFirstName     string // OWNER_FIRST_NAME
	OwnerLastName      string // OWNER_LAST_NAME
	OwnerAvatarURL     string // OWNER_AVATAR_URL

	// Skills is nil when the source did not provide skills data.
	// A nil slice means "do not touch skills"; non-nil (even empty) replaces all existing rows.
	Skills []string

	// Mentors is nil when the source did not provide mentor data.
	// A nil slice means "do not touch mentors"; non-nil (even empty) replaces all existing rows.
	Mentors []MentorshipMentor

	// Beneficiaries is nil when the source did not provide beneficiary data.
	// A nil slice means "do not touch beneficiaries"; an empty non-nil slice
	// means "source returned zero beneficiaries — delete all existing rows".
	Beneficiaries []MentorshipBeneficiary
}

// MentorshipBeneficiary is one approved beneficiary (selected mentee) on a program.
type MentorshipBeneficiary struct {
	Name  string // derived: first_name + " " + last_name
	Email string
}

// MentorshipMentor is one mentor on a program.
type MentorshipMentor struct {
	Name      string // derived: first_name + " " + last_name
	Email     string
	AvatarURL string
}
