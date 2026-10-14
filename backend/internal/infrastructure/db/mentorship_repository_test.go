// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// compile-time check that MentorshipRepositoryImpl satisfies domain.MentorshipRepository.
var _ domain.MentorshipRepository = (*MentorshipRepositoryImpl)(nil)

// A Snowflake program must never take over a non-mentorship initiative or
// rewrite an existing user's profile (lfx-self-serve-ops#277).
func TestMentorshipRepository_UpsertProgram_OnlyTouchesOwnedRows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "project-owner")
	project := seedInitiative(t, ctx, owner.ID, "Project Fund", "project-fund")
	repo := NewMentorshipRepository(testPool)

	hijack := models.MentorshipProgram{
		JobspringProjectID: project.ID, Name: "Hijacked", Status: "published",
		Slug: "hijacked", OwnerLFUsername: owner.Username, OwnerEmail: "attacker@example.com",
	}
	if _, err := repo.UpsertProgram(ctx, hijack); err == nil {
		t.Fatal("UpsertProgram() on a project initiative ID: want error, got nil")
	}

	got, err := NewInitiativeRepository(testPool).GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != "Project Fund" || got.InitiativeType != "project" || got.OwnerID != owner.ID {
		t.Errorf("project initiative modified: %+v", got)
	}
	var email string
	if err := testPool.QueryRow(ctx, `SELECT email FROM crowdfunding.users WHERE id = $1`, owner.ID).Scan(&email); err != nil {
		t.Fatalf("select email: %v", err)
	}
	if email != owner.Email {
		t.Errorf("existing user email = %q, want unchanged %q", email, owner.Email)
	}

	// A new mentorship ID still inserts, and re-upserting it updates it.
	program := models.MentorshipProgram{
		JobspringProjectID: uuid.New().String(), Name: "Mentorship", Status: "published",
		Slug: "mentorship", OwnerLFUsername: "mentor",
	}
	if _, err := repo.UpsertProgram(ctx, program); err != nil {
		t.Fatalf("UpsertProgram() insert error = %v", err)
	}
	program.Name = "Mentorship Renamed"
	if _, err := repo.UpsertProgram(ctx, program); err != nil {
		t.Fatalf("UpsertProgram() update error = %v", err)
	}
	var name string
	if err := testPool.QueryRow(ctx, `SELECT name FROM crowdfunding.initiatives WHERE id = $1`, program.JobspringProjectID).Scan(&name); err != nil {
		t.Fatalf("select name: %v", err)
	}
	if name != "Mentorship Renamed" {
		t.Errorf("mentorship name = %q, want %q", name, "Mentorship Renamed")
	}
}

// Jobspring allows two programs with the same slug; the second must still sync
// (with a suffixed slug) rather than fail the unique index forever.
func TestMentorshipRepository_UpsertProgram_SuffixesTakenSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")
	repo := NewMentorshipRepository(testPool)

	first := models.MentorshipProgram{
		JobspringProjectID: uuid.New().String(), Name: "Newlib Optimization", Status: "published",
		Slug: "newlib-optimization", OwnerLFUsername: "flashy",
	}
	if _, err := repo.UpsertProgram(ctx, first); err != nil {
		t.Fatalf("UpsertProgram() first error = %v", err)
	}

	dup := first
	dup.JobspringProjectID = uuid.New().String()
	dup.Status = "pending"
	if _, err := repo.UpsertProgram(ctx, dup); err != nil {
		t.Fatalf("UpsertProgram() duplicate slug error = %v", err)
	}

	slugOf := func(id string) string {
		var s string
		if err := testPool.QueryRow(ctx, `SELECT slug FROM crowdfunding.initiatives WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatalf("select slug for %s: %v", id, err)
		}
		return s
	}
	if got := slugOf(first.JobspringProjectID); got != "newlib-optimization" {
		t.Errorf("first slug = %q, want unchanged", got)
	}
	want := "newlib-optimization-" + dup.JobspringProjectID[:8]
	if got := slugOf(dup.JobspringProjectID); got != want {
		t.Errorf("duplicate slug = %q, want %q", got, want)
	}

	// Re-syncing both leaves slugs alone.
	for _, p := range []models.MentorshipProgram{first, dup} {
		if _, err := repo.UpsertProgram(ctx, p); err != nil {
			t.Fatalf("UpsertProgram() re-sync error = %v", err)
		}
	}
	if got := slugOf(first.JobspringProjectID); got != "newlib-optimization" {
		t.Errorf("first slug after re-sync = %q", got)
	}
	if got := slugOf(dup.JobspringProjectID); got != want {
		t.Errorf("duplicate slug after re-sync = %q, want %q", got, want)
	}
}
