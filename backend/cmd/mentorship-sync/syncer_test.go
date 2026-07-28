// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// ─── Mock implementations ────────────────────────────────────────────────────

type mockMentorshipSource struct {
	programs []models.MentorshipProgram
	err      error
}

func (m *mockMentorshipSource) FetchPrograms(_ context.Context) ([]models.MentorshipProgram, error) {
	return m.programs, m.err
}

type mockMentorshipRepo struct {
	upsertedPrograms []models.MentorshipProgram
	programErr       error
	jobspringIDs     []string
	listErr          error
}

func (m *mockMentorshipRepo) UpsertProgram(_ context.Context, p models.MentorshipProgram) (string, error) {
	if m.programErr != nil {
		return "", m.programErr
	}
	m.upsertedPrograms = append(m.upsertedPrograms, p)
	return "initiative-uuid-" + p.JobspringProjectID, nil
}

func (m *mockMentorshipRepo) ListJobspringIDs(_ context.Context) ([]string, error) {
	return m.jobspringIDs, m.listErr
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestSyncer_Run_upsertsAllPrograms(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Prog A", Status: "published", OwnerLFUsername: "alice", Beneficiaries: []models.MentorshipBeneficiary{{Name: "Alice", Email: "a@x.com"}}},
			{JobspringProjectID: "js-2", Name: "Prog B", Status: "pending", OwnerLFUsername: "bob", Beneficiaries: nil},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	result, err := s.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.total != 2 {
		t.Errorf("total: got %d, want 2", result.total)
	}
	if result.upserted != 2 {
		t.Errorf("upserted: got %d, want 2", result.upserted)
	}
	if result.errors != 0 {
		t.Errorf("errors: got %d, want 0", result.errors)
	}
}

func TestSyncer_Run_normalisesStatusToLowercase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"Published", "published"},
		{"Pending", "pending"},
		{"Hidden", "hidden"},
		{"Rejected", "rejected"},
		{"hide", "hidden"}, // Jobspring legacy value
	}

	for _, tc := range cases {
		src := &mockMentorshipSource{
			programs: []models.MentorshipProgram{
				{JobspringProjectID: "js-1", Name: "Test", Status: tc.input, OwnerLFUsername: "testowner"},
			},
		}
		repo := &mockMentorshipRepo{}

		s := newSyncer(repo, src, discardLogger())
		if _, err := s.Run(context.Background()); err != nil {
			t.Fatalf("input %q: unexpected error: %v", tc.input, err)
		}
		if len(repo.upsertedPrograms) != 1 {
			t.Fatalf("input %q: expected 1 upsert, got %d", tc.input, len(repo.upsertedPrograms))
		}
		if got := repo.upsertedPrograms[0].Status; got != tc.want {
			t.Errorf("input %q: status got %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSyncer_Run_beneficiariesUpsertedForInitiative(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{
				JobspringProjectID: "js-1",
				Name:               "Prog A",
				Status:             "published",
				OwnerLFUsername:    "alice",
				Beneficiaries: []models.MentorshipBeneficiary{
					{Name: "Alice", Email: "alice@x.com"},
				},
			},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upsertedPrograms) != 1 {
		t.Fatalf("expected 1 upserted program, got %d", len(repo.upsertedPrograms))
	}
	bens := repo.upsertedPrograms[0].Beneficiaries
	if len(bens) != 1 || bens[0].Email != "alice@x.com" {
		t.Errorf("beneficiaries: got %+v", bens)
	}
}

func TestSyncer_Run_nilBeneficiariesSkipsUpsert(t *testing.T) {
	t.Parallel()

	// nil Beneficiaries = source did not provide beneficiary data (e.g. Snowflake client)
	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Prog A", Status: "published", OwnerLFUsername: "alice", Beneficiaries: nil},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upsertedPrograms) != 1 {
		t.Fatalf("expected 1 upserted program, got %d", len(repo.upsertedPrograms))
	}
	if repo.upsertedPrograms[0].Beneficiaries != nil {
		t.Error("Beneficiaries should be nil when not provided by source")
	}
}

func TestSyncer_Run_emptyBeneficiariesDeletesExisting(t *testing.T) {
	t.Parallel()

	// non-nil empty slice = source explicitly returned zero beneficiaries
	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Prog A", Status: "published", OwnerLFUsername: "alice", Beneficiaries: []models.MentorshipBeneficiary{}},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upsertedPrograms) != 1 {
		t.Fatalf("expected 1 upserted program, got %d", len(repo.upsertedPrograms))
	}
	bens := repo.upsertedPrograms[0].Beneficiaries
	if bens == nil {
		t.Fatal("Beneficiaries should be non-nil empty slice to signal delete-all")
	}
	if len(bens) != 0 {
		t.Errorf("expected empty beneficiaries slice, got %+v", bens)
	}
}

func TestSyncer_Run_emptySourceReturnsZeroResult(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{programs: nil}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	result, err := s.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.total != 0 || result.upserted != 0 || result.errors != 0 {
		t.Errorf("expected all-zero result, got %+v", result)
	}
}

func TestSyncer_Run_propagatesSourceError(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{err: errors.New("snowflake unavailable")}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	_, err := s.Run(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSyncer_Run_countsUpsertErrorsWithoutHalting(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Good", OwnerLFUsername: "alice"},
			{JobspringProjectID: "js-2", Name: "Bad", OwnerLFUsername: "bob"},
		},
	}
	repo := &mockMentorshipRepo{}

	failRepo := &failOnSecondRepo{base: repo}

	s := newSyncer(failRepo, src, discardLogger())
	result, err := s.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if result.upserted != 1 {
		t.Errorf("upserted: got %d, want 1", result.upserted)
	}
	if result.errors != 1 {
		t.Errorf("errors: got %d, want 1", result.errors)
	}
}

// failOnSecondRepo fails on the second UpsertProgram call.
type failOnSecondRepo struct {
	base  *mockMentorshipRepo
	calls int
}

func (r *failOnSecondRepo) UpsertProgram(ctx context.Context, p models.MentorshipProgram) (string, error) {
	r.calls++
	if r.calls == 2 {
		return "", errors.New("db error")
	}
	return r.base.UpsertProgram(ctx, p)
}

func (r *failOnSecondRepo) ListJobspringIDs(ctx context.Context) ([]string, error) {
	return r.base.ListJobspringIDs(ctx)
}

func TestSyncer_Run_skillsUpsertedForInitiative(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{
				JobspringProjectID: "js-1",
				Name:               "Prog A",
				Status:             "published",
				OwnerLFUsername:    "alice",
				Skills:             []string{"Golang", "Kubernetes"},
			},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upsertedPrograms) != 1 {
		t.Fatalf("expected 1 upserted program, got %d", len(repo.upsertedPrograms))
	}
	skills := repo.upsertedPrograms[0].Skills
	if len(skills) != 2 || skills[0] != "Golang" || skills[1] != "Kubernetes" {
		t.Errorf("skills: got %v", skills)
	}
}

func TestSyncer_Run_nilSkillsSkipsUpsert(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Prog A", Status: "published", OwnerLFUsername: "alice", Skills: nil},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upsertedPrograms) != 1 {
		t.Fatalf("expected 1 upserted program, got %d", len(repo.upsertedPrograms))
	}
	if repo.upsertedPrograms[0].Skills != nil {
		t.Errorf("Skills should be nil when not provided, got %v", repo.upsertedPrograms[0].Skills)
	}
}

func TestSyncer_Run_mentorsUpsertedForInitiative(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{
				JobspringProjectID: "js-1",
				Name:               "Prog A",
				Status:             "published",
				OwnerLFUsername:    "alice",
				Mentors: []models.MentorshipMentor{
					{Name: "Jane Smith", Email: "jane@example.com", AvatarURL: "https://example.com/jane.png"},
				},
			},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upsertedPrograms) != 1 {
		t.Fatalf("expected 1 upserted program, got %d", len(repo.upsertedPrograms))
	}
	mentors := repo.upsertedPrograms[0].Mentors
	if len(mentors) != 1 || mentors[0].Email != "jane@example.com" {
		t.Errorf("mentors: got %+v", mentors)
	}
}

func TestSyncer_Run_nilMentorsSkipsUpsert(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Prog A", Status: "published", OwnerLFUsername: "alice", Mentors: nil},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upsertedPrograms) != 1 {
		t.Fatalf("expected 1 upserted program, got %d", len(repo.upsertedPrograms))
	}
	if repo.upsertedPrograms[0].Mentors != nil {
		t.Errorf("Mentors should be nil when not provided, got %v", repo.upsertedPrograms[0].Mentors)
	}
}

func TestSyncer_Run_skipsAndCountsErrorForEmptyOwnerLFUsername(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "No Owner", Status: "published", OwnerLFUsername: ""},
			{JobspringProjectID: "js-2", Name: "Has Owner", Status: "published", OwnerLFUsername: "alice"},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	result, err := s.Run(context.Background())

	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if result.upserted != 1 {
		t.Errorf("upserted: got %d, want 1", result.upserted)
	}
	if result.errors != 1 {
		t.Errorf("errors: got %d, want 1 (missing OwnerLFUsername counts as error)", result.errors)
	}
	if len(repo.upsertedPrograms) != 1 || repo.upsertedPrograms[0].JobspringProjectID != "js-2" {
		t.Errorf("only js-2 should be upserted, got %+v", repo.upsertedPrograms)
	}
}
