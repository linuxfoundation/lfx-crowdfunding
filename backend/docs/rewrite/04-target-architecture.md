<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Target Architecture

This document describes the intended architecture for the rewritten LFX Crowdfunding platform.

---

## System Overview

Architecture validated against diagram (May 2026). The purple "NEW" box is everything
deployed to Kubernetes.

```text
━━━━━━━━━━━━━━━━━━━━━━━ NEW (Kubernetes) ━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Beneficiary / Donor / CF Admin
          │
          ▼
  ┌───────────────────┐
  │  Nuxt 4 Frontend  │  K8s Deployment + Ingress
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

  Old LFF Lambda + DynamoDB + OpenSearch  (decommissioned post-cutover)
```

---

## Frontend — Nuxt 4

### Tech Stack

| Concern | Choice | Notes |
|---|---|---|
| Framework | Nuxt 4 + Vue 3 | LFX platform standard |
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

```text
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

```text
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
| `ledger-stats-sync` | CronJob | Every hour | Calls Ledger HTTP API to sync pre-aggregated financial stats as cached columns on `crowdfunding.initiatives`. Required for correctness — the only mechanism that reflects Expensify debit-side disbursements. |

Jobs removed from old system (not in new architecture):
- `amountraised` / `amountraised-entities` → replaced by `ledger-stats-sync` CronJob
- `export-projects`, `export-organizations`, `export-users`, `entities-sync` → OpenSearch dropped; search replaced by Postgres full-text search
- `ledger-viewmodel` → no longer needed
- `expensify-sync` → stays on old Lambda; not part of the new CF service
- `cii-badge` → not in scope
- `sqs-consumer` → dropped; replaced by `mentorship-sync` Snowflake CronJob

### Internal Endpoints (for Reimbursement Service)

A narrow read-only M2M endpoint for RS to replace its OpenSearch reads of CF-owned owner data. Authenticated via Auth0 **`access:manage`** M2M token (`client_credentials` grant) — see `09-authentication-architecture.md` Flow 3. On the public HTTPS ingress — RS Lambda can reach it the same way it reaches any other public HTTPS service.

| Method | Path | Returns | Used by |
|---|---|---|---|
| `GET` | `/v1/initiatives/{slug}/owner-info` | `{email, name}` of the initiative owner | RS — resolve owner email for expense/beneficiary notifications |

`{slug}` is the initiative slug. No status filter is applied — the endpoint works for any initiative status so RS can resolve owner details regardless of publication state.

### Stripe Webhook

`POST /v1/stripe/webhook` — handles `customer.subscription.deleted` → cancel subscription in Postgres. Stripe signature verification required.

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

## External Services

Services that CF integrates with but does not own:

| Service | Location | Notes |
|---|---|---|
| Ledger Service | AWS Lambda | Own Postgres (Ledger DB). CF calls it read-only via HTTP. Ledger calls CF HTTP API for donation notification emails. |
| Reimbursement Service | AWS Lambda | Reads CF initiative data via `/v1/internal/*` endpoints. Cannot reach shared RDS directly (separate AWS account/VPC). |
| Mentorship (jobspring) | AWS Lambda | Publishes data to Snowflake. No direct calls to CF. |

---

## Deployment — Kubernetes

All CF components are deployed to Kubernetes via ArgoCD.

| Component | K8s Resource | Notes |
|---|---|---|
| Nuxt 4 frontend | `Deployment` + `Service` + `Ingress` | TLS termination at Ingress |
| Go HTTP API | `Deployment` + `Service` + `Ingress` | Chi router, long-running |
| Crowdfunding Postgres | Shared AWS RDS instance | LFX standard — `crowdfunding` schema on `lfx-v2` RDS; app connects via `rds-postgres.lfx:5432` |
| mentorship-sync job | `CronJob` | Daily or a few times/day; Snowflake → CF Postgres |
| ledger-stats-sync job | `CronJob` | Every hour; syncs financial stats from Ledger API into cached columns on `initiatives` |
| Secrets | External Secrets Operator → AWS Secrets Manager | LFX standard — ESO syncs secrets from AWS Secrets Manager into K8s Secrets; service account uses IRSA |
| ArgoCD app | `linuxfoundation/lfx-v2-argocd` | `crowdfunding` namespace; `lfx-v2-applications.yaml` |

URLs:
- Dev: `https://crowdfunding.dev.lfx.dev/`
- Staging: `https://crowdfunding.staging.lfx.dev/`
- Prod: `https://crowdfunding.linuxfoundation.org/`

---

## What Is Intentionally Not in This Architecture

- OpenSearch — replaced by Postgres full-text search
- AWS Lambda — application code runs in Kubernetes
- DynamoDB — data lives in PostgreSQL
- Serverless Framework — replaced by K8s manifests + ArgoCD
- CloudWatch Events — replaced by K8s CronJobs
- DynamoDB Streams — stream-triggered logic moved to jobs or eliminated
- SNS/SQS — replaced by Snowflake CronJob for Mentorship sync
- `entities` / `projects` split — merged into unified `crowdfunding.initiatives` table with `initiative_type` discriminator
- `initiative` fund type — merged into `general fund` during migration
- `travel_fund` / `other` entity types — stored as `initiative_type = 'other'`
- `community` entity type — 3 rows from 2019 migrated as `initiative_type = 'community'`
- Datadog RUM — not in scope
- Intercom — not in scope
- Stacks / security vulnerabilities — not in scope
- Sponsor Tiers — not in scope
- CII badge — not in scope
