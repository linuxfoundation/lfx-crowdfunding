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
			"owner_lf_username": "cncf-admin",
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
		t.Errorf("JobspringProjectID: got %q, want fe38c553-a066-44b0-8192-f5a5bee5074b", p.JobspringProjectID)
	}
	if p.Name != "Linux Kernel Mentorship" {
		t.Errorf("Name: got %q, want Linux Kernel Mentorship", p.Name)
	}
	if p.Status != "Published" {
		t.Errorf("Status: got %q, want Published", p.Status)
	}
	if len(p.Beneficiaries) != 1 {
		t.Fatalf("Beneficiaries: got %d, want 1", len(p.Beneficiaries))
	}
	if p.Beneficiaries[0].Name != "Alice Mentee" {
		t.Errorf("Beneficiaries[0].Name: got %q, want Alice Mentee", p.Beneficiaries[0].Name)
	}
	if p.Beneficiaries[0].Email != "alice@example.com" {
		t.Errorf("Beneficiaries[0].Email: got %q, want alice@example.com", p.Beneficiaries[0].Email)
	}
	if p.OwnerLFUsername != "cncf-admin" {
		t.Errorf("OwnerLFUsername: got %q, want cncf-admin", p.OwnerLFUsername)
	}
}

func TestFixtureSource_FetchPrograms_absentFieldProducesNilBeneficiaries(t *testing.T) {
	t.Parallel()

	// When "beneficiaries" key is absent from JSON, Beneficiaries must be nil
	// (not a non-nil empty slice) so the syncer skips UpsertBeneficiaries.
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
	if programs[0].Beneficiaries != nil {
		t.Errorf("expected nil Beneficiaries when field absent, got %v", programs[0].Beneficiaries)
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
