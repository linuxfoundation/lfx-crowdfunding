# Current System Inventory

Source repos explored: `LFF` (backend), `lfx-crowdfunding-upgrade` (frontend),
`ledger-service`, `reimbursement-service`, `jobspring` (mentorship), `lfx-v1-sync-helper`.

---

## Production Data Inventory (as of May 2026)

Total: **2,013 rows** across projects and entities tables. **1,374 published** (active/live).

| Source | Type | Submitted | Published | Declined | Hidden | Total |
|---|---|---|---|---|---|---|
| projects | mentorship | 108 | 1,249 | 21 | 98 | 1,476 |
| projects | project | 171 | 82 | 99 | 3 | 356 |
| entities | initiative | 81 | 24 | 15 | 1 | 121 |
| entities | other (travel) | 3 | 8 | 2 | 13 | 26 |
| entities | event | 6 | 8 | 6 | — | 20 |
| entities | ostif | 7 | 3 | 1 | — | 11 |
| entities | community | 2 | 0 | 1 | — | 3 |
| **Total** | | **378** | **1,374** | **145** | **115** | **2,013** |

### Key observations

- **Mentorship projects dominate** — 1,476 of 1,832 project rows (80%) are mentorship-type. These come from the Mentorship service, not from Crowdfunding users directly. In the new system they are synced from Snowflake via a K8s CronJob (SNS/SQS is not used).
- **"community" entity type** — 3 rows with type `community` not seen in the Go code's entity type enum (`project`, `initiative`, `general-fund`). Must be handled in migration (map to a known type or add `community` to the enum). **Needs clarification.**
- **"other (travel)" entity type** — likely `general-fund` subtype used for travel funds. Confirm the exact DynamoDB type value.
- **Non-published records matter** — 639 non-published rows (submitted, declined, hidden). Migration must include all statuses, not just published. Active subscriptions may exist against non-published projects.
- **Small entity volume** — 181 entity rows total; straightforward to migrate and validate manually if needed.

---

## Backend (LFF — Go + AWS Lambda + DynamoDB)

### Module & Runtime

- Module: `github.com/LF-Engineering/LFF/backend`
- Go 1.21.1
- Runtime: AWS Lambda (`provided.al2023`)
- Framework: Serverless Framework v2
- Router: Chi (`github.com/go-chi/chi`)
- DI: hand-rolled per domain (`*/di/init.go`)
- Architecture: Domain-Driven Design — each domain has `domain/`, `usecases/`, `interfaces/repository/`, `interfaces/restapi/`

---

## HTTP Endpoints

All under `/v1/`. Protected routes require Auth0 JWT via the `authorizer` Lambda.

### Projects

| Method | Path | Handler | Auth |
|---|---|---|---|
| `POST` | `/v1/me/projects` | CreateProjectHandler | required |
| `GET` | `/v1/me/projects` | PrivateProjectsHandler | required |
| `GET` | `/v1/me/projects/{projectIdOrSlug}` | PrivateProjectDetailsHandler | required |
| `PATCH` | `/v1/me/projects/{projectId}` | UpdateProjectHandler | required |
| `POST` | `/v1/me/projects/{projectId}/status/{status}` | HideUnhideProjectStatusHandler | required |
| `GET` | `/v1/projects` | PublicProjectsHandler | public |
| `GET` | `/v1/projects/{projectIdOrSlug}` | PublicProjectDetailsHandler | public |
| `GET` | `/v1/projects/with-subscriptions` | PublicProjectsWithSubHandler | public |
| `GET` | `/v1/projects/with-subscriptions/{projectIdOrSlug}` | PublicProjectDetailsWithSubHandler | public |
| `GET` | `/v1/projects/cache` | PublicProjectsCacheHandler | public |
| `GET` | `/v1/projects/cache/{projectIdOrSlug}` | PublicCacheProjectHandler | public |
| `GET` | `/v1/projects/cache/run` | PublicProjectsRunCacheHandler | public |
| `GET` | `/v1/projects/paginate` | PaginateProjectsCacheHandler | public |
| `GET` | `/v1/projects/search` | PublicProjectSearchHandler | public |
| `GET` | `/v1/projects/donor-stats` | PublicDonorStatsHandler | public |
| `GET` | `/v1/projects/{projectId}/funding` | GetProjectFundingStatusHandler | public |
| `GET` | `/v1/projects/{projectId}/{nameOrSlug}/sync` | GetUpdatedProjectSlugHandler | public |
| `GET` | `/v1/projects/badge/{id}` | BadgeHandler | public |
| `GET` | `/v1/projects/sponsors/{projectId}` | ProjectSponsorsHandler | public |
| `POST` | `/v1/projects/exists` | ProjectExistsByNameHandler | public |
| `POST` | `/v1/projects/slug-check` | ProjectSlugCheckHandler | public |
| `POST` | `/v1/projects/title-check` | ProjectTitleCheckHandler | public |
| `POST` | `/v1/projects/approvals` | EmailApproveProjectHandler | required |
| `POST` | `/v1/projects/approvals/{action}/{reportId}` | ApprovalFlowExpenseHandler | required |
| `POST` | `/v1/projects/approval-flow` | ApprovalFlowProjectHandler | required |
| `POST` | `/v1/presigned-url` | PresignedLogoUploadHandler | required |

### Entities (General Fund / OSTIF / Initiative / Event subtypes)

| Method | Path | Handler | Auth |
|---|---|---|---|
| `POST` | `/v1/entities` | CreateEntityHandler | required |
| `GET` | `/v1/entities` | GetEntitiesHandler | public |
| `GET` | `/v1/generic/entities` | GetEntitiesGenericHandler | public |
| `GET` | `/v1/entities/paginate` | PaginateEntitiesCacheHandler | public |
| `GET` | `/v1/entities/{entityId}` | GetEntityHandler | public |
| `GET` | `/v1/entities/{entityId}/funding` | GetEntityFundingStatusHandler | public |
| `PUT` | `/v1/me/entities/{entityId}` | UpdateEntityHandler | required |
| `GET` | `/v1/me/entities` | GetPrivateEntitiesHandler | required |
| `POST` | `/v1/me/entities/{entityId}/status/{status}` | HideUnhideEntityStatusHandler | required |
| `POST` | `/v1/entities/{entityId}/addbeneficiary` | EntityAddBeneficiaryHandler | required |
| `POST` | `/v1/entities/{entityId}/removebeneficiary` | EntityRemoveBeneficiaryHandler | required |
| `POST` | `/v1/entities/approval-flow` | ApprovalFlowEntityHandler | required |
| `POST` | `/v1/entities/slug-check` | EntitySlugCheckHandler | public |
| `POST` | `/v1/entities/title-check` | EntityTitleCheckHandler | public |
| `GET` | `/v1/events` | GetEventsHandler | public |
| `GET` | `/v1/me/events` | GetMyEventsHandler | required |

### Subscriptions & Donations

| Method | Path | Handler | Auth |
|---|---|---|---|
| `POST` | `/v1/me/subscriptions` | CreateSubscriptionsHandler | required |
| `GET` | `/v1/me/subscriptions` | ListSubscriptionsHandler | required |
| `GET` | `/v1/me/subscriptions/{projectId}` | GetSubscriptionHandler | required |
| `PATCH` | `/v1/me/subscriptions/{projectId}` | UpdateSubscriptionHandler | required |
| `DELETE` | `/v1/me/subscriptions/{projectId}` | CancelSubscriptionHandler | required |
| `POST` | `/v1/me/entities/subscriptions` | CreateEntitySubscriptionsHandler | required |
| `GET` | `/v1/me/entities/subscriptions` | ListEntitySubscriptionsHandler | required |
| `GET` | `/v1/me/entities/subscriptions/{entityId}` | GetEntitySubscriptionHandler | required |
| `PATCH` | `/v1/me/entities/subscriptions/{entityId}` | UpdateEntitySubscriptionHandler | required |
| `DELETE` | `/v1/me/entities/subscriptions/{entityId}` | CancelEntitySubscriptionHandler | required |
| `POST` | `/v1/me/donations` | CreateDonationHandler | required |
| `POST` | `/v1/hooks/stripe` | StripeWebhookHandler | public (signed) |
| `GET` | `/v1/projects/{projectId}/backers` | ListPublicBackersHandler | public |
| `GET` | `/v1/entities/{entityId}/backers` | ListPublicEntityBackersHandler | public |

### Organizations

| Method | Path | Handler | Auth |
|---|---|---|---|
| `POST` | `/v1/me/organizations` | CreateOrganizationsHandler | required |
| `GET` | `/v1/me/organizations` | GetOrganizationsHandler | required |
| `PUT` | `/v1/me/organizations/{organizationId}` | UpdateOrganizationHandler | required |
| `DELETE` | `/v1/me/organizations/{organizationId}` | DeleteOrganizationHandler | required |
| `GET` | `/v1/organizations/{organizationId}` | GetPublicOrganizationHandler | public |
| `GET` | `/v1/organizations/users/{userID}` | GetPublicUserHandler | public |
| `GET` | `/v1/organizations/exists/{organizationName}` | CreateOrganizationExistsHandler | public |
| `GET` | `/v1/organizations/approvals` | EmailApproveOrganizationHandler | public |

### Users & Payment Accounts

| Method | Path | Handler | Auth |
|---|---|---|---|
| `GET` | `/v1/me` | UpdateUser | required |
| `PUT` | `/v1/me/payment-account` | CreateCustomerHandler | required |
| `PATCH` | `/v1/me/payment-account` | UpdateCustomerHandler | required |
| `GET` | `/v1/me/payment-account` | GetCustomerHandler | required |
| `DELETE` | `/v1/me/payment-card` | DeletePaymentCardHandler | required |
| `POST` | `/v1/me/accounts` | ConnectOAuthAccount | required |
| `GET` | `/v1/me/repositories` | ListRepositoriesHandler | required |

### Transactions

| Method | Path | Handler | Auth |
|---|---|---|---|
| `GET` | `/v1/projects/{projectId}/transactions` | GetTransactionsHandler | public |
| `GET` | `/v1/entities/{entityId}/transactions` | GetEntityTransactionsHandler | public |
| `GET` | `/v1/me/transactions` | GetTransactionsThatUserOwnsHandler | required |

---

## DynamoDB Tables

All tables prefixed: `lff-{stage}-{table}`. Stage values: `dev`, `staging`, `prod`.

### `lff-{stage}-projects`

Primary key: `projectId` (partition) + `status` (sort).

DynamoDB attribute names (JSON struct tags):
- `projectId`, `ownerId`, `name`, `status`, `planId` (StripePlanID), `productId` (StripeProductID)
- `details` (nested ProjectDetails), `cachedDetails` (GitHub + project stats)
- `createdOn`, `updatedOn`, `logoUrl`, `slug`, `amountRaised`

DynamoDB Streams: yes — triggers vulnerability registration handlers on INSERT/MODIFY/REMOVE.

### `lff-{stage}-subscriptions`

DynamoDB attribute names: `stripeSubscriptionId`, `projectId`, `userId`, `frequency`, `amountInCents`, `status`, `createdOn`, `updatedOn`.

DynamoDB Streams: yes — triggers `InsertProjectStatsHandler`, `ModifyProjectStatsHandler`, `RemoveProjectStatsHandler` to update aggregate stats on the project record.

### `lff-{stage}-donations`

DynamoDB attribute names: `stripeChargeId`, `projectId`, `userId`, `entityId`, `name`, `avatarUrl`, `paymentMethod`, `amountInCents`, `createdOn`, `status`, `poNumber`, `orgId`, `category`.

DynamoDB Streams: yes — triggers OpenSearch export on INSERT/MODIFY.

### `lff-{stage}-entities`

Primary key: `entityId`. Subtypes stored as a type field: `project`, `initiative`, `general-fund`.

DynamoDB Streams: yes — triggers OpenSearch export on INSERT/MODIFY.

### `lff-{stage}-entity-subscriptions`

Same shape as subscriptions but with `entityId` instead of `projectId`.

### `lff-{stage}-entity-donations`

Same shape as donations but entity-scoped.

### `lff-{stage}-users`

User profile + Stripe customer info + GitHub OAuth tokens.

### `lff-{stage}-organizations`

Primary key: `organizationId`. GSI: `orgByOwner` (ownerId).

Fields: `organizationId`, `ownerId`, `name`, `status`, `description`, `website`, `logoUrl`, `createdOn`, `updatedOn`, `approvedOn`, `rejectedOn`.

---

## Domain Data Models (Go structs)

### Project

```go
ProjectID, OwnerID, Name, Status, Industry, Website, CIIProjectID, Description
CreatedOn, UpdatedOn, Color, StripeProductID, StripePlanID, LogoURL
Diversity, Development, Marketing, Meetups, Travel, BugBounty, Documentation, Mentee, Other, Uncategorised
GithubStats, ProjectStats
Contributors[], Beneficiaries[], CustomWebsites[], Sponsors[], VulnerabilitySummary
CodeOfConduct, JobspringProjectID, StacksIdentifier, Slug, AmountRaised (int)
```

Budget category struct (Development, Marketing, etc.):
```go
Budget { AmountInCents int }  // 0–100,000,000 cents ($0–$1M per category)
Description, Goals string
IsActive bool
Skills[], Terms []string
```

### Donation

```go
UserID, ProjectID, EntityID, StripeChargeID, Name, AvatarURL
PaymentMethod enum: "card" | "invoice"
AmountInCents int64  // range: 100–99,999,999 ($1–$999,999.99)
CreatedOn, Status
PONumber*, OrgID*, Category* (optional)
```

### Subscription

```go
SubscriptionID, StripeSubscriptionID, ProjectID, UserID
Frequency enum: "monthly" | "annual"
AmountInCents int64, Status, CreatedOn, UpdatedOn
```

### Organization

```go
OrganizationID, OwnerID, Name, Status, Description, Website, LogoURL
CreatedOn, UpdatedOn, ApprovedOn, RejectedOn
```

### Entity

```go
EntityID, OwnerID, Name
Type enum: "project" | "initiative" | "general-fund"
Status, Description, Website, LogoURL
CreatedOn, UpdatedOn, ApprovedOn
Goals, Beneficiaries, Donations, Subscriptions, Vulnerabilities
AmountRaised
```

---

## Authentication

- Provider: Auth0
- Libraries: `github.com/auth0/go-jwt-middleware/v2`, `github.com/golang-jwt/jwt`
- Mechanism: Custom Lambda authorizer validates JWT; passes user context to downstream handlers
- Token claims extracted: `UserID`, `Email`, `LFID`, `Name`, `GivenName`, `FamilyName`, `AvatarURL`

Auth0 tenants by stage:
- Dev: `linuxfoundation-dev.auth0.com` / client `lzClGRsDYnfgMmio8J9vYXwTkFm51na2`
- Staging: `linuxfoundation-staging.auth0.com` / client `DnO2mm4jbiKO3HaFIo2TOwY3fkcKV5O3`
- Prod: `sso.linuxfoundation.org` / client `1sgQmtwRIKwMrCFoFSu6iAm8RtJGvPmf`

Project/entity approval uses signed JWT links emailed to admin (Sriji). No separate admin UI exists.

---

## External Integrations

### Stripe

- Purpose: one-time charges (donations), recurring subscriptions, Stripe Connect for project owners
- Stripe objects: Customer, Plan, Product, Subscription, Charge
- Webhook: `POST /v1/hooks/stripe` — handles `customer.subscription.deleted` (cancels in DynamoDB) and `invoice.payment_succeeded` (unmarshalled but not written to LFF — Ledger handles this directly via its own Stripe webhook)
- Metadata on Stripe objects: `userID`, `projectID`, `entityID`, `ownerID`, `orgID`, `category`
- Config: `STRIPE_CLIENT_SECRET`, `STRIPE_WEBHOOK_SIGNING_SECRET` (SSM)

### Ledger Service

- LFF calls Ledger HTTP API **read-only**. LFF never writes to Ledger.
- Ledger has its own Stripe webhook that writes CREDIT/DEBIT records to its own Postgres DB.
- LFF calls:
  - `GET /transactions` — list transactions with filters (projectID, startDate, endDate, page, perPage)
  - `GET /balance/{projectID}` — project balance
  - `GET /balance/{projectID}/entity` — entity balance
  - `GET /transactions` with userID filter — user transactions
- Auth: Bearer token (`TRANSACTIONS_API_SECRET` env var → `LEDGER_AUTHORIZATION_TOKEN`)
- Config: `TRANSACTIONS_API_URL`, `TRANSACTIONS_API_SECRET`

### Reimbursement Service

- LFF pushes project/entity metadata via:
  - `POST /reimbursement/{projectID}` — create Expensify policy for new project
  - `PATCH /reimbursement/{projectID}` — update policy when project changes
  - `POST /expense/{action}/{reportID}` — process expense approval/rejection
- Auth: `X-API-KEY` header + Bearer token
- Config: `REIMBURSEMENTS_API_URL`, `REIMBURSEMENTS_API_SECRET`

### Auth0

- See Authentication section above.

### Mandrill (email)

- Provider: Mailchimp Mandrill
- Key: `MANDRILL_API_KEY` (SSM)
- Emails sent: org approvals/rejections, entity approvals/rejections, project approvals, donation notifications (partially replaced by Ledger service — FUND-1055), invoicing, security audit submissions
- From: `noreply@` per stage

### GitHub API

- Repository info, issue counts (accounts for PRs vs issues)
- OAuth for user repo access during project creation
- Config: `GITHUB_TOKEN`, `GITHUB_OAUTH_CLIENT_ID`, `GITHUB_OAUTH_CLIENT_SECRET` (SSM)

### Mentorship (JobSpring) — Bidirectional integration

The integration runs in three channels, not one. **It is bidirectional.**

**Channel 1: SNS/SQS — shared topic, bidirectional via separate queues**

Both CF and Mentorship publish to the **same** SNS topic `lfx-topic-{stage}-project` and each subscribes via their own SQS queue.

CF → Mentorship:
- CF publishes `projectUpdated` and `projectUpdateStatus` events to `lfx-topic-{stage}-project` (via `SNS_PROJECT_TOPIC_ARN` env var → `LFF/backend/projects/di/init.go`)
- Triggered when: CF admin updates a project, or project status changes (approval flow)
- Mentorship consumes from: `jobspring-lfx-queues-{stage}-project`

Mentorship → CF:
- Mentorship publishes `projectCreated`, `projectUpdated`, `projectUpdateStatus`, `selfSync` to same topic
- CF consumes from: `fundspring-lfx-queues-{stage}-project`
- Event types consumed by CF:
  - `projectCreated` — create project record in DynamoDB
  - `projectUpdated` — update project record
  - `projectUpdateStatus` — update project status
  - `selfSync` — ignored
- Message format: `LfxEvent` JSON with `type`, `source_id: "jobspring"`, `data.projectId`, `data.name`, `data.status`, `data.projectDetails.mentee` (skills, terms, mentors, customTerm)
- **⚠️ Budget data is nested:** mentorship budget lives at `data.projectDetails.mentee`, NOT at `data.mentee`

**Channel 2: Mentorship → CF via direct HTTP calls**
Mentorship calls these CF API endpoints directly (against `FUNDING_API_URL`):
- `GET /v1/projects/{projectId}/{projectNameOrSlug}/sync` — slug sync after rename
- `GET /v1/projects/{projectId}/funding` — fetch funding/balance status
- `POST /v1/entities/{entityId}/addbeneficiary` — add beneficiary to entity (auth: `x-beneficiary-auth`)
- `POST /v1/entities/{entityId}/removebeneficiary` — remove beneficiary
- `POST /v1/projects/title-check` — check title uniqueness before creating a project

**These endpoints existed in the old system. In the new CF service, all five are eliminated — Mentorship no longer calls CF directly. See `04-target-architecture.md` for details.**

**Channel 3: CF → Mentorship via direct HTTP call (fallback only)**
- `GET https://{JOBSPRING_API_URL}/users/external/{lfid}` — fetches user profile from Mentorship when a user is not found in the CF database
- This is a fallback lookup, not a primary data flow

### OpenSearch

- Used for: project/entity search and discovery, caching, export
- Indices written by LFF: `projects`, `entities`, `lff-users`, `organizations`
- Indices read by LFF: `projects`, `entities` (for cache endpoints)
- Indices read by Reimbursement Service: `projects`, `entities`, `lff-users`, `spring-projects`, `spring-users`, `beneficiary-actions`, `travel-funds-tickets`
- Indices written by Reimbursement Service: `lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`

### Other External APIs

| Service | Config var | Purpose |
|---|---|---|
| Stacks / CommunityBridge | `STACKS_BASE_URL` | Security vulnerabilities, bug bounties |
| Diversity API | `DIVERSITY_BASE_URL` | Diversity data on projects |
| Vulnerability API | `VULNERABILITY_BASE_URL` | Vulnerability scanning registration |
| JobSpring API | `JOBSPRING_API_URL` | Mentorship/mentee integration |
| S3 | upload bucket `{project}-{stage}-uploads` | Logo/image uploads (presigned URLs) |

---

## Ledger Service (separate Go microservice)

- Module: `github.com/LF-Engineering/ledger-service`
- Database: **PostgreSQL** (single `ledger` table) — NOT DynamoDB
- Purpose: write-only financial audit ledger; records every CREDIT and DEBIT
- Data source: Stripe and Expensify webhooks hit Ledger directly
- LFF relationship: LFF reads from Ledger API (read-only); LFF never writes to Ledger

### `ledger` table schema

```sql
txn_id          uuid PRIMARY KEY  -- auto-generated
project_id      text
user_id         text
organization_id text
account_email   text              -- Expensify/Stripe account
submitter_name  text
merchant_name   text
report_name     text
txn_comment     text
source_type     text              -- 'STRIPE' | 'EXPENSIFY'
source_txn_id   text              -- external tx ID (dedup key)
source_account_id text
txn_type        text              -- 'CREDIT' | 'DEBIT'
txn_category    text              -- marketing | meetups | mentorship | development | travel | bugBounty | documentation | other | uncategorised
fee             integer           -- cents
amount          integer           -- cents (positive or negative)
txn_date        bigint            -- Unix epoch
created_at      bigint            -- Unix epoch, auto
```

### Ledger HTTP Endpoints

All require `LEDGER_AUTHORIZATION_TOKEN` Bearer header.

- `GET /health/`
- `GET /balance/{projectID}` — project balance with breakdown
- `GET /balance/{projectID}/entity` — entity balance
- `GET /transactions/` — list all
- `GET /transactions/v1/paginate` — paginated view model
- `GET /transactions/v1/donor-stats` — donor statistics
- `POST /transactions/` — add transaction (called by Stripe/Expensify webhooks, not by LFF)
- `PUT /transactions/` — update transaction
- `POST /hooks/stripe` — Stripe webhook (writes CREDIT records)

---

## Reimbursement Service

- Reads from OpenSearch indices: `projects`, `entities`, `lff-users`, `spring-projects`, `spring-users`, `beneficiary-actions`, `travel-funds-tickets`
- Writes to OpenSearch indices: `lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`
- Does NOT call the LFF API directly (constructs URL strings for email links only)
- Purpose: manages Expensify expense policies for Crowdfunding projects; handles beneficiary lifecycle, policy creation/updates, expense approval/rejection notifications

**OpenSearch dependency — migration plan (see OQ-7):**
- On CF release day: RS switches `projects`/`entities`/`lff-users` index reads to direct SQL on CF Postgres (`crowdfunding` schema, read-only role)
- CF release + 2 weeks: RS migrates `lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets` to `reimbursement` schema on CF Postgres; OpenSearch decommissioned

---

## lfx-v1-sync-helper

- Purpose: real-time bidirectional sync between LFX v1 (legacy) and LFX One (v2) via NATS KV
- Syncs: project metadata and committee data only
- Does NOT sync: Crowdfunding donations, subscriptions, organizations, or Ledger transactions
- Not useful for Crowdfunding DB migration — purpose-built for v1↔v2 platform metadata
