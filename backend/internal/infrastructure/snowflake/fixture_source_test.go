// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package snowflake_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/snowflake"
)

func TestFixtureSource_FetchPrograms_readsAllFields(t *testing.T) {
	t.Parallel()

	fixture := `[
		{
			"jobspring_project_id": "fe38c553-a066-44b0-8192-f5a5bee5074b",
			"name": "Linux Kernel Mentorship",
			"status": "Published",
			"description": "Learn kernel development",
			"slug": "linux-kernel-mentorship",
			"industry": "Open Source, Systems",
			"owner_lf_username": "cncf-admin",
			"skills": ["C", "Linux"],
			"mentors": [
				{"name": "Jane Smith", "email": "jane@example.com", "avatar_url": "https://example.com/jane.png"}
			],
			"beneficiaries": [
				{"name": "Alice Mentee", "email": "alice@example.com"}
			]
		},
		{
			"jobspring_project_id": "60410ceb-37ca-4ecf-9233-f907d1adf439",
			"name": "COBOL Programming Course",
			"status": "Published",
			"owner_lf_username": "omp-admin",
			"beneficiaries": []
		},
		{
			"jobspring_project_id": "92df3acf-9c1e-4a27-b9a4-cb5ed4293435",
			"name": "Hidden Program",
			"status": "Hidden",
			"owner_lf_username": "omp-admin",
			"beneficiaries": []
		}
	]`

	path := filepath.Join(t.TempDir(), "programs.json")
	if err := os.WriteFile(path, []byte(fixture), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	src := snowflake.NewFixtureSource(path)
	programs, err := src.FetchPrograms(context.Background())
	if err != nil {
		t.Fatalf("FetchPrograms: %v", err)
	}

	if len(programs) != 3 {
		t.Fatalf("got %d programs, want 3", len(programs))
	}

	p := programs[0]
	if p.JobspringProjectID != "fe38c553-a066-44b0-8192-f5a5bee5074b" {
		t.Errorf("JobspringProjectID: got %q", p.JobspringProjectID)
	}
	if p.Name != "Linux Kernel Mentorship" {
		t.Errorf("Name: got %q, want Linux Kernel Mentorship", p.Name)
	}
	if p.Status != "Published" {
		t.Errorf("Status: got %q, want Published", p.Status)
	}
	if p.Description != "Learn kernel development" {
		t.Errorf("Description: got %q, want Learn kernel development", p.Description)
	}
	if p.Slug != "linux-kernel-mentorship" {
		t.Errorf("Slug: got %q, want linux-kernel-mentorship", p.Slug)
	}
	if p.Industry != "Open Source, Systems" {
		t.Errorf("Industry: got %q, want Open Source, Systems", p.Industry)
	}
	if p.OwnerLFUsername != "cncf-admin" {
		t.Errorf("OwnerLFUsername: got %q, want cncf-admin", p.OwnerLFUsername)
	}

	// Skills.
	if len(p.Skills) != 2 || p.Skills[0] != "C" || p.Skills[1] != "Linux" {
		t.Errorf("Skills: got %v", p.Skills)
	}

	// Mentors.
	if len(p.Mentors) != 1 || p.Mentors[0].Email != "jane@example.com" {
		t.Errorf("Mentors: got %+v", p.Mentors)
	}
	if p.Mentors[0].Name != "Jane Smith" {
		t.Errorf("Mentors[0].Name: got %q, want Jane Smith", p.Mentors[0].Name)
	}
	if p.Mentors[0].AvatarURL != "https://example.com/jane.png" {
		t.Errorf("Mentors[0].AvatarURL: got %q", p.Mentors[0].AvatarURL)
	}

	// Beneficiaries.
	if len(p.Beneficiaries) != 1 {
		t.Fatalf("Beneficiaries: got %d, want 1", len(p.Beneficiaries))
	}
	if p.Beneficiaries[0].Name != "Alice Mentee" {
		t.Errorf("Beneficiaries[0].Name: got %q, want Alice Mentee", p.Beneficiaries[0].Name)
	}
	if p.Beneficiaries[0].Email != "alice@example.com" {
		t.Errorf("Beneficiaries[0].Email: got %q, want alice@example.com", p.Beneficiaries[0].Email)
	}
}

func TestFixtureSource_FetchPrograms_absentFieldsProduceNilSlices(t *testing.T) {
	t.Parallel()

	// When "beneficiaries", "skills", "mentors" keys are absent, all must be
	// nil so the syncer skips their respective upserts.
	fixture := `[{"jobspring_project_id": "abc", "name": "Test", "status": "Published"}]`

	path := filepath.Join(t.TempDir(), "programs.json")
	if err := os.WriteFile(path, []byte(fixture), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	src := snowflake.NewFixtureSource(path)
	programs, err := src.FetchPrograms(context.Background())
	if err != nil {
		t.Fatalf("FetchPrograms: %v", err)
	}
	p := programs[0]
	if p.Beneficiaries != nil {
		t.Errorf("expected nil Beneficiaries when field absent, got %v", p.Beneficiaries)
	}
	if p.Skills != nil {
		t.Errorf("expected nil Skills when field absent, got %v", p.Skills)
	}
	if p.Mentors != nil {
		t.Errorf("expected nil Mentors when field absent, got %v", p.Mentors)
	}
}

func TestFixtureSource_FetchPrograms_missingFileReturnsError(t *testing.T) {
	t.Parallel()

	src := snowflake.NewFixtureSource("/nonexistent/programs.json")
	_, err := src.FetchPrograms(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
