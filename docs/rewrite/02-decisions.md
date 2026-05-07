# Architectural Decisions

Decisions made during discovery (May 2026). Each decision has a rationale.
Update this file when decisions change — note what changed and why.

---

## Scope — Initial Release

### What is IN scope

- Project CRUD (create, edit, publish, hide) — campaign_type: `project`
- Mentorship campaign display and Snowflake-driven sync — campaign_type: `mentorship`
- Fund CRUD (general_fund, event, ostif) — replaces "entities"
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
- RS Category 2 Postgres migration (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`) — scheduled CF release + 2 weeks
- OpenSearch decommission — scheduled CF release + 2 weeks (see OQ-7)
- CII badge validation (not in initial release UI)
- Diversity API integration
- Vulnerability API integration
- `travel_fund` as a distinct fund type (merged into `general_fund` during migration)
- `initiative` as a distinct fund type (merged into `general_fund` during migration)
- `community` entity type (3 dead rows discarded during migration)

---

## Database

### PostgreSQL, not DynamoDB

Target database is PostgreSQL. DynamoDB is decommissioned after successful migration and cutover.

### Schema separation — two schemas, one Postgres instance

One Postgres instance, two schemas:

- `crowdfunding` — owned by CF Go API (campaigns, orgs, subscriptions, donations, users)
- `reimbursement` — owned by Reimbursement Service (expense_log, beneficiary_actions, travel_fund_tickets)

Reimbursement Service connects directly to this Postgres instance:
- Read-write on `reimbursement.*` (its own tables)
- Read-only on `crowdfunding.campaigns` (replaces OpenSearch `projects`/`entities`/`lff-users` index reads)

Enforced at the DB level: RS gets a read-only Postgres role for `crowdfunding` schema. Not by convention.

No internal HTTP API endpoints between CF and RS — direct DB reads are simpler, faster, and remove an unnecessary layer given both services are on the same team and the same Postgres instance.

Rationale: RS has no existing Postgres. Adding a separate RS Postgres instance for three tiny tables (~65 lines of SQL total) is unjustified operational overhead at this scale. The blast radius concern from schema co-location is theoretical at 2,013 rows and a small team. RS data is isolated in its own schema.

Ledger Service keeps its own separate Postgres DB (Ledger DB). It is not migrated in the initial release. CF calls the Ledger HTTP API read-only for transaction stats and balance data — exactly as today.

Future (post-initial-release): Ledger DB merges into Crowdfunding DB as a `ledger` schema on the same Postgres instance. At that point `project_funding_summary` view becomes active and the Ledger HTTP API call for balance data is replaced by a direct SQL query. This is a separate tracked project, not part of the initial release.

Rationale: co-locating Ledger DB now requires migrating Ledger's Postgres data, reconfiguring Ledger Service (a change we want to avoid), and coordinating two cutover windows simultaneously. The tech debt of keeping one HTTP call is minimal and localized. Deliver CF first, migrate Ledger after.

### `projects` + `funds` → unified `campaigns` table

The old `projects` and `funds` tables are merged into a single `crowdfunding.campaigns` table with a `campaign_type` discriminator column.

`campaign_type` values: `project` | `mentorship` | `general_fund` | `event` | `ostif`

Rationale: all campaign types share the same donor/subscription FK, the same Ledger balance lookup via `legacy_id`, the same status workflow, and the same discovery/search surface. Two tables required a polymorphic FK (`project_id OR fund_id`) with a CHECK constraint and no hard FK enforcement. One table gives a clean `campaign_id` FK on subscriptions and donations, simpler queries, and a unified search/discovery endpoint.

Type-specific sparse columns (e.g. `event_start_date`, `mentorship_program_id`, `city`) are nullable and only populated for the relevant `campaign_type`. This is the standard discriminated-union pattern for a sparse table — preferable to two tables with a UNION when the shared columns dominate, which they do here.

### Budget categories → JSONB keyed object

Budget categories (development, marketing, meetups, travel, bug_bounty, documentation, mentee, other, diversity) stored as a JSONB keyed object on `crowdfunding.campaigns`.

Rationale: new budget categories may be added in future — columns would require a migration per new category. JSONB allows adding new categories without schema changes. Categories are always read/written as a unit. No queries filter by individual category amounts at the DB level.

**Shape:** keyed by category name (not an array). The keyed object enforces category uniqueness by structure and allows O(1) lookup by key. A `CHECK (jsonb_typeof(budgets) = 'object')` constraint enforces the shape at the DB level.

```json
{
  "development": {"amount_in_cents": 100000, "description": "...", "goals": "...", "is_active": true},
  "marketing":   {"amount_in_cents": 50000,  "description": "...", "goals": "...", "is_active": false}
}
```

Mentorship campaigns extend the `mentee` key with additional fields:
```json
{
  "mentee": {
    "amount_in_cents": 600000,
    "is_active": true,
    "skills": ["Go", "Kubernetes"],
    "terms": ["Spring 2026"],
    "mentors": [{"name": "...", "email": "...", "introduction": "...", "avatar_url": "..."}],
    "custom_term": {"start_month": "March", "end_month": "August", "term_name": "Spring", "year": 2026}
  }
}
```

**Migration:** fund `goals` arrays from DynamoDB are converted to keyed objects during migration. Project budgets are already keyed objects — no conversion needed.

### Campaign workflow timestamps

Three nullable timestamps track the approval workflow: `submitted_at`, `approved_at`, `published_at`. Set by the API on status transitions. Never updated once set.

Migration sets `approved_at = created_at` for all existing rows with `status IN ('approved', 'published')` as a safe approximation — no historical data exists for the actual approval moment.

Rationale: enables audit trails, SLA measurement ("how long does approval take"), and donor-facing "approved on" display. Zero schema cost.

### `category` validation on subscriptions and donations

The `category` column on `subscriptions` and `donations` references a budget category on the target campaign. Validated at two levels:

1. **DB level:** CHECK constraint against the fixed known set (`development`, `marketing`, `meetups`, `travel`, `bug_bounty`, `documentation`, `mentee`, `other`, `diversity`). Catches garbage values at insert time.
2. **API level:** validates that the category exists and `is_active = true` on the target campaign's `budgets` JSONB before accepting a payment.

If a new budget category is added, the CHECK constraint is updated via migration at the same time.

### `stripe_charge_id` nullable on donations

`stripe_charge_id` is nullable. Invoice-based donations have no Stripe charge ID at creation time. The UNIQUE constraint still applies — Postgres UNIQUE allows multiple NULLs, so invoice donations without a charge ID do not conflict.

### Donor name snapshot on donations

`donations.name` stores the donor display name at time of donation. This differs from the Auth0 profile for org/invoice donations where the payer is an individual but the displayed donor is the company name. `avatar_url` is not stored — always fetched from Auth0 at render time to avoid stale cache.

### Campaign workflow timestamps — `submitted_at`, `approved_at`, `published_at`

Covered above under "Campaign workflow timestamps."

### `amount_in_cents` → `bigint`

All monetary amounts stored as `bigint` (int8).

Rationale: max donation is $999,999.99 = 99,999,999 cents, which fits in `int4` (~2.1B max). However `bigint` costs nothing on modern Postgres, eliminates overflow risk on aggregated totals, and matches the existing Go code which uses `int64` throughout. No reason to use `int4`.

### Two separate tables for subscriptions and donations

`subscriptions` and `donations` are separate tables, not merged with a `payment_type` enum.

Rationale: they represent different Stripe object types (`Subscription` vs `Charge`), have different lifecycles (recurring vs one-time), different cancellation/update logic, and different fields (frequency, stripe_subscription_id on subscriptions; stripe_charge_id on donations). Merging would create nullable columns and make queries less obvious.

### `amount_raised` → Postgres view (not a cron job)

Replace the `amountraised` cron job (which wrote a denormalized `amountRaised` field back to the DynamoDB project record) with a Postgres view:

```sql
CREATE VIEW crowdfunding.project_funding_summary AS
SELECT project_id, SUM(amount) AS amount_raised_cents
FROM ledger.ledger
WHERE txn_type = 'CREDIT'
GROUP BY project_id;
```

Rationale: always fresh, no job to maintain, no staleness. If query performance becomes an issue at scale, materialize it then. Start simple.

Note: this view requires the `ledger` schema to be accessible from the `crowdfunding` schema. Until Ledger is on the same Postgres instance, `amount_raised` will need to be fetched via the Ledger API (`GET /balance/{projectID}`) as today, and optionally cached on the project record. Revisit when Ledger is co-located.

### No ORM — sqlc for type-safe queries

Use `sqlc` to generate type-safe Go code from SQL queries. No ORM (GORM, ent, etc.).

Rationale: the existing codebase uses raw AWS SDK calls with no ORM. `sqlc` gives compile-time query safety without the complexity and magic of an ORM. Consistent with the team's preference for explicit code.

### golang-migrate for schema migrations

Use `golang-migrate` for database schema migrations.

Rationale: simpler than Atlas, battle-tested, widely used in LFX ecosystem. Atlas's schema diffing is valuable but unnecessary complexity for this project.

---

## Domain Model

### `campaigns` table — unified campaign type

All fundable things live in one `crowdfunding.campaigns` table with `campaign_type` as the discriminator.

`campaign_type` values:
- `project` — created by users via the CF web UI; GitHub-linked; has budget categories; owned and editable by CF
- `mentorship` — created and managed by the Mentorship service; synced into CF via Snowflake CronJob; 91% of published rows; partially read-only in CF UI
- `general_fund` — fundraising fund (formerly `initiative`, `general-fund`, `travel_fund`)
- `event` — calendar/meetup-based fund
- `ostif` — security audit fund

`campaign_type` is set on creation and never changed.

### `mentorship_program_id` — canonical Mentorship link

Rename `jobspringProjectID` → `mentorship_program_id`. This is the string ID used by the Mentorship service to identify a program. It is:
- Populated during migration from the `jobspringProjectID` DynamoDB attribute on existing rows
- Used by the `mentorship-sync` CronJob to match Snowflake program records to existing Postgres rows (upsert key)
- Used as the `legacy_id` equivalent for Ledger API calls (same value — see migration notes)
- Never exposed in the public API response
- NULL on all non-mentorship campaigns

### Mentorship campaigns — partially editable in CF UI

Current system: fully editable — no restrictions on any field. This is a latent bug; Mentorship sync overwrites CF edits silently.

New system: field ownership split enforced at the API layer (`PATCH /v1/me/campaigns/:id` rejects Mentorship-owned fields when `campaign_type = mentorship`).

| Field group | Owner | Editable via CF UI |
|---|---|---|
| `name` | Mentorship | No — set by `mentorship-sync`, read-only in CF |
| `status` | Mentorship | No — controlled by `mentorship-sync` (mirrors Mentorship status) |
| `mentorship_program_id` | Mentorship | No — internal, never exposed |
| Skills, terms, mentors, custom term (inside `budgets.mentee`) | Mentorship | No |
| `logo_url`, `color`, `description`, `website` | CF | Yes |
| Budget goal amounts (per category) | CF | Yes |
| `beneficiaries` | Shared | Yes — CF manages who can draw funds |

### Type consolidations (applied during migration)

| Old DynamoDB type | New `campaign_type` | Rationale |
|---|---|---|
| `project` (with `jobspringProjectID`) | `mentorship` | Explicit type; was previously inferred from field presence |
| `project` (without `jobspringProjectID`) | `project` | Unchanged |
| `initiative` | `general_fund` | UI already labeled these "General Fund"; backend type was an alias |
| `general-fund` | `general_fund` | Normalize hyphen, same concept |
| `travel_fund` / `other` | `general_fund` | UI option commented out; 26 rows function identically to general funds |
| `event` | `event` | Unchanged |
| `ostif` | `ostif` | Unchanged |
| `community` | ~~discarded~~ | 3 rows from 2019, all declined/submitted, no UI, no active users |

Confirm with PM before migration: the 8 published `travel_fund` rows will display as "General Fund" after reclassification.

### `general_fund` vs `initiative` — same thing

In the old system: `initiative` was the backend type string; "General Fund" was the UI label; the subscription service explicitly mapped `'general fund'` → `ExpenseCategory.INITIATIVE`. They were always the same concept. The new schema uses `general_fund` everywhere and drops the alias.

### Budget JSONB schema — two shapes by campaign_type

The `budgets` JSONB column on `projects` has different internal structure depending on `campaign_type`:

**`campaign_type = project`** (CF-created):
```json
{
  "development":  {"amount_in_cents": 100000, "description": "...", "goals": "...", "is_active": true},
  "marketing":    {"amount_in_cents": 50000,  "description": "...", "goals": "...", "is_active": false},
  "meetups":      {"amount_in_cents": 0,      "description": "...", "goals": "...", "is_active": false},
  "travel":       {"amount_in_cents": 0,      "description": "...", "goals": "...", "is_active": false},
  "bug_bounty":   {"amount_in_cents": 0,      "description": "...", "goals": "...", "is_active": false},
  "documentation":{"amount_in_cents": 0,      "description": "...", "goals": "...", "is_active": false},
  "mentee":       {"amount_in_cents": 0,      "description": "...", "goals": "...", "is_active": false},
  "other":        {"amount_in_cents": 0,      "description": "...", "goals": "...", "is_active": false}
}
```

**`campaign_type = mentorship`** (Mentorship-managed):
```json
{
  "mentee": {
    "amount_in_cents": 600000,
    "is_active": true,
    "skills": ["Go", "Kubernetes"],
    "terms": ["Spring 2026"],
    "mentors": [{"name": "...", "email": "...", "introduction": "...", "avatar_url": "..."}],
    "custom_term": {"start_month": "March", "end_month": "August", "term_name": "Spring", "year": 2026}
  }
}
```

**Migration implication:** the `migrate-cf` tool must read `data.projectDetails.mentee` (nested) for mentorship projects — NOT `data.mentee` at the top level. Reading from the wrong path silently drops all mentorship metadata. This was the bug that caused the first SQL pass to miss 1,249 of 1,476 project rows.

**Application implication:** Go code that reads `budgets` must branch on `campaign_type`. The `mentee` budget category for a `project`-type campaign is just an amount + description. For a `mentorship`-type campaign, it contains the full mentorship program structure.

### `status` normalization

DynamoDB has dirty status values: `'hide'` (13 rows) and `'hidden'` (1 row) coexist. Normalize during migration:
- `'hide'` → `'hidden'`

Valid status values in the new schema: `draft`, `submitted`, `approved`, `published`, `hidden`, `declined`.

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
| `cmd/migrate/` | `Dockerfile.migrate` | one-off `Job` |

Rationale: a single container serving both HTTP requests and being invoked as a CronJob (via a flag) conflates two distinct runtime responsibilities. Separate images are minimal, contain only the code they need, and make it obvious what each K8s resource is doing. Shared business logic in `internal/` is compiled into each binary — no duplication of source, no runtime coupling.

### Chi router (same as today)

Keep Chi. No reason to change.

### Ledger integration → keep calling Ledger HTTP API

The new Go service calls the Ledger HTTP API (read-only GET calls) exactly as LFF does today. LFF has never written to Ledger directly — Ledger gets its data from its own Stripe/Expensify webhooks. No change to this contract.

### Mentorship sync — Snowflake pull, not SNS/SQS

CF syncs Mentorship program data from Snowflake via a periodic K8s CronJob (`mentorship-sync`). SNS/SQS is not used in the new system.

Rationale: Mentorship and CF run in separate AWS accounts, making cross-account SNS/SQS subscription complex and requiring Mentorship team involvement. Mentorship is also moving to Kubernetes in the coming months, making Lambda-era SQS infrastructure a poor long-term investment. Both services already mirror data into Snowflake. A 24h sync delay is acceptable — new mentorship programs are not immediately donation-ready, and beneficiaries don't access funds until mid-term (months after program creation).

The `mentorship-sync` CronJob:
- Queries Snowflake for mentorship programs and their approved beneficiaries
- Creates `campaign_type = mentorship` project rows in CF Postgres for new programs
- Updates Mentorship-owned fields (name, status, budgets.mentee) for existing rows
- Syncs approved beneficiary list onto each project record
- Normalizes `'hide'` → `'hidden'` on status

**There are no direct HTTP calls between Mentorship and CF.** All data flows through Snowflake. This is a clean separation — Mentorship owns program and beneficiary data; CF reads it from Snowflake on a scheduled basis.

**Why CF keeps beneficiary data despite the Snowflake sync:**
CF is the financial custodian of donated funds. It collects money from donors and must maintain visibility into who is approved to draw those funds via Expensify. Beneficiary records in CF serve two purposes:
1. **Financial governance** — CF can reconcile money collected against approved disbursement recipients
2. **Reimbursement Service** — RS reads beneficiaries directly from CF Postgres (per OQ-7 resolution) to manage Expensify policies

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
- `campaign` (project/mentorship/general_fund/event/ostif): owner (creator) has writer; donors have no elevated access; CF admin has writer for approval flow
- `organization`: owner has writer; members have no elevated access beyond the owner (org membership is used for donation attribution only, not access control)
- `subscription` / `donation`: owned by the creating user; read-only to CF admin
- Anonymous users: read-only access to published campaigns

**Campaigns are decoupled from LFX project entities.** There is no FK or relationship linking a CF campaign to an LFX project in the permissions graph. Access is determined solely by `owner_id` (Auth0 subject) and `campaign_type`. Role inheritance from LFX project roles (e.g., project auditor → campaign writer) is not possible without adding a `project_id` FK — this is a design decision deferred to Option C.

Consequence for non-LF projects (future): since campaigns are decoupled, supporting non-LF projects is a per-campaign policy decision, not a structural schema change.

**Known divergence from LFX v2 API patterns:** LFX v2 resource APIs never serve collections — all list queries go through the Query Service (OpenSearch-backed, access-control-aware). CF's initial release serves collection endpoints directly from the Go API (e.g., `GET /v1/campaigns`, `GET /v1/me/subscriptions`). These endpoints must be redesigned through the Query Service when Option C is implemented.

**LFX One integration auth:** For the initial release, LFX One reads CF data from Snowflake (Option B) — there are no live API calls from LFX One to the CF Go API. The question of how LFX One authenticates to the CF API (ID tokens, API key, M2M, Auth0 resource server) only arises if Option A or C is chosen. Deferred to OQ-11.

**Option C** (full platform stack integration) is a post-initial-release tracked project — not "later maybe." File a Jira epic when initial release ships.

### Authorization model — initial release

The initial release uses the same authorization model as the current LFF system. There is no RBAC — authorization is purely ownership-based, enforced at the usecase layer.

**Three mechanisms:**

**1. Ownership check (campaign/org CRUD)**
Any authenticated user can create a campaign. The creator's Auth0 subject (`jwt.sub`) is stored as `owner_id` at creation time. All mutating operations check `campaign.owner_id == jwt.sub` inline in the usecase. There is no "owner role" stored anywhere — ownership is structural.

```go
// enforced inline in usecases, same pattern as current LFF
if campaign.OwnerID != currentUser.ID {
    return ErrNotAuthorized
}
```

**2. CF admin / campaign approver — `CF_APPROVERS` env var**
The person who approves or rejects campaign submissions is identified by their LFID, stored in a `CF_APPROVERS` environment variable (comma- or pipe-separated). This is not an Auth0 role — it is a config value injected at deploy time.

Production value: `shubhrakar` (Sriji). Confirmed as the approver for the new system.
Dev/staging value: `*` (any authenticated user can approve — allows testing).

Stored in AWS Secrets Manager, injected via ESO. The approval check at the API layer:
```go
authorized := strings.Contains(os.Getenv("CF_APPROVERS"), user.LFID)
```

**3. Email approval links — HMAC-signed token, no Auth0**
Campaign and expense approval email links use HMAC HS256-signed tokens (not Auth0). The token encodes `{ campaignID, action: "approve"|"reject" }` and has an expiry. The `POST /v1/campaigns/approvals` endpoint verifies the HMAC signature — the signed token is the sole authorization mechanism for this flow. No Auth0 JWT is required or checked.

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

---

## Migration

### Dedicated `migrate-cf` Go CLI (not lfx-v1-sync-helper)

Write a standalone Go CLI tool (`migrate-cf`) for the one-time DynamoDB → Postgres migration.

Rationale: `lfx-v1-sync-helper` is purpose-built for project/committee metadata sync between LFX v1 and v2 via NATS KV. Bolting Crowdfunding migration logic onto it would be wrong — different concerns, different infrastructure, different data shapes. A 200–300 line Go CLI with `--validate` (dry-run, reads DynamoDB, reports what would be written) and `--execute` (writes to Postgres) phases is cleaner, more auditable, and easier to run/rerun.

### All new components deployed to Kubernetes

Everything inside the "NEW" purple box in the architecture diagram is deployed to Kubernetes:
- Crowdfunding Nuxt frontend — K8s Deployment + Service + Ingress
- Crowdfunding Go API — K8s Deployment + Service + Ingress
- Crowdfunding Postgres DB — shared LFX v2 RDS instance; DevOps adds `crowdfunding` DB + role to `lfx-v2-opentofu`
- `mentorship-sync` — K8s CronJob (daily or a few times/day)
- GitHub stats job — K8s CronJob

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
2. Reimbursement Service OpenSearch dependency is resolved (OQ-7)
3. DNS cutover is executed

Do not decommission the old stack before all three conditions are met.
