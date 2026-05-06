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
  │  Go HTTP API      │◄──────►│  Mentorship (direct HTTP)   │
  │  (Chi router)     │        └─────────────────────────────┘
  │  K8s Deployment   │        ┌─────────────────────────────┐
  │  + Ingress        │◄──────►│  Snowflake (mentorship sync)│
  └──┬────────────────┘        └─────────────────────────────┘
     │                         ┌─────────────────────────────┐
     │                    ────►│  Stripe (payments)          │
     │                         └─────────────────────────────┘
     │                         ┌─────────────────────────────┐
     │  Get Transactions Stats  │  Ledger Service (Lambda)   │
     │────────────────────────►│  read-only HTTP API calls   │
     │                         └─────────────────────────────┘
     │                         ┌─────────────────────────────┐
     │                         │  Mandrill (email)           │
     │                         └─────────────────────────────┘
     │                         ┌─────────────────────────────┐
     │                         │  GitHub API                 │
     │                         └─────────────────────────────┘
     ▼
  ┌───────────────────┐
  │  Crowdfunding DB  │  K8s (or managed Postgres)
  │  (PostgreSQL)     │  two schemas:
  │  crowdfunding.*   │  crowdfunding.* (CF-owned, RW)
  │  reimbursement.*  │  reimbursement.* (RS-owned, RW for RS; not accessible by CF)
  └───────────────────┘

  Background workers (K8s CronJobs):
  ┌──────────────────────────────────────────────────────┐
  │  mentorship-sync  K8s CronJob (daily or few x/day)   │
  │  github-stats     K8s CronJob (6-hourly)             │
  └──────────────────────────────────────────────────────┘

━━━━━━━━━━━━━━━━━━━ UNCHANGED (Lambda / external) ━━━━━━━━━━━━━━━━━━

  Stripe ──webhook──► Ledger Service ──► Ledger DB (own Postgres)
                           ▲
  Reimbursement Service ───┘ (writes reimbursement/debit transactions)
         │
         ▼
  Expensify ──► NetSuite ──► Finance Team

  Mentorship (jobspring Lambda) ──► CF APIs (direct HTTP, CF returns 404 gracefully if not synced yet)
  Mentorship data ──► Snowflake ──► CF mentorship-sync CronJob ──► CF Postgres

  Old LFF Lambda + DynamoDB + OpenSearch  (parallel, until cutover)

━━━━━━━━━━━━━━━━━━━━━ FUTURE (post-initial-release) ━━━━━━━━━━━━━━━━

  Ledger DB merges into Crowdfunding DB as ledger.* schema.
  Ledger Service reconfigured to point at combined Postgres.
  project_funding_summary view becomes active.
  CF API balance calls become direct SQL queries.
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

Auth0 tenants (unchanged from current system):
- Dev: `linuxfoundation-dev.auth0.com` / `lzClGRsDYnfgMmio8J9vYXwTkFm51na2`
- Staging: `linuxfoundation-staging.auth0.com` / `DnO2mm4jbiKO3HaFIo2TOwY3fkcKV5O3`
- Prod: `sso.linuxfoundation.org` / `1sgQmtwRIKwMrCFoFSu6iAm8RtJGvPmf`

### Pages / Routes (Nuxt file-based)

```
pages/
├── index.vue                          # Discovery (project + fund listing)
├── auth/
│   └── callback.vue                   # Auth0 callback redirect
├── stripe/
│   └── callback.vue                   # Stripe OAuth callback
├── github/
│   └── callback.vue                   # GitHub OAuth callback
├── email/
│   ├── approve.vue                    # Approve expense (JWT link)
│   ├── reject.vue                     # Reject expense (JWT link)
│   ├── approve-project.vue            # Approve project (JWT link)
│   └── approve-fund.vue               # Approve fund (JWT link)
├── projects/
│   ├── create/
│   │   ├── connect.vue                # GitHub OAuth step
│   │   ├── select-repo.vue            # Repository picker
│   │   └── details.vue                # Project details form
│   └── [slug]/
│       ├── index.vue                  # Project dashboard
│       ├── financial.vue              # Financial tab
│       ├── edit.vue                   # Edit project (CF-owned fields only for mentorship)
│       └── payments.vue               # Donation/subscription form
└── funds/
    ├── create/
    │   ├── general.vue                # General fund form
    │   ├── event.vue                  # Event form
    │   └── ostif.vue                  # OSTIF form
    └── [slug]/
        ├── index.vue                  # Fund dashboard
        ├── financial.vue              # Financial tab
        ├── edit.vue                   # Edit fund
        └── payments.vue               # Fund donation form
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

```
cmd/
├── api/         # HTTP server entrypoint
├── worker/      # SQS consumer entrypoint
└── migrate/     # DB migration runner (golang-migrate)

internal/
├── projects/
│   ├── domain/       # Domain structs (Project, CampaignType)
│   ├── usecases/     # Business logic (enforces field ownership by campaign_type)
│   └── repository/   # sqlc-generated SQL queries
├── funds/            # Formerly "entities"
│   ├── domain/       # Domain structs (Fund, FundType)
│   ├── usecases/
│   └── repository/
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
└── sqs/              # SQS consumer for Mentorship events

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
| `mentorship-sync` | CronJob | Daily (or a few times/day) | Pulls mentorship program data from Snowflake, creates/updates `campaign_type = mentorship` rows in CF Postgres |
| `github-stats` | CronJob | Every 6 hours | Fetches GitHub repo stats, updates project records |

Jobs removed from old system (not ported):
- `amountraised` / `amountraised-entities` → replaced by Ledger HTTP API calls (same as today); will be replaced by `project_funding_summary` view once Ledger DB is co-located (post-initial-release)
- `export-projects`, `export-organizations`, `export-users`, `entities-sync` → OpenSearch dropped; search replaced by Postgres full-text search
- `ledger-viewmodel` → no longer needed
- `expensify-sync` → stays on old Lambda, not ported for initial release
- `cii-badge` → deferred
- `sqs-consumer` → dropped; replaced by `mentorship-sync` Snowflake CronJob

### Stripe Webhook

`POST /v1/hooks/stripe` — handles `customer.subscription.deleted` → cancel subscription in Postgres. Stripe signature verification required.

`invoice.payment_succeeded` is handled by the Ledger Service's own Stripe webhook. This does not change.

### Mentorship sync — Snowflake CronJob

CF syncs Mentorship program data from Snowflake via the `mentorship-sync` K8s CronJob. SNS/SQS is not used.

The CronJob:
- Queries Snowflake for all mentorship programs
- For each program not yet in CF Postgres: inserts a row with `campaign_type = 'mentorship'`, populates `mentorship_program_id`, and reads budgets from the `mentee` field (skills, terms, mentors, custom_term)
- For each program already in CF Postgres: updates Mentorship-owned fields only (name, status, budgets.mentee); never overwrites CF-owned fields (logo_url, color, description, website, beneficiaries)
- Normalizes `'hide'` → `'hidden'` on status

A 24h sync window is acceptable: new mentorship programs are not immediately donation-ready, and beneficiaries don't access funds until mid-term.

### Mentorship → CF direct HTTP calls (must exist before DNS cutover)

Mentorship calls these endpoints directly against the CF API URL. Missing any one breaks Mentorship silently at cutover.

| Method | Path | Purpose | Auth |
|---|---|---|---|
| `GET` | `/v1/projects/{id}/{slug}/sync` | Slug sync after rename | public |
| `GET` | `/v1/projects/{id}/funding` | Funding status | public |
| `POST` | `/v1/projects/title-check` | Title uniqueness validation | public |
| `POST` | `/v1/entities/{id}/addbeneficiary` | Add beneficiary | `x-beneficiary-auth` header |
| `POST` | `/v1/entities/{id}/removebeneficiary` | Remove beneficiary | `x-beneficiary-auth` header |

Note: beneficiary endpoints use the fund ID (old entity ID). Mentorship currently calls `/v1/entities/{id}/...` — the new service **must keep this path** for compatibility. The `/v1/funds/{id}/...` path is the new canonical path and both routes will be supported. Mentorship can migrate to the new path in a follow-up update.

---

## Database — PostgreSQL

### Schema: `crowdfunding`

All monetary values `bigint` (cents). All primary keys `uuid`. All timestamps `timestamptz`.

**Terminology:**
- `projects` — CF projects (`campaign_type = 'project'`) and Mentorship campaigns (`campaign_type = 'mentorship'`)
- `funds` — fundraising funds, formerly "entities" (`fund_type`: `general_fund` | `event` | `ostif`)
- `campaign_type` values: `project` | `mentorship`
- `fund_type` values: `general_fund` | `event` | `ostif`
- `status` values: `draft` | `submitted` | `approved` | `published` | `hidden` | `declined`

```sql
-- Projects (CF projects + Mentorship campaigns)
CREATE TABLE crowdfunding.projects (
  id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_type         text NOT NULL,                -- 'project' | 'mentorship'
  owner_id              text NOT NULL,                -- Auth0 subject
  name                  text NOT NULL,
  slug                  text NOT NULL UNIQUE,
  status                text NOT NULL,
  website               text,
  description           text,
  color                 text,
  logo_url              text,
  code_of_conduct       text,
  cii_project_id        text,                         -- project type only
  stacks_id             text,                         -- project type only; deferred feature
  mentorship_program_id text UNIQUE,                  -- mentorship type only; DynamoDB projectId from SQS
  legacy_id             text UNIQUE,                  -- original DynamoDB projectId; for Ledger API calls
  stripe_plan_id        text,                         -- preserved from migration
  stripe_product_id     text,                         -- preserved from migration
  budgets               jsonb NOT NULL DEFAULT '{}',  -- shape differs by campaign_type (see decisions doc)
  github_stats          jsonb NOT NULL DEFAULT '{}',  -- project type only; cached from GitHub API
  beneficiaries         jsonb NOT NULL DEFAULT '[]',
  contributors          jsonb NOT NULL DEFAULT '[]',  -- project type only
  custom_websites       jsonb NOT NULL DEFAULT '[]',  -- project type only
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now()
);

-- Funds (formerly "entities": general_fund, event, ostif)
CREATE TABLE crowdfunding.funds (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  fund_type     text NOT NULL,                        -- 'general_fund' | 'event' | 'ostif'
  owner_id      text NOT NULL,                        -- Auth0 subject
  name          text NOT NULL,
  slug          text NOT NULL UNIQUE,
  status        text NOT NULL,
  legacy_id     text UNIQUE,                          -- original DynamoDB entityId; for Ledger API calls
  description   text,
  website       text,
  logo_url      text,
  budgets       jsonb NOT NULL DEFAULT '{}',
  beneficiaries jsonb NOT NULL DEFAULT '[]',
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Organizations
CREATE TABLE crowdfunding.organizations (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id    text NOT NULL,
  name        text NOT NULL,
  status      text NOT NULL,
  description text,
  website     text,
  logo_url    text,
  approved_at timestamptz,
  rejected_at timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Subscriptions (recurring — projects and funds in one table)
CREATE TABLE crowdfunding.subscriptions (
  id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  stripe_subscription_id text NOT NULL UNIQUE,       -- dedup key; preserved from migration
  project_id             uuid REFERENCES crowdfunding.projects(id),
  fund_id                uuid REFERENCES crowdfunding.funds(id),
  user_id                text NOT NULL,              -- Auth0 subject
  org_id                 uuid REFERENCES crowdfunding.organizations(id),
  frequency              text NOT NULL,              -- 'monthly' | 'annual'
  amount_in_cents        bigint NOT NULL,
  category               text,
  payment_method         text NOT NULL,              -- 'card' | 'invoice'
  status                 text NOT NULL,              -- 'active' | 'cancelled' | 'past_due'
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT project_or_fund CHECK (
    (project_id IS NOT NULL) != (fund_id IS NOT NULL)
  )
);

-- Donations (one-time — projects and funds in one table)
CREATE TABLE crowdfunding.donations (
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  stripe_charge_id text NOT NULL UNIQUE,             -- dedup key; preserved from migration
  project_id       uuid REFERENCES crowdfunding.projects(id),
  fund_id          uuid REFERENCES crowdfunding.funds(id),
  user_id          text NOT NULL,
  org_id           uuid REFERENCES crowdfunding.organizations(id),
  name             text,                             -- display name at time of donation
  avatar_url       text,
  amount_in_cents  bigint NOT NULL,
  category         text,
  payment_method   text NOT NULL,                   -- 'card' | 'invoice'
  po_number        text,
  status           text NOT NULL,
  created_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT project_or_fund CHECK (
    (project_id IS NOT NULL) != (fund_id IS NOT NULL)
  )
);

-- Users (minimal — auth in Auth0, payment account in Stripe)
CREATE TABLE crowdfunding.users (
  id                  text PRIMARY KEY,              -- Auth0 subject
  stripe_customer_id  text,
  github_access_token text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);
```

### View: `project_funding_summary`

**Not active in initial release.** Ledger DB is a separate Postgres instance.
Until Ledger DB is co-located, `amount_raised` and balance data are fetched via
the Ledger HTTP API (`GET /balance/{legacy_id}`) — same as today.

The view is written and committed to migrations but created with `-- INACTIVE` comment.
It is activated as part of the Ledger DB migration (post-initial-release).

Note: `ledger.ledger.project_id` stores the old DynamoDB string ID — match via `legacy_id`, not `id`.

```sql
-- Assumption: l.amount is always stored as a positive integer regardless of txn_type.
-- CREDIT rows increase the balance; DEBIT rows decrease it.
-- If Ledger stores DEBIT amounts as negative, replace balance_cents with SUM(l.amount).
-- Verify against Ledger DB before activating this view.
CREATE VIEW crowdfunding.project_funding_summary AS
SELECT
  p.id                                                                  AS project_id,
  SUM(CASE WHEN l.txn_type = 'CREDIT' THEN l.amount ELSE 0 END)        AS amount_raised_cents,
  SUM(CASE WHEN l.txn_type = 'DEBIT'  THEN l.amount ELSE 0 END)        AS amount_disbursed_cents,
  SUM(CASE WHEN l.txn_type = 'CREDIT' THEN l.amount ELSE -l.amount END) AS balance_cents
FROM crowdfunding.projects p
JOIN ledger.ledger l ON l.project_id = p.legacy_id
GROUP BY p.id;
```

### Indexes

```sql
-- Projects
CREATE INDEX ON crowdfunding.projects (campaign_type);
CREATE INDEX ON crowdfunding.projects (owner_id);
CREATE INDEX ON crowdfunding.projects (status);
CREATE INDEX ON crowdfunding.projects (mentorship_program_id); -- SQS event matching
CREATE INDEX ON crowdfunding.projects (legacy_id);             -- Ledger API lookups

-- Funds
CREATE INDEX ON crowdfunding.funds (fund_type);
CREATE INDEX ON crowdfunding.funds (owner_id);
CREATE INDEX ON crowdfunding.funds (status);
CREATE INDEX ON crowdfunding.funds (legacy_id);

-- Subscriptions / Donations
CREATE INDEX ON crowdfunding.subscriptions (user_id);
CREATE INDEX ON crowdfunding.subscriptions (project_id);
CREATE INDEX ON crowdfunding.subscriptions (fund_id);
CREATE INDEX ON crowdfunding.donations (user_id);
CREATE INDEX ON crowdfunding.donations (project_id);
CREATE INDEX ON crowdfunding.donations (fund_id);

-- Full-text search (replaces OpenSearch for discovery)
CREATE INDEX ON crowdfunding.projects
  USING gin(to_tsvector('english', name || ' ' || coalesce(description, '')));
CREATE INDEX ON crowdfunding.funds
  USING gin(to_tsvector('english', name || ' ' || coalesce(description, '')));
```

---

## Services That Remain Unchanged (initial release)

| Service | Location | Notes |
|---|---|---|
| Ledger Service | AWS Lambda | Unchanged. Own Postgres (Ledger DB). CF calls it read-only via HTTP. |
| Ledger DB | AWS RDS / Postgres | Separate from Crowdfunding DB. Migrated post-initial-release. |
| Reimbursement Service | AWS Lambda | On CF release: switches CF data reads to CF Postgres (read-only). Own tables (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`) migrate to `reimbursement` schema on CF Postgres within 2 weeks of CF release. |
| Mentorship (jobspring) | AWS Lambda | Unchanged. Publishes to SNS. Calls CF API directly. |
| Old LFF Lambda | AWS Lambda | Runs in parallel until cutover. Keeps OpenSearch fed for Reimbursement Service. |
| DynamoDB tables | AWS DynamoDB | Read during migration. Kept until decommission is confirmed safe. |
| OpenSearch | AWS OpenSearch | Kept alive until RS Category 2 migration completes (CF release + 2 weeks). Decommissioned at that point. |

---

## Deployment — Kubernetes (all NEW components)

Every component inside the "NEW" purple box is deployed to Kubernetes via ArgoCD.
Nothing in the initial release runs on Lambda or Serverless Framework.

| Component | K8s Resource | Notes |
|---|---|---|
| Nuxt 3 frontend | `Deployment` + `Service` + `Ingress` | TLS termination at Ingress |
| Go HTTP API | `Deployment` + `Service` + `Ingress` | Chi router, long-running |
| Crowdfunding Postgres | `StatefulSet` or managed Postgres | Confirm with DevOps — RDS vs in-cluster |
| SQS consumer | `Deployment` | Long-running poll loop, not a CronJob |
| GitHub stats job | `CronJob` | Every 6 hours |
| Secrets | K8s `Secret` or external secrets operator | Confirm LFX standard with DevOps |
| ArgoCD app | New entry in `linuxfoundation/lfx-v2-argocd` | See OQ-5 |

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
- `entities` table name — replaced by `funds`
- `initiative` and `travel_fund` fund types — merged into `general_fund`
- `community` entity type — 3 dead rows discarded
- Datadog RUM — deferred
- Intercom — deferred
- Stacks / security vulnerabilities — deferred
- Sponsor Tiers — deferred
- CII badge — deferred
