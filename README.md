# LFX Crowdfunding (Standalone UI & API)

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![DCO](https://img.shields.io/badge/DCO-required-brightgreen.svg)](https://developercertificate.org/)

LFX Crowdfunding enables open source projects to raise funds for development, security audits, mentorship programs, events, and community initiatives. This is the rewritten platform — a Kubernetes-native monorepo replacing the original AWS Lambda + DynamoDB system.

---

> ### Not the Self Serve integration
> This repo is the **standalone Crowdfunding UI and API** — it is separate from [LFX Self Serve](https://github.com/linuxfoundation/lfx-v2-ui). Crowdfunding data ("My Donations", "My Initiatives") that appears in Self Serve originates from this application.

---

## Repository Layout

```text
lfx-crowdfunding/
├── backend/                    # Go API service
│   ├── cmd/
│   │   ├── initiatives-api/    # HTTP API entrypoint
│   │   └── ledger-stats-sync/  # CronJob: syncs financial stats from Ledger HTTP API
│   ├── internal/
│   │   ├── domain/             # Domain models and repository interfaces
│   │   ├── service/            # Business logic / orchestration
│   │   ├── handler/            # HTTP handlers (Chi router)
│   │   └── infrastructure/     # DB, external clients, auth middleware
│   ├── db/
│   │   ├── migrations/         # SQL migration files (golang-migrate)
│   │   ├── scripts/            # One-time DynamoDB → Postgres migration (Python)
│   │   └── seed.sql            # Development seed data
│   └── charts/                 # Helm chart
├── frontend/                   # Nuxt 4 frontend (Vue 3, TypeScript, Tailwind, PrimeVue)
├── docker-compose.yml          # Local Postgres
└── backend/docs/
    └── rewrite/                # Architecture decisions, open questions, migration plan
```

## Architecture

The platform is split into two independently deployable services:

**Frontend** — Nuxt 4 BFF. Handles Auth0 PKCE authentication, HTTP-only session cookies, and Stripe.js. Calls the Go API to serve pages.

**Go HTTP API** — Chi router. Owns all business logic: initiative CRUD, Stripe payment processing, webhook handling, email, and read-only Ledger integration.

Both are deployed as Kubernetes Deployments behind an Ingress. Background jobs run as K8s CronJobs.

See [`docs/architecture.md`](docs/architecture.md) for the full system diagram and component breakdown.

## Documentation

| Document | Contents |
|---|---|
| [`backend/docs/rewrite/01-current-system.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/backend/docs/rewrite/01-current-system.md) | Inventory of the current Lambda system — endpoints, DynamoDB tables, integrations |
| [`backend/docs/rewrite/02-decisions.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/backend/docs/rewrite/02-decisions.md) | All architectural decisions with rationale |
| [`backend/docs/rewrite/03-open-questions.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/backend/docs/rewrite/03-open-questions.md) | Open questions with owners and blocking status |
| [`backend/docs/rewrite/04-target-architecture.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/backend/docs/rewrite/04-target-architecture.md) | Target system design — tech stack, repo layout, API surface, K8s resources |
| [`backend/docs/rewrite/05-migration-plan.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/backend/docs/rewrite/05-migration-plan.md) | Step-by-step migration and cutover plan |

## Tech Stack

### Frontend

| Concern | Choice |
|---|---|
| Framework | Nuxt 4 + Vue 3 |
| Language | TypeScript (strict) |
| Styling | Tailwind CSS + PrimeVue v4 |
| State | Pinia + Vue Query |
| Auth | OAuth2 PKCE, HTTP-only cookies |
| Payments | Stripe.js |

### Backend

| Concern | Choice |
|---|---|
| Language | Go (latest stable) |
| Router | Chi |
| Database | PostgreSQL via `pgx/v5` |
| Migrations | `golang-migrate` |
| Auth | Auth0 JWT middleware |
| Logging | `slog` (stdlib) |
| Tracing | OpenTelemetry |

### Infrastructure

- **Kubernetes** (LFX v2 shared cluster) — Deployments, CronJobs, Ingress
- **PostgreSQL** — shared LFX v2 RDS instance, `crowdfunding` schema
- **Snowflake** — mentorship program sync (inbound via CronJob); CF data sync to LFX Self Serve (outbound via Fivetran)
- **ArgoCD** — GitOps deployment via `linuxfoundation/lfx-v2-argocd`

## Key Integrations

| Service | Direction | Purpose |
|---|---|---|
| Stripe | CF → Stripe | Charges, subscriptions, Stripe Connect |
| Stripe webhook | Stripe → CF | `customer.subscription.deleted` |
| Ledger Service | CF → Ledger (read-only) | Balance and transaction data |
| Ledger Service | Ledger → CF | Donation email callbacks (`GET /v1/projects/{id}`) |
| Reimbursement Service | Bidirectional | Expense policy management, beneficiary lifecycle |
| Mandrill | CF → Mandrill | Transactional email (approvals, rejections, invoices) |
| GitHub | CF → GitHub | Repo stats, OAuth for project creation |
| Snowflake | Snowflake → CF | Mentorship program sync (inbound CronJob) |
| Snowflake | CF → Snowflake | Fivetran CF→Snowflake sync for LFX Self Serve |
| Auth0 | CF → Auth0 | JWT validation, user identity |

## Background Jobs

| Job | Schedule | Purpose |
|---|---|---|
| `ledger-stats-sync` | Hourly | Syncs financial stats (balance, sponsors) from Ledger HTTP API into cached columns on `initiatives` |

## Database

The `crowdfunding` schema lives on the shared LFX v2 RDS instance. The initial migration file is at [`backend/db/migrations/001_initial.up.sql`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/backend/db/migrations/001_initial.up.sql).

One-time DynamoDB → Postgres data migration script: [`backend/db/scripts/migrate_dynamo_to_postgres.py`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/backend/db/scripts/migrate_dynamo_to_postgres.py).

See [`backend/docs/rewrite/05-migration-plan.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/backend/docs/rewrite/05-migration-plan.md) for cutover procedure.

## Development Setup

### Prerequisites

- Go (latest stable)
- Node.js 22+ and pnpm 9+
- Docker (for Postgres)

### 1. Start Postgres

```bash
docker compose up -d
```

### 2. Backend

```bash
cd backend
cp .env.example .env   # then fill in values — see below
psql "$DATABASE_URL" -f db/migrations/001_initial.up.sql
go run ./cmd/initiatives-api/
```

To seed development data after the DB schema is applied:

```bash
make db-seed
```

**Required env vars:**

| Var | Notes |
|-----|-------|
| `DATABASE_URL` | `postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding` (matches docker-compose) |
| `ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS` | Must be set to `true` to enable `DISABLED_MOCK_LOCAL_PRINCIPAL` |
| `DISABLED_MOCK_LOCAL_PRINCIPAL` | Set to any non-empty string to skip JWT validation locally (requires `ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS=true`) |
| `STRIPE_SECRET_KEY` | Stripe test key |
| `STRIPE_WEBHOOK_SECRET` | Stripe test webhook secret |
| `STRIPE_RETURN_URL` | Frontend URL Stripe redirects to after 3DS (e.g. `http://localhost:3000/payment/complete`) |
| `LEDGER_BASE_URL` | Ledger service URL |
| `LEDGER_API_KEY` | Ledger API key |
| `FRONTEND_BASE_URL` | Frontend base URL for email links (e.g. `http://localhost:3000`) |
| `S3_UPLOAD_BUCKET` | S3 bucket name for logo uploads |
| `S3_REGION` | AWS region for S3 bucket |

`JWKS_URL` and `DISABLED_MOCK_LOCAL_PRINCIPAL` are mutually exclusive — the server rejects startup if both are set. When using the mock principal locally, leave `JWKS_URL` unset or empty. Both `ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS=true` and `DISABLED_MOCK_LOCAL_PRINCIPAL` must be set together to enable the bypass.

### 3. Frontend

Node 22 is required (pnpm 9+ and the husky hooks need it):

```bash
nvm use 22
cd frontend
pnpm install
cp .env.example .env   # then fill in values — see below
pnpm dev
```

**Required env vars:**

| Var | Notes |
|-----|-------|
| `NUXT_PUBLIC_AUTH0_CLIENT_ID` | Auth0 client ID for the dev tenant |
| `NUXT_AUTH0_CLIENT_SECRET` | Auth0 client secret for the dev tenant |
| `NUXT_JWT_SECRET` | Any random string for local session signing |

Auth0 domain is set automatically to `linuxfoundation-dev.auth0.com` when `NUXT_APP_ENV=development` (the default).

## Contributing

All commits must be signed off per the [Developer Certificate of Origin](https://developercertificate.org/):

```bash
git commit --signoff
```

See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

- Code: [MIT](LICENSE)
- Documentation: [MIT](LICENSE-docs)

Copyright The Linux Foundation and each contributor to LFX.
