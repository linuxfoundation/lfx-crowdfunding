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

**Why not a DEV Snowflake schema?**
Snowflake does not have a DEV namespace for Mentorship data. Fivetran mirrors the `jobspring` production database into Snowflake — setting up a separate DEV connector would require Mentorship team involvement, an additional Fivetran connector (cost), and ongoing maintenance as the schema evolves. The payoff is low because the CronJob's Snowflake query is simple and separately unit-tested.

**Why an env var rather than `APP_ENV`?**
`MENTORSHIP_SYNC_FIXTURE_FILE` is an explicit, self-describing opt-in. Setting it to a path makes the intent unambiguous in the ArgoCD values file. It also allows the fixture source to be used locally without changing the global `APP_ENV`, and it can be applied to staging temporarily if needed (e.g. to test a schema change before Fivetran is updated).

**What this does not test**
The fixture source bypasses the Snowflake driver and SQL query entirely. That gap is covered by a unit test in `internal/infrastructure/snowflake/client_test.go` that asserts the expected SQL query string against a mock driver. The sync logic itself (upsert algorithm, field mapping, beneficiary handling) runs identically regardless of source — the fixture source exercises it fully.

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

## File Inventory

| File | Purpose |
|---|---|
| `internal/domain/models/mentorship_sync.go` | `MentorshipProgram`, `MentorshipBeneficiary` domain types |
| `internal/infrastructure/snowflake/client.go` | Real Snowflake client; queries and maps rows |
| `internal/infrastructure/snowflake/client_test.go` | Unit test asserting the SQL query against a mock driver |
| `internal/infrastructure/snowflake/fixture_source.go` | JSON-file-backed source for DEV |
| `cmd/mentorship-sync/main.go` | Entry point; wires source based on `MENTORSHIP_SYNC_FIXTURE_FILE` |
| `cmd/mentorship-sync/syncer.go` | Sync algorithm; works identically against either source |
| `cmd/mentorship-sync/testdata/programs.json` | Fixture data committed to the repository |
| `Dockerfile.mentorship-sync` | Container image build |
| `lfx-v2-argocd` values (dev/staging/prod) | Environment-specific source selection |
