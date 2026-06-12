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

---

## Database

### PostgreSQL, not DynamoDB

Target database is PostgreSQL. DynamoDB is decommissioned after successful migration and cutover.

### Schema separation — one schema, one Postgres instance

The CF Postgres instance has one schema:

- `crowdfunding` — owned by CF Go API (initiatives, orgs, subscriptions, donations, users)

RS continues using OpenSearch for its own three tables (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`) until it moves to Kubernetes. For the initial CF release, RS reads CF-owned data via three narrow internal HTTP endpoints on the CF Go API. See OQ-7 for the migration plan.

Ledger Service keeps its own separate Postgres DB (Ledger DB) in a separate AWS account. It is not changed in the initial release.

Ledger DB co-location and raw table mirroring (cross-account DB access) were both considered and rejected by the architect (Eric, May 2026) — see OQ-18. The confirmed approach is stats-sync via Ledger HTTP API.

**Confirmed approach:** a `ledger-stats-sync` K8s CronJob calls the Ledger HTTP API to sync pre-aggregated financial stats per initiative and stores them in the `initiative_ledger_stats` table in CF DB. This enables sorting and filtering by financial metrics (most funded, most supporters) at query time with a simple index scan. See the `ledger-stats-sync` section under Backend for full specification.

### `projects` + `entities` → unified `initiatives` table

The old `projects` and `entities` tables are merged into a single `crowdfunding.initiatives` table with an `initiative_type` discriminator column (`project` | `mentorship` | `general fund` | `event` | `ostif`; legacy: `other`, `community`). Type-specific sparse columns are nullable. See Domain Model section for full type definitions and field ownership rules.

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

Projects produce up to 8 rows (one per fixed category). Entities produce one row per element in `entity.Goals[]`. Mentorship-specific data (skills, terms, mentors, config, custom term) lives in the dedicated child tables: `initiative_mentors`, `initiative_program_info_skills`, `initiative_program_info_terms`, `initiative_program_info_config`, `initiative_program_info_custom_term`.

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

### Ledger-derived financial fields — `initiative_ledger_stats` table

Ledger-derived financial data is stored in the `initiative_ledger_stats` table (one row per initiative), kept fresh by the `ledger-stats-sync` CronJob. See the `ledger-stats-sync` section under Backend for the full specification.

The `initiative_stats` table that existed in an earlier schema version has been removed entirely. `initiative_ledger_stats` is its replacement — see the `initiative_stats` removal entry in this document.

The Stripe webhook handler (`POST /v1/hooks/stripe`) does **not** call the Ledger API — it handles only `customer.subscription.deleted`. Calling Ledger from the webhook would require a 5-second delay to avoid a race condition with Ledger's own webhook handler.

**Stripe webhook auth:** HMAC-SHA256 via `webhook.ConstructEvent(body, sig, endpointSecret)`, `STRIPE_WEBHOOK_SECRET` env var. Must not be protected by Auth0 JWT middleware — Stripe cannot send a Bearer token.

**Cutover:** run `ledger-stats-sync` once manually before DNS cutover to pre-populate `initiative_ledger_stats` for all migrated initiatives. Rows are absent before the first run — the `initiative_repository` LEFT JOINs `initiative_ledger_stats` and COALESCEs all financial values to `0`, so missing rows display cleanly as zero. See OQ-15 for post-cutover ID strategy.


### No ORM — raw pgx with explicit query functions

Use raw `pgx/v5` with explicit query functions. No ORM (GORM, ent, etc.), no code generator (sqlc).

Rationale: the existing codebase uses raw AWS SDK calls with no ORM. Raw pgx keeps SQL visible and explicit without tooling overhead. Consistent with the team's preference for explicit code.

### GitHub stats — lazy refresh, no CronJob

GitHub stats (forks, stars, open issues) are display-only metrics on project cards. There are ~82 published `project`-type initiatives. These stats have no financial or operational consequence if slightly stale.

When the Go API serves a project detail response, it checks `github_stats->>'fetched_at'` against a 6-hour TTL. If stale (or absent), it fetches fresh data from the GitHub API, writes it back to `github_stats`, and returns it. If the GitHub API is unavailable, it returns the cached value without error.

Rationale: a dedicated CronJob for ~82 projects is disproportionate overhead. Lazy refresh is simpler, requires no separate deployment artifact, and produces identical staleness characteristics.

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

Key non-obvious mappings: `project` rows with `jobspringProjectID` → `mentorship`; `initiative` and `general-fund` DynamoDB types → `general fund`; `other` entity rows (26) and `community` rows (3) stored as-is. See `db/scripts/migrate_dynamo_to_postgres.py` for the full mapping.

### Budget goals — `initiative_goals` child table, mentorship data in dedicated tables

All initiative budget goals are stored in the `initiative_goals` child table (one row per category per initiative). Mentorship-specific metadata lives in five additional dedicated tables: `initiative_mentors`, `initiative_program_info_skills`, `initiative_program_info_terms`, `initiative_program_info_config`, and `initiative_program_info_custom_term`.

**Project-type initiatives** have up to 8 `initiative_goals` rows with fixed names: `development`, `marketing`, `meetups`, `travel`, `bugBounty`, `documentation`, `other`, `mentee`.

**Mentorship-type initiatives** have one `initiative_goals` row with `name = 'mentee'` carrying the `amount_in_cents` goal, plus rows in the mentor/skills/terms tables for program metadata.

---

## API Design

### No OpenAPI spec for initial release

We will NOT write an OpenAPI spec before implementing the Go handlers.

Rationale: the API surface is well-understood from the existing system inventory. Writing a full OpenAPI spec first adds overhead without clear benefit at this stage. If the Nuxt frontend team and Go backend team are the same team (or closely coordinated), shared TypeScript types generated from Go structs are sufficient. Revisit if the team structure changes or if external consumers need a formal contract.

Update: if this decision is reversed, use `oapi-codegen` (Go server stubs) and `openapi-typescript` (Nuxt client types).

### API version stays `/v1/`

No versioning changes. New endpoints keep `/v1/` prefix to preserve compatibility with any existing consumers.

### URLs

- Dev: `https://crowdfunding.dev.lfx.dev/`
- Staging: `https://crowdfunding.staging.lfx.dev/`
- Prod: `https://crowdfunding.linuxfoundation.org/`

The rewrite uses new hostnames (not the legacy `funding.dev.platform.linuxfoundation.org` / `crowdfunding.lfx.linuxfoundation.org`). DNS for the old hostnames can be decommissioned separately after the legacy Lambda stack is retired.

---

## Post-Release Cleanup Convention

Some code in the initial release is intentionally temporary — legacy shims, compatibility wrappers, or features kept alive only until a dependent service migrates. To make the post-release cleanup tractable, all such code must be tagged at the point of writing using a standard `TODO(<label>)` comment so it can be found with a single `grep`.

### Tag convention

```go
// TODO(<label>): <one-line reason>
// <what to do when the trigger is met and any context needed to act on it>
```

Tags go on the first line of the relevant handler, function, or block — not on a line inside the implementation. This makes `grep -rn 'TODO(label)'` return exactly the entry points, not noise.

### Labels in use

| Label | Trigger for removal | What to remove |
|---|---|---|
| `TODO(ledger-k8s)` | Ledger Service migrated to Kubernetes and updated to call `/v1/initiatives/{id}` | `GET /v1/projects/{id}`, `GET /v1/entities/{id}`, `GET /v1/organizations/{id}` legacy shim handlers and their middleware |
| `TODO(rs-k8s)` | Reimbursement Service migrated to Kubernetes | Internal RS-facing HTTP endpoints, any RS-specific auth middleware, OpenSearch queue writes |
| `TODO(post-cutover)` | Old LFF Lambda stack decommissioned | DynamoDB string ID → UUID resolution fallback in any endpoint that accepts legacy IDs; `source_dynamo_table` column and any code that reads it |

Add new labels to this table when new temporary code is introduced. Each label must have a named trigger (a concrete event, not "someday") and a named owner or Jira epic if one exists.

### Finding all cleanup points

```bash
grep -rn 'TODO(' cmd/ internal/ | grep -E 'TODO\((ledger-k8s|rs-k8s|post-cutover)\)'
```

Run this after each follow-on milestone to find what can now be deleted.

## Backend

### Architecture — separate Go API, not embedded in Nuxt

The backend is a **standalone Go HTTP service**, separate from the Nuxt frontend. Business logic (DB queries, Stripe, webhooks, email) lives in Go. Nuxt's server layer handles auth (PKCE, HTTP-only cookies, session) and calls the Go API to build pages.

Embedding business logic in Nuxt server routes was considered and rejected — Stripe webhooks require a stable dedicated endpoint, LFX Self Serve may call the CF API directly, and Go is the team's existing language. Nuxt is the BFF for the CF frontend; the Go API is the resource service for all transactional logic.

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
| `cmd/ledger-stats-sync/` | `Dockerfile.ledger-stats-sync` | `CronJob` |
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

**Donation and subscription acknowledgement emails are sent by Ledger, not CF.**
This was migrated in FUND-1055. LFF kept stub implementations of `SendDonationSubscriptionEmail`, `SendDonationNoticeForOwner`, etc. that return `nil` immediately — they exist only to avoid changing callers. The new CF service must not implement these methods at all. CF's email responsibilities are: initiative/entity approval and rejection, expense approval, invoice notifications to the internal CB team, security audit submission confirmations, and GitHub Connect confirmations. Everything donation/subscription-acknowledgement-related is Ledger's job.

**Ledger calls CF HTTP API for notification emails — three endpoints, auth header, legacy ID:**
Ledger's `SendNotifications()` function calls the CF API to resolve project name, user name, and org name for donation confirmation emails. It calls three endpoints using the `project_id` / `user_id` / `organization_id` values stored in the Ledger DB:

- `GET /v1/projects/{id}` — resolves project name and owner details
- `GET /v1/entities/{id}` — fallback if project lookup returns empty
- `GET /v1/organizations/{id}` — resolves org name for org/invoice donations (called unauthenticated — no auth header sent; must remain a public endpoint)

These three routes are **Ledger-only legacy shims** — they exist solely because the Ledger Service has these URLs hardcoded and cannot be changed without a Ledger PR. They are not part of the public CF API and should not be used by any other caller. When the Ledger Service is migrated to Kubernetes, it must be updated to call `GET /v1/initiatives/{id}` instead, and these three routes must be removed.

Mark each handler in the Go code with:

```go
// TODO(ledger-k8s): remove once Ledger Service is migrated to Kubernetes.
// Ledger calls this path from fundspring.go to resolve initiative name/owner for notification emails.
// Replace with GET /v1/initiatives/{id} in the Ledger Service, then delete this handler.
```

Search for `TODO(ledger-k8s)` to find all removal points.

After DNS cutover, these requests hit the new CF Go API. If any of these lookups fail, donation confirmation emails silently fail (Ledger logs a Slack error but posts the transaction anyway).

**Auth header: `x-ledger-auth` → must be changed to `Authorization: Bearer` before cutover**

Ledger authenticates its CF API calls with a custom `x-ledger-auth` header (`fundspring.go:102`), sending the raw value of `LEDGER_AUTHORIZATION_TOKEN` (with the `Bearer ` prefix stripped). The old LFF Lambda read this from the API Gateway event — a Lambda/API Gateway artifact, not a design decision.

The new CF Go API must **not** implement `x-ledger-auth` support. Instead, the Ledger Service must be updated to send a standard `Authorization: Bearer <token>` header before CF cutover. This is a one-line change in `fundspring.go:getToken()` and `request.Header.Set(...)`. Rationale: accepting a non-standard auth header creates a confusing third auth mechanism alongside the Auth0 Bearer tokens used by all other callers. `Authorization: Bearer` is the HTTP standard and is handled correctly by every proxy, middleware, and security scanner.

The CF Go API accepts `Authorization: Bearer` on the project/entity detail endpoints and validates the token against a shared secret (`CF_LEDGER_AUTH_TOKEN` env var, same value as Ledger's `LEDGER_AUTHORIZATION_TOKEN`). This is a service-to-service shared secret, not a JWT — validated with a constant-time comparison (e.g. `hmac.Equal`) in a dedicated middleware applied only to these endpoints. Do not use `==` or `strings.EqualFold` — both are vulnerable to timing attacks.

**`GET /v1/organizations/{id}` — Ledger bug: called unauthenticated**

Ledger's `GetOrganizationName()` (`fundspring.go:63`) uses plain `http.Get()` with no auth header — an oversight; `GetProject()` on the same file sends `Authorization: Bearer`. The fix is the same one-line change: use `http.Client` and `request.Header.Set("Authorization", "Bearer "+getToken())`. This should be fixed in the same Ledger PR that changes `x-ledger-auth` → `Authorization: Bearer`. The CF endpoint should require the same `Authorization: Bearer` header as the project/entity detail endpoints — no special public treatment needed.

**Legacy ID lookups on project/entity endpoints:**
The `project_id` stored in the Ledger DB is the old DynamoDB string ID. The new CF Go API must accept both the original DynamoDB string ID and the Postgres UUID on `GET /v1/projects/{id}` and `GET /v1/entities/{id}`, resolving via the DynamoDB string ID first. This covers both existing rows (keyed by original ID) and post-cutover rows (keyed by UUID — see OQ-15).

**Stripe metadata must use the correct project ID for new donations after cutover:**
The ID placed in Stripe object metadata fields `projectID` / `entityID` at charge-creation time determines what `project_id` gets written to the Ledger DB and what key `GET /balance/{id}` must use.

For post-cutover initiatives (no DynamoDB origin), the recommended approach is to use the Postgres UUID directly as the project ID: CF puts the UUID in Stripe metadata at charge-creation time; Ledger stores it verbatim; `GET /balance/{uuid}` finds it because Ledger's regex (`^[0-9a-zA-Z\_\-]+$`) accepts UUIDs and the `WHERE project_id = $1` query matches. No Ledger code changes required. Lewis must confirm no Ledger code path assumes `project_id` is in a non-UUID format before this is adopted — see OQ-15.

This is an implementation constraint on the new CF Go API Stripe integration — must be enforced at code review.

### `ledger-stats-sync` CronJob — specification

**What it is:** a standalone Go binary at `cmd/ledger-stats-sync/`, deployed as a K8s CronJob. It runs on a schedule, pulls balance data from the Ledger HTTP API, and upserts rows into `initiative_ledger_stats` in CF Postgres. It has no HTTP server — it runs, syncs, logs a summary, and exits.

**Schedule:** hourly. Stats are at most 1 hour stale, acceptable for funding totals displayed in the UI.

**Algorithm:**

1. **Load initiatives to sync** — query CF DB for all initiatives where `status NOT IN ('archived', 'draft')`. Sync all active initiatives (not just published) so that when an initiative is published its stats are already populated and don't show as zero on first load.

2. **Fetch all balances from Ledger** — call `GET /balance` (bulk endpoint) once. This returns all project balances in a single HTTP response regardless of initiative count (~2,000 rows). Do not call `GET /balance/{id}` per initiative in a loop.

3. **Build a lookup map** — index the Ledger response by `projectID` so each CF initiative ID resolves in O(1).

4. **Upsert into `initiative_ledger_stats`** — for each initiative that has a matching Ledger entry, run:
   ```sql
   INSERT INTO initiative_ledger_stats
     (initiative_id, total_raised_cents, total_debited_cents,
      total_balance_cents, available_balance_cents, fee_balance_cents,
      supporters, updated_on)
   VALUES (...)
   ON CONFLICT (initiative_id) DO UPDATE SET
     total_raised_cents      = EXCLUDED.total_raised_cents,
     total_debited_cents     = EXCLUDED.total_debited_cents,
     total_balance_cents     = EXCLUDED.total_balance_cents,
     available_balance_cents = EXCLUDED.available_balance_cents,
     fee_balance_cents       = EXCLUDED.fee_balance_cents,
     supporters              = EXCLUDED.supporters,
     updated_on              = NOW()
   ```

   Column mapping from the Ledger `Balance` struct:
   | `initiative_ledger_stats` column | Ledger field | Notes |
   |---|---|---|
   | `total_raised_cents` | `totalCredit` | always positive |
   | `total_debited_cents` | `ABS(totalDebit)` | Ledger stores debits as negative integers |
   | `total_balance_cents` | `totalBalance` | |
   | `available_balance_cents` | `availableBalance` | stored as-is; Ledger computes this as `totalBalance + feeBalance` (feeBalance is negative) |
   | `fee_balance_cents` | `ABS(feeBalance)` | Ledger stores fee balance as negative |
   | `supporters` | `supporters` | count of distinct user IDs with `amount > 0` |

5. **Skip initiatives with no Ledger entry** — new initiatives not yet in the Ledger are silently skipped. Their `initiative_ledger_stats` row either doesn't exist yet (COALESCE to 0 in queries) or retains the last known values from a previous sync.

6. **Log a sync summary** — at completion, log: total initiatives in CF DB, how many had a Ledger match, how many were upserted, how many were skipped, and total wall-clock duration.

7. **Exit cleanly** — exit 0 on success, non-zero on any error. K8s uses the exit code to determine CronJob success or failure and will alert/retry accordingly.

**ID mapping constraint (OQ-15):** The Ledger `projectID` field must match `initiatives.id` (UUID) in CF DB for the lookup to work. This must be validated before the CronJob goes live — see OQ-15.

**Skeleton:** `cmd/ledger-stats-sync/main.go` exists in the repo as a documented skeleton for Lewis to implement.

---

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

### Mentorship UI — "available funds" display removed

The Mentorship UI currently shows an "available funds" balance for each mentorship program, sourced from the CF API (which proxies Ledger). This display is removed and not carried into any future Mentorship rewrite.

Rationale: CF is the authoritative UI for financial data. Duplicating the balance in Mentorship blurs the product boundary and creates an integration dependency with no clear owner. Users who need the funding balance (finance team, CF admins, donors) are already in CF. Mentees care about receiving their stipend — handled downstream by Expensify/NetSuite — not the raw balance figure.

When Mentorship moves to Kubernetes and gets a new UI design, a "View funding on LFX Crowdfunding" link on the program page is sufficient to surface the information without an integration dependency.

### CF → Snowflake sync via Fivetran

CF Postgres must be synced to Snowflake via Fivetran so that CF data (projects, funds, donations, subscriptions, organizations) is available for analytics and reporting alongside data from other LFX products.

This is **required for the initial release** — not "shortly after". A Jira ticket to configure the Fivetran connector must be created once the CF DB is up and populated with production data. Owner: DevOps. This is a release-blocking task.

Note: the `mentorship-sync` CronJob reads **Mentorship data from Snowflake into CF** — this is the Mentorship team's Fivetran responsibility, not CF's. CF→Snowflake is a separate Fivetran connector that CF DevOps owns.

### LFX Self Serve integration — Snowflake via Fivetran

The PM has requested CF data surfaces in LFX Self Serve ("My Donations", "My Initiatives", and potentially more — full list TBD, see OQ-11). LFX Self Serve reads CF data from **Snowflake** via the Fivetran CF→Snowflake sync (the same pattern used for My Trainings, My Meetings, etc.).

Rationale: the Snowflake pattern already exists in LFX Self Serve and requires minimal new code. The 24h Fivetran sync delay is acceptable for summary widgets — real-time payment confirmation is handled on `crowdfunding.linuxfoundation.org`, not in LFX Self Serve. This approach has no runtime dependency between LFX Self Serve and the CF API service. See OQ-11 for scope.

**No LFX Self Serve integration code will be written until:**
1. The PM provides a UI design for the LFX Self Serve CF widgets (what data, what layout)
2. The full list of CF data needed in LFX Self Serve is confirmed (see OQ-11)
3. The Fivetran CF→Snowflake connector is live with production data

**Future path:** Full platform stack integration (Traefik ingress, OpenFGA authorization, platform indexer, Query Service) is the long-term correct architecture for CF as an LFX v2 citizen. This is deferred post-initial-release and must be tracked as a separate Jira epic. It is not "later maybe" — it is a scheduled follow-on project.

### Expensify sync — keep on old Lambda for initial release

The `expensify-sync` cron job (SyncExpensifyHandler) pushes project/entity metadata to the Reimbursement Service. This is NOT end-user visible and NOT part of the initial release. The old Lambda continues running this job unchanged until the Reimbursement Service is migrated.

**Constraint for future expensify-sync port:** When expensify-sync is eventually ported to the new CF service, it must send the original DynamoDB string ID (`projectId`) as the project ID to RS — not the new Postgres UUID. RS stores the project ID in Expensify as a custom reporting field and uses it as the policy lookup key; sending the Postgres UUID would silently break the approval chain. The original DynamoDB ID is not recoverable by inverting the `uuid5` mapping — it must be preserved explicitly in a dedicated column or mapping table during migration.

### EMAIL_DRY_RUN mode

Add `EMAIL_DRY_RUN=true` environment variable. When set, email service logs the would-be email payload instead of calling Mandrill. Used when testing with production data to prevent accidental emails.

### LFX v2 platform stack — standalone for initial release

CF does **not** integrate with the LFX v2 platform stack (Traefik gateway, Heimdall, OpenFGA, platform indexer, Query Service) for the initial release. CF uses a standalone Kubernetes Ingress and handles auth internally via its own Auth0 JWT middleware — the same approach as `lfx-v2-ui` and `lfx-changelog`.

This was reviewed with Eric Searcy (chief architect, May 2026). His position: participating in the access control graph is architecturally correct long-term but will slow down delivery significantly. Given the end-of-May deadline, standalone is the right call for initial release.

**Tech debt created is bounded and intentional:**
- CF's own JWT middleware → replaceable with Heimdall when full platform stack integration is implemented
- CF's own Ingress → replaceable with Traefik HTTPRoute
- Postgres FTS for search → stays; Query Service is additive, not a replacement
- No OpenFGA types defined → must be designed and implemented as part of full platform stack integration

**To keep the future platform integration achievable:** access control intent must be documented now — who can do what to which resource — even though it is not implemented via OpenFGA yet. This is captured in the OpenFGA design notes below.

**Access control intent (for future OpenFGA model):**
- `initiative` (project/mentorship/`general fund`/event/ostif): owner (creator) has writer; donors have no elevated access; CF admin has writer for approval flow
- `organization`: owner has writer; members have no elevated access beyond the owner (org membership is used for donation attribution only, not access control)
- `subscription` / `donation`: owned by the creating user; read-only to CF admin
- Anonymous users: read-only access to published initiatives

**Initiatives are decoupled from LFX project entities.** There is no FK or relationship linking a CF initiative to an LFX project in the permissions graph. Access is determined solely by `owner_id` (Auth0 subject) and `initiative_type`. Role inheritance from LFX project roles (e.g., project auditor → initiative writer) is not possible without adding a `project_id` FK — this is a design decision deferred to full platform stack integration.

Consequence for non-LF projects (future): since initiatives are decoupled, supporting non-LF projects is a per-initiative policy decision, not a structural schema change.

**Known divergence from LFX v2 API patterns:** LFX v2 resource APIs never serve collections — all list queries go through the Query Service (OpenSearch-backed, access-control-aware). CF's initial release serves collection endpoints directly from the Go API (e.g., `GET /v1/initiatives`, `GET /v1/me/subscriptions`). These endpoints must be redesigned through the Query Service when full platform stack integration is implemented.

**LFX Self Serve integration auth:** LFX Self Serve reads CF data from Snowflake — there are no live API calls from LFX Self Serve to the CF Go API. Auth between LFX Self Serve and the CF API is not needed until full platform stack integration (see below) is implemented. Deferred to OQ-11.

**Full platform stack integration** is a post-initial-release tracked project — not "later maybe." File a Jira epic when initial release ships.

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

**2. CF admin / initiative approver — `ALLOWED_APPROVERS` env var**
The person who approves or rejects initiative submissions is identified by their LFID, stored in a `ALLOWED_APPROVERS` environment variable (comma-separated). This is not an Auth0 role — it is a config value injected at deploy time.

Production value: `shubhrakar` (Sriji). Confirmed as the approver for the new system.
Dev/staging value: `*` (any authenticated user can approve — allows testing).

Stored in AWS Secrets Manager, injected via ESO. The env var is parsed into a list at startup and the approval check does a case-insensitive exact match:
```go
// parsed at startup: parseCommaList(getEnv("ALLOWED_APPROVERS", ""))
// check in handler:
for _, a := range h.allowedApprovers {
    if strings.EqualFold(a, principal.Username) {
        return true
    }
}
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

### `diversity` — budget category kept, Diversity API deferred

`diversity` is both a budget category (active — used in production `subscriptions` and `donations` data, must be migrated as an `initiative_goals` row like `development`, `marketing`, etc.) and a separate Diversity API integration (deferred — fetches demographic stats for display on project pages). These are unrelated. The deferred API flag does not mean the budget category is removed.

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
- `ledger-stats-sync` — K8s CronJob (hourly)

Nothing in the initial release runs on Lambda or Serverless Framework.

### Database — shared AWS RDS instance (managed Postgres)

Crowdfunding Postgres runs on the shared LFX v2 RDS instance (`aws_db_instance.lfv_v2`, Postgres 17.4), defined in `linuxfoundation/lfx-v2-opentofu`. This is the LFX platform standard — every service (changelog, lens, member-onboarding, sanctions-screening, litellm, openfga) uses the same shared RDS instance with per-service databases and roles. No per-service RDS instance, no in-cluster StatefulSet.

To add CF: a `crowdfunding` entry is added to the `databases` map in `lfx-v2-opentofu/postgres.tf` — PR already open at `linuxfoundation/lfx-v2-opentofu#181`. Credentials are auto-rotated every 30 days via AWS Secrets Manager and injected by ESO.

From the app's perspective the connection string is `rds-postgres.lfx:5432` — an in-cluster ExternalName service with a socat proxy defined in `lfx-v2-opentofu/k8s-database-proxy.tf`.

### Secrets — External Secrets Operator + AWS Secrets Manager

All secrets are stored in AWS Secrets Manager and synced into K8s Secrets by the External Secrets Operator (ESO). The service account uses IRSA to authenticate to AWS Secrets Manager. This is the LFX platform standard.

AWS Secrets Manager path convention (following LFX pattern): `/cloudops/managed-secrets/crowdfunding/{env}/...` — confirm exact path with DevOps.

#### Go API — required env vars

| Env var | Description | Source / notes |
|---|---|---|
| `DATABASE_URL` | Postgres connection string for CF DB | Auto-provisioned via `lfx-v2-opentofu`; auto-rotated every 30 days |
| `JWKS_URL` | Auth0 JWKS endpoint for JWT validation | New — see `../../../docs/authentication-architecture.md` Configuration Reference |
| `JWT_ISSUER` | Expected `iss` claim | New — environment-specific; see `09` |
| `JWT_AUDIENCE` | Expected `aud` claim | New — `https://crowdfunding-api.{env}.lfx.dev`; see `09` |
| `STRIPE_SECRET_KEY` | Stripe secret API key | Same key as LFF `STRIPE_CLIENT_SECRET` |
| `STRIPE_WEBHOOK_SECRET` | Per-endpoint signing secret for `POST /v1/stripe/webhook` | Same key as LFF; registered in Stripe dashboard against the CF webhook URL |
| `MANDRILL_API_KEY` | Transactional email via Mandrill/Mailchimp | Same key as LFF `MANDRILL_API_KEY` |
| `GITHUB_TOKEN` | GitHub API token for GitHub stats (repo metadata, stars, etc.) | Same token as LFF |
| `GITHUB_OAUTH_CLIENT_ID` | GitHub OAuth app client ID (GitHub Connect for project owners) | Same as LFF |
| `GITHUB_OAUTH_CLIENT_SECRET` | GitHub OAuth app client secret | Same as LFF |
| `CF_LEDGER_AUTH_TOKEN` | Shared secret for authenticating Ledger→CF API calls (`Authorization: Bearer`) | Same value as Ledger's `LEDGER_AUTHORIZATION_TOKEN`; must match what Ledger sends after the `fundspring.go` auth header fix |
| *(removed)* | RS→CF auth uses Auth0 M2M (`access:manage` scope) — no shared secret on the CF side. RS mints a token via `client_credentials` grant; CF validates via JWKS. See `../../../docs/authentication-architecture.md` Flow 3. |
| `CF_APPROVAL_SIGNING_SECRET` | HMAC secret for initiative/expense approval email links | New — replaces LFF `EMAIL_TOKEN_SIGNING_KEY` |
| `ALLOWED_APPROVERS` | Comma-separated list of LFIDs who can approve initiatives | Replaces LFF `APPROVERS` env var |
| `SNOWFLAKE_ACCOUNT` | Snowflake account identifier (for `mentorship-sync` CronJob) | Follow LFX platform pattern (see `lfx-lens` ArgoCD values) |
| `SNOWFLAKE_USER` | Snowflake user for CF service account | New — to be provisioned by DevOps |
| `SNOWFLAKE_PRIVATE_KEY` | Snowflake private key (key-pair auth) | New — LFX platform standard (no password auth) |
| `SNOWFLAKE_WAREHOUSE` | Snowflake warehouse name | Follow LFX platform pattern |
| `SNOWFLAKE_ROLE` | Snowflake role for CF queries | New — to be provisioned by DevOps |
| `LEDGER_API_URL` | Base URL of the Ledger HTTP API | Replaces LFF `TRANSACTIONS_API_URL` |
| `EMAIL_DRY_RUN` | Set to `true` to suppress Mandrill calls (logs instead) | Non-secret config; used for testing with production data |
| `CF_NOTIFICATION_SOURCE_EMAIL` | From address for outgoing emails | Replaces LFF `NOTIFICATION_SOURCE_EMAIL` |
| `CF_ADMIN_EMAIL` | Admin contact email (used in approval flows) | Replaces LFF `ADMIN_EMAIL` |

**Dropped from LFF (not needed in new system):**

| LFF var | Reason dropped |
|---|---|
| `TRANSACTIONS_API_SECRET` / `BENEFICIARY_API_SECRET` | Replaced by `CF_LEDGER_AUTH_TOKEN` (Ledger shared secret) — RS auth is now Auth0 M2M, no CF-side token needed |
| `SNS_PROJECT_TOPIC_ARN` | SNS/SQS dropped; Mentorship sync is Snowflake pull, not push |
| `REIMBURSEMENTS_API_SECRET` / `CLIENT_SECRET` / `CLIENT_ID` / `AUTH0_URL` | RS→CF auth uses Auth0 M2M (`access:manage` scope); CF validates via JWKS — no shared secret needed on the CF side. See `../../../docs/authentication-architecture.md` Flow 3. |
| `DIVERSITY_BASE_URL` | Diversity API integration deferred |
| `JOBSPRING_API_URL` | Mentorship data now comes from Snowflake, not Jobspring HTTP API |
| `STAGE` / `REGION` / `APP_NAME` | Lambda-era config; replaced by K8s environment convention |
| `TRAVEL_SCHOLARSHIP_SLUG` / `DEFAULT_FUNDING_AMOUNT_IN_CENTS` | Audit whether still needed; likely hardcoded constants in the new service |

#### Nuxt frontend — required env vars

Already documented in the Frontend section above (`NUXT_AUTH0_*`, `NUXT_STRIPE_SECRET_KEY`, `NUXT_JWT_SECRET`, `NUXT_GITHUB_OAUTH_*`, `NUXT_PUBLIC_*`).

#### Pre-cutover checklist item

All Go API env vars must be provisioned in AWS Secrets Manager and verified via ESO before cutover. Add a smoke-test step that starts the Go service with `EMAIL_DRY_RUN=true` and confirms all required vars are present (the service panics on startup if any are missing — same pattern as LFF).

### Old Lambda stack runs in parallel during initial release

The old LFF Lambda + DynamoDB + OpenSearch stack continues running until:
1. New system is fully validated in production
2. RS Phase 1 internal endpoints are live on the CF Go API (RS switched off OpenSearch for CF-owned reads — see OQ-7)
3. DNS cutover is executed

Do not decommission the old stack before all three conditions are met. OpenSearch itself is NOT decommissioned at this point — RS still owns three live indices there (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`). OpenSearch decommission is deferred until RS moves to Kubernetes (timeline TBD, see OQ-7).
