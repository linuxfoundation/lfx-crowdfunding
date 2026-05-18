# LFX Crowdfunding — Architecture

This document describes the architecture of the rewritten LFX Crowdfunding platform. It reflects the target system: a Kubernetes-native monorepo replacing the original AWS Lambda + DynamoDB stack.

---

## System Overview

```mermaid
graph TB
    subgraph Users
        U[Donor / Beneficiary / Admin]
    end

    subgraph K8s["Kubernetes (LFX v2 cluster)"]
        FE["Nuxt 3 Frontend\nDeployment + Ingress"]
        API["Go HTTP API\nDeployment + Ingress"]
        LSS["ledger-stats-sync\nCronJob (hourly)"]
        DB[("PostgreSQL\ncrowdfunding schema")]

        FE -- "$fetch HTTPS" --> API
        API --> DB
        LSS --> DB
    end

    subgraph External["External Services"]
        AUTH0[Auth0]
        STRIPE[Stripe]
        LEDGER["Ledger Service\n(AWS Lambda)"]
        RS["Reimbursement Service\n(AWS Lambda)"]
        MENTORSHIP["Mentorship Service\n(AWS Lambda / jobspring)"]
        MANDRILL[Mandrill]
        GITHUB[GitHub API]
    end

    U --> FE
    FE -- "PKCE auth" --> AUTH0
    API -- "JWT validation" --> AUTH0
    API -- "payments" --> STRIPE
    STRIPE -- "webhook" --> API
    API -- "balance + transactions\n(read-only)" --> LEDGER
    LSS -- "batch balance sync" --> LEDGER
    API <--> RS
    API <--> MENTORSHIP
    API -- "transactional email" --> MANDRILL
    API -- "repo stats" --> GITHUB
```

---

## Components

### Frontend — Nuxt 3

Server-side rendered Vue 3 application. Acts as a BFF: handles authentication, session cookies, and Stripe.js. All data fetched from the Go API.

| Concern | Choice |
|---|---|
| Framework | Nuxt 3 + Vue 3 |
| Language | TypeScript (strict) |
| Styling | Tailwind CSS + PrimeVue v4 |
| State | Pinia (app state) + Vue Query (server state) |
| Auth | OAuth2 PKCE, HTTP-only session cookies |
| Payments | Stripe.js |

**Authentication flow:**

1. User clicks login → `GET /api/auth/login` → server generates PKCE challenge → returns Auth0 redirect URL
2. Auth0 authenticates → redirects to `/auth/callback`
3. Server exchanges code for tokens → stores in HTTP-only cookies
4. All API calls include `credentials: 'include'` — token sent automatically

**Pages:**

```
pages/
├── index.vue                  # Initiative discovery (listing)
├── auth/callback.vue          # Auth0 OAuth callback
├── stripe/callback.vue        # Stripe OAuth callback
├── github/callback.vue        # GitHub OAuth callback
├── email/
│   ├── approve.vue            # Approve expense (email JWT link)
│   ├── reject.vue             # Reject expense (email JWT link)
│   └── approve-project.vue    # Approve initiative (email JWT link)
├── projects/
│   ├── create/                # GitHub OAuth → repo select → details form
│   └── [slug]/
│       ├── index.vue          # Project overview
│       ├── financial.vue      # Donations & expenses
│       ├── edit.vue           # Edit project
│       └── payments.vue       # Donate / subscribe
└── funds/
    ├── create/                # General fund / event / OSTIF creation form
    └── [slug]/
        ├── index.vue          # Fund overview
        ├── financial.vue      # Donations & expenses
        ├── edit.vue           # Edit fund
        └── payments.vue       # Donate / subscribe
```

---

### Backend — Go HTTP API

Chi router. Owns all business logic: initiative CRUD, Stripe payments, webhook processing, transactional email, and read-only Ledger integration. Structured as a layered DDD application.

| Concern | Choice |
|---|---|
| Language | Go (latest stable) |
| Router | Chi |
| Database | PostgreSQL via `pgx/v5` |
| Migrations | `golang-migrate` |
| Auth | Auth0 JWT middleware |
| Logging | `slog` (stdlib) |
| Tracing | OpenTelemetry |

**Package layout:**

```
backend/
├── cmd/
│   ├── initiatives-api/     # HTTP server entrypoint
│   └── ledger-stats-sync/   # CronJob entrypoint
├── internal/
│   ├── domain/              # Domain models + repository interfaces
│   ├── service/             # Business logic / orchestration
│   ├── handler/             # HTTP handlers
│   └── infrastructure/
│       ├── db/              # PostgreSQL repository implementations
│       ├── clients/         # Ledger + Stripe HTTP clients
│       └── auth/            # JWT middleware
└── db/
    ├── migrations/          # golang-migrate SQL files
    └── seed.sql             # Development seed data
```

**API surface:**

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/initiatives` | List initiatives (filterable, paginated) |
| `POST` | `/v1/initiatives` | Create initiative |
| `GET` | `/v1/initiatives/{id}` | Get initiative by UUID or slug |
| `PUT` | `/v1/initiatives/{id}` | Update initiative |
| `DELETE` | `/v1/initiatives/{id}` | Delete initiative |
| `GET` | `/v1/initiatives/{id}/transactions` | Donations and expenses |
| `POST` | `/v1/initiatives/{id}/payment-intent` | Create Stripe payment intent |
| `POST` | `/v1/initiatives/{id}/subscription` | Create Stripe subscription |
| `DELETE` | `/v1/subscriptions/{id}` | Cancel subscription |
| `POST` | `/v1/hooks/stripe` | Stripe webhook receiver |

**Mentorship compatibility endpoints** (called directly by the Mentorship service):

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/v1/projects/{id}/{slug}/sync` | Slug sync after rename |
| `GET` | `/v1/projects/{id}/funding` | Funding status |
| `POST` | `/v1/projects/title-check` | Title uniqueness validation |
| `POST` | `/v1/entities/{id}/addbeneficiary` | Add beneficiary |
| `POST` | `/v1/entities/{id}/removebeneficiary` | Remove beneficiary |

---

### Background Jobs

| Job | K8s resource | Schedule | Purpose |
|---|---|---|---|
| `ledger-stats-sync` | CronJob | Hourly | Fetches balance and sponsor data from Ledger HTTP API; caches into `initiative_ledger_stats` |

---

### Database — PostgreSQL

`crowdfunding` schema on the shared LFX v2 RDS instance. All monetary values stored as `bigint` (cents). All primary keys are UUIDs.

**Core tables:**

| Table | Purpose |
|---|---|
| `initiatives` | Unified table for all initiative types (project, event, mentorship, general_fund, security_audit, ostif, other) |
| `initiative_goals` | Funding goals per initiative; donated/spent enriched live from Ledger |
| `initiative_ledger_stats` | Hourly-cached financial stats and sponsors (written by CronJob) |
| `initiative_beneficiaries` | Beneficiaries linked to an initiative |
| `initiative_contributors` | Contributors (project type only) |
| `initiative_mentors` | Mentors (mentorship type only) |
| `users` | LFX user identity; Auth0 subject as primary key |
| `organizations` | Donor organizations |
| `donations` | One-time donation records |
| `subscriptions` | Recurring subscription records |

**initiative_type values:**

| Type | Description |
|---|---|
| `project` | Open source software project |
| `mentorship` | Mentorship program (managed by Mentorship service) |
| `event` | Conference or community event |
| `general_fund` | General-purpose fundraising fund |
| `security_audit` | OSTIF security audit |
| `ostif` | Legacy OSTIF type (migrated rows only) |
| `other` | Legacy general type (migrated rows only) |

**Financial data flow:**

```
Ledger Service (Lambda)
        │
        │  GET /api/balance/{projectID}
        ▼
ledger-stats-sync CronJob (hourly)
        │
        │  writes total_raised, available_balance,
        │  supporters, sponsors JSONB
        ▼
initiative_ledger_stats
        │
        │  JOIN on every initiative read
        ▼
GET /v1/initiatives/{id}  ←── also calls Ledger live
                               for per-goal donated/spent
```

---

## Data Flows

### User Authentication

```mermaid
sequenceDiagram
    actor User
    participant FE as Nuxt Frontend
    participant Auth0

    User->>FE: Click "Sign In"
    FE->>FE: Generate PKCE code_verifier + code_challenge
    FE-->>User: Redirect to Auth0 /authorize
    User->>Auth0: Credentials
    Auth0-->>FE: Redirect /auth/callback?code=...
    FE->>Auth0: Exchange code + code_verifier for tokens
    Auth0-->>FE: access_token + id_token
    FE->>FE: Store tokens in HTTP-only cookies
    FE-->>User: Authenticated session
```

Tokens never touch the browser directly — stored server-side in HTTP-only cookies. Subsequent API calls from the frontend automatically include the token via `credentials: 'include'`. The Go API validates the JWT on every protected request against Auth0's JWKS endpoint.

---

### Initiative Detail Page Load

Every request for an initiative detail page triggers two parallel data sources: a DB read for initiative fields + cached financials, and a live Ledger call for per-goal donated/spent.

```mermaid
sequenceDiagram
    actor User
    participant FE as Nuxt Frontend
    participant API as Go API
    participant DB as PostgreSQL
    participant Ledger as Ledger Service

    User->>FE: GET /projects/kubernetes
    FE->>API: GET /v1/initiatives/kubernetes
    API->>DB: SELECT initiatives + initiative_ledger_stats\nWHERE slug = 'kubernetes'
    DB-->>API: initiative row + cached financials + sponsors JSONB
    API->>DB: SELECT initiative_goals WHERE initiative_id = ?
    DB-->>API: goals[]
    API->>Ledger: GET /api/balance/{projectID}
    Ledger-->>API: balance + per-category subTotals
    API->>API: Enrich each goal with donated_cents / spent_cents\nfrom Ledger subTotals (case-insensitive match)
    API->>API: Flatten + sort sponsors from cached JSONB
    API-->>FE: Initiative JSON (ETag + Cache-Control: max-age=60)
    FE-->>User: Rendered page
```

**Caching:** The API response includes `Cache-Control: public, max-age=60, stale-while-revalidate=300` and an `ETag`. Ledger unavailability is non-fatal — goals are returned with zero donated/spent rather than failing the request.

---

### One-Time Donation

```mermaid
sequenceDiagram
    actor Donor
    participant FE as Nuxt Frontend
    participant API as Go API
    participant DB as PostgreSQL
    participant Stripe

    Donor->>FE: Fill donation form, click Pay
    FE->>API: POST /v1/initiatives/{id}/donations\n{amount_cents, category, payment_method}
    API->>DB: INSERT INTO donations (status = 'pending')
    DB-->>API: donation record
    API->>Stripe: Create PaymentIntent\n(amount, currency, metadata.initiative_id)
    Stripe-->>API: client_secret
    API-->>FE: {donation_id, client_secret}
    FE->>Stripe: stripe.confirmPayment(client_secret)
    Stripe-->>FE: Payment result
    FE-->>Donor: Confirmation screen
    Stripe->>API: POST /v1/stripe/webhook\n(payment_intent.succeeded)
    API->>API: Validate Stripe-Signature header
    API->>DB: UPDATE donations SET status = 'succeeded'\nWHERE stripe_charge_id = ?
```

The Ledger Service also receives a Stripe webhook independently and records the transaction in its own database. CF reads balance data from Ledger — it does not maintain its own running balance.

---

### Recurring Subscription

```mermaid
sequenceDiagram
    actor Donor
    participant FE as Nuxt Frontend
    participant API as Go API
    participant DB as PostgreSQL
    participant Stripe

    Donor->>FE: Choose monthly/annual, click Subscribe
    FE->>API: POST /v1/initiatives/{id}/subscriptions\n{amount_cents, frequency, payment_method}
    API->>Stripe: Create Customer (if new)\nCreate Subscription (price, metadata)
    Stripe-->>API: subscription_id + latest_invoice.payment_intent.client_secret
    API->>DB: INSERT INTO subscriptions (status = 'active')
    API-->>FE: {subscription_id, client_secret}
    FE->>Stripe: stripe.confirmPayment(client_secret)
    Stripe-->>FE: Confirmed

    Note over Stripe,API: Later — donor cancels or card expires
    Stripe->>API: POST /v1/stripe/webhook\n(customer.subscription.deleted)
    API->>API: Validate Stripe-Signature header
    API->>DB: UPDATE subscriptions SET status = 'cancelled'
```

---

### Financial Stats Sync (CronJob)

The `ledger-stats-sync` CronJob runs hourly and pre-warms the `initiative_ledger_stats` cache so initiative list and detail responses do not need a live Ledger call for aggregate financial figures.

```mermaid
sequenceDiagram
    participant Cron as ledger-stats-sync\n(K8s CronJob)
    participant Ledger as Ledger Service
    participant DB as PostgreSQL

    Cron->>Ledger: GET /api/balance (all projects)
    Ledger-->>Cron: [{projectID, total_raised, balance, sponsors[]}]
    loop For each initiative
        Cron->>DB: UPSERT initiative_ledger_stats\n(total_raised_cents, available_balance_cents,\nsupporters, sponsors JSONB)
    end
    Note over DB: initiative detail reads JOIN\nthis table — no live Ledger call\nneeded for aggregate figures
```

---

### Transaction History (Donations & Expenses)

Transaction data lives in the Ledger Service, not in CF's database. The CF API proxies and enriches it with donor names and avatars from the CF DB.

```mermaid
sequenceDiagram
    actor User
    participant FE as Nuxt Frontend
    participant API as Go API
    participant DB as PostgreSQL
    participant Ledger as Ledger Service

    User->>FE: Open "Donations" tab
    FE->>API: GET /v1/initiatives/{id}/transactions?type=donations
    API->>DB: SELECT id FROM initiatives WHERE slug = ? AND status = 'published'
    DB-->>API: initiative UUID
    API->>Ledger: GET /api/transactions\n{projectID, type: 'donation', page, size}
    Ledger-->>API: [{txn_id, amount, date, user_id, org_id}]
    API->>DB: SELECT * FROM users WHERE user_id = ANY(?)
    API->>DB: SELECT * FROM organizations WHERE id = ANY(?)
    DB-->>API: user + org records
    API->>API: Merge donor name + avatar onto each transaction\n(org takes priority over user;\ngenerate avatar URL if no DB record)
    API-->>FE: Enriched transaction list
    FE-->>User: Donations table
```

---

### Mentorship Program Sync

Mentorship programs in LFX are created and managed by the Mentorship service (jobspring). CF mirrors them as `mentorship`-type initiatives so they appear in the Crowdfunding UI and can receive donations.

```mermaid
sequenceDiagram
    participant MS as Mentorship Service\n(jobspring Lambda)
    participant SNS as AWS SNS
    participant SQS as AWS SQS
    participant CF as CF Go API
    participant DB as PostgreSQL

    MS->>SNS: Publish projectCreated / projectUpdated / projectUpdateStatus
    SNS->>SQS: Fan-out to CF queue
    CF->>SQS: Poll (SQS consumer — long-running Deployment)
    SQS-->>CF: Event message

    alt projectCreated
        CF->>DB: INSERT INTO initiatives\n(initiative_type='mentorship', jobspring_project_id=?)
    else projectUpdated
        CF->>DB: UPDATE initiatives\n(Mentorship-owned fields only;\nnever overwrite logo, color, description)
    else projectUpdateStatus
        CF->>DB: UPDATE initiatives SET status = ?
    end

    Note over MS,CF: Mentorship also calls CF directly (HTTP)\nfor slug sync, funding status, title checks,\nand beneficiary management
```

---

## External Integrations

| Service | Direction | Purpose |
|---|---|---|
| Auth0 | CF → Auth0 | JWT validation; user identity |
| Stripe | CF → Stripe | Charges, subscriptions, Stripe Connect |
| Stripe webhook | Stripe → CF | `customer.subscription.deleted` → cancel in DB |
| Ledger Service | CF → Ledger (read-only) | Balance, per-goal subtotals, transaction history |
| Ledger Service | Ledger → CF | Donation callbacks (`GET /v1/projects/{id}`) |
| Reimbursement Service | Bidirectional | Expense policy, beneficiary lifecycle |
| Mentorship Service | Bidirectional | Program sync via SNS/SQS + direct HTTP calls |
| Mandrill | CF → Mandrill | Transactional email |
| GitHub | CF → GitHub | Repo stats; OAuth for project creation |

---

## Deployment

All application components run in Kubernetes, deployed via ArgoCD from [`linuxfoundation/lfx-v2-argocd`](https://github.com/linuxfoundation/lfx-v2-argocd).

| Component | K8s resource |
|---|---|
| Nuxt 3 frontend | `Deployment` + `Service` + `Ingress` |
| Go HTTP API | `Deployment` + `Service` + `Ingress` |
| `ledger-stats-sync` | `CronJob` |
| PostgreSQL | Managed RDS (shared LFX v2 instance) |
| Secrets | External Secrets Operator |

**URLs:**

| Environment | URL |
|---|---|
| Dev | `https://funding.dev.platform.linuxfoundation.org/` |
| Prod | `https://crowdfunding.lfx.linuxfoundation.org/` |

---

## What Was Intentionally Removed

The rewrite drops the following from the original Lambda system:

| Removed | Replaced by |
|---|---|
| AWS Lambda (application code) | Kubernetes Deployments |
| DynamoDB | PostgreSQL |
| OpenSearch | Postgres full-text search |
| Serverless Framework | Helm charts + ArgoCD |
| CloudWatch Events / DynamoDB Streams | K8s CronJobs |
| `travel_fund` initiative type | Merged into `general_fund` |
| `community` initiative type | 3 dead rows discarded at migration |
