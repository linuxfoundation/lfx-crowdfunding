# Postgres Schema & Go Project Scaffold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the Go project scaffold and Postgres schema migration files that all other components (API, migration CLI, CronJobs) depend on.

**Architecture:** A Go monorepo with separate binaries under `cmd/` and shared business logic under `internal/`. The schema lives in `db/migrations/` as golang-migrate SQL files. A local Docker Compose Postgres is used for validation — no cloud infrastructure needed at this stage.

**Tech Stack:** Go 1.23+, PostgreSQL 17, `golang-migrate/migrate` CLI, `pgx/v5`, `sqlc`, Docker Compose.

---

## File Map

### Created by this plan

```
go.mod                                  # module: github.com/linuxfoundation/lfx-crowdfunding
go.sum
.env.local.example                      # local dev env vars template
docker-compose.yml                      # local Postgres for dev/testing

cmd/
  api/
    main.go                             # HTTP server entrypoint (stub — wire-up in next plan)
  mentorship-sync/
    main.go                             # CronJob entrypoint (stub)
  amount-raised-sync/
    main.go                             # CronJob entrypoint (stub)
  migrate/
    main.go                             # golang-migrate runner entrypoint

tools/
  migrate-cf/
    main.go                             # DynamoDB→Postgres one-time CLI (stub)

internal/
  initiatives/
    domain/
      initiative.go                     # Initiative struct + InitiativeType + Status constants
  subscriptions/
    domain/
      subscription.go                   # Subscription struct + Frequency + Status constants
  donations/
    domain/
      donation.go                       # Donation struct + PaymentMethod + Category constants
  organizations/
    domain/
      organization.go                   # Organization struct
  users/
    domain/
      user.go                           # User struct

db/
  migrations/
    001_initial.up.sql                  # Full schema: crowdfunding schema + all tables + indexes + FTS
    001_initial.down.sql                # DROP everything in reverse order

sqlc.yaml                               # sqlc config (queries written in next plan)
```

---

## Task 1: Go module + dependencies

**Files:**
- Create: `go.mod`
- Create: `go.sum` (generated)

- [ ] **Step 1: Initialise Go module**

```bash
cd /Users/michal/src/github/linuxfoundation/lfx-crowdfunding
go mod init github.com/linuxfoundation/lfx-crowdfunding
```

Expected output: `go: creating new go.mod: module github.com/linuxfoundation/lfx-crowdfunding`

- [ ] **Step 2: Add runtime dependencies**

```bash
go get github.com/jackc/pgx/v5@latest
go get github.com/go-chi/chi/v5@latest
go get github.com/golang-migrate/migrate/v4@latest
go get github.com/golang-migrate/migrate/v4/database/pgx/v5
go get github.com/golang-migrate/migrate/v4/source/file
```

- [ ] **Step 3: Add sqlc**

```bash
go get github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

- [ ] **Step 4: Tidy**

```bash
go mod tidy
```

Expected: `go.sum` created/updated, no errors.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit --signoff -m "chore: initialise Go module with core dependencies"
```

---

## Task 2: Docker Compose for local Postgres

**Files:**
- Create: `docker-compose.yml`
- Create: `.env.local.example`

- [ ] **Step 1: Write docker-compose.yml**

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: crowdfunding
      POSTGRES_PASSWORD: crowdfunding
      POSTGRES_DB: crowdfunding
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U crowdfunding"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

- [ ] **Step 2: Write .env.local.example**

```bash
# .env.local.example — copy to .env.local for local development
DATABASE_URL=postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding?sslmode=disable
```

- [ ] **Step 3: Start Postgres and verify connection**

```bash
docker compose up -d
docker compose ps
```

Expected: `postgres` container shows `healthy`.

```bash
docker compose exec postgres psql -U crowdfunding -c "SELECT version();"
```

Expected: PostgreSQL 17.x version string.

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml .env.local.example
git commit --signoff -m "chore: add docker-compose for local Postgres dev environment"
```

---

## Task 3: Domain structs — initiatives

**Files:**
- Create: `internal/initiatives/domain/initiative.go`

- [ ] **Step 1: Write initiative.go**

```go
// internal/initiatives/domain/initiative.go
package domain

import (
	"time"

	"github.com/google/uuid"
)

type InitiativeType string

const (
	InitiativeTypeProject     InitiativeType = "project"
	InitiativeTypeMentorship  InitiativeType = "mentorship"
	InitiativeTypeGeneralFund InitiativeType = "general_fund"
	InitiativeTypeEvent       InitiativeType = "event"
	InitiativeTypeOSTIF       InitiativeType = "ostif"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusSubmitted Status = "submitted"
	StatusApproved  Status = "approved"
	StatusPublished Status = "published"
	StatusHidden    Status = "hidden"
	StatusDeclined  Status = "declined"
)

// Initiative represents a fundable entity in the crowdfunding system.
// All initiative types (project, mentorship, general_fund, event, ostif)
// share this struct. Type-specific fields are nullable and only populated
// for the relevant InitiativeType.
type Initiative struct {
	ID             uuid.UUID      `db:"id"`
	InitiativeType InitiativeType `db:"initiative_type"`
	OwnerID        string         `db:"owner_id"`
	Name           string         `db:"name"`
	Slug           string         `db:"slug"`
	Status         Status         `db:"status"`
	Website        *string        `db:"website"`
	Description    *string        `db:"description"`
	Color          *string        `db:"color"`
	LogoURL        *string        `db:"logo_url"`
	Industry       *string        `db:"industry"`
	LegacyID       *string        `db:"legacy_id"`
	StripePlanID   *string        `db:"stripe_plan_id"`
	StripeProductID *string       `db:"stripe_product_id"`

	// project/mentorship only
	CodeOfConduct      *string `db:"code_of_conduct"`
	CIIProjectID       *string `db:"cii_project_id"`
	StacksID           *string `db:"stacks_id"`
	MentorshipProgramID *string `db:"mentorship_program_id"`

	// general_fund/event/ostif only
	City           *string    `db:"city"`
	Country        *string    `db:"country"`
	IsOnline       bool       `db:"is_online"`
	AcceptFunding  bool       `db:"accept_funding"`
	ApplicationURL *string    `db:"application_url"`
	EventStartDate *time.Time `db:"event_start_date"`
	EventEndDate   *time.Time `db:"event_end_date"`
	EventbriteURL  *string    `db:"eventbrite_url"`

	// JSONB columns stored as raw JSON bytes; unmarshalled by callers as needed
	Budgets        []byte `db:"budgets"`
	GithubStats    []byte `db:"github_stats"`
	Beneficiaries  []byte `db:"beneficiaries"`
	Contributors   []byte `db:"contributors"`
	CustomWebsites []byte `db:"custom_websites"`
	Sponsors       []byte `db:"sponsors"`

	// Cached financial column — kept fresh by amount-raised-sync CronJob
	// NULL before first cron run after a donation; display as 0.
	AmountRaisedCents *int64 `db:"amount_raised_cents"`

	// Workflow timestamps — set on status transitions, never updated after set
	SubmittedAt *time.Time `db:"submitted_at"`
	ApprovedAt  *time.Time `db:"approved_at"`
	PublishedAt *time.Time `db:"published_at"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
```

- [ ] **Step 2: Add uuid dependency**

```bash
go get github.com/google/uuid@latest
go mod tidy
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/initiatives/...
```

Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add internal/initiatives/domain/initiative.go go.mod go.sum
git commit --signoff -m "feat(domain): add Initiative domain struct with type and status constants"
```

---

## Task 4: Domain structs — subscriptions, donations, organizations, users

**Files:**
- Create: `internal/subscriptions/domain/subscription.go`
- Create: `internal/donations/domain/donation.go`
- Create: `internal/organizations/domain/organization.go`
- Create: `internal/users/domain/user.go`

- [ ] **Step 1: Write subscription.go**

```go
// internal/subscriptions/domain/subscription.go
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Frequency string

const (
	FrequencyMonthly Frequency = "monthly"
	FrequencyAnnual  Frequency = "annual"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
)

type Subscription struct {
	ID                       uuid.UUID          `db:"id"`
	StripeSubscriptionID     string             `db:"stripe_subscription_id"`
	StripeSubscriptionItemID *string            `db:"stripe_subscription_item_id"`
	InitiativeID             uuid.UUID          `db:"initiative_id"`
	UserID                   string             `db:"user_id"`
	OrgID                    *uuid.UUID         `db:"org_id"`
	Frequency                Frequency          `db:"frequency"`
	AmountInCents            int64              `db:"amount_in_cents"`
	Category                 *string            `db:"category"`
	PaymentMethod            *string            `db:"payment_method"`
	Status                   SubscriptionStatus `db:"status"`
	CreatedAt                time.Time          `db:"created_at"`
	UpdatedAt                time.Time          `db:"updated_at"`
}
```

- [ ] **Step 2: Write donation.go**

```go
// internal/donations/domain/donation.go
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Donation struct {
	ID              uuid.UUID  `db:"id"`
	StripeChargeID  *string    `db:"stripe_charge_id"` // NULL for invoice payments
	InitiativeID    uuid.UUID  `db:"initiative_id"`
	UserID          string     `db:"user_id"`
	OrgID           *uuid.UUID `db:"org_id"`
	Name            *string    `db:"name"`
	AmountInCents   int64      `db:"amount_in_cents"`
	Category        *string    `db:"category"`
	PaymentMethod   *string    `db:"payment_method"`
	PONumber        *string    `db:"po_number"`
	Status          *string    `db:"status"`
	CreatedAt       time.Time  `db:"created_at"`
}
```

- [ ] **Step 3: Write organization.go**

```go
// internal/organizations/domain/organization.go
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID        uuid.UUID `db:"id"`
	OwnerID   string    `db:"owner_id"`
	Name      string    `db:"name"`
	Status    string    `db:"status"`
	AvatarURL *string   `db:"avatar_url"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
```

- [ ] **Step 4: Write user.go**

```go
// internal/users/domain/user.go
package domain

import "time"

type User struct {
	ID                string    `db:"id"` // Auth0 subject
	StripeCustomerID  *string   `db:"stripe_customer_id"`
	GithubAccessToken *string   `db:"github_access_token"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}
```

- [ ] **Step 5: Verify all compile**

```bash
go build ./internal/...
```

Expected: no output (success).

- [ ] **Step 6: Commit**

```bash
git add internal/subscriptions/ internal/donations/ internal/organizations/ internal/users/
git commit --signoff -m "feat(domain): add Subscription, Donation, Organization, User domain structs"
```

---

## Task 5: golang-migrate up migration — schema creation

**Files:**
- Create: `db/migrations/001_initial.up.sql`

- [ ] **Step 1: Create migrations directory**

```bash
mkdir -p db/migrations
```

- [ ] **Step 2: Write 001_initial.up.sql**

```sql
-- db/migrations/001_initial.up.sql

CREATE SCHEMA IF NOT EXISTS crowdfunding;

-- Initiatives: unified table for all fundable things.
-- initiative_type discriminator: 'project' | 'mentorship' | 'general_fund' | 'event' | 'ostif'
-- status values: 'draft' | 'submitted' | 'approved' | 'published' | 'hidden' | 'declined'
-- Type-specific columns are nullable; only populated for the relevant initiative_type.
CREATE TABLE crowdfunding.initiatives (
  id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_type       text        NOT NULL,
  owner_id              text        NOT NULL,
  name                  text        NOT NULL,
  slug                  text        NOT NULL UNIQUE,
  status                text        NOT NULL,
  website               text,
  description           text,
  color                 text,
  logo_url              text,
  industry              text,
  legacy_id             text        UNIQUE,
  stripe_plan_id        text,
  stripe_product_id     text,

  -- project/mentorship only
  code_of_conduct       text,
  cii_project_id        text,
  stacks_id             text,
  mentorship_program_id text        UNIQUE,

  -- general_fund/event/ostif only
  city                  text,
  country               text,
  is_online             boolean     NOT NULL DEFAULT false,
  accept_funding        boolean     NOT NULL DEFAULT true,
  application_url       text,
  event_start_date      date,
  event_end_date        date,
  eventbrite_url        text,

  -- JSONB columns
  budgets               jsonb       NOT NULL DEFAULT '{}',
  github_stats          jsonb       NOT NULL DEFAULT '{}',
  beneficiaries         jsonb       NOT NULL DEFAULT '[]',
  contributors          jsonb       NOT NULL DEFAULT '[]',
  custom_websites       jsonb       NOT NULL DEFAULT '[]',
  sponsors              jsonb       NOT NULL DEFAULT '[]',

  -- Cached financial column — kept fresh by amount-raised-sync CronJob (hourly).
  -- NULL before first cron run after a donation; application treats NULL as 0.
  -- amount-raised-sync UPDATE must NOT touch updated_at (not a real initiative change).
  amount_raised_cents   bigint,

  -- Workflow timestamps — set once on status transition, never overwritten.
  submitted_at          timestamptz,
  approved_at           timestamptz,
  published_at          timestamptz,

  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT initiatives_initiative_type_valid CHECK (
    initiative_type IN ('project', 'mentorship', 'general_fund', 'event', 'ostif')
  ),
  CONSTRAINT initiatives_status_valid CHECK (
    status IN ('draft', 'submitted', 'approved', 'published', 'hidden', 'declined')
  ),
  CONSTRAINT budgets_is_object CHECK (jsonb_typeof(budgets) = 'object')
);

CREATE INDEX ON crowdfunding.initiatives (owner_id);
CREATE INDEX ON crowdfunding.initiatives (initiative_type);
CREATE INDEX ON crowdfunding.initiatives (status);

-- Full-text search index — replaces OpenSearch for discovery.
CREATE INDEX ON crowdfunding.initiatives
  USING gin(to_tsvector('english', name || ' ' || coalesce(description, '')));

-- Organizations
CREATE TABLE crowdfunding.organizations (
  id         uuid        PRIMARY KEY,
  owner_id   text        NOT NULL,
  name       text        NOT NULL,
  status     text        NOT NULL,
  avatar_url text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON crowdfunding.organizations (owner_id);

-- Subscriptions (recurring payments)
CREATE TABLE crowdfunding.subscriptions (
  id                          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  stripe_subscription_id      text        NOT NULL UNIQUE,
  stripe_subscription_item_id text,
  initiative_id               uuid        NOT NULL REFERENCES crowdfunding.initiatives(id),
  user_id                     text        NOT NULL,
  org_id                      uuid        REFERENCES crowdfunding.organizations(id),
  frequency                   text        NOT NULL,
  amount_in_cents             bigint      NOT NULL,
  category                    text        CHECK (category IN (
                                'development', 'marketing', 'meetups', 'travel',
                                'bug_bounty', 'documentation', 'mentee', 'other', 'diversity'
                              )),
  payment_method              text,
  status                      text        NOT NULL,
  created_at                  timestamptz NOT NULL DEFAULT now(),
  updated_at                  timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT subscriptions_frequency_valid CHECK (frequency IN ('monthly', 'annual')),
  CONSTRAINT subscriptions_status_valid    CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX ON crowdfunding.subscriptions (initiative_id);
CREATE INDEX ON crowdfunding.subscriptions (user_id);
CREATE INDEX ON crowdfunding.subscriptions (org_id);

-- Donations (one-time payments)
-- stripe_charge_id is UNIQUE but nullable: invoice donations have no charge ID at creation time.
-- Postgres UNIQUE allows multiple NULLs, so invoice rows do not conflict.
CREATE TABLE crowdfunding.donations (
  id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
  stripe_charge_id text        UNIQUE,
  initiative_id    uuid        NOT NULL REFERENCES crowdfunding.initiatives(id),
  user_id          text        NOT NULL,
  org_id           uuid        REFERENCES crowdfunding.organizations(id),
  name             text,
  amount_in_cents  bigint      NOT NULL,
  category         text        CHECK (category IN (
                     'development', 'marketing', 'meetups', 'travel',
                     'bug_bounty', 'documentation', 'mentee', 'other', 'diversity'
                   )),
  payment_method   text,
  po_number        text,
  status           text,
  created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON crowdfunding.donations (initiative_id);
CREATE INDEX ON crowdfunding.donations (user_id);

-- Users (minimal profile — auth lives in Auth0, payment account in Stripe)
-- github_access_token is plain text matching current LFF behavior.
-- Should be encrypted (KMS envelope) post-initial-release.
CREATE TABLE crowdfunding.users (
  id                  text        PRIMARY KEY,
  stripe_customer_id  text,
  github_access_token text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);
```

- [ ] **Step 3: Commit**

```bash
git add db/migrations/001_initial.up.sql
git commit --signoff -m "feat(db): add initial crowdfunding schema migration (up)"
```

---

## Task 6: golang-migrate down migration — schema teardown

**Files:**
- Create: `db/migrations/001_initial.down.sql`

- [ ] **Step 1: Write 001_initial.down.sql**

```sql
-- db/migrations/001_initial.down.sql
-- Drop tables in reverse dependency order (subscriptions/donations before initiatives/orgs).

DROP TABLE IF EXISTS crowdfunding.donations;
DROP TABLE IF EXISTS crowdfunding.subscriptions;
DROP TABLE IF EXISTS crowdfunding.users;
DROP TABLE IF EXISTS crowdfunding.initiatives;
DROP TABLE IF EXISTS crowdfunding.organizations;

DROP SCHEMA IF EXISTS crowdfunding;
```

- [ ] **Step 2: Commit**

```bash
git add db/migrations/001_initial.down.sql
git commit --signoff -m "feat(db): add initial crowdfunding schema migration (down)"
```

---

## Task 7: migrate entrypoint + run migration locally

**Files:**
- Create: `cmd/migrate/main.go`

- [ ] **Step 1: Write cmd/migrate/main.go**

```go
// cmd/migrate/main.go
// Runs golang-migrate schema migrations against the database.
// Usage: DATABASE_URL=... ./migrate [up|down]
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	migrationsPath := flag.String("migrations", "db/migrations", "path to migrations directory")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	sourceURL := fmt.Sprintf("file://%s", *migrationsPath)
	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		slog.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("migration up failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			slog.Error("migration down failed", "error", err)
			os.Exit(1)
		}
		slog.Info("migrations rolled back successfully")
	default:
		slog.Error("unknown direction", "direction", *direction)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build the migrate binary**

```bash
go build -o bin/migrate ./cmd/migrate/
```

Expected: `bin/migrate` created, no errors.

- [ ] **Step 3: Run migrations against local Postgres**

```bash
DATABASE_URL="postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding?sslmode=disable" \
  ./bin/migrate -direction=up -migrations=db/migrations
```

Expected output: `migrations applied successfully`

- [ ] **Step 4: Verify schema was created**

```bash
docker compose exec postgres psql -U crowdfunding -c "\dt crowdfunding.*"
```

Expected output lists: `initiatives`, `organizations`, `subscriptions`, `donations`, `users`

```bash
docker compose exec postgres psql -U crowdfunding -c "\d crowdfunding.initiatives"
```

Expected: full column list matches schema in `001_initial.up.sql`.

- [ ] **Step 5: Test rollback**

```bash
DATABASE_URL="postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding?sslmode=disable" \
  ./bin/migrate -direction=down -migrations=db/migrations
```

Expected output: `migrations rolled back successfully`

```bash
docker compose exec postgres psql -U crowdfunding -c "\dt crowdfunding.*"
```

Expected: `Did not find any relation named "crowdfunding.*"` (schema dropped).

- [ ] **Step 6: Re-apply migrations (leave DB ready for development)**

```bash
DATABASE_URL="postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding?sslmode=disable" \
  ./bin/migrate -direction=up -migrations=db/migrations
```

- [ ] **Step 7: Commit**

```bash
git add cmd/migrate/main.go
git commit --signoff -m "feat(cmd): add migrate entrypoint using golang-migrate"
```

---

## Task 8: Stub entrypoints for api, mentorship-sync, amount-raised-sync, tools/migrate-cf

**Files:**
- Create: `cmd/api/main.go`
- Create: `cmd/mentorship-sync/main.go`
- Create: `cmd/amount-raised-sync/main.go`
- Create: `tools/migrate-cf/main.go`

These stubs establish the binary entry points so `go build ./...` works cleanly. They are wired up in subsequent plans.

- [ ] **Step 1: Write cmd/api/main.go**

```go
// cmd/api/main.go
package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	slog.Info("crowdfunding API starting", "addr", addr)
	if err := http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Write cmd/mentorship-sync/main.go**

```go
// cmd/mentorship-sync/main.go
package main

import "log/slog"

func main() {
	slog.Info("mentorship-sync: not yet implemented")
}
```

- [ ] **Step 3: Write cmd/amount-raised-sync/main.go**

```go
// cmd/amount-raised-sync/main.go
package main

import "log/slog"

func main() {
	slog.Info("amount-raised-sync: not yet implemented")
}
```

- [ ] **Step 4: Write tools/migrate-cf/main.go**

```go
// tools/migrate-cf/main.go
// One-time DynamoDB → Postgres data migration CLI.
// Delete this tool after cutover is validated.
package main

import "log/slog"

func main() {
	slog.Info("migrate-cf: not yet implemented")
}
```

- [ ] **Step 5: Verify all binaries build**

```bash
go build ./cmd/... ./tools/...
```

Expected: no output (success). Binaries deposited in working directory (or use `-o bin/` if preferred).

- [ ] **Step 6: Commit**

```bash
git add cmd/api/ cmd/mentorship-sync/ cmd/amount-raised-sync/ tools/migrate-cf/
git commit --signoff -m "feat(cmd): add stub entrypoints for api, mentorship-sync, amount-raised-sync, migrate-cf"
```

---

## Task 9: sqlc configuration

**Files:**
- Create: `sqlc.yaml`
- Create: `db/queries/.gitkeep` (placeholder — queries written in next plan)

sqlc reads SQL queries from `db/queries/` and generates type-safe Go code into `internal/*/repository/`. This task sets up the config so queries can be added in the Go API plan without changing the scaffold.

- [ ] **Step 1: Install sqlc CLI**

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

Verify: `sqlc version` — expected: `v1.x.x`

- [ ] **Step 2: Write sqlc.yaml**

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/migrations/"
    gen:
      go:
        package: "repository"
        out: "internal/repository"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_db_tags: true
        emit_pointers_for_null_types: true
        overrides:
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
          - db_type: "pg_catalog.int8"
            go_type: "int64"
```

- [ ] **Step 3: Create placeholder query directory**

```bash
mkdir -p db/queries
touch db/queries/.gitkeep
```

- [ ] **Step 4: Verify sqlc generates cleanly (no queries yet — expect empty output)**

```bash
sqlc generate
```

Expected: no output, no errors. (With no `.sql` query files, sqlc generates nothing — that's correct at this stage.)

- [ ] **Step 5: Commit**

```bash
git add sqlc.yaml db/queries/.gitkeep
git commit --signoff -m "chore: add sqlc config for type-safe query generation"
```

---

## Task 10: .gitignore and bin/ directory

**Files:**
- Modify: `.gitignore` (create if absent)

- [ ] **Step 1: Add Go-appropriate .gitignore entries**

Append to `.gitignore` (create the file if it doesn't exist):

```gitignore
# Go binaries
bin/
*.exe

# Local env
.env.local
.env.*.local

# Editor
.idea/
.vscode/
*.swp

# Docker volumes (if using local bind mounts)
.postgres-data/
```

- [ ] **Step 2: Create bin/ directory with .gitkeep**

```bash
mkdir -p bin
touch bin/.gitkeep
```

- [ ] **Step 3: Commit**

```bash
git add .gitignore bin/.gitkeep
git commit --signoff -m "chore: add .gitignore and bin/ placeholder"
```

---

## Task 11: Verify full build and schema round-trip

This task is the acceptance gate for the entire plan. Everything must pass before moving to the next plan (Go API handlers).

- [ ] **Step 1: Clean build of all packages**

```bash
go build ./...
```

Expected: no output (success).

- [ ] **Step 2: Verify no vet issues**

```bash
go vet ./...
```

Expected: no output (success).

- [ ] **Step 3: Schema round-trip test**

```bash
# Down (clean state)
DATABASE_URL="postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding?sslmode=disable" \
  ./bin/migrate -direction=down -migrations=db/migrations

# Up (create schema)
DATABASE_URL="postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding?sslmode=disable" \
  ./bin/migrate -direction=up -migrations=db/migrations

# Verify tables
docker compose exec postgres psql -U crowdfunding -c \
  "SELECT table_name FROM information_schema.tables WHERE table_schema = 'crowdfunding' ORDER BY table_name;"
```

Expected table_name output:
```
 donations
 initiatives
 organizations
 subscriptions
 users
```

- [ ] **Step 4: Verify CHECK constraints fire**

```bash
docker compose exec postgres psql -U crowdfunding -c \
  "INSERT INTO crowdfunding.initiatives (initiative_type, owner_id, name, slug, status)
   VALUES ('invalid_type', 'auth0|test', 'Test', 'test-slug', 'published');"
```

Expected: `ERROR: new row for relation "initiatives" violates check constraint "initiatives_initiative_type_valid"`

```bash
docker compose exec postgres psql -U crowdfunding -c \
  "INSERT INTO crowdfunding.initiatives (initiative_type, owner_id, name, slug, status, budgets)
   VALUES ('project', 'auth0|test', 'Test', 'test-slug2', 'published', '[\"not\",\"an\",\"object\"]');"
```

Expected: `ERROR: new row for relation "initiatives" violates check constraint "budgets_is_object"`

- [ ] **Step 5: Verify FTS index exists**

```bash
docker compose exec postgres psql -U crowdfunding -c \
  "SELECT indexname FROM pg_indexes WHERE tablename = 'initiatives' AND indexdef LIKE '%gin%';"
```

Expected: one row with `initiatives_to_tsvector_idx` (or similar gin index name).

- [ ] **Step 6: Final commit**

```bash
git add bin/.gitkeep
git commit --signoff -m "chore: verify schema round-trip and constraint checks pass"
```

---

## Self-Review Checklist (completed)

- **Spec coverage:**
  - ✅ `crowdfunding` schema with all 5 tables
  - ✅ `amount_raised_cents` cached column (not the view — post-release)
  - ✅ All JSONB columns (`budgets`, `github_stats`, `beneficiaries`, `contributors`, `custom_websites`, `sponsors`)
  - ✅ CHECK constraints: `initiative_type`, `status`, `budgets_is_object`, `category` on subscriptions and donations
  - ✅ UNIQUE constraints: `slug`, `legacy_id`, `mentorship_program_id`, `stripe_subscription_id`, `stripe_charge_id`
  - ✅ FTS GIN index on `initiatives`
  - ✅ golang-migrate runner (`cmd/migrate/`)
  - ✅ Stub entrypoints for all binaries (`cmd/api`, `cmd/mentorship-sync`, `cmd/amount-raised-sync`, `tools/migrate-cf`)
  - ✅ sqlc config ready for next plan
  - ✅ Docker Compose local Postgres
  - ✅ Domain structs for all 5 entities

- **Not in this plan (next plans):**
  - Go API handlers and sqlc queries
  - DynamoDB→Postgres migration CLI implementation
  - CronJob implementations
  - Nuxt frontend
  - Kubernetes manifests

- **Type consistency:** All domain structs use `uuid.UUID` from `github.com/google/uuid` consistently. `db:` tags match SQL column names exactly.

- **No placeholders:** All steps contain exact commands, exact SQL, exact Go code.
