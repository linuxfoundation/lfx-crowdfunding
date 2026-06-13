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
	upsertedPrograms      []models.MentorshipProgram
	upsertedBeneficiaries map[string][]models.MentorshipBeneficiary
	programErr            error
	beneficiaryErr        error
	jobspringIDs          []string
	listErr               error
}

func (m *mockMentorshipRepo) UpsertProgram(_ context.Context, p models.MentorshipProgram) (string, error) {
	if m.programErr != nil {
		return "", m.programErr
	}
	m.upsertedPrograms = append(m.upsertedPrograms, p)
	return "initiative-uuid-" + p.JobspringProjectID, nil
}

func (m *mockMentorshipRepo) UpsertBeneficiaries(_ context.Context, initiativeID string, beneficiaries []models.MentorshipBeneficiary) error {
	if m.beneficiaryErr != nil {
		return m.beneficiaryErr
	}
	if m.upsertedBeneficiaries == nil {
		m.upsertedBeneficiaries = make(map[string][]models.MentorshipBeneficiary)
	}
	m.upsertedBeneficiaries[initiativeID] = beneficiaries
	return nil
}

func (m *mockMentorshipRepo) ListJobspringIDs(_ context.Context) ([]string, error) {
	return m.jobspringIDs, m.listErr
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestSyncer_Run_upsertsAllPrograms(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Prog A", Status: "published", Beneficiaries: []models.MentorshipBeneficiary{{Name: "Alice", Email: "a@x.com"}}},
			{JobspringProjectID: "js-2", Name: "Prog B", Status: "pending", Beneficiaries: nil},
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
				{JobspringProjectID: "js-1", Name: "Test", Status: tc.input},
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

	initiativeID := "initiative-uuid-js-1"
	bens, ok := repo.upsertedBeneficiaries[initiativeID]
	if !ok {
		t.Fatalf("no beneficiaries upserted for initiative %q", initiativeID)
	}
	if len(bens) != 1 || bens[0].Email != "alice@x.com" {
		t.Errorf("beneficiaries: got %+v", bens)
	}
}

func TestSyncer_Run_nilBeneficiariesSkipsUpsert(t *testing.T) {
	t.Parallel()

	// nil Beneficiaries = source did not provide beneficiary data (e.g. Snowflake client)
	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Prog A", Status: "published", Beneficiaries: nil},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, called := repo.upsertedBeneficiaries["initiative-uuid-js-1"]; called {
		t.Error("UpsertBeneficiaries should not be called when Beneficiaries is nil")
	}
}

func TestSyncer_Run_emptyBeneficiariesDeletesExisting(t *testing.T) {
	t.Parallel()

	// non-nil empty slice = source explicitly returned zero beneficiaries
	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Prog A", Status: "published", Beneficiaries: []models.MentorshipBeneficiary{}},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bens, called := repo.upsertedBeneficiaries["initiative-uuid-js-1"]
	if !called {
		t.Fatal("UpsertBeneficiaries should be called when Beneficiaries is non-nil empty")
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
			{JobspringProjectID: "js-1", Name: "Good"},
			{JobspringProjectID: "js-2", Name: "Bad"},
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

func (r *failOnSecondRepo) UpsertBeneficiaries(ctx context.Context, id string, b []models.MentorshipBeneficiary) error {
	return r.base.UpsertBeneficiaries(ctx, id, b)
}

func (r *failOnSecondRepo) ListJobspringIDs(ctx context.Context) ([]string, error) {
	return r.base.ListJobspringIDs(ctx)
}
