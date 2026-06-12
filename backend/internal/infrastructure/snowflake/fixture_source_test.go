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
			"jobspring_project_id": "proj-1",
			"name": "Linux Kernel",
			"status": "published",
			"mentee_goal_cents": 500000,
			"beneficiaries": [
				{"name": "Alice", "email": "alice@example.com"}
			]
		},
		{
			"jobspring_project_id": "proj-2",
			"name": "Pending Program",
			"status": "pending",
			"mentee_goal_cents": 0,
			"beneficiaries": []
		},
		{
			"jobspring_project_id": "proj-3",
			"name": "Hidden Program",
			"status": "hide",
			"mentee_goal_cents": 100000,
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
	if p.JobspringProjectID != "proj-1" {
		t.Errorf("JobspringProjectID: got %q, want proj-1", p.JobspringProjectID)
	}
	if p.Name != "Linux Kernel" {
		t.Errorf("Name: got %q, want Linux Kernel", p.Name)
	}
	if p.Status != "published" {
		t.Errorf("Status: got %q, want published", p.Status)
	}
	if p.MenteeGoalCents != 500000 {
		t.Errorf("MenteeGoalCents: got %d, want 500000", p.MenteeGoalCents)
	}
	if len(p.Beneficiaries) != 1 {
		t.Fatalf("Beneficiaries: got %d, want 1", len(p.Beneficiaries))
	}
	if p.Beneficiaries[0].Name != "Alice" {
		t.Errorf("Beneficiaries[0].Name: got %q, want Alice", p.Beneficiaries[0].Name)
	}
	if p.Beneficiaries[0].Email != "alice@example.com" {
		t.Errorf("Beneficiaries[0].Email: got %q, want alice@example.com", p.Beneficiaries[0].Email)
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
