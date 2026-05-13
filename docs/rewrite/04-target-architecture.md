<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Target Architecture

This document describes the intended architecture for the rewritten LFX Crowdfunding platform.

---

## System Overview

Architecture validated against diagram (May 2026). The purple "NEW" box is everything
deployed to Kubernetes. Everything outside the box is unchanged for the initial release.

```
━━━━━━━━━━━━━━━━━━━━━━━ NEW (Kubernetes) ━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Beneficiary / Donor / CF Admin
          │
          ▼
  ┌───────────────────┐
  │  Nuxt 3 Frontend  │  K8s Deployment + Ingress
  │  (Vue 3 / TS)     │
  └────────┬──────────┘
           │ $fetch HTTPS
  ┌────────▼──────────┐        ┌─────────────────────────────┐
  │  Go HTTP API      │        │  Snowflake (mentorship sync)│
  │  (Chi router)     │◄──────►│  programs + beneficiaries   │
  │  K8s Deployment   │        └─────────────────────────────┘
  │  + Ingress        │
  └──┬────────────────┘
     │                         ┌─────────────────────────────┐
     │                    ────►│  Stripe (payments)          │
     │                         └─────────────────────────────┘
     │                         ┌─────────────────────────────┐
     │  GET /balance, /txns    │  Ledger Service (Lambda)   │
     │────────────────────────►│                             │
     │◄────────────────────────│  GET /v1/projects/{id}      │
     │  (donation email data)  │  GET /v1/entities/{id}      │
     │                         └─────────────────────────────┘
     │                         ┌─────────────────────────────┐
     │                         │  Mandrill (email)           │
     │                         └─────────────────────────────┘
     │                         ┌─────────────────────────────┐
     │                         │  GitHub API                 │
     │                         └─────────────────────────────┘
     ▼
  ┌───────────────────┐
  │  Crowdfunding DB  │  Shared LFX v2 RDS (private, K8s-only)
  │  (PostgreSQL)     │  one schema:
  │  crowdfunding.*   │  crowdfunding.* (CF-owned, RW)
  └───────────────────┘

  Background workers (K8s CronJobs):
  ┌──────────────────────────────────────────────────────┐
  │  mentorship-sync     K8s CronJob (daily or few x/day) │
  │  ledger-stats-sync   K8s CronJob (hourly)             │
  └──────────────────────────────────────────────────────┘

━━━━━━━━━━━━━━━━━━━ UNCHANGED (Lambda / external) ━━━━━━━━━━━━━━━━━━

  Stripe ──webhook──► Ledger Service ──► Ledger DB (own Postgres)
                           ▲
  Reimbursement Service ───┘ (writes reimbursement/debit transactions)
         │
         ▼
  Expensify ──► NetSuite ──► Finance Team

  Mentorship data ──► Snowflake ──► CF mentorship-sync CronJob ──► CF Postgres
  (no direct HTTP calls between Mentorship and CF)

  Old LFF Lambda + DynamoDB + OpenSearch  (parallel, until cutover)
```

---

## Frontend — Nuxt 3

### Tech Stack

| Concern | Choice | Notes |
|---|---|---|
| Framework | Nuxt 3 (latest) + Vue 3 | LFX platform standard |
| Language | TypeScript strict | Follow Insights repo |
| Styling | Tailwind CSS + CSS variables | Follow Insights |
| Component library | PrimeVue v4 (theme: none) | Custom Tailwind styles applied |
| State management | Pinia (app state) + Vue Query (server state) | Follow Insights |
| HTTP client | `$fetch` (ofetch) | Built into Nuxt |
| Auth | OAuth2 PKCE, HTTP-only cookies | Server-side token exchange |
| LFX Header | `@linuxfoundation/lfx-ui-core`, `<lfx-navbar />` | Dynamic import, client-only |
| Form validation | Vuelidate | Follow Insights |
| Payments | Stripe.js (external script) | Same as current system |

### Project Structure (follow Insights)

```
frontend/
├── app/
│   ├── assets/            # Images, icons, styles
│   ├── components/
│   │   ├── modules/       # Feature components (project, fund, donation, etc.)
│   │   └── shared/        # Layout, header, footer, common UI
│   ├── layouts/           # default.vue
│   ├── middleware/        # Route guards (auth, owner)
│   ├── pages/             # File-based routing
│   └── plugins/           # auth.client.ts, stripe.client.ts
├── server/
│   ├── api/
│   │   └── auth/          # login.get.ts, callback.ts, logout.post.ts, user.get.ts
│   └── utils/
├── composables/           # useAuth.ts, useProject.ts, useFund.ts, etc.
├── types/                 # TypeScript type definitions
└── nuxt.config.ts
```

### Authentication Flow (follow Insights exactly)

1. User clicks login → `GET /api/auth/login` → server generates PKCE challenge + state → returns Auth0 URL
2. Auth0 authenticates user → redirects to `/auth/callback`
3. `/api/auth/callback` exchanges code for tokens → stores in HTTP-only cookies
4. All API calls use `credentials: 'include'` — token sent automatically
5. `composables/useAuth.ts` provides reactive auth state client-side

Auth0 tenants — **new Auth0 application required** (see OQ-8). Tenants are unchanged but a new app must be created in each via `linuxfoundation/auth0-terraform`. Client IDs below are for the old system and must not be reused:
- Dev: `linuxfoundation-dev.auth0.com`
- Staging: `linuxfoundation-staging.auth0.com`
- Prod: `sso.linuxfoundation.org`

### Pages / Routes (Nuxt file-based)

```
pages/
├── index.vue                          # Discovery (project + fund listing)
├── auth/
│   └── callback.vue                   # Auth0 callback redirect
├── github/
│   └── callback.vue                   # GitHub OAuth callback
├── email/
│   └── approve-initiative.vue         # Approve/reject initiative (JWT link, no Auth0 required)
├── expense-email/
│   ├── approve/
│   │   └── [reportId].vue             # Approve Expensify expense report (Auth0 required; calls POST /v1/projects/approvals/approve/{reportId})
│   └── reject/
│       └── [reportId].vue             # Reject Expensify expense report (Auth0 required; calls POST /v1/projects/approvals/reject/{reportId})
└── initiatives/
    ├── create/
    │   ├── project/
    │   │   ├── connect.vue            # GitHub OAuth step
    │   │   ├── select-repo.vue        # Repository picker
    │   │   └── details.vue            # Project details form
    │   ├── general.vue                # General fund form
    │   ├── event.vue                  # Event form
    │   └── ostif.vue                  # OSTIF form
    └── [slug]/
        ├── index.vue                  # Initiative dashboard
        ├── financial.vue              # Financial tab
        ├── edit.vue                   # Edit initiative (CF-owned fields only for mentorship)
        └── payments.vue               # Donation/subscription form
```

### Environment Config

Server-side only (via `useRuntimeConfig()` server context):
- `NUXT_AUTH0_CLIENT_SECRET`
- `NUXT_JWT_SECRET`
- `NUXT_API_URL` (Go backend internal URL)
- `NUXT_STRIPE_SECRET_KEY`
- `NUXT_GITHUB_OAUTH_CLIENT_SECRET`

Public (accessible client-side):
- `NUXT_PUBLIC_AUTH0_DOMAIN`
- `NUXT_PUBLIC_AUTH0_CLIENT_ID`
- `NUXT_PUBLIC_AUTH0_REDIRECT_URI`
- `NUXT_PUBLIC_STRIPE_PUBLIC_KEY`
- `NUXT_PUBLIC_GITHUB_OAUTH_CLIENT_ID`
- `NUXT_PUBLIC_APP_ENV` (dev / staging / prod)

---

## Backend — Go HTTP Service

### Tech Stack

| Concern | Choice | Notes |
|---|---|---|
| Language | Go (latest stable) | Same as current system |
| Router | Chi | Same as current system |
| DB driver | `sqlc` + `pgx/v5` | Type-safe queries, no ORM |
| Migrations | `golang-migrate` | SQL migration files |
| Auth middleware | Auth0 JWT validation | Same logic as current authorizer Lambda |
| Config | env vars | Keep it simple |
| Logging | `slog` (stdlib) | Upgrade from logrus |

### Architecture Pattern (same DDD as LFF)

```text
cmd/
├── api/                # HTTP server entrypoint
├── mentorship-sync/    # Snowflake CronJob entrypoint
├── ledger-stats-sync/  # Ledger financial stats CronJob entrypoint
└── migrate/            # DB schema migration runner (golang-migrate)

db/
└── scripts/
    └── migrate_dynamo_to_postgres.py  # One-time DynamoDB → Postgres data migration (Python, complete)

internal/
├── initiatives/
│   ├── domain/       # Domain structs (Initiative, InitiativeType)
│   ├── usecases/     # Business logic (enforces field ownership by initiative_type)
│   └── repository/   # sqlc-generated SQL queries
├── subscriptions/
├── donations/
├── organizations/
├── users/
├── transactions/     # Read-only pass-through to Ledger API
└── auth/             # JWT middleware

services/
├── stripe/           # Stripe SDK wrapper
├── email/            # Mandrill wrapper + EMAIL_DRY_RUN support
├── github/           # GitHub API wrapper
├── ledger/           # Ledger API HTTP client (read-only)
├── reimbursement/    # Reimbursement API HTTP client
└── snowflake/        # Snowflake client (mentorship-sync CronJob)

db/
└── migrations/       # golang-migrate SQL files (001_initial.up.sql, etc.)
```

### EMAIL_DRY_RUN

When `EMAIL_DRY_RUN=true`:
- Email service logs the full email payload (to, subject, template, vars) at INFO level
- Does not call Mandrill API
- Returns success to caller (no error surface)
- Use when testing with production data to prevent accidental sends

### Background Jobs

| Job | K8s resource | Schedule | What it does |
|---|---|---|---|
| `mentorship-sync` | CronJob | Daily (or a few times/day) | Pulls mentorship program data from Snowflake, creates/updates `initiative_type = mentorship` rows in CF Postgres |
| GitHub stats | Lazy refresh (no CronJob) | On page load, TTL 6h | See decision in `02-decisions.md`. |
| `ledger-stats-sync` | CronJob | Every hour | Calls Ledger HTTP API to sync pre-aggregated financial stats as cached columns on `crowdfunding.initiatives`. **Required for correctness** — the only mechanism that reflects Expensify debit-side disbursements. Must run once manually before DNS cutover. Initial release syncs `amount_raised_in_cents` only (`GET /balance/{id}`). Full set of stats columns (backer count, subscription totals, etc.) defined after UI design review — see OQ-11 and OQ-19. |

Jobs removed from old system (not ported):
- `amountraised` / `amountraised-entities` → replaced by `ledger-stats-sync` CronJob
- `export-projects`, `export-organizations`, `export-users`, `entities-sync` → OpenSearch dropped; search replaced by Postgres full-text search
- `ledger-viewmodel` → no longer needed
- `expensify-sync` → stays on old Lambda, not ported for initial release
- `cii-badge` → deferred
- `sqs-consumer` → dropped; replaced by `mentorship-sync` Snowflake CronJob

### Internal Endpoints (for Reimbursement Service)

Three narrow read-only endpoints for RS to replace its OpenSearch reads of CF-owned data. Authenticated via `X-Internal-Token` shared secret (not Auth0). On the public HTTPS ingress — RS Lambda can reach them the same way it reaches any other public HTTPS service.

| Method | Path | Returns | Replaces | Used by |
|---|---|---|---|---|
| `GET` | `/internal/v1/initiatives?slug={slug}` | `{id, name, owner_id, status, initiative_type}` | `projects` + `entities` per-slug reads | `getEmailBySlug()` |
| `GET` | `/internal/v1/initiatives?status=published` | `[{id, name}]` (all published) | `projects` + `entities` bulk reads | `RefreshTags()` cron (every 3h) |
| `GET` | `/internal/v1/users/{owner_id}` | `{id, email}` | `lff-users` reads | `getEmailBySlug()` |

`X-Internal-Token` secret stored in AWS Secrets Manager, injected via ESO into both CF and RS at deploy time.

**The bulk endpoint is release-blocking.** Once CF DNS cuts over, OpenSearch receives no new CF writes and goes stale. `RefreshTags()` must switch to the bulk endpoint on cutover day or new projects will never appear as Expensify tags — beneficiaries cannot submit expenses against them. This is a silent financial failure.

`id` in the response is `initiatives.id` (Postgres UUID). For migrated initiatives whose DynamoDB ID was already UUID-form (the vast majority), `initiatives.id` is identical to the original DynamoDB string ID — RS can use it directly as the Expensify GL code. For the small number of non-UUID legacy IDs, `initiatives.id` is a `uuid5`-derived value that differs from the original DynamoDB string ID; RS must maintain a mapping in that case to match existing Expensify GL codes.

### Stripe Webhook

`POST /v1/hooks/stripe` — handles `customer.subscription.deleted` → cancel subscription in Postgres. Stripe signature verification required.

`invoice.payment_succeeded` is handled by the Ledger Service's own Stripe webhook. This does not change.

### Mentorship sync — Snowflake CronJob

CF syncs Mentorship program data from Snowflake via the `mentorship-sync` K8s CronJob. SNS/SQS is not used.

The CronJob:
- Queries Snowflake for all mentorship programs and their approved beneficiaries
- For each program not yet in CF Postgres: inserts a row with `initiative_type = 'mentorship'`, populates `jobspring_project_id`, `initiative_goals` mentee row (amount_in_cents), and approved beneficiary list
- For each program already in CF Postgres: updates Mentorship-owned fields only (name, status, the `initiative_goals` mentee row amount, mentorship mentors/skills/terms/custom_term tables, beneficiaries); never overwrites CF-owned fields (logo_url, color, description, website)
- Normalizes `'hide'` → `'hidden'` on status

A 24h sync window is acceptable: new mentorship programs are not immediately donation-ready, and beneficiaries don't draw funds until mid-term (months after approval).

CF keeps beneficiary data for two reasons: financial governance (CF is the custodian of donated funds and must know who is approved to draw them) and Reimbursement Service integration (CF pushes beneficiary add/remove actions to RS via the `beneficiary-actions` OpenSearch queue; RS cannot reach CF Postgres directly).

### Mentorship → CF: no direct HTTP calls

All Mentorship data (programs, statuses, beneficiaries) flows to CF via the `mentorship-sync` Snowflake CronJob. There are no direct HTTP calls from Mentorship to CF in the new system.

**Eliminated calls (previously in old system):**

| Method | Path | Why eliminated |
|---|---|---|
| `GET` | `/v1/projects/{id}/{slug}/sync` | CF slug is CF-internal; Mentorship doesn't need it |
| `GET` | `/v1/projects/{id}/funding` | Ledger data is in Snowflake — Mentorship queries directly |
| `POST` | `/v1/projects/title-check` | No user-initiated creation from Mentorship into CF |
| `POST` | `/v1/entities/{id}/addbeneficiary` | Beneficiaries synced via Snowflake CronJob |
| `POST` | `/v1/entities/{id}/removebeneficiary` | Beneficiaries synced via Snowflake CronJob |

These endpoints do not need to exist in the new CF service. The old Lambda kept them alive for the old integration — they are not ported.

---

## Database — PostgreSQL

### Schema: `crowdfunding`

All monetary values `bigint` (cents). All primary keys `uuid`. All timestamps `timestamptz`.

**Terminology:**
- `initiatives` — unified table for all fundable things; formerly split into `projects` and `entities`
- `initiative_type` values: `project` | `mentorship` | `general fund` | `event` | `ostif` (plus legacy migrated: `other` | `community` — present in production data but no new rows expected)
- `status` values: `submitted` | `published` | `declined` | `hidden`

The schema is defined in `db/migrations/001_initial.up.sql` (version 2.0.0) and reproduced here as a reference. The authoritative source is the SQL file.

Key structural notes:
- Budget goals are normalized into `initiative_goals` child table — no JSONB `budgets` column
- GitHub stats, mentors, skills, terms, beneficiaries, contributors etc. are in dedicated child tables
- Column names use `_on` suffix for timestamps (`created_on`, `updated_on`)
- Monetary amounts use `current_amount_in_cents` on transactions; `amount_raised_in_cents` on initiatives
- `source_dynamo_table` column on initiatives is migration-only and must be dropped post-cutover
- `stripe_subscription_id` and `stripe_charge_id` are nullable `VARCHAR(255)` with no UNIQUE constraint in the schema; uniqueness is enforced by Stripe

See `db/migrations/001_initial.up.sql` and `docs/rewrite/data-design_and_migration.md` for the full DDL and field-by-field rationale.

### Indexes

See `db/migrations/001_initial.up.sql` for the full index definitions (23 indexes). The migration sets `search_path TO crowdfunding, public` so all unqualified table names resolve to the `crowdfunding` schema. Key indexes:

```sql
-- Initiatives (unqualified names resolve to crowdfunding schema via SET LOCAL search_path)
CREATE INDEX IF NOT EXISTS idx_initiatives_owner_id       ON initiatives(owner_id);
CREATE INDEX IF NOT EXISTS idx_initiatives_type           ON initiatives(initiative_type);
CREATE INDEX IF NOT EXISTS idx_initiatives_status         ON initiatives(status);
CREATE INDEX IF NOT EXISTS idx_initiatives_slug           ON initiatives(slug);
CREATE INDEX IF NOT EXISTS idx_initiatives_amount_raised  ON initiatives(amount_raised_in_cents DESC);
```

---

## Services That Remain Unchanged (initial release)

| Service | Location | Notes |
|---|---|---|
| Ledger Service | AWS Lambda | Unchanged. Own Postgres (Ledger DB). CF calls it read-only via HTTP. Ledger calls CF HTTP API (`GET /v1/projects/{id}`, `GET /v1/entities/{id}`, `GET /v1/organizations/{id}`) for donation notification emails — new CF API must support legacy ID lookups on these paths (see decisions doc). |
| Ledger DB | AWS RDS / Postgres | Separate from Crowdfunding DB. Migrated post-initial-release. |
| Reimbursement Service | AWS Lambda | On CF release: switches CF data reads (`projects`, `entities`, `lff-users` OpenSearch indices) to three internal HTTPS endpoints on the CF Go API. Own tables (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`) stay on OpenSearch until RS moves to K8s. Cannot reach shared RDS directly (separate AWS account/VPC; RDS is private). |
| Mentorship (jobspring) | AWS Lambda | Unchanged. Publishes data to Snowflake. No direct calls to CF in new system. |
| Old LFF Lambda | AWS Lambda | Runs in parallel until cutover. Keeps OpenSearch fed for Reimbursement Service. |
| DynamoDB tables | AWS DynamoDB | Read during migration. Kept until decommission is confirmed safe. |
| OpenSearch | AWS OpenSearch | Kept alive until RS moves to K8s and migrates its three owned indices (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`) to Postgres. Timeline TBD. Must NOT be decommissioned before that point. |

---

## Deployment — Kubernetes (all NEW components)

Every component inside the "NEW" purple box is deployed to Kubernetes via ArgoCD.
Nothing in the initial release runs on Lambda or Serverless Framework.

| Component | K8s Resource | Notes |
|---|---|---|
| Nuxt 3 frontend | `Deployment` + `Service` + `Ingress` | TLS termination at Ingress |
| Go HTTP API | `Deployment` + `Service` + `Ingress` | Chi router, long-running |
| Crowdfunding Postgres | Shared AWS RDS instance | LFX standard — DevOps adds `crowdfunding` DB + role to existing `lfx-v2` RDS in `lfx-v2-opentofu/postgres.tf`; app connects via `rds-postgres.lfx:5432` |
| mentorship-sync job | `CronJob` | Daily or a few times/day; Snowflake → CF Postgres |
| ledger-stats-sync job | `CronJob` | Every hour; syncs financial stats from Ledger API into cached columns on `initiatives` |
| Secrets | External Secrets Operator → AWS Secrets Manager | LFX standard — ESO syncs secrets from AWS Secrets Manager into K8s Secrets; service account uses IRSA |
| ArgoCD app | New entry in `linuxfoundation/lfx-v2-argocd` | `crowdfunding` namespace; `lfx-v2-applications.yaml` |

URLs (unchanged — DNS cutover at go-live):
- Dev: `https://funding.dev.platform.linuxfoundation.org/`
- Prod: `https://crowdfunding.lfx.linuxfoundation.org/`

Cutover: switch Ingress/DNS from old Lambda API Gateway to new K8s Ingress.
Rollback: revert Ingress. Old Lambda stack stays live until explicitly decommissioned.

---

## What Is Intentionally Not in This Architecture

- OpenSearch — replaced by Postgres full-text search
- AWS Lambda — application code moves to K8s
- DynamoDB — data moves to Postgres
- Serverless Framework — replaced by K8s manifests + ArgoCD
- CloudWatch Events — replaced by K8s CronJobs
- DynamoDB Streams — stream-triggered logic moved to jobs or eliminated
- SNS/SQS — replaced by Snowflake CronJob for Mentorship sync
- `entities` / `projects` split — merged into unified `crowdfunding.initiatives` table with `initiative_type` discriminator
- `initiative` fund type — merged into `general fund` during migration
- `travel_fund` / `other` entity types — stored as `initiative_type = 'other'` (not merged into `general fund`)
- `community` entity type — 3 rows from 2019 migrated as `initiative_type = 'community'` (not discarded)
- Datadog RUM — deferred
- Intercom — deferred
- Stacks / security vulnerabilities — deferred
- Sponsor Tiers — deferred
- CII badge — deferred
