# Mentorship Sync — DEV Environment Testing

## Problem

The `mentorship-sync` CronJob pulls Mentorship program data from Snowflake and upserts it into CF Postgres. Snowflake holds only production data — it is a Fivetran mirror of the `jobspring` production database. There is no DEV layer in Snowflake, so the CronJob cannot run against real data in the dev environment without connecting to production Snowflake.

This creates two problems:

1. **No test data** — running the CronJob in DEV against production Snowflake reads live program records into the DEV database, which is undesirable and potentially destructive.
2. **Credential coupling** — requiring Snowflake credentials in DEV adds operational overhead and a security surface for a non-production environment.

## Decision

The `mentorship-sync` CronJob supports a **fixture source** controlled by a single environment variable. When `MENTORSHIP_SYNC_FIXTURE_FILE` is set, the CronJob reads program data from a JSON file on disk instead of querying Snowflake. When the variable is absent, it connects to Snowflake as normal.

- **DEV** — `MENTORSHIP_SYNC_FIXTURE_FILE=/app/testdata/programs.json` is set; no Snowflake credentials are required.
- **Staging / Production** — `MENTORSHIP_SYNC_FIXTURE_FILE` is absent; `SNOWFLAKE_DSN` is set from a Kubernetes Secret.

The fixture file is committed to the repository under `cmd/mentorship-sync/testdata/programs.json` and contains a small set of realistic records sufficient to exercise all upsert code paths.

## Rationale

**Why not `ANALYTICS_DEV` right now?**
`ANALYTICS_DEV` exists in Snowflake and is supported by the Data Lake team (confirmed June 2026). However it is currently missing ~8 of the ~10 raw source tables required to build the gold model. Only two DEV tables exist today:
- `FIVETRAN_INGEST.DYNAMODB_PRODUCT_US_EAST1_DEV.JOBSPRING_DEV_PROJECTS`
- `FIVETRAN_INGEST.DYNAMODB_PRODUCT_US_EAST1_DEV.JOBSPRING_DEV_TASKS`

The gold model cannot be built until the remaining tables are added by the Data Lake team. The fixture source unblocks implementation without waiting on that timeline.

**Why an env var rather than `APP_ENV`?**
`MENTORSHIP_SYNC_FIXTURE_FILE` is an explicit, self-describing opt-in. Setting it to a path makes the intent unambiguous in the ArgoCD values file. It also allows the fixture source to be used locally without changing the global `APP_ENV`, and it can be applied to staging temporarily if needed (e.g. to test a schema change before Fivetran is updated).

**What this does not test**
The fixture source bypasses the Snowflake driver and SQL query entirely. That gap is covered by a unit test in `internal/infrastructure/snowflake/client_test.go` that asserts the expected SQL query string and field mapping against a mock driver. A manual smoke test against `ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS` (read-only, prod credentials) must be run by the deploying developer before the first DEV and staging deployments — documented in `docs/go-live-checklist.md`.

## Implementation

### Interface

A `mentorshipSource` interface is defined at the point of consumption in `cmd/mentorship-sync/syncer.go`:

```go
type mentorshipSource interface {
    FetchPrograms(ctx context.Context) ([]models.MentorshipProgram, error)
}
```

Two implementations satisfy it:

| Implementation | Package | When used |
|---|---|---|
| `snowflake.Client` | `internal/infrastructure/snowflake` | Production / staging |
| `snowflake.FixtureSource` | `internal/infrastructure/snowflake` | DEV / local |

### Wire-up in `main.go`

```go
var src mentorshipSource
if cfg.FixtureFile != "" {
    src = snowflake.NewFixtureSource(cfg.FixtureFile)
} else {
    var err error
    src, err = snowflake.NewClient(cfg.SnowflakeDSN)
    if err != nil {
        return fmt.Errorf("snowflake client: %w", err)
    }
}
syncer := newSyncer(repo, src, logger)
```

### Fixture file

`cmd/mentorship-sync/testdata/programs.json` contains 3–5 records covering:

- An active program with approved beneficiaries
- A pending program with no beneficiaries
- A hidden program (exercises `'hide'` → `'hidden'` status normalization)

The fixture schema mirrors `models.MentorshipProgram` exactly, so any field additions to the domain model require a corresponding update to the fixture file.

### ArgoCD values

```yaml
# values/dev/mentorship-sync.yaml
env:
  - name: MENTORSHIP_SYNC_FIXTURE_FILE
    value: /app/testdata/programs.json
  # SNOWFLAKE_DSN is not set in DEV

# values/prod/mentorship-sync.yaml
env:
  - name: SNOWFLAKE_DSN
    valueFrom:
      secretKeyRef:
        name: mentorship-sync-secrets
        key: snowflake-dsn
  # MENTORSHIP_SYNC_FIXTURE_FILE is not set in prod
```

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

## File Inventory

| File | Purpose |
|---|---|
| `internal/domain/models/mentorship_sync.go` | `MentorshipProgram`, `MentorshipBeneficiary` domain types |
| `internal/domain/repository.go` | `MentorshipRepository` interface (added) |
| `internal/infrastructure/snowflake/client.go` | Real Snowflake client; queries `ANALYTICS.GOLD_FACT.MENTORSHIP_PROGRAMS` |
| `internal/infrastructure/snowflake/client_test.go` | Unit test: SQL query string + struct field mapping via mock driver |
| `internal/infrastructure/snowflake/fixture_source.go` | JSON-file-backed source for DEV |
| `internal/infrastructure/snowflake/fixture_source_test.go` | Unit tests for fixture source |
| `internal/infrastructure/db/mentorship_repository.go` | `MentorshipRepository` implementation |
| `internal/infrastructure/db/mentorship_repository_test.go` | Compile-time interface check |
| `cmd/mentorship-sync/main.go` | Entry point; wires source based on `MENTORSHIP_SYNC_FIXTURE_FILE` |
| `cmd/mentorship-sync/syncer.go` | Sync algorithm; `mentorshipSource` interface |
| `cmd/mentorship-sync/syncer_test.go` | Table-driven tests for syncer algorithm |
| `cmd/mentorship-sync/helpers_test.go` | `discardLogger` test helper |
| `cmd/mentorship-sync/testdata/programs.json` | Fixture data (3 records) committed to repo |
| `Dockerfile.mentorship-sync` | Container image build |
| `lfx-v2-argocd` values (dev/staging/prod) | Environment-specific source selection |
