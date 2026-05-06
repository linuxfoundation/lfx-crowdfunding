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
- SQS consumer for Mentorship events
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

- `crowdfunding` — owned by CF Go API (projects, funds, orgs, subscriptions, donations, users)
- `reimbursement` — owned by Reimbursement Service (expense_log, beneficiary_actions, travel_fund_tickets)

Reimbursement Service connects directly to this Postgres instance:
- Read-write on `reimbursement.*` (its own tables)
- Read-only on `crowdfunding.projects` and `crowdfunding.funds` (replaces OpenSearch `projects`/`entities`/`lff-users` index reads)

Enforced at the DB level: RS gets a read-only Postgres role for `crowdfunding` schema. Not by convention.

No internal HTTP API endpoints between CF and RS — direct DB reads are simpler, faster, and remove an unnecessary layer given both services are on the same team and the same Postgres instance.

Rationale: RS has no existing Postgres. Adding a separate RS Postgres instance for three tiny tables (~65 lines of SQL total) is unjustified operational overhead at this scale. The blast radius concern from schema co-location is theoretical at 2,013 rows and a small team. RS data is isolated in its own schema.

Ledger Service keeps its own separate Postgres DB (Ledger DB). It is not migrated in the initial release. CF calls the Ledger HTTP API read-only for transaction stats and balance data — exactly as today.

Future (post-initial-release): Ledger DB merges into Crowdfunding DB as a `ledger` schema on the same Postgres instance. At that point `project_funding_summary` view becomes active and the Ledger HTTP API call for balance data is replaced by a direct SQL query. This is a separate tracked project, not part of the initial release.

Rationale: co-locating Ledger DB now requires migrating Ledger's Postgres data, reconfiguring Ledger Service (a change we want to avoid), solving the Reimbursement Service → K8s Postgres network path (OQ-2, unresolved), and coordinating two cutover windows simultaneously. The tech debt of keeping one HTTP call is minimal and localized. Deliver CF first, migrate Ledger after.

### Budget categories → JSONB

Budget categories (Development, Marketing, Meetups, Travel, BugBounty, Documentation, Mentee, Other, Diversity) stored as `JSONB` on the `projects` and `funds` tables.

Rationale: categories are always read/written as a unit alongside the project record. No queries filter by individual category values (e.g., "find all projects with Development budget > $X"). A normalized `project_budgets` table would add joins with zero query benefit. JSONB is simpler and faster for this access pattern.

Example structure:
```json
{
  "development": {"amount_in_cents": 100000, "description": "...", "goals": "...", "is_active": true},
  "marketing":   {"amount_in_cents": 50000,  "description": "...", "goals": "...", "is_active": true}
}
```

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

### `projects` table — two campaign types, not one

The `projects` table carries a `campaign_type` column with two values:

- `project` — created by users via the CF web UI; GitHub-linked; has budget categories; owned and editable by CF
- `mentorship` — created and managed by the Mentorship service (jobspring) via SQS events; 91% of published rows; partially read-only in CF UI

Rationale: 1,249 of 1,374 published rows are mentorship campaigns. Treating `mentorship` as a flag or a budget category (as the old system did via `jobspringProjectID` presence) misrepresents the actual data distribution and makes queries unnecessarily awkward.

`campaign_type` is set on creation and never changed. Mentorship campaigns are created by the SQS consumer; project campaigns are created via the API.

### `mentorship_program_id` — canonical Mentorship link

Rename `jobspringProjectID` → `mentorship_program_id` in the Postgres schema. This is the DynamoDB string ID used by the Mentorship service to identify a project. It is:
- Set by the SQS consumer on `projectCreated` events
- Used to match subsequent `projectUpdated` / `projectUpdateStatus` events to the correct Postgres row
- Used as the `legacy_id` equivalent for Ledger API calls (same value — see migration notes)
- Never exposed in the public API response

The Go SQS handler code continues to reference `jobspringProjectId` from the event payload (the Mentorship event schema does not change). Only the Postgres column name changes.

### Mentorship campaigns — partially editable in CF UI

Current system: fully editable — no restrictions on any field. This is a latent bug; Mentorship SQS events overwrite CF edits silently.

New system: field ownership split enforced at the API layer (`PATCH /v1/me/projects/:id` rejects Mentorship-owned fields when `campaign_type = mentorship`).

| Field group | Owner | Editable via CF UI |
|---|---|---|
| `name` | Mentorship | No — set by SQS, read-only in CF |
| `status` | Mentorship | No — controlled by `projectUpdateStatus` SQS event |
| `mentorship_program_id` | Mentorship | No — internal, never exposed |
| Skills, terms, mentors, custom term (inside `budgets.mentee`) | Mentorship | No |
| `logo_url`, `color`, `description`, `website` | CF | Yes |
| Budget goal amounts (per category) | CF | Yes |
| `beneficiaries` | Shared | Yes — CF manages who can draw funds |

### `funds` table — replaces `entities`

The `entities` table is renamed `funds`. "Entity" is a DDD implementation term that leaked into the schema. These are fundraising funds — the correct domain term.

`fund_type` enum:
- `general_fund` — formerly `initiative`, `general-fund`, and `travel_fund` (all merged; see below)
- `event` — calendar/meetup-based
- `ostif` — security audit funds

### Type consolidations (applied during migration)

| Old type | New type | Rationale |
|---|---|---|
| `initiative` | `general_fund` | UI already labeled these "General Fund"; backend type was an alias |
| `general-fund` | `general_fund` | Normalize hyphen, same concept |
| `travel_fund` / `other` | `general_fund` | UI option commented out; 26 rows function identically to general funds; no special mechanics |
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

### Go — same language, same patterns

New backend is Go, same DDD pattern as LFF (domain/, usecases/, interfaces/repository/). Not a framework change.

### Kubernetes, not Lambda

Deployed as a long-running Go HTTP service on Kubernetes, not Lambda. Background jobs become Kubernetes CronJobs (not CloudWatch Events).

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

This is required before or shortly after CF goes live in production. It is independent of the Mentorship→K8s migration.

Note: the `mentorship-sync` CronJob reads **Mentorship data from Snowflake into CF** — this is the Mentorship team's Fivetran responsibility, not CF's. CF→Snowflake is a separate Fivetran connector that CF DevOps owns.

### Expensify sync — keep on old Lambda for initial release

The `expensify-sync` cron job (SyncExpensifyHandler) pushes project/entity metadata to the Reimbursement Service. This is NOT end-user visible and NOT part of the initial release. The old Lambda continues running this job unchanged until the Reimbursement Service is migrated.

### EMAIL_DRY_RUN mode

Add `EMAIL_DRY_RUN=true` environment variable. When set, email service logs the would-be email payload instead of calling Mandrill. Used when testing with production data to prevent accidental emails.

### Reimbursement and Ledger on Lambda — network path

Both services remain on Lambda (API Gateway endpoints). The new CF Go service on K8s calls them over HTTPS. Network path assumed to be reachable (public HTTPS API Gateway). Architect to confirm (see open questions).

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
- Crowdfunding Postgres DB — K8s (or managed Postgres — confirm with DevOps)
- SQS consumer — K8s Deployment (long-running, not a CronJob)
- GitHub stats job — K8s CronJob

Nothing in the initial release runs on Lambda or Serverless Framework.

### Old Lambda stack runs in parallel during initial release

The old LFF Lambda + DynamoDB + OpenSearch stack continues running until:
1. New system is fully validated in production
2. Reimbursement Service OpenSearch dependency is resolved (OQ-7)
3. DNS cutover is executed

Do not decommission the old stack before all three conditions are met.
