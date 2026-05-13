<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Architectural Decisions

Decisions made during discovery (May 2026). Each decision has a rationale.
Update this file when decisions change — note what changed and why.

---

## Scope — Initial Release

### What is IN scope

- Project CRUD (create, edit, publish, hide) — initiative_type: `project`
- Mentorship initiative display and Snowflake-driven sync — initiative_type: `mentorship`
- Fund CRUD (`general fund`, event, ostif) — replaces "entities"
- Organization CRUD
- Donations (one-time, card and invoice)
- Subscriptions (recurring monthly/annual, card and invoice)
- Subscription management (cancel, update payment method)
- Payment account management (Stripe Connect for project owners)
- Backer/supporter list
- Financial view (transaction history from Ledger)
- Funding status / amount raised
- Project/entity search and discovery
- GitHub stats on projects
- Auth0 authentication
- Email approval flows (project, entity, expense via JWT links)
- Admin email dry-run mode (`EMAIL_DRY_RUN=true`)
- ArgoCD deployment to Kubernetes

### What is OUT of scope for initial release

- Security vulnerabilities / Stacks integration
- Sponsor Tiers (shown in UI prototype but deferred)
- Datadog RUM
- Intercom
- Ledger Service changes (kept unchanged on Lambda)
- Mentorship backend changes
- Mentorship UI changes (Mentorship is not rewritten until it moves to Kubernetes)
- RS Category 2 Postgres migration (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`) — deferred until RS moves to Kubernetes (timeline TBD, see OQ-7)
- OpenSearch decommission — deferred until RS moves to Kubernetes (see OQ-7)
- CII badge validation (not in initial release UI)
- Diversity API integration
- Vulnerability API integration
- `travel_fund` as a distinct fund type (the 26 DynamoDB `entityType = 'other'` rows are stored as `initiative_type = 'other'` during migration — they do not display as "General Fund")
- `initiative` as a distinct fund type (merged into `general fund` during migration)
- `community` entity type (3 rows from 2019, all declined/submitted; migrated as `initiative_type = 'community'` — no active UI, no new rows expected)

---

## Database

### PostgreSQL, not DynamoDB

Target database is PostgreSQL. DynamoDB is decommissioned after successful migration and cutover.

### Schema separation — one schema, one Postgres instance

The CF Postgres instance has one schema:

- `crowdfunding` — owned by CF Go API (initiatives, orgs, subscriptions, donations, users)

There is no `reimbursement` schema on the CF Postgres instance. The Reimbursement Service is a Lambda running in a separate AWS VPC and cannot reach the shared LFX v2 RDS (which is `publicly_accessible = false`, reachable only from within the K8s cluster VPC). Co-locating RS tables on CF Postgres would require making the shared RDS publicly accessible or establishing cross-account VPC peering — both unacceptable.

RS continues using OpenSearch for its own three tables (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`) until it moves to Kubernetes. When RS moves to K8s it will get its own database on the shared RDS via the same `lfx-v2-opentofu` pattern as CF (one four-line entry in `postgres.tf`). At that point RS can also read CF data directly from the `crowdfunding` schema via a read-only Postgres role.

For the initial CF release, RS reads CF-owned data (project/entity/user lookups) via three narrow internal HTTP endpoints on the CF Go API — the same network path RS uses today for other CF calls, over public HTTPS. See OQ-7 for the migration plan.

Ledger Service keeps its own separate Postgres DB (Ledger DB) in a separate AWS account. It is not changed in the initial release. CF calls the Ledger HTTP API read-only for transaction stats and balance data — exactly as today.

Ledger DB co-location on the same RDS instance as CF DB was considered and rejected. Ledger DB is RDS in AWS Account B; CF DB is on the shared LFX v2 RDS in AWS Account A (K8s). Even if both were on the same RDS instance, they would be separate Postgres databases — plain SQL JOINs across databases on the same RDS instance are not possible without `postgres_fdw` or `dblink`, neither of which is provisioned. Co-location brings no practical benefit given the mirroring approach below.

**Plan A (initial release, pending OQ-18 architect approval):** a `ledger-sync` K8s CronJob mirrors the Ledger `ledger` table into `crowdfunding.ledger_transactions` in CF DB via a direct read-only cross-account DB connection. Once mirrored, the `initiative_funding_summary` materialized view aggregates all financial fields from `ledger_transactions` directly — no Ledger HTTP API calls needed at read time. The `amount-raised-sync` CronJob is decommissioned at that point. See OQ-16 and OQ-17 for open questions that must be resolved before implementing the sync.

**Plan B (fallback if architect rejects cross-account DB access):** a `ledger-stats-sync` K8s CronJob calls the Ledger HTTP API to sync pre-aggregated stats (e.g. `amount_raised_in_cents`, backer count) per initiative and stores them as cached columns on `crowdfunding.initiatives`. This is essentially an extension of the current `amount-raised-sync` approach — more fields, same mechanism. It requires Ledger API + CronJob + CF DB schema changes per new UI field. OQ-18 must be resolved before implementation begins.

### `projects` + `funds` → unified `initiatives` table

The old `projects` and `funds` tables are merged into a single `crowdfunding.initiatives` table with an `initiative_type` discriminator column.

`initiative_type` values: `project` | `mentorship` | `general fund` | `event` | `ostif` | `other` (26 legacy migrated rows) | `community` (3 legacy migrated rows)

Rationale: all initiative types share the same donor/subscription FK, the same Ledger balance lookup via original DynamoDB string ID, the same status workflow, and the same discovery/search surface. Two tables required a polymorphic FK (`project_id OR fund_id`) with a CHECK constraint and no hard FK enforcement. One table gives a clean `initiative_id` FK on subscriptions and donations, simpler queries, and a unified search/discovery endpoint.

Type-specific sparse columns (e.g. `event_start_date`, `jobspring_project_id`, `city`) are nullable and only populated for the relevant `initiative_type`. This is the standard discriminated-union pattern for a sparse table — preferable to two tables with a UNION when the shared columns dominate, which they do here.

### Budget categories → normalized `initiative_goals` child table

Budget categories (development, marketing, meetups, travel, bugBounty, documentation, mentee, other) are stored in a normalized `initiative_goals` child table — one row per category per initiative. This is schema v2.0.0 and supersedes an earlier JSONB design.

**Schema:**
```sql
CREATE TABLE initiative_goals (
  id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id   UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name            TEXT         NOT NULL,
  amount_in_cents BIGINT       NOT NULL DEFAULT 0 CHECK (amount_in_cents >= 0),
  allocation      TEXT,
  repo_link       TEXT,         -- 'development' category only
  description     TEXT,         -- entity goals only
  color           VARCHAR(10),  -- entity goals only
  icon            TEXT,         -- entity goals only
  sort_order      INTEGER       DEFAULT 0,
  UNIQUE (initiative_id, name)
);
```

Projects produce up to 8 rows (one per fixed category). Entities produce one row per element in `entity.Goals[]`. Mentorship-specific data (skills, terms, mentors, custom term) lives in the dedicated child tables: `initiative_mentors`, `initiative_program_info_skills`, `initiative_program_info_terms`, `initiative_program_info_custom_term`.

**`total_annual_goal_in_cents`** is derived at read time:
```sql
SELECT initiative_id, SUM(amount_in_cents) FROM initiative_goals GROUP BY initiative_id;
```

**Migration:** Project budget categories map to `initiative_goals` rows. Entity `goals[]` arrays map to `initiative_goals` rows with free-form names.

### `category` validation on subscriptions and donations

The `category` column on `subscriptions` and `donations` is a free-form `TEXT` field referencing a budget category name on the target initiative. Validation is enforced at the API layer only (not at the DB level with a CHECK constraint, since entity goal names are free-form).

**API level:** validates that the category exists as an `initiative_goals.name` row on the target initiative before accepting a payment. This covers both the fixed project categories (`development`, `marketing`, `meetups`, `travel`, `bugBounty`, `documentation`, `mentee`, `other`) and the free-form entity goal names.

### `stripe_charge_id` nullable on donations

`stripe_charge_id` is nullable. Invoice-based donations have no Stripe charge ID at creation time. There is no UNIQUE constraint on `stripe_charge_id` in the schema — the column is plain `VARCHAR(255) NULL`.

### Donor details snapshot on donations

Donor display name and avatar are stored as a snapshot inside `donations.cached_details` JSONB at the time of donation, as `{"backerDetails": {"name": "...", "avatarURL": "..."}}`. There is no separate `name` or `avatar_url` column on the `donations` table. This snapshot differs from the live Auth0 profile for org/invoice donations where the payer is an individual but the displayed donor is the company name.

### `amount_in_cents` → `bigint`

All monetary amounts stored as `bigint` (int8).

Rationale: max donation is $999,999.99 = 99,999,999 cents, which fits in `int4` (~2.1B max). However `bigint` costs nothing on modern Postgres, eliminates overflow risk on aggregated totals, and matches the existing Go code which uses `int64` throughout. No reason to use `int4`.

### Two separate tables for subscriptions and donations

`subscriptions` and `donations` are separate tables, not merged with a `payment_type` enum.

Rationale: they represent different Stripe object types (`Subscription` vs `Charge`), have different lifecycles (recurring vs one-time), different cancellation/update logic, and different fields (frequency, stripe_subscription_id on subscriptions; stripe_charge_id on donations). Merging would create nullable columns and make queries less obvious.

### Ledger-derived financial fields — Plan A (ledger-sync + materialized view) or Plan B (stats-sync cached columns)

The approach for initial release depends on OQ-18 architect approval (see above).

**If Plan A (ledger-sync) is approved:** The `ledger-sync` CronJob and `initiative_funding_summary` materialized view ship as part of the initial release. The `amount-raised-sync` CronJob is decommissioned. Additional financial fields (backer count, subscription totals, etc.) are served from the view without cached columns.

**If Plan B (fallback):** Ledger-derived financial data is stored as cached columns on `crowdfunding.initiatives`, kept fresh by CronJob calls to the Ledger HTTP API. The initial release caches `amount_raised_in_cents` only (via `amount-raised-sync`). Additional UI fields require separate Ledger API endpoints + CronJob changes + new cached columns — they are not individually back-filled without those changes.

`amount_raised_in_cents` is stored as a cached column on `crowdfunding.initiatives` and kept fresh solely by the `amount-raised-sync` CronJob.

**Sole mechanism: `amount-raised-sync` CronJob (hourly)**

The CronJob (`cmd/amount-raised-sync/`) runs every hour and calls `GET /balance/{id}` for all published initiatives, updating `amount_raised_in_cents`. It uses `initiatives.id` as the Ledger `{id}`. For migrated initiatives, that means: if the legacy DynamoDB ID was already UUID-form, `initiatives.id` matches that original ID; otherwise `initiatives.id` is the deterministic UUID generated during migration via the `uuid5` mapping, not the original non-UUID DynamoDB string. For post-cutover initiatives with no DynamoDB origin, `id` is the Postgres UUID (pending OQ-15 confirmation). It is the **only** mechanism for keeping `amount_raised_in_cents` current. It covers all balance change sources:

- **Expensify disbursements** — when beneficiaries draw funds, Ledger records a DEBIT. This produces no Stripe event. Without the cron, `amount_raised_in_cents` would only ever increase, never reflecting disbursements. This is a correctness requirement, not optional.
- **Donations and subscription renewals** — Stripe charges are processed by Ledger's own webhook. The cron reads the authoritative balance from Ledger after Ledger has processed it.
- **Ledger corrections** — manual transaction corrections produce no CF signal; the cron picks them up on the next run.

The Stripe webhook handler (`POST /v1/hooks/stripe`) does **not** call the Ledger API. It handles only `customer.subscription.deleted` (cancel subscription in Postgres). It does not call `GET /balance/` — that is the cron's job. Rationale: a Stripe webhook triggering a Ledger API call requires a 5-second delay to avoid a race condition with Ledger's own webhook handler — a timing hack. The hourly cron makes this unnecessary and gives the same freshness guarantee with simpler code.

The cron UPDATE must **not** include `updated_on` in the SET clause. Background reconciliation is not a meaningful initiative change and must not produce false-positive change signals for Fivetran sync, RS bulk endpoint, or audit logs:

```sql
-- correct: updated_on is not touched
UPDATE crowdfunding.initiatives
SET amount_raised_in_cents = $1
WHERE id = $2
```

**`NULL` treated as `0` in display layer**

`amount_raised_in_cents` is `NULL` on a new initiative before the first cron run after a donation. Display as `0`. No spinner or special-case Ledger call needed.

**Migration cutover requirement**

The `amount-raised-sync` CronJob must be run once manually as part of the cutover procedure, before DNS switches to the new system. This pre-populates `amount_raised_in_cents` for all 1,374 migrated published initiatives so no card incorrectly shows `$0 raised` on day one. See migration plan Phase 4.

**Open question (OQ-15): ID for post-cutover initiatives**

New initiatives created after cutover have no DynamoDB origin and therefore no original string ID. See OQ-15 in `03-open-questions.md`.

**Migration path (Plan A only)**

If Plan A ships: `ledger-sync`, the `initiative_funding_summary` materialized view, and the removal of `amount-raised-sync` are all part of the initial release — no follow-on migration needed.

If Plan B ships instead and Plan A is later approved: once `ledger-sync` is running and `crowdfunding.ledger_transactions` is populated:
- Add the `initiative_funding_summary` materialized view as a new migration
- Remove the `amount_raised_in_cents` cached column
- Remove the `amount-raised-sync` / `ledger-stats-sync` CronJob

No API contract changes required — the cached column and the view return the same value for `amount_raised_in_cents`. Additional fields (backer count, subscription totals) become available via the view without schema changes to the API response.

### No ORM — sqlc for type-safe queries

Use `sqlc` to generate type-safe Go code from SQL queries. No ORM (GORM, ent, etc.).

Rationale: the existing codebase uses raw AWS SDK calls with no ORM. `sqlc` gives compile-time query safety without the complexity and magic of an ORM. Consistent with the team's preference for explicit code.

### GitHub stats — lazy refresh, no CronJob

GitHub stats (forks, stars, open issues) are display-only metrics on project cards. There are ~82 published `project`-type initiatives. These stats have no financial or operational consequence if slightly stale.

When the Go API serves a project detail response, it checks `github_stats->>'fetched_at'` against a 6-hour TTL. If stale (or absent), it fetches fresh data from the GitHub API, writes it back to `github_stats`, and returns it. If the GitHub API is unavailable, it returns the cached value without error.

Rationale: a dedicated CronJob binary, Dockerfile, and K8s CronJob manifest for 82 GitHub API calls every 6 hours is disproportionate overhead. Lazy refresh is simpler, requires no separate deployment artifact, and produces identical staleness characteristics (6h TTL either way). The only tradeoff is ~200ms added latency for the first page load after TTL expiry — acceptable for display-only vanity metrics.

### `migrate_dynamo_to_postgres.py` — migration tool

The one-time DynamoDB → Postgres migration is implemented as a Python script (`db/scripts/migrate_dynamo_to_postgres.py`), not a Go CLI. Migration status: **complete**.

Rationale: a Python script with `boto3` and `psycopg2` is straightforward to write, run, and audit for a one-shot migration. The script is idempotent (upsert-based) and fully disposable after cutover is validated.

### Deterministic UUID5 for surrogate PKs — stable across re-runs

All surrogate PKs are generated deterministically via `uuid5(UUID_NS, "{scope}:{natural_key}")` so re-runs produce the same IDs and FK relationships remain stable. The namespace UUID `6ba7b810-9dad-11d1-80b4-00c04fd430c8` must never change.

Rationale: deterministic UUIDs mean the migration can be re-run against a fresh DB and produce identical rows — no orphaned FKs, no need to persist a separate mapping table. Subscriptions and donations that reference old DynamoDB string IDs (`projectId`, `entityId`) resolve to the correct Postgres UUID by recomputing the same `uuid5` of the old ID during that phase of the migration.

### golang-migrate for schema migrations

Use `golang-migrate` for database schema migrations.

Rationale: simpler than Atlas, battle-tested, widely used in LFX ecosystem. Atlas's schema diffing is valuable but unnecessary complexity for this project.

---

## Domain Model

### `initiatives` table — unified initiative type

All fundable things live in one `crowdfunding.initiatives` table with `initiative_type` as the discriminator.

`initiative_type` values:
- `project` — created by users via the CF web UI; GitHub-linked; has budget categories; owned and editable by CF
- `mentorship` — created and managed by the Mentorship service; synced into CF via Snowflake CronJob; 91% of published rows; partially read-only in CF UI
- `general fund` — fundraising fund (formerly `initiative` and `general-fund` DynamoDB entity types; note: `other`/travel-fund type rows are stored as `other`, not merged into `general fund`)
- `event` — calendar/meetup-based fund
- `ostif` — security audit fund

`initiative_type` is set on creation and never changed.

### `jobspring_project_id` — canonical Mentorship link

The column is named `jobspring_project_id` (preserving the DynamoDB field name `jobspringProjectId`). This is the string ID used by the Mentorship service to identify a program. It is:
- Populated during migration from the `jobspringProjectId` DynamoDB attribute on existing rows
- Used by the `mentorship-sync` CronJob to match Snowflake program records to existing Postgres rows (upsert key)
- Used as the original DynamoDB string ID for Ledger API calls (same value — see migration notes)
- Never exposed in the public API response
- NULL on all non-mentorship initiatives

### Mentorship initiatives — partially editable in CF UI

Current system: fully editable — no restrictions on any field. This is a latent bug; Mentorship sync overwrites CF edits silently.

New system: field ownership split enforced at the API layer (`PATCH /v1/me/initiatives/:id` rejects Mentorship-owned fields when `initiative_type = mentorship`).

| Field group | Owner | Editable via CF UI |
|---|---|---|
| `name` | Mentorship | No — set by `mentorship-sync`, read-only in CF |
| `status` | Mentorship | No — controlled by `mentorship-sync` (mirrors Mentorship status) |
| `jobspring_project_id` | Mentorship | No — internal, never exposed |
| Skills, terms, mentors, custom term (in `initiative_goals` + mentor/skills/terms tables) | Mentorship | No |
| `logo_url`, `color`, `description`, `website` | CF | Yes |
| Budget goal amounts (per category) | CF | Yes |
| `beneficiaries` | Shared | Yes — CF manages who can draw funds |

### Type consolidations (applied during migration)

| Old DynamoDB type | New `initiative_type` | Rationale |
|---|---|---|
| `project` (with `jobspringProjectID`) | `mentorship` | Explicit type; was previously inferred from field presence |
| `project` (without `jobspringProjectID`) | `project` | Unchanged |
| `initiative` | `general fund` | UI already labeled these "General Fund"; backend type was an alias |
| `general-fund` | `general fund` | Normalize hyphen, same concept |
| `other` | `other` | Unchanged — 26 entity rows with DynamoDB `entityType = 'other'` stored as-is |
| `event` | `event` | Unchanged |
| `ostif` | `ostif` | Unchanged |
| `community` | `community` | Unchanged — 3 rows from 2019 migrated as-is; no active UI, no new rows expected |

### `general fund` vs `initiative` (old type) — same thing

In the old system: `initiative` was the backend type string; "General Fund" was the UI label; the subscription service explicitly mapped `'general fund'` → `ExpenseCategory.INITIATIVE`. They were always the same concept. The new schema uses `general fund` everywhere and drops the alias. Note: `initiative` as an `initiative_type` value does NOT exist — the table is named `initiatives` but the active type values are `project`, `mentorship`, `general fund`, `event`, `ostif`. Legacy migrated rows also carry `other` (26) and `community` (3).

### Budget goals — `initiative_goals` child table, mentorship data in dedicated tables

All initiative budget goals are stored in the `initiative_goals` child table (one row per category per initiative). Mentorship-specific metadata lives in four additional dedicated tables: `initiative_mentors`, `initiative_program_info_skills`, `initiative_program_info_terms`, `initiative_program_info_config`, and `initiative_program_info_custom_term`.

**Project-type initiatives** have up to 8 `initiative_goals` rows with fixed names: `development`, `marketing`, `meetups`, `travel`, `bugBounty`, `documentation`, `other`, `mentee`.

**Mentorship-type initiatives** have one `initiative_goals` row with `name = 'mentee'` carrying the `amount_in_cents` goal, plus rows in the mentor/skills/terms tables for program metadata.

**Migration implication:** the migration script reads `data.projectDetails.mentee` (nested) for mentorship projects — NOT `data.mentee` at the top level. Reading from the wrong path silently drops all mentorship metadata. This was the bug that caused the first migration pass to miss all 1,486 reclassified rows.

### `status` normalization

DynamoDB has dirty status values: `'hide'` (13 rows) and `'hidden'` (1 row) coexist. Normalize during migration:
- `'hide'` → `'hidden'`

Production status values observed: `submitted`, `published`, `declined`, `hidden`. The schema column is `VARCHAR(50)` with no CHECK constraint.

---

## API Design

### No OpenAPI spec for initial release

We will NOT write an OpenAPI spec before implementing the Go handlers.

Rationale: the API surface is well-understood from the existing system inventory. Writing a full OpenAPI spec first adds overhead without clear benefit at this stage. If the Nuxt frontend team and Go backend team are the same team (or closely coordinated), shared TypeScript types generated from Go structs are sufficient. Revisit if the team structure changes or if external consumers need a formal contract.

Update: if this decision is reversed, use `oapi-codegen` (Go server stubs) and `openapi-typescript` (Nuxt client types).

### API version stays `/v1/`

No versioning changes. New endpoints keep `/v1/` prefix to preserve compatibility with any existing consumers.

### Same URL as current system

- Dev: `https://funding.dev.platform.linuxfoundation.org/`
- Prod: `https://crowdfunding.lfx.linuxfoundation.org/`

DNS cutover at go-live. Old and new systems must not run concurrently on the same URL. Cutover plan: switch DNS/ingress from Lambda API Gateway to K8s ingress. Rollback: switch back.

---

## Backend

### Architecture — separate Go API, not embedded in Nuxt

The backend is a **standalone Go HTTP service**, separate from the Nuxt frontend. Business logic (DB queries, Stripe, webhooks, email) lives in Go. Nuxt's server layer handles auth (PKCE, HTTP-only cookies, session) and calls the Go API to build pages.

This was explicitly considered and rejected: embedding all backend logic inside Nuxt's server routes (as suggested during architecture review) would couple the UI framework to payment processing, Stripe webhook handling, and LFX One integration in ways that are fragile and hard to reverse. Specifically:

- **Stripe webhooks** must be handled by a stable, dedicated HTTPS endpoint — not a Nuxt server route subject to SSR configuration changes
- **LFX One integration** — LFX One's Express BFF may call the CF API directly (see LFX One integration decision); this requires an independently addressable service, not a Nuxt route
- **Go is the team's existing language** — the codebase, migration tooling, and team knowledge are all Go; a TypeScript rewrite of business logic adds risk with no benefit

Nuxt is the BFF for the CF frontend. The Go API is the resource service for all transactional logic.

### Go — same language, same patterns

New backend is Go, same DDD pattern as LFF (domain/, usecases/, interfaces/repository/). Not a framework change.

### Kubernetes, not Lambda

Deployed as a long-running Go HTTP service on Kubernetes, not Lambda. Background jobs become Kubernetes CronJobs (not CloudWatch Events).

### Background jobs — monorepo, separate binaries, separate container images

All code lives in one repository (monorepo). Each entrypoint under `cmd/` builds to a **separate container image** via its own Dockerfile:

| Entrypoint | Dockerfile | K8s resource |
|---|---|---|
| `cmd/api/` | `Dockerfile.api` | `Deployment` |
| `cmd/mentorship-sync/` | `Dockerfile.mentorship-sync` | `CronJob` |
| `cmd/amount-raised-sync/` | `Dockerfile.amount-raised-sync` | `CronJob` |
| `cmd/migrate/` | `Dockerfile.migrate` | one-off `Job` |

Rationale: a single container serving both HTTP requests and being invoked as a CronJob (via a flag) conflates two distinct runtime responsibilities. Separate images are minimal, contain only the code they need, and make it obvious what each K8s resource is doing. Shared business logic in `internal/` is compiled into each binary — no duplication of source, no runtime coupling.

### Chi router (same as today)

Keep Chi. No reason to change.

### HTTP caching — public endpoints use `Cache-Control: public`

Reviewed with Eric Searcy (chief architect, May 2026). Any Go API endpoint that serves public data (i.e. the response is identical regardless of whether the caller is authenticated) must return `Cache-Control: public, max-age=<N>` with no `Vary: Cookie`. This lets CDN edge nodes, corporate forward proxies, and browsers all cache and reuse the response without hitting the origin.

**Rules:**

- **Public endpoints** (anonymous-safe responses, same content for all callers): `Cache-Control: public, max-age=<N>`. Do NOT include `Vary: Cookie`. Include an `ETag` header (hash of the response body) so clients and CDN pops can do conditional `If-None-Match` revalidation and receive 304s instead of re-fetching the full payload.
- **Private / authenticated endpoints** (response varies by user identity or contains personal data): `Cache-Control: private, max-age=<N>`. Do not expose to CDN caches. Browser caching is acceptable — `private` ensures the response is only stored by the end-user's browser, not shared caches. Use `must-revalidate` if stale responses must never be served.
- **Unauthenticated view of a public page that also has an authenticated view**: serve `Vary: Cookie` so CDN does not re-serve an anonymous-user response to a logged-in user with a different cookie.

**What qualifies as public:** initiative listing (`GET /v1/initiatives`), initiative detail (`GET /v1/initiatives/{id}`), backer list, organization detail — any read endpoint that does not expose per-user data.

**ETag implementation:** compute `ETag` as the hex-encoded MD5 or xxHash of the JSON response body. Chi middleware is the right place to intercept the response writer, capture the body, hash it, and set the header. Return 304 if the request `If-None-Match` matches.

**`stale-while-revalidate`:** For high-traffic list endpoints (e.g. the initiative discovery page), set `Cache-Control: public, max-age=60, stale-while-revalidate=300`. The CDN serves the cached copy immediately and asynchronously re-fetches in the background — users are never blocked on origin latency.

**ValKey / in-app caching:** Not used for the initial release. HTTP caching at the CDN layer provides equivalent or better benefits without introducing a second cache TTL to manage. Revisit only if origin load data (Datadog) shows a specific bottleneck that HTTP caching cannot address.

**Why this matters for CF specifically:** initiative listing is accessible to unauthenticated users (search engines, anonymous visitors, link previews). Without public caching, every card scroll triggers an origin hit. With `Cache-Control: public` + `stale-while-revalidate`, the CDN serves most traffic — origin only sees revalidation requests, not full payloads.

### Ledger integration → keep calling Ledger HTTP API

The new Go service calls the Ledger HTTP API (read-only GET calls) exactly as LFF does today. LFF has never written to Ledger directly — Ledger gets its data from its own Stripe/Expensify webhooks. No change to this contract.

**Ledger calls CF HTTP API for notification emails — three endpoints, auth header, legacy ID:**
Ledger's `SendNotifications()` function calls the CF API to resolve project name, user name, and org name for donation confirmation emails. It calls three endpoints using the `project_id` / `user_id` / `organization_id` values stored in the Ledger DB:

- `GET /v1/projects/{id}` — resolves project name and owner details
- `GET /v1/entities/{id}` — fallback if project lookup returns empty
- `GET /v1/organizations/{id}` — resolves org name for org/invoice donations (called unauthenticated — no auth header sent; must remain a public endpoint)

After DNS cutover, these requests hit the new CF Go API. If any of these lookups fail, donation confirmation emails silently fail (Ledger logs a Slack error but posts the transaction anyway).

**Auth header: `x-ledger-auth` → must be changed to `Authorization: Bearer` before cutover**

Ledger authenticates its CF API calls with a custom `x-ledger-auth` header (`fundspring.go:102`), sending the raw value of `LEDGER_AUTHORIZATION_TOKEN` (with the `Bearer ` prefix stripped). The old LFF Lambda read this from the API Gateway event — a Lambda/API Gateway artifact, not a design decision.

The new CF Go API must **not** implement `x-ledger-auth` support. Instead, the Ledger Service must be updated to send a standard `Authorization: Bearer <token>` header before CF cutover. This is a one-line change in `fundspring.go:getToken()` and `request.Header.Set(...)`. Rationale: accepting a non-standard auth header creates a confusing third auth mechanism alongside Auth0 Bearer and `X-Internal-Token` (RS). `Authorization: Bearer` is the HTTP standard and is handled correctly by every proxy, middleware, and security scanner.

The CF Go API accepts `Authorization: Bearer` on the project/entity detail endpoints and validates the token against a shared secret (`CF_LEDGER_AUTH_TOKEN` env var, same value as Ledger's `LEDGER_AUTHORIZATION_TOKEN`). This is a service-to-service shared secret, not a JWT — validated with a direct string comparison in a dedicated middleware applied only to these endpoints.

**`GET /v1/organizations/{id}` — Ledger bug: called unauthenticated**

Ledger's `GetOrganizationName()` (`fundspring.go:63`) uses plain `http.Get()` with no auth header — an oversight; `GetProject()` on the same file sends `Authorization: Bearer`. The fix is the same one-line change: use `http.Client` and `request.Header.Set("Authorization", "Bearer "+getToken())`. This should be fixed in the same Ledger PR that changes `x-ledger-auth` → `Authorization: Bearer`. The CF endpoint should require the same `Authorization: Bearer` header as the project/entity detail endpoints — no special public treatment needed.

**Legacy ID lookups on project/entity endpoints:**
The `project_id` stored in the Ledger DB is the old DynamoDB string ID. The new CF Go API must accept both the original DynamoDB string ID and the Postgres UUID on `GET /v1/projects/{id}` and `GET /v1/entities/{id}`, resolving via the DynamoDB string ID first. This covers both existing rows (keyed by original ID) and post-cutover rows (keyed by UUID — see OQ-15).

**Stripe metadata must use the correct project ID for new donations after cutover:**
See OQ-15. The ID placed in Stripe object metadata fields `projectID` / `entityID` at charge-creation time determines what `project_id` gets written to the Ledger DB and what key `GET /balance/{id}` must use. This is unresolved for post-cutover initiatives and is the subject of OQ-15.

This is an implementation constraint on the new CF Go API Stripe integration — must be enforced at code review.

### Mentorship sync — Snowflake pull, not SNS/SQS

CF syncs Mentorship program data from Snowflake via a periodic K8s CronJob (`mentorship-sync`). SNS/SQS is not used in the new system.

Rationale: Mentorship and CF run in separate AWS accounts, making cross-account SNS/SQS subscription complex and requiring Mentorship team involvement. Mentorship is also moving to Kubernetes in the coming months, making Lambda-era SQS infrastructure a poor long-term investment. Both services already mirror data into Snowflake. A 24h sync delay is acceptable — new mentorship programs are not immediately donation-ready, and beneficiaries don't access funds until mid-term (months after program creation).

The `mentorship-sync` CronJob:
- Queries Snowflake for mentorship programs and their approved beneficiaries
- Creates `initiative_type = mentorship` project rows in CF Postgres for new programs
- Updates Mentorship-owned fields (name, status, mentee goal row in `initiative_goals`) for existing rows
- Syncs approved beneficiary list onto each project record
- Normalizes `'hide'` → `'hidden'` on status

**There are no direct HTTP calls between Mentorship and CF.** All data flows through Snowflake. This is a clean separation — Mentorship owns program and beneficiary data; CF reads it from Snowflake on a scheduled basis.

**Why CF keeps beneficiary data despite the Snowflake sync:**
CF is the financial custodian of donated funds. It collects money from donors and must maintain visibility into who is approved to draw those funds via Expensify. Beneficiary records in CF serve two purposes:
1. **Financial governance** — CF can reconcile money collected against approved disbursement recipients
2. **Reimbursement Service** — when CF adds or removes a beneficiary, it writes an action to the RS `beneficiary-actions` OpenSearch queue; RS processes that queue to manage Expensify policies. RS cannot reach CF Postgres directly (separate VPC/account).

CF does not use beneficiary data for payment routing (Stripe charges are donor→program, not donor→beneficiary). The 24h sync delay is acceptable — mentees do not access funds until mid-term, months after approval.

A 24h delay on beneficiary sync is acceptable by the same logic as program sync — mentees don't draw funds until mid-term, months after being approved.

### Mentorship UI — "available funds" display removed

The Mentorship UI currently shows an "available funds" balance for each mentorship program, sourced from the CF API (which proxies Ledger). This display is removed and not carried into any future Mentorship rewrite.

Rationale: CF is the authoritative UI for financial data. Duplicating the balance in Mentorship blurs the product boundary and creates an integration dependency with no clear owner. Users who need the funding balance (finance team, CF admins, donors) are already in CF. Mentees care about receiving their stipend — handled downstream by Expensify/NetSuite — not the raw balance figure.

When Mentorship moves to Kubernetes and gets a new UI design, a "View funding on LFX Crowdfunding" link on the program page is sufficient to surface the information without an integration dependency.

### CF → Snowflake sync via Fivetran

CF Postgres must be synced to Snowflake via Fivetran so that CF data (projects, funds, donations, subscriptions, organizations) is available for analytics and reporting alongside data from other LFX products.

This is **required for the initial release** — not "shortly after". A Jira ticket to configure the Fivetran connector must be created once the CF DB is up and populated with production data. Owner: DevOps. This is a release-blocking task.

Note: the `mentorship-sync` CronJob reads **Mentorship data from Snowflake into CF** — this is the Mentorship team's Fivetran responsibility, not CF's. CF→Snowflake is a separate Fivetran connector that CF DevOps owns.

### LFX One integration — Snowflake (Option B) for initial release

The PM has requested CF data surfaces in LFX One ("My Donations", "My Initiatives", and potentially more — full list TBD, see OQ-11). For the initial release, LFX One will read CF data from **Snowflake** (the same pattern used for My Trainings, My Meetings, etc.).

Rationale: the Snowflake pattern already exists in LFX One and requires minimal new code. The 24h Fivetran sync delay is acceptable for summary widgets — real-time payment confirmation is handled on `crowdfunding.lfx.linuxfoundation.org`, not in LFX One. This approach has no runtime dependency between LFX One and the CF API service.

Option A (LFX One calls CF Go API directly) was raised during the architecture review (Eric Searcy, May 2026) as a valid alternative once design is confirmed. This decision will be revisited once the LFX One UI design for CF widgets is delivered — see OQ-11.

**No LFX One integration code will be written until:**
1. The PM provides a UI design for the LFX One CF widgets (what data, what layout)
2. The full list of CF data needed in LFX One is confirmed (see OQ-11)
3. The Fivetran CF→Snowflake connector is live with production data

**Future path (Option C):** Full platform stack integration (Traefik ingress, OpenFGA authorization, platform indexer, Query Service) is the long-term correct architecture for CF as an LFX v2 citizen. This is deferred post-initial-release and must be tracked as a separate Jira epic. It is not "later maybe" — it is a scheduled follow-on project.

### Expensify sync — keep on old Lambda for initial release

The `expensify-sync` cron job (SyncExpensifyHandler) pushes project/entity metadata to the Reimbursement Service. This is NOT end-user visible and NOT part of the initial release. The old Lambda continues running this job unchanged until the Reimbursement Service is migrated.

**Constraint for future expensify-sync port:** When expensify-sync is eventually ported to the new CF service, it must send the old DynamoDB string ID (`projectId`) as the project ID to RS — not the new Postgres UUID. RS stores the project ID in Expensify as a custom reporting field and uses it as the policy lookup key. The Reimbursement Service has two project IDs hardcoded in `chrisProjectList` (service.go:197) for special Expensify handling (Kubernetes and CoreDNS — these projects route expense approval through Chris Aniszczyk as an auditor). These are DynamoDB-era project UUIDs (`2d438b9a...` = Kubernetes, `6705be57...` = CoreDNS — confirmed via RS Postman collection). RS receives and stores these IDs from LFF at policy-creation time (`POST /reimbursement/{projectID}`), where `projectID` is the DynamoDB `project.ProjectID`. Sending the new Postgres UUID instead would silently break the `chrisProjectList` check and the Expensify approval chain for these two projects. The original DynamoDB ID is not recoverable from the Postgres `initiatives.id` UUID by inverting the deterministic `uuid5` mapping; if the new service needs to send the legacy ID after migration, that DynamoDB `projectId` must be preserved explicitly in a dedicated column or mapping table during migration.

### EMAIL_DRY_RUN mode

Add `EMAIL_DRY_RUN=true` environment variable. When set, email service logs the would-be email payload instead of calling Mandrill. Used when testing with production data to prevent accidental emails.

### LFX v2 platform stack — standalone for initial release

CF does **not** integrate with the LFX v2 platform stack (Traefik gateway, Heimdall, OpenFGA, platform indexer, Query Service) for the initial release. CF uses a standalone Kubernetes Ingress and handles auth internally via its own Auth0 JWT middleware — the same approach as `lfx-v2-ui` and `lfx-changelog`.

This was reviewed with Eric Searcy (chief architect, May 2026). His position: participating in the access control graph is architecturally correct long-term but will slow down delivery significantly. Given the end-of-May deadline, standalone is the right call for initial release.

**Tech debt created is bounded and intentional:**
- CF's own JWT middleware → replaceable with Heimdall when Option C is implemented
- CF's own Ingress → replaceable with Traefik HTTPRoute
- Postgres FTS for search → stays; Query Service is additive, not a replacement
- No OpenFGA types defined → must be designed and implemented as part of Option C

**To keep Option C achievable:** access control intent must be documented now — who can do what to which resource — even though it is not implemented via OpenFGA yet. This is captured in the OpenFGA design notes below.

**Access control intent (for future OpenFGA model):**
- `initiative` (project/mentorship/`general fund`/event/ostif): owner (creator) has writer; donors have no elevated access; CF admin has writer for approval flow
- `organization`: owner has writer; members have no elevated access beyond the owner (org membership is used for donation attribution only, not access control)
- `subscription` / `donation`: owned by the creating user; read-only to CF admin
- Anonymous users: read-only access to published initiatives

**Initiatives are decoupled from LFX project entities.** There is no FK or relationship linking a CF initiative to an LFX project in the permissions graph. Access is determined solely by `owner_id` (Auth0 subject) and `initiative_type`. Role inheritance from LFX project roles (e.g., project auditor → initiative writer) is not possible without adding a `project_id` FK — this is a design decision deferred to Option C.

Consequence for non-LF projects (future): since initiatives are decoupled, supporting non-LF projects is a per-initiative policy decision, not a structural schema change.

**Known divergence from LFX v2 API patterns:** LFX v2 resource APIs never serve collections — all list queries go through the Query Service (OpenSearch-backed, access-control-aware). CF's initial release serves collection endpoints directly from the Go API (e.g., `GET /v1/initiatives`, `GET /v1/me/subscriptions`). These endpoints must be redesigned through the Query Service when Option C is implemented.

**LFX One integration auth:** For the initial release, LFX One reads CF data from Snowflake (Option B) — there are no live API calls from LFX One to the CF Go API. The question of how LFX One authenticates to the CF API (ID tokens, API key, M2M, Auth0 resource server) only arises if Option A or C is chosen. Deferred to OQ-11.

**Option C** (full platform stack integration) is a post-initial-release tracked project — not "later maybe." File a Jira epic when initial release ships.

### Authorization model — initial release

The initial release uses the same authorization model as the current LFF system. There is no RBAC — authorization is purely ownership-based, enforced at the usecase layer.

**Three mechanisms:**

**1. Ownership check (initiative/org CRUD)**
Any authenticated user can create an initiative. The creator's Auth0 subject (`jwt.sub`) is stored as `owner_id` at creation time. All mutating operations check `initiative.owner_id == jwt.sub` inline in the usecase. There is no "owner role" stored anywhere — ownership is structural.

```go
// enforced inline in usecases, same pattern as current LFF
if initiative.OwnerID != currentUser.ID {
    return ErrNotAuthorized
}
```

**2. CF admin / initiative approver — `CF_APPROVERS` env var**
The person who approves or rejects initiative submissions is identified by their LFID, stored in a `CF_APPROVERS` environment variable (comma- or pipe-separated). This is not an Auth0 role — it is a config value injected at deploy time.

Production value: `shubhrakar` (Sriji). Confirmed as the approver for the new system.
Dev/staging value: `*` (any authenticated user can approve — allows testing).

Stored in AWS Secrets Manager, injected via ESO. The approval check at the API layer:
```go
authorized := strings.Contains(os.Getenv("CF_APPROVERS"), user.LFID)
```

**3. Email approval links — HMAC-signed token, no Auth0**
Initiative and expense approval email links use HMAC HS256-signed tokens (not Auth0). The token encodes `{ initiativeID, action: "approve"|"reject" }` and has an expiry. The `POST /v1/initiatives/approvals` endpoint verifies the HMAC signature — the signed token is the sole authorization mechanism for this flow. No Auth0 JWT is required or checked.

This is intentional: the approver clicks a link in email without needing to be logged in to CF. The HMAC secret is stored in AWS Secrets Manager (`CF_APPROVAL_SIGNING_SECRET`).

### Reimbursement and Ledger on Lambda — network path

Both services remain on Lambda (API Gateway endpoints). The new CF Go service on K8s calls them over public HTTPS (both are API Gateway endpoints). Network path confirmed reachable — see OQ-1 and OQ-2.

---

## Frontend

### Nuxt 3 + Vue 3

New UI is Nuxt 3 (latest) + Vue 3, TypeScript strict mode. Follows Insights repo patterns.

Rationale: this is the LFX platform standard for new frontends. Angular 15 is the current system but is not the target platform.

### Follow Insights repo architecture

Follow `linuxfoundation/insights` frontend for:
- Project structure (`app/`, `server/`, `composables/`, `types/`)
- Auth pattern: OAuth2 PKCE with HTTP-only cookies, server-side token exchange via `server/api/auth/`
- LFX Header: `@linuxfoundation/lfx-ui-core`, dynamic import client-only
- HTTP client: `$fetch` (ofetch) + Vue Query (TanStack) for server state caching
- State: Pinia for app state, Composition API for local state
- Styling: Tailwind CSS + CSS variables
- Environment: `useRuntimeConfig()`, `NUXT_` prefixed vars, server-only secrets

### PrimeVue as component library

Use PrimeVue (v4+) as the primary component library, with theme set to `none` and custom Tailwind styling applied. Do not build a custom UIKit from scratch.

Rationale: Insights builds its own UIKit (44 component families) which is appropriate for a large platform. Crowdfunding has a narrower scope — forms, cards, modals, tables — which PrimeVue covers adequately. Custom components only where PrimeVue doesn't fit the prototype design.

### No Datadog RUM, no Intercom (initial release)

Deferred. Can be added later as they are non-functional additions.

### Same LFX Header as Insights

`@linuxfoundation/lfx-ui-core` package, `<lfx-navbar />` web component, dynamic import on client only.

### UI design source

Prototype: `https://github.com/jonathimer/lfx-crowdfunding-prototype` — treat as the design reference for the initial release UI. Sponsor Tiers shown in prototype are out of scope for initial release.

### `/stripe` OAuth callback route — dropped (dead code)

The old CF Angular app had a `/stripe` route (`RedirectingModule`) with an empty `ngOnInit()`. It appeared to be a Stripe Connect OAuth callback but has no logic — the component does nothing. Investigation of the backend confirmed that `ConnectOAuthAccount()` only handles `github` as a provider type; there is no Stripe Connect OAuth flow in the current codebase.

The `/stripe/callback` route is **not ported** to the new Nuxt frontend. If Stripe Connect OAuth is ever needed in the future, it must be implemented from scratch. There is no existing implementation to migrate.

### `diversity` — budget category kept, Diversity API deferred

`diversity` appears in two distinct and unrelated contexts:

**1. `diversity` budget category (active, must be kept)**
`diversity` is a valid entry in `entity_goals.json` (displayed as "Diversity Funding") and is used as a payment/subscription category in the current frontend and backend. Donors can designate their contribution to the `diversity` budget. This category exists in production data on `subscriptions` and `donations` DynamoDB records.

`diversity` must be a valid `category` value on `crowdfunding.subscriptions` and `crowdfunding.donations` and must be migratable as an `initiative_goals` row — exactly as `development`, `marketing`, etc.

**2. Diversity API integration (deferred)**
The old system fetches demographic diversity stats (`malePercentage`, `femalePercentage`, etc.) from a separate Diversity API and displays them on the project page. This API integration is **deferred** — not in the initial release. It has nothing to do with the budget category above.

Do not conflate these two: the Diversity API deferred flag does not mean the `diversity` budget category is removed. Keep the category; defer the API.

---

## Migration

### Python script for one-time DynamoDB → Postgres migration

The migration is implemented as `db/scripts/migrate_dynamo_to_postgres.py` — a Python script using `boto3` and `psycopg2`. Migration is complete (status: exit 0, May 2026).

Rationale: `lfx-v1-sync-helper` is purpose-built for project/committee metadata sync between LFX v1 and v2 via NATS KV — wrong tool for this. The Python script is short (~600 lines), uses idempotent upserts, and is fully disposable after cutover is confirmed.

### All new components deployed to Kubernetes

Everything inside the "NEW" purple box in the architecture diagram is deployed to Kubernetes:
- Crowdfunding Nuxt frontend — K8s Deployment + Service + Ingress
- Crowdfunding Go API — K8s Deployment + Service + Ingress
- Crowdfunding Postgres DB — shared LFX v2 RDS instance; DevOps adds `crowdfunding` DB + role to `lfx-v2-opentofu`
- `mentorship-sync` — K8s CronJob (daily or a few times/day)
- `amount-raised-sync` — K8s CronJob (hourly)

Nothing in the initial release runs on Lambda or Serverless Framework.

### Database — shared AWS RDS instance (managed Postgres)

Crowdfunding Postgres runs on the shared LFX v2 RDS instance (`aws_db_instance.lfv_v2`, Postgres 17.4), defined in `linuxfoundation/lfx-v2-opentofu`. This is the LFX platform standard — every service (changelog, lens, member-onboarding, sanctions-screening, litellm, openfga) uses the same shared RDS instance with per-service databases and roles. No per-service RDS instance, no in-cluster StatefulSet.

To add CF: a `crowdfunding` entry is added to the `databases` map in `lfx-v2-opentofu/postgres.tf` — PR already open at `linuxfoundation/lfx-v2-opentofu#181`. Credentials are auto-rotated every 30 days via AWS Secrets Manager and injected by ESO.

From the app's perspective the connection string is `rds-postgres.lfx:5432` — an in-cluster ExternalName service with a socat proxy defined in `lfx-v2-opentofu/k8s-database-proxy.tf`.

### Secrets — External Secrets Operator + AWS Secrets Manager

All secrets (DB credentials, Auth0 client secret, Stripe keys, GitHub OAuth secret, JWT secret, Mandrill key, Snowflake credentials) are stored in AWS Secrets Manager and synced into K8s Secrets by the External Secrets Operator. The service account uses IRSA (IAM Roles for Service Accounts) to authenticate to AWS Secrets Manager. This is the LFX platform standard — confirmed by reviewing ESO config in `lfx-v2-argocd` for existing services.

AWS Secrets Manager path convention (following LFX pattern): `/cloudops/managed-secrets/crowdfunding/{env}/...` — confirm exact path with DevOps.

### Old Lambda stack runs in parallel during initial release

The old LFF Lambda + DynamoDB + OpenSearch stack continues running until:
1. New system is fully validated in production
2. RS Phase 1 internal endpoints are live on the CF Go API (RS switched off OpenSearch for CF-owned reads — see OQ-7)
3. DNS cutover is executed

Do not decommission the old stack before all three conditions are met. OpenSearch itself is NOT decommissioned at this point — RS still owns three live indices there (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`). OpenSearch decommission is deferred until RS moves to Kubernetes (timeline TBD, see OQ-7).
