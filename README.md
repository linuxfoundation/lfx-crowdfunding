# LFX Crowdfunding (Standalone UI & API)

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![DCO](https://img.shields.io/badge/DCO-required-brightgreen.svg)](https://developercertificate.org/)

LFX Crowdfunding enables open source projects to raise funds for development, security audits, mentorship programs, events, and community initiatives. This is the rewritten platform — a Kubernetes-native monorepo replacing the original AWS Lambda + DynamoDB system.

---

> ### Not the Self Serve integration
> This repo is the **standalone Crowdfunding UI and API** — it is separate from [LFX Self Serve](https://github.com/linuxfoundation/lfx-v2-ui). Crowdfunding data ("My Donations", "My Initiatives") that appears in Self Serve originates from this application.

---

## Repository Layout

```
lfx-crowdfunding/
├── frontend/          # Nuxt 3 frontend (Vue 3, TypeScript, Tailwind, PrimeVue)
├── cmd/
│   ├── api/           # Go HTTP API entrypoint
│   ├── mentorship-sync/   # CronJob: syncs mentorship data from Snowflake
│   ├── ledger-stats-sync/ # CronJob: syncs financial stats from Ledger HTTP API
│   └── migrate/       # DB migration runner (golang-migrate)
├── internal/          # Go domain packages (DDD: domain/, usecases/, repository/)
├── services/          # Go external service clients (Stripe, Mandrill, GitHub, Ledger, Snowflake)
├── db/
│   ├── migrations/    # SQL migration files (golang-migrate)
│   └── scripts/       # One-time DynamoDB → Postgres migration (Python)
└── docs/
    └── rewrite/       # Architecture decisions, open questions, migration plan
```

## Architecture

The platform is split into two independently deployable services:

**Frontend** — Nuxt 3 BFF. Handles Auth0 PKCE authentication, HTTP-only session cookies, and Stripe.js. Calls the Go API to serve pages.

**Go HTTP API** — Chi router. Owns all business logic: initiative CRUD, Stripe payment processing, webhook handling, email, and read-only Ledger integration.

Both are deployed as Kubernetes Deployments behind an Ingress. Background jobs run as K8s CronJobs.

See [`docs/rewrite/04-target-architecture.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/docs/rewrite/04-target-architecture.md) for the full system diagram and component breakdown.

## Documentation

| Document | Contents |
|---|---|
| [`docs/rewrite/01-current-system.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/docs/rewrite/01-current-system.md) | Inventory of the current Lambda system — endpoints, DynamoDB tables, integrations |
| [`docs/rewrite/02-decisions.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/docs/rewrite/02-decisions.md) | All architectural decisions with rationale |
| [`docs/rewrite/03-open-questions.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/docs/rewrite/03-open-questions.md) | Open questions with owners and blocking status |
| [`docs/rewrite/04-target-architecture.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/docs/rewrite/04-target-architecture.md) | Target system design — tech stack, repo layout, API surface, K8s resources |
| [`docs/rewrite/05-migration-plan.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/docs/rewrite/05-migration-plan.md) | Step-by-step migration and cutover plan |

## Tech Stack

### Frontend

| Concern | Choice |
|---|---|
| Framework | Nuxt 3 + Vue 3 |
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
| Database | PostgreSQL via `sqlc` + `pgx/v5` |
| Migrations | `golang-migrate` |
| Auth | Auth0 JWT middleware |
| Logging | `slog` (stdlib) |

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
| `mentorship-sync` | Daily | Pulls mentorship program data from Snowflake into CF Postgres |
| `ledger-stats-sync` | Hourly | Syncs financial stats from Ledger HTTP API into cached columns on `initiatives` |

## Database

The `crowdfunding` schema lives on the shared LFX v2 RDS instance. The initial migration file is at [`db/migrations/001_initial.up.sql`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/db/migrations/001_initial.up.sql).

One-time DynamoDB → Postgres data migration script: [`db/scripts/migrate_dynamo_to_postgres.py`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/db/scripts/migrate_dynamo_to_postgres.py).

See [`docs/rewrite/05-migration-plan.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/docs/rewrite/05-migration-plan.md) for cutover procedure.

## Development Setup

### Frontend

```bash
cd frontend
pnpm install
cp .env.example .env.local   # fill in Auth0, Stripe, API URL
pnpm dev
```

Required env vars (server-side): `NUXT_AUTH0_CLIENT_SECRET`, `NUXT_JWT_SECRET`, `NUXT_API_URL`, `NUXT_STRIPE_SECRET_KEY`, `NUXT_GITHUB_OAUTH_CLIENT_SECRET`

Public env vars: `NUXT_PUBLIC_AUTH0_DOMAIN`, `NUXT_PUBLIC_AUTH0_CLIENT_ID`, `NUXT_PUBLIC_STRIPE_PUBLIC_KEY`, `NUXT_PUBLIC_APP_ENV`

### Backend

```bash
# Run DB migrations
go run ./cmd/migrate

# Start API server
go run ./cmd/api
```

Required env vars: see [`docs/rewrite/04-target-architecture.md`](https://github.com/linuxfoundation/lfx-crowdfunding/blob/main/docs/rewrite/04-target-architecture.md) for the full inventory.

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
