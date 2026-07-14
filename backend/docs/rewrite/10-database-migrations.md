# Automated Database Migrations — Proposal

**Status:** Proposal (for review with the platform/DevOps team)
**Author:** (draft)
**Context:** We currently have **no automated migration mechanism**. Schema changes in
`backend/db/migrations/` only reach a database when someone runs the `.sql` by hand.
That is how the recent prod incident happened: an `ALTER TABLE` (`004_sponsorship_tiers`)
run manually in a SQL editor left an open, uncommitted transaction holding an
`ACCESS EXCLUSIVE` lock, which blocked all live app reads on `initiatives` and
`initiative_sponsorship_tiers` until the transaction was released. This proposal
removes the human-in-the-loop step entirely.

---

## 1. Current state

| Where a migration mechanism could live | Present? |
| -------------------------------------- | -------- |
| Go code / app startup (`golang-migrate` import) | ❌ not in `go.mod` |
| `Makefile` (`migrate` target) | ❌ only `db-seed` (seed data, localhost-guarded) |
| Dockerfile (ships `db/migrations/`) | ❌ files are **not** copied into the image |
| Helm chart (migration Job / hook) | ❌ none |
| CI | ❌ none |

Migration files follow the `golang-migrate` convention (`NNN_name.up.sql` /
`NNN_name.down.sql`), and `CLAUDE.md` claims migrations run "via golang-migrate" —
but that tooling was never wired up. Migrations `001`–`003` reached prod by manual
application.

---

## 2. How other LFX v2 K8s services do it

Two directly-comparable v2 services deploy the same way we do (Helm chart synced by
ArgoCD onto the shared v2 cluster). They represent the **two viable patterns**:

### A. `lfx-changelog` — Helm pre-upgrade hook Job (ArgoCD PreSync)
- A `templates/migration-job.yaml` with `helm.sh/hook: pre-install,pre-upgrade`.
- ArgoCD renders the chart, sees the hook annotation, and runs the Job as a **PreSync**
  resource **before** applying the Deployment.
- The migration binary/CLI + migration files ship **inside the app image**, so the Job
  runs the exact migrations that match the app version being deployed.
- **Fails the deploy on a bad migration:** if the Job fails, the ArgoCD sync fails and
  the **old pods keep running** — a broken migration doesn't roll out new code on top of
  itself. This is *not* full protection, though: a PreSync migration is **committed before
  the new Deployment is applied**, so a schema change that is incompatible with the
  currently-running (old) pods can break production the moment it commits, and stays
  committed even if the subsequent rollout fails. The hook alone does not prevent this —
  see the expand/contract requirement in §4.
- Documented in `lfx-changelog/docs/database-migrations.md`. (Tool there is Prisma, but
  the mechanism is tool-agnostic.)

### B. `lfx-sanctions-screening` — migrate on app startup, in-process
- Uses **`golang-migrate/v4`** (the *same* tool our files are written for) with the
  *same* `NNN_name.up.sql/.down.sql` convention and `//go:embed migrations/*.sql`.
- `store.RunMigrations()` is called from `cmd/server/main.go` **at startup**, before the
  pool opens. Idempotent (`ErrNoChange` tolerated).
- Chart runs **`replicaCount: 2`** — so on every deploy, two pods call `RunMigrations`
  concurrently. `golang-migrate`'s advisory lock stops them from colliding, but see the
  critique below.

---

## 3. Recommendation

**Adopt pattern A (ArgoCD PreSync hook Job), using `golang-migrate`.**

This is a deliberate blend of the two precedents:

- **Execution model = `lfx-changelog`** (PreSync hook Job). It is the correct model for a
  GitOps + shared-prod service, and it is already documented and proven on this platform.
- **Tool = `golang-migrate`**, matching `lfx-sanctions-screening` *and* our existing
  migration files and `CLAUDE.md`. No file conversion, no new tool to standardize on.

Concretely:

1. **Ship migrations in the image.** `//go:embed` is relative to the package that
   declares it and cannot reach across directories with `..`, so the embed must live in a
   package that *contains* `migrations/` as a subdirectory. Put the embed + `RunMigrations`
   in a `db` package alongside the SQL (e.g. `db/migrate.go` with
   `//go:embed migrations/*.sql`) — mirroring how `lfx-sanctions-screening` does it
   (`internal/store/migrations.go` embeds its sibling `migrations/`). Then add a tiny
   `cmd/migrate` binary that imports that package and calls `RunMigrations()` / `m.Up()`.
   Update the Dockerfile to build the binary (the app image already exists; add one binary).
2. **Add a Helm hook Job** (`templates/migrate-job.yaml`) modeled on changelog's:
   ```yaml
   annotations:
     "helm.sh/hook": pre-install,pre-upgrade
     "helm.sh/hook-weight": "-1"
     "helm.sh/hook-delete-policy": before-hook-creation
   spec:
     backoffLimit: 3
     activeDeadlineSeconds: 120
     template:
       spec:
         restartPolicy: Never
         containers:
           - name: migrate
             image: {{ include "...image" . }}
             command: ["/app/migrate"]
   ```
   Reuse the app's existing DB credentials block. ArgoCD runs it PreSync; a failure fails
   the sync and leaves current pods running.
3. **Bake in the lock safety rails** (the direct lesson from the incident). The migrate
   connection sets a short `lock_timeout` and a `statement_timeout`, e.g. DSN
   `?options=-c%20lock_timeout%3D5s%20-c%20statement_timeout%3D30s`. If a migration can't
   acquire its `ACCESS EXCLUSIVE` lock quickly (table busy with prod traffic), it **fails
   fast and rolls back** instead of queuing and freezing the app. Fail-and-retry beats
   hang. The `activeDeadlineSeconds: 120` is the outer backstop.

### Why not pattern B (migrate on startup) for us

I considered it — it's less code (no chart Job, no Dockerfile target) and uses our exact
tool. I'm recommending against it for a **shared prod** service:

- **Replica race + slow lock = worse blast radius.** With `replicaCount > 1`, every pod
  runs migrations on boot. The advisory lock serializes them, but a migration that takes a
  long `ACCESS EXCLUSIVE` lock now blocks pod **readiness** across the rollout — the same
  "everything waits" failure we just had, just relocated into the deploy.
- **Couples schema change to process restart**, and re-attempts on every crash/restart,
  not just deploys.
- **No clean fail-closed story.** A failing migration crash-loops pods rather than
  cleanly failing a sync and leaving the old version serving.

Pattern B is fine for `lfx-sanctions-screening` (smaller, newer, low traffic). For a
service on shared prod RDS where a lock affects other tenants, the PreSync Job's ordering
and fail-closed behavior are worth the extra ~40 lines.

---

## 4. Prerequisites & open questions (need DevOps)

1. **Baseline 001–003.** They are already applied to prod but there is no
   `schema_migrations` tracking table. Before the first automated `up`, we must create the
   table and mark 001–003 as applied (`migrate force 3` after verifying the schema
   matches), or the first run will try to re-apply them. **This is a one-time, deliberate
   prod step** and must be done by/with DevOps.
2. **Never automate `down` on prod.** Only 004/005 even have `.down.sql`. Wire **only
   `up`** into automation; treat `down` as a local/testing aid. Prod rollbacks = forward-fix
   migrations, not `down`.
3. **Network path & credentials for the Job.** The Job runs *inside* the cluster, so it
   reaches RDS directly (no tunnel/WARP needed) — but it needs the DB secret mounted the
   same way the app gets it. Confirm the ArgoCD-managed values wire this.
4. **Heavy/locking migrations still need care.** The constant-default *`ADD COLUMN`* part
   of our 004/005 is cheap (metadata-only). But both add an inline **validated `CHECK`
   constraint** (`donation_mode IN (...)`, `donation_tier IN (...)`), and validating a
   `CHECK` makes PostgreSQL **scan every existing row while holding `ACCESS EXCLUSIVE`** —
   on a large table that recreates the exact blocking condition this proposal is trying to
   prevent. For constraints on large tables, add them in two steps: `ADD CONSTRAINT ...
   NOT VALID` (fast, takes a brief lock) then `VALIDATE CONSTRAINT` (scans without the
   exclusive lock) in a later statement/migration. Similarly, future volatile-default
   columns or non-concurrent indexes take long locks; `lock_timeout` limits the damage,
   and indexes should use `CREATE INDEX CONCURRENTLY`. Note `CREATE INDEX CONCURRENTLY`
   cannot run inside a transaction — `golang-migrate`'s postgres driver has **no per-file
   directive** for this; the documented approach is to put such a statement in its **own
   dedicated migration file** (with multi-statement mode off, which is the default) so it
   is not wrapped in a transaction.
5. **Separately: tighten `idle_in_transaction_session_timeout`.** Prod is currently `1d`
   (24h) with `lock_timeout=0`. Even with automated migrations, a per-role
   `ALTER ROLE crowdfunding SET idle_in_transaction_session_timeout = '120s'` would have
   auto-killed the incident's stuck session. Platform-level change — raise with DevOps.

---

## 5. Proposed next steps

1. Review this doc with the platform/DevOps team; confirm the pattern-A choice and the
   baseline plan for 001–003.
2. PR: add `cmd/migrate` (embed + `Up()`), Dockerfile target, and
   `templates/migrate-job.yaml`.
3. Coordinate the one-time `schema_migrations` baseline on dev → staging → prod.
4. Update `CLAUDE.md` to describe the real mechanism.
