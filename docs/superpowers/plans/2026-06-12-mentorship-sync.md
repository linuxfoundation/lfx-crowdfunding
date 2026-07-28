<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Mentorship Sync CronJob Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the `mentorship-sync` K8s CronJob that pulls Mentorship program data from Snowflake (or a JSON fixture in DEV) and upserts `initiative_type = mentorship` rows into CF Postgres.

**Architecture:** A `mentorshipSource` interface is defined in `cmd/mentorship-sync/syncer.go` and satisfied by two implementations in `internal/infrastructure/snowflake/`: `Client` (real Snowflake, used in staging/prod) and `FixtureSource` (JSON file, used in DEV and local dev). `cmd/mentorship-sync/main.go` wires the correct source based on `MENTORSHIP_SYNC_FIXTURE_FILE`. The sync algorithm in `syncer.go` runs identically regardless of source. A `MentorshipRepository` in `internal/infrastructure/db/` owns all DB writes. This mirrors the `ledger-stats-sync` pattern exactly.

**Tech Stack:** Go 1.25, `pgx/v5` (DB), `gosnowflake` driver (Snowflake), `encoding/json` (fixture), `log/slog` (structured logging), `go.opentelemetry.io/otel` (tracing on Snowflake client).

---

## File Map

| File | Action | Purpose |
|---|---|---|
| `internal/domain/models/mentorship_sync.go` | **Create** | `MentorshipProgram`, `MentorshipBeneficiary` domain types |
| `internal/domain/repository.go` | **Modify** | Add `MentorshipRepository` interface |
| `internal/infrastructure/snowflake/client.go` | **Create** | Real Snowflake client; `FetchPrograms` runs the SQL query |
| `internal/infrastructure/snowflake/client_test.go` | **Create** | Unit test: asserts SQL query string + struct field mapping via mock driver |
| `internal/infrastructure/snowflake/fixture_source.go` | **Create** | `FixtureSource` reads `programs.json` and returns `[]MentorshipProgram` |
| `internal/infrastructure/snowflake/fixture_source_test.go` | **Create** | Unit test: reads a small embedded fixture and asserts all fields |
| `internal/infrastructure/db/mentorship_repository.go` | **Create** | `MentorshipRepository` impl: `UpsertProgram`, `ListJobspringIDs`, `UpsertBeneficiaries` |
| `internal/infrastructure/db/mentorship_repository_test.go` | **Create** | Unit tests for upsert logic with mock pgx rows |
| `cmd/mentorship-sync/main.go` | **Create** | Entry point; wires source from env var, runs syncer, exits |
| `cmd/mentorship-sync/syncer.go` | **Create** | `mentorshipSource` interface + `Syncer.Run` algorithm |
| `cmd/mentorship-sync/syncer_test.go` | **Create** | Table-driven tests for syncer algorithm using mock source + mock repo |
| `cmd/mentorship-sync/testdata/programs.json` | **Create** | Fixture: 3 records (active+beneficiaries, pending, hidden) |
| `Dockerfile.mentorship-sync` | **Create** | Multi-stage build producing minimal container image |
| `lfx-v2-argocd` `values/dev/lfx-crowdfunding-mentorship-sync.yaml` | **Create** | DEV: `MENTORSHIP_SYNC_FIXTURE_FILE` set, no Snowflake creds |
| `lfx-v2-argocd` `values/staging/lfx-crowdfunding-mentorship-sync.yaml` | **Create** | Staging: Snowflake creds from K8s Secret |
| `lfx-v2-argocd` `values/prod/lfx-crowdfunding-mentorship-sync.yaml` | **Create** | Prod: same as staging |
| `backend/docs/rewrite/10-mentorship-sync-dev-testing.md` | **Update** | Rewrite rationale + add ANALYTICS_DEV follow-on section + client validation section |

---

## Task 1: Domain Models

**Files:**
- Create: `backend/internal/domain/models/mentorship_sync.go`
- Modify: `backend/internal/domain/repository.go`

- [ ] **Step 1: Create domain model file**

```go
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
```

- [ ] **Step 2: Add `MentorshipRepository` to repository.go**

Open `backend/internal/domain/repository.go` and add at the end:

```go
// MentorshipRepository defines persistence operations used by mentorship-sync.
// All methods are scoped to the batch upsert pattern of that CronJob.
type MentorshipRepository interface {
	// UpsertProgram creates or updates the initiative row for a mentorship program.
	// The upsert key is jobspring_project_id. Returns the initiative UUID.
	UpsertProgram(ctx context.Context, p models.MentorshipProgram) (string, error)

	// UpsertBeneficiaries replaces the beneficiary list for the given initiative.
	// All existing rows for initiativeID are deleted then re-inserted.
	UpsertBeneficiaries(ctx context.Context, initiativeID string, beneficiaries []models.MentorshipBeneficiary) error

	// ListJobspringIDs returns the jobspring_project_id values for all existing
	// mentorship initiatives. Used to detect programs that have been removed from
	// Snowflake (not currently acted on, but useful for future reconciliation).
	ListJobspringIDs(ctx context.Context) ([]string, error)
}
```

- [ ] **Step 3: Run tests to confirm nothing broken**

```bash
cd backend && make test
```

Expected: all existing tests pass.

- [ ] **Step 4: Commit**

```bash
cd backend
git add internal/domain/models/mentorship_sync.go internal/domain/repository.go
git commit --signoff -m "feat(mentorship-sync): add MentorshipProgram domain model and repository interface"
```

---

## Task 2: Fixture Source

**Files:**
- Create: `backend/internal/infrastructure/snowflake/fixture_source.go`
- Create: `backend/internal/infrastructure/snowflake/fixture_source_test.go`

- [ ] **Step 1: Write the failing test**

```go
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

	// Write a minimal fixture file to a temp dir.
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
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd backend && go test ./internal/infrastructure/snowflake/... -run TestFixtureSource -v
```

Expected: `FAIL` — package does not exist yet.

- [ ] **Step 3: Implement FixtureSource**

Create `backend/internal/infrastructure/snowflake/fixture_source.go`:

```go
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package snowflake

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// fixtureProgram mirrors MentorshipProgram with JSON tags for fixture file decoding.
type fixtureProgram struct {
	JobspringProjectID string               `json:"jobspring_project_id"`
	Name               string               `json:"name"`
	Status             string               `json:"status"`
	MenteeGoalCents    int64                `json:"mentee_goal_cents"`
	Beneficiaries      []fixtureBeneficiary `json:"beneficiaries"`
}

type fixtureBeneficiary struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// FixtureSource implements mentorshipSource by reading a JSON file from disk.
// Used in DEV and local development — requires no Snowflake credentials.
type FixtureSource struct {
	path string
}

// NewFixtureSource returns a FixtureSource that reads programs from the given path.
func NewFixtureSource(path string) *FixtureSource {
	return &FixtureSource{path: path}
}

// FetchPrograms reads the fixture file and returns its contents as domain models.
func (f *FixtureSource) FetchPrograms(_ context.Context) ([]models.MentorshipProgram, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return nil, fmt.Errorf("read fixture file %q: %w", f.path, err)
	}

	var raw []fixtureProgram
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse fixture file %q: %w", f.path, err)
	}

	programs := make([]models.MentorshipProgram, len(raw))
	for i, r := range raw {
		beneficiaries := make([]models.MentorshipBeneficiary, len(r.Beneficiaries))
		for j, b := range r.Beneficiaries {
			beneficiaries[j] = models.MentorshipBeneficiary{Name: b.Name, Email: b.Email}
		}
		programs[i] = models.MentorshipProgram{
			JobspringProjectID: r.JobspringProjectID,
			Name:               r.Name,
			Status:             r.Status,
			MenteeGoalCents:    r.MenteeGoalCents,
			Beneficiaries:      beneficiaries,
		}
	}
	return programs, nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd backend && go test ./internal/infrastructure/snowflake/... -run TestFixtureSource -v
```

Expected: `PASS` for both tests.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/infrastructure/snowflake/fixture_source.go internal/infrastructure/snowflake/fixture_source_test.go
git commit --signoff -m "feat(mentorship-sync): add FixtureSource for DEV environment testing"
```

---

## Task 3: Snowflake Client

**Files:**
- Create: `backend/internal/infrastructure/snowflake/client.go`
- Create: `backend/internal/infrastructure/snowflake/client_test.go`

This task adds the real Snowflake driver dependency and implements the `Client` that runs the production SQL query.

- [ ] **Step 1: Add gosnowflake dependency**

```bash
cd backend && go get github.com/snowflakedb/gosnowflake@latest
```

Confirm it appears in `go.mod` and `go.sum`.

- [ ] **Step 2: Write the failing test**

```go
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
func (c *mockConn) Close() error  { return nil }
func (c *mockConn) Begin() (driver.Tx, error) { return nil, nil }

type mockStmt struct{ driver *mockSnowflakeDriver }

func (s *mockStmt) Close() error               { return nil }
func (s *mockStmt) NumInput() int               { return 0 }
func (s *mockStmt) Exec(_ []driver.Value) (driver.Result, error) { return nil, nil }
func (s *mockStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return &mockRows{rows: s.driver.rows, pos: 0}, nil
}

type mockRows struct {
	rows [][]driver.Value
	pos  int
}

func (r *mockRows) Columns() []string {
	return []string{"jobspring_project_id", "name", "status", "mentee_goal_cents"}
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

	mock := &mockSnowflakeDriver{
		rows: [][]driver.Value{
			{"proj-1", "Linux Kernel", "published", int64(500000)},
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
	if !contains(mock.lastQuery, wantTable) {
		t.Errorf("query does not reference %q\ngot: %s", wantTable, mock.lastQuery)
	}

	// Assert field mapping: one row returned.
	if len(programs) != 1 {
		t.Fatalf("got %d programs, want 1", len(programs))
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
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
```

- [ ] **Step 3: Run test to confirm it fails**

```bash
cd backend && go test ./internal/infrastructure/snowflake/... -run TestClient -v
```

Expected: `FAIL` — `snowflake.NewClientFromDB` not defined yet.

- [ ] **Step 4: Implement Snowflake Client**

Create `backend/internal/infrastructure/snowflake/client.go`:

```go
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package snowflake

import (
	"context"
	"database/sql"
	"fmt"

	gosnowflake "github.com/snowflakedb/gosnowflake"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

const fetchProgramsQuery = `
SELECT
	p.jobspring_project_id,
	p.name,
	p.status,
	COALESCE(p.mentee_goal_cents, 0) AS mentee_goal_cents
FROM ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS p
WHERE p.jobspring_project_id IS NOT NULL
`

// ClientConfig holds credentials for connecting to Snowflake via key-pair auth.
type ClientConfig struct {
	Account    string
	User       string
	Warehouse  string
	Database   string
	Role       string
	PrivateKey string // PEM-encoded PKCS8 private key
}

// Client queries Snowflake for Mentorship program data.
type Client struct {
	db *sql.DB
}

// NewClient opens a Snowflake connection using key-pair authentication.
// The caller must call Close() when done.
func NewClient(cfg ClientConfig) (*Client, error) {
	dsn, err := gosnowflake.DSN(&gosnowflake.Config{
		Account:       cfg.Account,
		User:          cfg.User,
		Warehouse:     cfg.Warehouse,
		Database:      cfg.Database,
		Role:          cfg.Role,
		Authenticator: gosnowflake.AuthTypeJwt,
		PrivateKey:    cfg.PrivateKey,
	})
	if err != nil {
		return nil, fmt.Errorf("build snowflake DSN: %w", err)
	}

	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		return nil, fmt.Errorf("open snowflake connection: %w", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	return &Client{db: db}, nil
}

// NewClientFromDB constructs a Client from an existing *sql.DB.
// Used in tests to inject a mock driver.
func NewClientFromDB(db *sql.DB) *Client {
	return &Client{db: db}
}

// Close releases the underlying database connection pool.
func (c *Client) Close() error {
	return c.db.Close()
}

// FetchPrograms runs the Snowflake query and returns all Mentorship programs.
// Beneficiaries are not included — they are fetched in a separate query or
// embedded in the gold model; adjust the query when the schema is confirmed.
func (c *Client) FetchPrograms(ctx context.Context) ([]models.MentorshipProgram, error) {
	rows, err := c.db.QueryContext(ctx, fetchProgramsQuery)
	if err != nil {
		return nil, fmt.Errorf("query mentorship programs: %w", err)
	}
	defer rows.Close()

	var programs []models.MentorshipProgram
	for rows.Next() {
		var p models.MentorshipProgram
		if err := rows.Scan(
			&p.JobspringProjectID,
			&p.Name,
			&p.Status,
			&p.MenteeGoalCents,
		); err != nil {
			return nil, fmt.Errorf("scan mentorship program row: %w", err)
		}
		programs = append(programs, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mentorship program rows: %w", err)
	}
	return programs, nil
}
```

- [ ] **Step 5: Run tests to confirm they pass**

```bash
cd backend && go test ./internal/infrastructure/snowflake/... -v
```

Expected: all 3 tests pass (`TestFixtureSource_*` x2, `TestClient_FetchPrograms_queriesExpectedSQL`).

- [ ] **Step 6: Commit**

```bash
cd backend
git add internal/infrastructure/snowflake/client.go internal/infrastructure/snowflake/client_test.go go.mod go.sum
git commit --signoff -m "feat(mentorship-sync): add Snowflake client with mock-driver unit test"
```

---

## Task 4: Mentorship Repository

**Files:**
- Create: `backend/internal/infrastructure/db/mentorship_repository.go`
- Create: `backend/internal/infrastructure/db/mentorship_repository_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db_test

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/db"
)

// These tests use a stub pool — they verify the repository compiles and
// exposes the correct interface. Integration tests against a real DB are
// outside scope for the CronJob unit test suite.

func TestMentorshipRepository_implementsInterface(t *testing.T) {
	// Compile-time check: MentorshipRepository implements domain.MentorshipRepository.
	// If the interface changes and the implementation doesn't follow, this test
	// file will fail to compile.
	_ = func() {
		var _ interface {
			UpsertProgram(context.Context, models.MentorshipProgram) (string, error)
			UpsertBeneficiaries(context.Context, string, []models.MentorshipBeneficiary) error
			ListJobspringIDs(context.Context) ([]string, error)
		} = (*db.MentorshipRepositoryImpl)(nil)
	}
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd backend && go test ./internal/infrastructure/db/... -run TestMentorshipRepository -v
```

Expected: `FAIL` — `db.MentorshipRepositoryImpl` not defined.

- [ ] **Step 3: Implement the repository**

Create `backend/internal/infrastructure/db/mentorship_repository.go`:

```go
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// MentorshipRepositoryImpl implements domain.MentorshipRepository against CF Postgres.
type MentorshipRepositoryImpl struct {
	pool *pgxpool.Pool
}

// NewMentorshipRepository returns a MentorshipRepositoryImpl backed by pool.
func NewMentorshipRepository(pool *pgxpool.Pool) *MentorshipRepositoryImpl {
	return &MentorshipRepositoryImpl{pool: pool}
}

// UpsertProgram inserts or updates the mentorship initiative row identified by
// jobspring_project_id. Returns the initiative UUID.
func (r *MentorshipRepositoryImpl) UpsertProgram(ctx context.Context, p models.MentorshipProgram) (string, error) {
	// Normalise Mentorship status: 'hide' → 'hidden'
	status := p.Status
	if status == "hide" {
		status = "hidden"
	}

	const q = `
INSERT INTO initiatives (
	id,
	initiative_type,
	jobspring_project_id,
	name,
	status,
	created_on,
	updated_on
) VALUES (
	gen_random_uuid(),
	'mentorship',
	$1,
	$2,
	$3,
	NOW(),
	NOW()
)
ON CONFLICT (jobspring_project_id) DO UPDATE SET
	name       = EXCLUDED.name,
	status     = EXCLUDED.status,
	updated_on = NOW()
RETURNING id
`
	var id string
	err := r.pool.QueryRow(ctx, q, p.JobspringProjectID, p.Name, status).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert mentorship program %q: %w", p.JobspringProjectID, err)
	}
	return id, nil
}

// UpsertBeneficiaries replaces all beneficiary rows for the given initiative.
// Runs in a transaction: delete existing → insert new.
func (r *MentorshipRepositoryImpl) UpsertBeneficiaries(ctx context.Context, initiativeID string, beneficiaries []models.MentorshipBeneficiary) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx for beneficiaries %q: %w", initiativeID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`DELETE FROM initiative_beneficiaries WHERE initiative_id = $1`, initiativeID,
	); err != nil {
		return fmt.Errorf("delete beneficiaries for %q: %w", initiativeID, err)
	}

	for _, b := range beneficiaries {
		if _, err := tx.Exec(ctx,
			`INSERT INTO initiative_beneficiaries (id, initiative_id, name, email, created_on, updated_on)
			 VALUES ($1, $2, $3, $4, $5, $5)`,
			uuid.New().String(), initiativeID, b.Name, b.Email, time.Now(),
		); err != nil {
			return fmt.Errorf("insert beneficiary %q for %q: %w", b.Email, initiativeID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit beneficiaries tx for %q: %w", initiativeID, err)
	}
	return nil
}

// ListJobspringIDs returns the jobspring_project_id for all existing mentorship initiatives.
func (r *MentorshipRepositoryImpl) ListJobspringIDs(ctx context.Context) ([]string, error) {
	const q = `
SELECT jobspring_project_id
FROM initiatives
WHERE initiative_type = 'mentorship'
  AND jobspring_project_id IS NOT NULL
`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list jobspring IDs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan jobspring ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd backend && go test ./internal/infrastructure/db/... -run TestMentorshipRepository -v
```

Expected: `PASS`.

- [ ] **Step 5: Commit**

```bash
cd backend
git add internal/infrastructure/db/mentorship_repository.go internal/infrastructure/db/mentorship_repository_test.go
git commit --signoff -m "feat(mentorship-sync): add MentorshipRepository with upsert and beneficiary replacement"
```

---

## Task 5: Fixture Data File

**Files:**
- Create: `backend/cmd/mentorship-sync/testdata/programs.json`

- [ ] **Step 1: Create fixture data**

```json
[
  {
    "jobspring_project_id": "jobspring-active-001",
    "name": "Linux Kernel Mentorship",
    "status": "published",
    "mentee_goal_cents": 500000,
    "beneficiaries": [
      {"name": "Alice Mentee", "email": "alice@example.com"},
      {"name": "Bob Mentee", "email": "bob@example.com"}
    ]
  },
  {
    "jobspring_project_id": "jobspring-pending-002",
    "name": "Cloud Native Mentorship",
    "status": "pending",
    "mentee_goal_cents": 0,
    "beneficiaries": []
  },
  {
    "jobspring_project_id": "jobspring-hidden-003",
    "name": "Hidden Program",
    "status": "hide",
    "mentee_goal_cents": 100000,
    "beneficiaries": []
  }
]
```

This covers three code paths in `Syncer.Run`: active program with beneficiaries, pending program (no goal), and a hidden program that exercises the `'hide'` → `'hidden'` status normalisation.

- [ ] **Step 2: Verify fixture parses cleanly**

```bash
cd backend && go run -v ./cmd/mentorship-sync/... 2>&1 | head -5
```

(The binary won't exist yet — that's fine. This just confirms the testdata directory is in place for the next task.)

Alternatively, verify with:

```bash
cat backend/cmd/mentorship-sync/testdata/programs.json | python3 -m json.tool > /dev/null && echo "valid JSON"
```

Expected: `valid JSON`.

- [ ] **Step 3: Commit**

```bash
cd backend
git add cmd/mentorship-sync/testdata/programs.json
git commit --signoff -m "feat(mentorship-sync): add fixture data file for DEV environment testing"
```

---

## Task 6: Syncer and CronJob Entry Point

**Files:**
- Create: `backend/cmd/mentorship-sync/syncer.go`
- Create: `backend/cmd/mentorship-sync/syncer_test.go`
- Create: `backend/cmd/mentorship-sync/main.go`

- [ ] **Step 1: Write the failing syncer tests**

```go
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
	upsertedPrograms    []models.MentorshipProgram
	upsertedBeneficiaries map[string][]models.MentorshipBeneficiary
	programErr          error
	beneficiaryErr      error
	jobspringIDs        []string
	listErr             error
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

func TestSyncer_Run_normalisesHideStatus(t *testing.T) {
	t.Parallel()

	src := &mockMentorshipSource{
		programs: []models.MentorshipProgram{
			{JobspringProjectID: "js-1", Name: "Hidden", Status: "hide"},
		},
	}
	repo := &mockMentorshipRepo{}

	s := newSyncer(repo, src, discardLogger())
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.upsertedPrograms) != 1 {
		t.Fatalf("expected 1 upsert, got %d", len(repo.upsertedPrograms))
	}
	if got := repo.upsertedPrograms[0].Status; got != "hidden" {
		t.Errorf("status: got %q, want hidden", got)
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
	callCount := 0
	repo := &mockMentorshipRepo{}
	repo.programErr = nil

	// Override to fail on second call.
	failRepo := &failOnSecondRepo{base: repo}

	s := newSyncer(failRepo, src, discardLogger())
	result, err := s.Run(context.Background())

	_ = callCount
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

// failOnSecondRepo wraps mockMentorshipRepo and fails on the second UpsertProgram call.
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
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd backend && go test ./cmd/mentorship-sync/... -v 2>&1 | head -20
```

Expected: `FAIL` — package does not exist yet.

- [ ] **Step 3: Implement syncer.go**

```go
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// mentorshipSource is the interface Syncer needs from its data source.
// Defined at the point of consumption — both snowflake.Client and
// snowflake.FixtureSource satisfy this interface.
type mentorshipSource interface {
	FetchPrograms(ctx context.Context) ([]models.MentorshipProgram, error)
}

// syncResult carries the per-run counters logged on completion.
type syncResult struct {
	total    int // programs returned by source
	upserted int // programs successfully upserted (program + beneficiaries)
	errors   int // programs that failed to upsert (logged, not fatal)
}

// Syncer orchestrates a single mentorship-sync run.
type Syncer struct {
	repo   domain.MentorshipRepository
	source mentorshipSource
	logger *slog.Logger
}

// newSyncer returns a configured Syncer ready to call Run.
func newSyncer(repo domain.MentorshipRepository, source mentorshipSource, logger *slog.Logger) *Syncer {
	return &Syncer{repo: repo, source: source, logger: logger}
}

// Run executes the full sync algorithm:
//  1. Fetch all programs from source (Snowflake or fixture).
//  2. For each program: normalise status, upsert initiative row, upsert beneficiaries.
//  3. Log per-program errors without halting the run.
//  4. Return per-run counters for summary logging.
func (s *Syncer) Run(ctx context.Context) (syncResult, error) {
	programs, err := s.source.FetchPrograms(ctx)
	if err != nil {
		return syncResult{}, fmt.Errorf("fetch programs: %w", err)
	}

	result := syncResult{total: len(programs)}

	for _, p := range programs {
		// Normalise Mentorship status before upsert.
		if p.Status == "hide" {
			p.Status = "hidden"
		}

		initiativeID, err := s.repo.UpsertProgram(ctx, p)
		if err != nil {
			s.logger.ErrorContext(ctx, "upsert program failed",
				"jobspring_project_id", p.JobspringProjectID,
				"error", err,
			)
			result.errors++
			continue
		}

		if err := s.repo.UpsertBeneficiaries(ctx, initiativeID, p.Beneficiaries); err != nil {
			s.logger.ErrorContext(ctx, "upsert beneficiaries failed",
				"initiative_id", initiativeID,
				"jobspring_project_id", p.JobspringProjectID,
				"error", err,
			)
			result.errors++
			continue
		}

		result.upserted++
	}

	return result, nil
}
```

- [ ] **Step 4: Implement main.go**

```go
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// mentorship-sync pulls Mentorship program data from Snowflake (or a JSON
// fixture in DEV) and upserts initiative_type=mentorship rows into CF Postgres.
//
// See docs/rewrite/10-mentorship-sync-dev-testing.md for the DEV testing strategy.
//
// Usage: run as a K8s CronJob (daily schedule). Exits 0 on success,
// non-zero on any error. K8s uses the exit code to track CronJob health.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/db"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/snowflake"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("mentorship-sync failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx := context.Background()
	start := time.Now()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Database pool — same pattern as initiatives-api and ledger-stats-sync.
	pool, err := db.NewPool(ctx, db.PoolConfig{
		DSN:      cfg.DatabaseURL,
		MaxConns: 5,
		MinConns: 1,
	})
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	defer pool.Close()

	// Wire the mentorship source: fixture (DEV) or real Snowflake (staging/prod).
	var src mentorshipSource
	if cfg.FixtureFile != "" {
		logger.Info("using fixture source", "path", cfg.FixtureFile)
		src = snowflake.NewFixtureSource(cfg.FixtureFile)
	} else {
		client, err := snowflake.NewClient(snowflake.ClientConfig{
			Account:    cfg.SnowflakeAccount,
			User:       cfg.SnowflakeUser,
			Warehouse:  cfg.SnowflakeWarehouse,
			Database:   cfg.SnowflakeDatabase,
			Role:       cfg.SnowflakeRole,
			PrivateKey: cfg.SnowflakePrivateKey,
		})
		if err != nil {
			return fmt.Errorf("snowflake client: %w", err)
		}
		defer client.Close()
		src = client
	}

	repo := db.NewMentorshipRepository(pool)
	syncer := newSyncer(repo, src, logger)

	logger.Info("mentorship-sync starting")

	result, err := syncer.Run(ctx)
	if err != nil {
		return fmt.Errorf("sync run: %w", err)
	}

	logger.Info("mentorship-sync complete",
		"duration", time.Since(start).String(),
		"total", result.total,
		"upserted", result.upserted,
		"errors", result.errors,
	)
	return nil
}

// config holds the runtime configuration for mentorship-sync.
type config struct {
	DatabaseURL        string
	FixtureFile        string // set in DEV; absence means use Snowflake
	SnowflakeAccount   string
	SnowflakeUser      string
	SnowflakeWarehouse string
	SnowflakeDatabase  string
	SnowflakeRole      string
	SnowflakePrivateKey string
}

func loadConfig() (*config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	cfg := &config{
		DatabaseURL: dbURL,
		FixtureFile: os.Getenv("MENTORSHIP_SYNC_FIXTURE_FILE"),
	}

	// Snowflake vars are only required when not using fixture.
	if cfg.FixtureFile == "" {
		for _, pair := range []struct{ key, field string }{
			{"SNOWFLAKE_ACCOUNT", cfg.SnowflakeAccount},
			{"SNOWFLAKE_USER", cfg.SnowflakeUser},
			{"SNOWFLAKE_WAREHOUSE", cfg.SnowflakeWarehouse},
			{"SNOWFLAKE_DATABASE", cfg.SnowflakeDatabase},
			{"SNOWFLAKE_ROLE", cfg.SnowflakeRole},
			{"SNOWFLAKE_PRIVATE_KEY", cfg.SnowflakePrivateKey},
		} {
			v := os.Getenv(pair.key)
			if v == "" {
				return nil, fmt.Errorf("%s is required when MENTORSHIP_SYNC_FIXTURE_FILE is not set", pair.key)
			}
		}
		cfg.SnowflakeAccount = os.Getenv("SNOWFLAKE_ACCOUNT")
		cfg.SnowflakeUser = os.Getenv("SNOWFLAKE_USER")
		cfg.SnowflakeWarehouse = os.Getenv("SNOWFLAKE_WAREHOUSE")
		cfg.SnowflakeDatabase = os.Getenv("SNOWFLAKE_DATABASE")
		cfg.SnowflakeRole = os.Getenv("SNOWFLAKE_ROLE")
		cfg.SnowflakePrivateKey = os.Getenv("SNOWFLAKE_PRIVATE_KEY")
	}

	return cfg, nil
}
```

Add the `discardLogger` helper to a new `helpers_test.go` in the same package (mirrors `ledger-stats-sync`):

```go
// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import "log/slog"

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
```

- [ ] **Step 5: Run all tests to confirm they pass**

```bash
cd backend && make test
```

Expected: all tests pass including the new syncer tests.

- [ ] **Step 6: Confirm binary compiles**

```bash
cd backend && go build ./cmd/mentorship-sync/...
```

Expected: no errors. Binary produced at `bin/mentorship-sync` (or default location).

- [ ] **Step 7: Commit**

```bash
cd backend
git add cmd/mentorship-sync/syncer.go cmd/mentorship-sync/syncer_test.go cmd/mentorship-sync/main.go cmd/mentorship-sync/helpers_test.go
git commit --signoff -m "feat(mentorship-sync): implement syncer and CronJob entry point"
```

---

## Task 7: Dockerfile

**Files:**
- Create: `backend/Dockerfile.mentorship-sync`

- [ ] **Step 1: Check existing Dockerfile pattern**

```bash
cat backend/Dockerfile.ledger-stats-sync
```

Use that as the template — same multi-stage build pattern.

- [ ] **Step 2: Create Dockerfile**

```dockerfile
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /mentorship-sync ./cmd/mentorship-sync

FROM gcr.io/distroless/static-debian12 AS runner
COPY --from=builder /mentorship-sync /mentorship-sync
# Copy fixture data so it's available when MENTORSHIP_SYNC_FIXTURE_FILE is set.
COPY cmd/mentorship-sync/testdata/ /app/testdata/
ENTRYPOINT ["/mentorship-sync"]
```

- [ ] **Step 3: Build the image locally to confirm it works**

```bash
cd backend && docker build -f Dockerfile.mentorship-sync -t mentorship-sync:local .
```

Expected: build succeeds, no errors.

- [ ] **Step 4: Commit**

```bash
cd backend
git add Dockerfile.mentorship-sync
git commit --signoff -m "feat(mentorship-sync): add Dockerfile for CronJob container image"
```

---

## Task 8: ArgoCD Values

**Files (in `lfx-v2-argocd` repo):**
- Create: `values/dev/lfx-crowdfunding-mentorship-sync.yaml`
- Create: `values/staging/lfx-crowdfunding-mentorship-sync.yaml`
- Create: `values/prod/lfx-crowdfunding-mentorship-sync.yaml`

- [ ] **Step 1: Check existing CronJob values pattern**

```bash
ls /Users/michal/src/github/linuxfoundation/lfx-v2-argocd/values/dev/ | grep -v crowdfunding
# Look for any CronJob example to understand the expected schema
cat /Users/michal/src/github/linuxfoundation/lfx-v2-argocd/values/dev/lfx-crowdfunding-backend.yaml
```

Adapt the structure from `lfx-crowdfunding-backend.yaml` but for a CronJob resource.

- [ ] **Step 2: Create DEV values**

```yaml
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
---
# DEV environment — uses fixture source; no Snowflake credentials required.
# TODO(analytics-dev): when ANALYTICS_DEV gold models are live, remove
# MENTORSHIP_SYNC_FIXTURE_FILE and set SNOWFLAKE_DATABASE: ANALYTICS_DEV instead.

image:
  repository: ghcr.io/linuxfoundation/lfx-crowdfunding-mentorship-sync
  tag: development
  pullPolicy: Always

schedule: "0 2 * * *"  # daily at 02:00 UTC

config:
  MENTORSHIP_SYNC_FIXTURE_FILE: /app/testdata/programs.json
```

- [ ] **Step 3: Create staging values**

```yaml
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
---
# Staging environment — real Snowflake credentials from K8s Secret.

image:
  repository: ghcr.io/linuxfoundation/lfx-crowdfunding-mentorship-sync
  tag: staging
  pullPolicy: Always

schedule: "0 2 * * *"  # daily at 02:00 UTC

config:
  SNOWFLAKE_DATABASE: ANALYTICS

secrets:
  - name: SNOWFLAKE_ACCOUNT
    secretName: mentorship-sync-secrets
    secretKey: snowflake-account
  - name: SNOWFLAKE_USER
    secretName: mentorship-sync-secrets
    secretKey: snowflake-user
  - name: SNOWFLAKE_WAREHOUSE
    secretName: mentorship-sync-secrets
    secretKey: snowflake-warehouse
  - name: SNOWFLAKE_ROLE
    secretName: mentorship-sync-secrets
    secretKey: snowflake-role
  - name: SNOWFLAKE_PRIVATE_KEY
    secretName: mentorship-sync-secrets
    secretKey: snowflake-private-key
```

- [ ] **Step 4: Create prod values**

Same as staging but with `tag: latest` and prod-appropriate schedule:

```yaml
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
---
# Production environment — real Snowflake credentials from K8s Secret.

image:
  repository: ghcr.io/linuxfoundation/lfx-crowdfunding-mentorship-sync
  tag: latest
  pullPolicy: IfNotPresent

schedule: "0 2 * * *"  # daily at 02:00 UTC

config:
  SNOWFLAKE_DATABASE: ANALYTICS

secrets:
  - name: SNOWFLAKE_ACCOUNT
    secretName: mentorship-sync-secrets
    secretKey: snowflake-account
  - name: SNOWFLAKE_USER
    secretName: mentorship-sync-secrets
    secretKey: snowflake-user
  - name: SNOWFLAKE_WAREHOUSE
    secretName: mentorship-sync-secrets
    secretKey: snowflake-warehouse
  - name: SNOWFLAKE_ROLE
    secretName: mentorship-sync-secrets
    secretKey: snowflake-role
  - name: SNOWFLAKE_PRIVATE_KEY
    secretName: mentorship-sync-secrets
    secretKey: snowflake-private-key
```

- [ ] **Step 5: Commit to lfx-v2-argocd**

```bash
cd /Users/michal/src/github/linuxfoundation/lfx-v2-argocd
git add values/dev/lfx-crowdfunding-mentorship-sync.yaml \
        values/staging/lfx-crowdfunding-mentorship-sync.yaml \
        values/prod/lfx-crowdfunding-mentorship-sync.yaml
git commit --signoff -m "feat(crowdfunding): add mentorship-sync CronJob ArgoCD values for dev/staging/prod"
```

---

## Task 9: Update PR #95 Documentation

**Files:**
- Modify: `backend/docs/rewrite/10-mentorship-sync-dev-testing.md` (on the PR #95 branch)

- [ ] **Step 1: Check out the PR #95 branch**

```bash
cd /Users/michal/src/github/linuxfoundation/lfx-crowdfunding
git fetch origin
git checkout docs/mentorship-sync-dev-testing  # adjust branch name if different
```

Confirm: `git log --oneline -3` shows the doc commit from PR #95.

- [ ] **Step 2: Replace "Why not a DEV Snowflake schema?" section**

Find this section in `backend/docs/rewrite/10-mentorship-sync-dev-testing.md` and replace it with:

```markdown
**Why not `ANALYTICS_DEV` right now?**
`ANALYTICS_DEV` exists in Snowflake and is supported by the Data Lake team (confirmed June 2026). However it is currently missing ~8 of the ~10 raw source tables required to build the gold model. Only two DEV tables exist today:
- `FIVETRAN_INGEST.DYNAMODB_PRODUCT_US_EAST1_DEV.JOBSPRING_DEV_PROJECTS`
- `FIVETRAN_INGEST.DYNAMODB_PRODUCT_US_EAST1_DEV.JOBSPRING_DEV_TASKS`

The gold model cannot be built until the remaining tables are added by the Data Lake team. The fixture source unblocks implementation without waiting on that timeline.

**Why an env var rather than `APP_ENV`?**
`MENTORSHIP_SYNC_FIXTURE_FILE` is an explicit, self-describing opt-in. Setting it to a path makes the intent unambiguous in the ArgoCD values file. It also allows the fixture source to be used locally without changing the global `APP_ENV`, and it can be applied to staging temporarily if needed (e.g. to test a schema change before Fivetran is updated).

**What this does not test**
The fixture source bypasses the Snowflake driver and SQL query entirely. That gap is covered by a unit test in `internal/infrastructure/snowflake/client_test.go` that asserts the expected SQL query string and field mapping against a mock driver. A manual smoke test against `ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS` (read-only, prod credentials) must be run by the deploying developer before the first DEV and staging deployments — documented in `docs/go-live-checklist.md`.
```

- [ ] **Step 3: Add the `ANALYTICS_DEV` follow-on section**

Append a new section after "Implementation":

```markdown
## Follow-on: `ANALYTICS_DEV` Path

When the missing raw DEV tables are added by the Data Lake team, CF will build bronze/silver/gold dbt models in `lf-dbt` so that `ANALYTICS_DEV.GOLD_FACT.MENTORSHIP_PROGRAMS` mirrors production. The DEV ArgoCD values are then updated: remove `MENTORSHIP_SYNC_FIXTURE_FILE`, set `SNOWFLAKE_DATABASE=ANALYTICS_DEV`. No code change needed — the `mentorshipSource` interface already supports it.

Steps in order:

| Step | Owner | Blocked on |
|---|---|---|
| 1. Add remaining ~8 raw DEV tables to `FIVETRAN_INGEST.DYNAMODB_PRODUCT_US_EAST1_DEV` | Data Lake (Shane/David) | Jira ticket to file |
| 2. Build bronze/silver/gold dbt models in `lf-dbt` | CF | Step 1 |
| 3. Provision DEV Snowflake service account (read-only to `ANALYTICS_DEV`) | Data Lake | Step 2 |
| 4. Update DEV ArgoCD values | CF | Step 3 |

The fixture source stays available for local development indefinitely — `MENTORSHIP_SYNC_FIXTURE_FILE` can always be set locally even after `ANALYTICS_DEV` is ready.

A `TODO(analytics-dev)` comment in `values/dev/lfx-crowdfunding-mentorship-sync.yaml` marks the swap point.
```

- [ ] **Step 4: Run the test plan from the PR**

```bash
# Confirm doc renders in GitHub — review markdown in editor
# Check file inventory in the doc matches the files created in Tasks 1-8
```

- [ ] **Step 5: Commit and push to the PR branch**

```bash
cd /Users/michal/src/github/linuxfoundation/lfx-crowdfunding
git add backend/docs/rewrite/10-mentorship-sync-dev-testing.md
git commit --signoff -m "docs(mentorship-sync): update DEV testing doc — correct ANALYTICS_DEV rationale, add follow-on path"
git push
```

---

## Self-Review Checklist

- [x] **Domain models** (`MentorshipProgram`, `MentorshipBeneficiary`) defined in Task 1, used consistently in Tasks 2–6
- [x] **`MentorshipRepository` interface** defined in Task 1, implemented in Task 4, consumed by syncer in Task 6
- [x] **`mentorshipSource` interface** defined in syncer (Task 6), satisfied by `FixtureSource` (Task 2) and `Client` (Task 3)
- [x] **Status normalisation** (`'hide'` → `'hidden'`) done in `Syncer.Run` (Task 6), tested in `TestSyncer_Run_normalisesHideStatus`
- [x] **Fixture file** created in Task 5, embedded in Dockerfile (Task 7), referenced in DEV ArgoCD values (Task 8)
- [x] **Snowflake vars required only when fixture absent** — validated in `loadConfig` (Task 6)
- [x] **`discardLogger` helper** added to `helpers_test.go` (Task 6), used by all syncer tests
- [x] **Method name consistency**: `FetchPrograms` used in Tasks 2, 3, and 6; `UpsertProgram`/`UpsertBeneficiaries`/`ListJobspringIDs` used in Tasks 1, 4, and 6
- [x] **`syncResult.errors` field** defined in syncer (Task 6), counted per-program, tested in `TestSyncer_Run_countsUpsertErrorsWithoutHalting`
- [x] **Doc update** (Task 9) aligned with design spec — corrects false rationale, adds follow-on section
