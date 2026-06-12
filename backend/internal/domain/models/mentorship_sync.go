// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

// MentorshipProgram holds the Mentorship program fields that mentorship-sync
// reads from Snowflake and writes into CF Postgres.
// Field names mirror the ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS columns.
type MentorshipProgram struct {
	JobspringProjectID string // upsert key — matches initiatives.jobspring_project_id
	Name               string
	Status             string // Snowflake value; 'hide' is normalised → 'hidden' in syncer
	MenteeGoalCents    int64  // mentee budget goal in cents
	Beneficiaries      []MentorshipBeneficiary
}

// MentorshipBeneficiary is one approved beneficiary on a program.
type MentorshipBeneficiary struct {
	Name  string
	Email string
}
