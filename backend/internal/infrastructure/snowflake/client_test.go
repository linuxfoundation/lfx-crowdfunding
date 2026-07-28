// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package snowflake_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/snowflake"
)

// mockSnowflakeDriver records the last query executed against it.
type mockSnowflakeDriver struct {
	lastQuery string
	rows      [][]driver.Value
}

func (d *mockSnowflakeDriver) Open(_ string) (driver.Conn, error) {
	return &mockConn{driver: d}, nil
}

type mockConn struct{ driver *mockSnowflakeDriver }

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	c.driver.lastQuery = query
	return &mockStmt{driver: c.driver}, nil
}
func (c *mockConn) Close() error              { return nil }
func (c *mockConn) Begin() (driver.Tx, error) { return nil, nil }

type mockStmt struct{ driver *mockSnowflakeDriver }

func (s *mockStmt) Close() error                                 { return nil }
func (s *mockStmt) NumInput() int                                { return 0 }
func (s *mockStmt) Exec(_ []driver.Value) (driver.Result, error) { return nil, nil }
func (s *mockStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &mockRows{rows: s.driver.rows, pos: 0}, nil
}

type mockRows struct {
	rows [][]driver.Value
	pos  int
}

func (r *mockRows) Columns() []string {
	return []string{
		"PROGRAM_ID",
		"PROGRAM_NAME",
		"PROGRAM_STATUS",
		"PROGRAM_DESCRIPTION",
		"program_slug",
		"OWNER_LF_USERNAME",
		"OWNER_EMAIL",
		"OWNER_FIRST_NAME",
		"OWNER_LAST_NAME",
		"OWNER_AVATAR_URL",
		"PROGRAM_TECHNOLOGY",
		"SELECTED_MENTEES",
		"mentors",
		"program_skills",
	}
}
func (r *mockRows) Close() error { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.pos])
	r.pos++
	return nil
}

func TestClient_FetchPrograms_queriesExpectedSQL(t *testing.T) {
	t.Parallel()

	menteesJSON := `[{"first_name":"Alice","last_name":"Mentee","email":"alice@example.com","avatar_url":""}]`
	mentorsJSON := `[{"first_name":"Jane","last_name":"Smith","email":"jane@example.com","avatar_url":"https://example.com/jane.png"}]`
	skillsJSON := `["Golang","Kubernetes"]`

	mock := &mockSnowflakeDriver{
		rows: [][]driver.Value{
			{
				"fe38c553-a066-44b0-8192-f5a5bee5074b",
				"Linux Kernel Mentorship",
				"Published",
				"A great program",
				"linux-kernel-mentorship",
				"cncf-admin",
				"cncf-admin@example.com",
				"Admin",
				"CNCF",
				"https://example.com/cncf-admin.png",
				"Cloud Native",
				menteesJSON,
				mentorsJSON,
				skillsJSON,
			},
		},
	}
	driverName := "snowflake_mock_" + t.Name()
	sql.Register(driverName, mock)

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	client := snowflake.NewClientFromDB(db)
	programs, err := client.FetchPrograms(context.Background())
	if err != nil {
		t.Fatalf("FetchPrograms: %v", err)
	}

	// Assert SQL contains the expected table.
	wantTable := "ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS"
	if mock.lastQuery == "" {
		t.Fatal("no query was executed")
	}
	if !containsSubstr(mock.lastQuery, wantTable) {
		t.Errorf("query does not reference %q\ngot: %s", wantTable, mock.lastQuery)
	}

	// Assert field mapping.
	if len(programs) != 1 {
		t.Fatalf("got %d programs, want 1", len(programs))
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
	if p.OwnerLFUsername != "cncf-admin" {
		t.Errorf("OwnerLFUsername: got %q, want cncf-admin", p.OwnerLFUsername)
	}
	if p.OwnerEmail != "cncf-admin@example.com" {
		t.Errorf("OwnerEmail: got %q, want cncf-admin@example.com", p.OwnerEmail)
	}
	if p.OwnerFirstName != "Admin" {
		t.Errorf("OwnerFirstName: got %q, want Admin", p.OwnerFirstName)
	}
	if p.OwnerLastName != "CNCF" {
		t.Errorf("OwnerLastName: got %q, want CNCF", p.OwnerLastName)
	}
	if p.OwnerAvatarURL != "https://example.com/cncf-admin.png" {
		t.Errorf("OwnerAvatarURL: got %q, want https://example.com/cncf-admin.png", p.OwnerAvatarURL)
	}
	if p.Description != "A great program" {
		t.Errorf("Description: got %q, want A great program", p.Description)
	}
	if p.Slug != "linux-kernel-mentorship" {
		t.Errorf("Slug: got %q, want linux-kernel-mentorship", p.Slug)
	}
	if p.Industry != "Cloud Native" {
		t.Errorf("Industry: got %q, want Cloud Native", p.Industry)
	}

	// Beneficiaries (SELECTED_MENTEES).
	if len(p.Beneficiaries) != 1 || p.Beneficiaries[0].Email != "alice@example.com" {
		t.Errorf("Beneficiaries: got %+v", p.Beneficiaries)
	}
	if p.Beneficiaries[0].Name != "Alice Mentee" {
		t.Errorf("Beneficiaries[0].Name: got %q, want Alice Mentee", p.Beneficiaries[0].Name)
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

	// Skills.
	if len(p.Skills) != 2 || p.Skills[0] != "Golang" || p.Skills[1] != "Kubernetes" {
		t.Errorf("Skills: got %v", p.Skills)
	}
}

func containsSubstr(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
