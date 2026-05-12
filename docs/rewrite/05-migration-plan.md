# Migration Plan

DynamoDB → PostgreSQL data migration and cutover plan.

**Owner:** Lewis  
**Schema DDL:** [`db/migrations/001_initial.up.sql`](../../db/migrations/001_initial.up.sql)

> **Migration status (2026-05-11):** Initial run complete — 2,021 initiatives, 16,840 users, 1,598 donations, 277 subscriptions migrated. FK integrity: 0 orphaned goals, 0 orphaned donations, 0 NULL owner IDs.

---

## Overview

### Scope: Crowdfunding DB only

This migration covers DynamoDB → Crowdfunding Postgres (`crowdfunding` schema) only.

**Not in scope for initial release:**
- Ledger DB migration (Ledger Service keeps its own Postgres, unchanged)
- Reimbursement Service data
- OpenSearch data (old stack stays live for Reimbursement Service)

Ledger DB migration is a separate post-release project. When it happens, the `ledger` schema is added to the Crowdfunding Postgres instance, the `project_funding_summary` view is activated, and CF API balance calls switch from HTTP to direct SQL.

### Phases

1. **Schema** — apply schema changes in `001_initial.up.sql`
2. **Data migration** — Go CLI `tools/migrate-cf/` reads all DynamoDB tables and upserts into Postgres
3. **Validation** — reconcile record counts, spot-checks, Stripe cross-check
4. **Cutover** — switch DNS/Ingress from old Lambda API Gateway to new K8s service
5. **Decommission** — tear down old Lambda stack, DynamoDB; OpenSearch decommission is a separate later phase (blocked on RS moving to K8s, see OQ-7)

These phases are sequential and gated by human review. Do not proceed to the next phase without explicit sign-off.

---

## Phase 1: Schema

Schema DDL is in [`db/migrations/001_initial.up.sql`](../../db/migrations/001_initial.up.sql). Applied via the `cmd/migrate/` golang-migrate runner.

The schema uses a **normalized 20-table design**: `users`, `organizations`, `initiatives`, `donations`, `subscriptions`, plus 15 child tables for repeating groups (`initiative_goals`, `initiative_beneficiaries`, `initiative_contributors`, `initiative_mentors`, `initiative_program_info_*`, `initiative_sponsorship_tiers`, `initiative_ostif_detail`, `initiative_contacts`, `initiative_github_stats`, `initiative_stats`, `initiative_entity_details`, `initiative_custom_websites`). See `data-design_and_migration.md` for the full data dictionary.

---

## Phase 2: Data Migration (`tools/migrate-cf/`)

### Tool: Dedicated Go CLI

**Location:** `tools/migrate-cf/` in this repo.

> The Python migration script is at [`tools/migrate-cf/migrate_dynamo_to_postgres.py`](../../tools/migrate-cf/migrate_dynamo_to_postgres.py).

**Note on Python script:** Lewis's original migration work used a Python script (`migrate_dynamo_to_postgres.py`) with the normalized 20-table schema now in `001_initial.up.sql`. The production Postgres DB currently contains the output of that script. The Go CLI replacement targets the same normalized schema.

`cmd/migrate/` is reserved for the golang-migrate schema runner — that is a separate tool.

### Two modes

```text
migrate-cf --mode=validate   # Read DynamoDB, report what would be written. No writes.
migrate-cf --mode=execute    # Read DynamoDB, write to Postgres.
```

Always run `--validate` first against prod data. Fix any issues. Then run `--execute`.

### DynamoDB Tables to Migrate

| DynamoDB Table | Target Postgres Table | Notes |
|---|---|---|
| `lff-prod-projects` | `crowdfunding.initiatives` | Set `initiative_type` (`project` or `mentorship`); branch budget path by type |
| `lff-prod-entities` | `crowdfunding.initiatives` | Reclassify DynamoDB `entityType` → `initiative_type` per type mapping table below |
| `lff-prod-organizations` | `crowdfunding.organizations` | Straightforward |
| `lff-prod-subscriptions` | `crowdfunding.subscriptions` | Preserve `stripe_subscription_id`; resolve old `projectId` → `initiative_id` |
| `lff-prod-entity-subscriptions` | `crowdfunding.subscriptions` | Merge with subscriptions; resolve old `entityId` → `initiative_id` |
| `lff-prod-donations` | `crowdfunding.donations` | Preserve `stripe_charge_id`; resolve old `projectId` → `initiative_id` |
| `lff-prod-entity-donations` | `crowdfunding.donations` | Merge with donations; resolve old `entityId` → `initiative_id` |
| `lff-prod-users` | `crowdfunding.users` | Minimal — just `stripe_customer_id` and `github_access_token` |

**Not migrated:** Ledger transactions remain in the Ledger Service Postgres DB unchanged.

### Field Mapping: Projects

**Critical:** DynamoDB project rows contain two distinct data shapes depending on origin.
The migration tool MUST detect the type and branch accordingly.

**Detecting initiative_type:**
- All projects are inserted with `initiative_type = 'project'` initially
- After all rows are inserted, a Phase 3 UPDATE reclassifies to `'mentorship'` any initiative that has a `mentee` goal with `amount_in_cents > 0` in `initiative_goals`

**Status normalization (apply to all rows):**
- `'hide'` → `'hidden'` (13 rows in prod have this dirty value)
- All other status values pass through as-is

| DynamoDB attribute | Postgres column | Transform |
|---|---|---|
| `projectId` | `id` | Cast directly to UUID — DynamoDB `projectId` is already a UUID v4 string; same value used by Ledger as `project_id` |
| `ownerId` | `owner_id` | Direct |
| `name` | `name` | Direct |
| `slug` | `slug` | Direct |
| `status` | `status` | Normalize: `'hide'` → `'hidden'` |
| `projectDetails.website` | `website_url` | Nested under `projectDetails` |
| `projectDetails.description` | `description` | Nested under `projectDetails` |
| `projectDetails.color` | `color` | Nested under `projectDetails` |
| `logoUrl` | `logo_url` | Direct (top-level) |
| `projectDetails.codeOfConduct.link` | `coc_url` | Nested; extract `.link` string |
| `projectDetails.ciiProjectID` | `cii_project_id` | Nested under `projectDetails` |
| `projectDetails.stacksIdentifier` | `stacks_identifier` | Nested under `projectDetails` |
| `projectDetails.industry` | `industry` | Nested under `projectDetails`; comma-separated topic tags string |
| `jobspringProjectId` | `mentorship_program_id` | Direct; field name uses lowercase `d` |
| `planId` | `stripe_plan_id` | **Critical — must be preserved exactly** |
| `productId` | `stripe_product_id` | **Critical — must be preserved exactly** |
| `projectDetails.contributors` | → `initiative_contributors` rows | Insert one row per contributor |
| `projectDetails.beneficiaries` | → `initiative_beneficiaries` rows | Insert one row per beneficiary |
| `projectDetails.customWebsites` | → `initiative_custom_websites` rows | Insert one row per URL |
| `projectDetails.sponsors` | **Drop** | No write path; computed at read time |
| `cachedDetails.GithubStats` | → `initiative_github_stats` row | Insert one row per initiative |
| `amountRaised` | — | **Drop** — replaced by `amount_raised_in_cents` cached column + CronJob |
| `createdOn` | `created_on` | Parse timestamp |
| `updatedOn` | `updated_on` | Parse timestamp |

**Budget mapping — insert rows into `initiative_goals`:**

For `initiative_type = 'project'` (read from `projectDetails.*`), insert one `initiative_goals` row per category:

```text
projectDetails.development.budget.amount   → initiative_goals (name='development',  amount_in_cents, allocation, repo_link=development.repoLink)
projectDetails.marketing.budget.amount     → initiative_goals (name='marketing',    amount_in_cents, allocation)
projectDetails.meetups.budget.amount       → initiative_goals (name='meetups',      amount_in_cents, allocation)
projectDetails.travel.budget.amount        → initiative_goals (name='travel',       amount_in_cents, allocation)
projectDetails.bugBounty.budget.amount     → initiative_goals (name='bugBounty',    amount_in_cents, allocation)
projectDetails.documentation.budget.amount → initiative_goals (name='documentation',amount_in_cents, allocation)
projectDetails.other.budget.amount         → initiative_goals (name='other',        amount_in_cents, allocation)
projectDetails.mentee.budget.amount        → initiative_goals (name='mentee',       amount_in_cents, allocation)
```

**⚠️ Critical:** `Budget.AmountInCents` has JSON tag `"amount"` — read from DynamoDB as `budget["amount"]`, NOT `budget["amountInCents"]`.

For `initiative_type = 'mentorship'`, the mentee goal is still inserted into `initiative_goals`. The additional structured fields from `projectDetails.mentee` go into child tables:

```text
projectDetails.mentee.skills[]       → initiative_program_info_skills rows
projectDetails.mentee.terms[]        → initiative_program_info_terms rows
projectDetails.mentee.mentor[]       → initiative_mentors rows
projectDetails.mentee.customTerm     → initiative_program_info_custom_term row
projectDetails.mentee.termsConditions → initiative_program_info_config row
```

For entity goals, `entity.goals[]` (top-level array) → one `initiative_goals` row per element (uses `Goal.Name`, `Goal.Description`, `Goal.goalColor`, `Goal.goalIcon`, `budget["amount"]`).

**⚠️ Do NOT read the top-level `mentee` attribute.** The actual data is always nested under `projectDetails.mentee`. Reading from the wrong path silently drops all mentorship metadata — this was the bug that caused the first migration pass to miss 1,249 of 1,476 rows.

**Drop from migration:**
- `amountRaised` — replaced by `amount_raised_in_cents` cached column + CronJob
- `cachedDetails.ProjectStats` — recomputed
- `details.Diversity` — deferred feature
- `details.VulnerabilitySummary` — deferred feature

### Field Mapping: Entities (→ `crowdfunding.initiatives`)

These rows migrate into the same `crowdfunding.initiatives` table as projects.

**Type reclassification (applied during migration):**

| DynamoDB `entityType` value | Postgres `initiative_type` | Row count | Notes |
|---|---|---|---|
| `initiative` | `general_fund` | 121 | UI label was already "General Fund" |
| `general-fund` | `general_fund` | (subset of above) | Normalize hyphen |
| `other` (travel) | `general_fund` | 26 | Functionally identical to general funds; UI option was commented out. **Retained in DB with `initiative_type = 'general_fund'`** for historic reference; not shown in UI. Confirm with PM: 8 published rows will display as "General Fund" after reclassification. |
| `event` | `event` | 20 | Unchanged |
| `ostif` | `ostif` | 11 | Unchanged |
| `community` | **discard** | 3 | All declined/submitted 2019; no active users |

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `entityId` | `id` | Cast directly to UUID — DynamoDB `entityId` is already a UUID v4 string; same value used by Ledger as `project_id` |
| `ownerId` | `owner_id` | Direct |
| `name` | `name` | Direct |
| `slug` | `slug` | Direct |
| `entityType` | `initiative_type` | Reclassify per table above |
| `status` | `status` | Normalize `'hide'` → `'hidden'` |
| `description` | `description` | Direct |
| `websiteURL` | `website_url` | DynamoDB field is `websiteURL` |
| `logoUrl` | `logo_url` | Direct |
| `city` | `city` | Direct; nullable |
| `country` | `country` | Direct; nullable |
| `isOnline` | `is_online` | Direct boolean; default false if missing |
| `acceptFunding` | `accept_funding` | Direct boolean; default true if missing |
| `applicationURL` | `application_url` | Direct |
| `eventStartDate` | `event_start_date` | Parse date; event type only |
| `eventEndDate` | `event_end_date` | Parse date; event type only |
| `eventbriteId` | `eventbrite_id` | Despite the DynamoDB field name, the stored value is a URL; handle at application layer |
| `industry` | `industry` | Direct; comma-separated topic tags string |
| `details.Beneficiaries` | → `initiative_beneficiaries` rows | Insert one row per beneficiary |
| `goals` | → `initiative_goals` rows | DynamoDB field is `goals` (top-level array) — insert one row per element |
| `amountRaised` | — | **Drop** |
| `createdOn` | `created_on` | Parse timestamp |
| `updatedOn` | `updated_on` | Parse timestamp |

### Field Mapping: Subscriptions

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `stripeSubscriptionId` | `stripe_subscription_id` | |
| `stripeSubscriptionItemId` | `stripe_subscription_item_id` | Nullable |
| `projectId` | `initiative_id` | Resolve old `projectId` → new Postgres UUID via in-memory map |
| `userId` | `user_id` | Resolve Auth0 subject via `users.user_id` → `users.id` |
| `orgId` | `organization_id` | Resolve to Postgres UUID via in-memory map if set |
| `frequency` | `frequency` | Direct (`monthly` \| `annual`); normalize `yearly` → `annual` |
| `currentAmountInCents` | `current_amount_in_cents` | DynamoDB field is `currentAmountInCents` |
| `status` | `status` | Prod values: `"active"` \| `"inactive"` |
| `createdOn` | `created_on` | ISO 8601 format |
| *(absent)* | `updated_at` | Does not exist in DynamoDB — default to `created_on` value on migration |

For `lff-prod-entity-subscriptions`: same mapping, resolve old `entityId` → new Postgres UUID.

### Field Mapping: Donations

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `stripeChargeId` | `stripe_charge_id` | Null for invoice payments |
| `projectId` | `initiative_id` | Resolve old `projectId` → new Postgres UUID via in-memory map |
| `userId` | `user_id` | Resolve Auth0 subject via `users.user_id` → `users.id` |
| `orgId` | `organization_id` | Resolve to Postgres UUID via in-memory map if set |
| `currentAmountInCents` | `current_amount_in_cents` | DynamoDB field is `currentAmountInCents` |
| `category` | `category` | Direct |
| `paymentmethod` | `payment_method` | DynamoDB key is `paymentmethod` (all lowercase) |
| `ponumber` | `po_number` | DynamoDB key is `ponumber` (all lowercase) |
| `status` | `status` | Direct; null on all production rows — migrate as null |
| `createdOn` | `created_on` | Parse timestamp |

For `lff-prod-entity-donations`: same mapping, resolve old `entityId` → new Postgres UUID.

### Field Mapping: Organizations

Organization IDs in DynamoDB are already UUIDs — migrate `organizationId` directly to `id`.

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `organizationId` | `id` | Already UUID — preserve as-is |
| `ownerId` | `owner_id` | Direct (Auth0 subject) |
| `name` | `name` | Direct |
| `status` | `status` | All prod rows are `"approved"` |
| `avatarUrl` | `avatar_url` | DynamoDB field is `avatarUrl` (capital U, lowercase rl) |

No `description`, `website`, `approved_at`, or `rejected_at` — those fields don't exist in DynamoDB.

### Field Mapping: Users

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `id` | `user_id` | Auth0 subject (e.g. `auth0|abc123`) — stored as natural key; `id` UUID is generated |
| `email` | `email` | Direct |
| `givenname` | `given_name` | DynamoDB key is `givenname` (no camelCase) |
| `familyName` | `family_name` | Direct |
| `name` | `name` | Direct |
| `avatarUrl` | `avatar_url` | Direct |
| `stripeCustomerId` | `stripe_customer_id` | Direct |
| *(OAuth token)* | `github_access_token` | GitHub OAuth token if present |

### ID Mapping (in-memory)

DynamoDB uses string IDs. Postgres uses UUIDs. The migration CLI holds a `map[string]uuid.UUID` in memory to resolve old `projectId` and `entityId` references when migrating subscriptions and donations. No DB table is created — see `02-decisions.md`.

### Data Quality Checks (run in `--validate` mode)

- Subscriptions referencing non-existent project or entity IDs → log as warnings, skip or null-out
- Donations referencing non-existent project or entity IDs → same
- Duplicate `stripe_subscription_id` values → error
- Duplicate `stripe_charge_id` values → error
- Missing `slug` on initiatives → error (UNIQUE NOT NULL)
- Initiatives with same slug → error
- Null `owner_id` → error
- Invalid `status` values → log and map to closest valid status

### Migration Execution Order

1. `organizations`
2. `initiatives` (from `lff-prod-projects` — projects and mentorship)
3. `initiatives` (from `lff-prod-entities` — general funds, events, OSTIF; appended to same table)
4. `users`
5. `subscriptions` (from `lff-prod-subscriptions` + `lff-prod-entity-subscriptions`, merged)
6. `donations` (from `lff-prod-donations` + `lff-prod-entity-donations`, merged)

---

## Production Data Size (as of 2026-05-11)

2,013 total rows from DynamoDB → `crowdfunding.initiatives` (minus 3 discarded `community` rows = 2,010).

| Table | Expected rows | Source |
|---|---|---|
| `crowdfunding.initiatives` | 2,010 | `lff-prod-projects` + `lff-prod-entities` |
| `crowdfunding.organizations` | ~606 | `lff-prod-organizations` |
| `crowdfunding.subscriptions` | ~277 | `lff-prod-subscriptions` + `lff-prod-entity-subscriptions` |
| `crowdfunding.donations` | ~1,598 | `lff-prod-donations` + `lff-prod-entity-donations` |
| `crowdfunding.users` | ~16,840 | `lff-prod-users` |

**Initiative type breakdown (post-migration):**

| `initiative_type` | Count |
|---|---:|
| `mentorship` | ~1,249 |
| `project` | ~590 |
| `general_fund` | ~148 (includes reclassified `other`/travel rows) |
| `event` | 20 |
| `ostif` | 11 |

---

## Phase 3: Validation

Before cutover, run a reconciliation pass:

1. **Record counts:** DynamoDB item count == Postgres row count per table
2. **Spot checks:** Sample 20 records per table, compare DynamoDB vs Postgres field by field
3. **Active subscriptions:** Every `stripeSubscriptionId` in DynamoDB is present in Postgres with `status = 'active'`
4. **Stripe cross-check:** Query Stripe API for active subscriptions; verify each one has a matching Postgres record
5. **Financial totals:** Sum of `currentAmountInCents` in DynamoDB donations == sum of `current_amount_in_cents` in Postgres donations per initiative

Document results. Keep the validation report alongside migration logs.

---

## Phase 4: Cutover

### Prerequisites

- [ ] New CF service fully deployed and tested in dev and staging
- [ ] Migration executed and validated in staging (against a staging DynamoDB copy)
- [ ] Migration executed and validated in prod (validate mode first, then execute)
- [ ] Auth0 callback/CORS URLs updated for new service
- [ ] `EMAIL_DRY_RUN=true` confirmed for prod migration testing
- [ ] Reimbursement Service OpenSearch dependency acknowledged (old Lambda keeps running)
- [ ] Rollback procedure tested in staging
- [x] OQ-15 resolved — `initiatives.id` IS the Ledger `project_id`; no `legacy_id` column needed (confirmed via source code trace, see `03-open-questions.md`)
- [ ] Ledger Service updated and deployed — auth headers fixed in `fundspring.go`: `x-ledger-auth` → `Authorization: Bearer` for `GetProject`/`GetUserName`; `Authorization: Bearer` added to `GetOrganizationName` (currently sends no auth header — a Ledger bug); must be live before DNS cutover or donation confirmation emails break immediately
- [ ] `amount_raised_in_cents` pre-populated via `amount-raised-sync` CronJob before DNS switch

### Cutover Steps

1. Put old system in read-only mode (disable writes) — or accept brief dual-write window
2. Run final incremental migration (any records created since the last full migration)
3. **Run `amount_raised_in_cents` pre-population** — execute the `amount-raised-sync` CronJob manually against prod Ledger API to populate `amount_raised_in_cents` for all migrated initiatives before DNS switches. This ensures no published initiative card shows `$0 raised` incorrectly on day one.
4. Switch DNS / K8s ingress from old Lambda API Gateway to new K8s service
5. Smoke test: login, view projects, make a test donation (test card), check subscription list
6. Monitor for errors (Go service logs, Postgres errors)
7. Keep old Lambda running for 2 weeks minimum (rollback window)

### Rollback

Switch DNS / K8s ingress back to old Lambda API Gateway. Old DynamoDB data is untouched during migration (migration is read-only from DynamoDB). Rollback is safe at any point before decommission.

---

## Phase 5: Decommission (post-cutover)

Only after:
- New system has been stable for minimum 2 weeks
- Reimbursement Service OpenSearch dependency is resolved (see OQ-7)
- All active Stripe subscriptions verified working under new system

Steps:
1. Decommission old LFF Lambda functions
2. Decommission DynamoDB tables (after backup)
3. Decommission OpenSearch (after Reimbursement Service migrated off it — see OQ-7; also blocked on Ledger Expensify fallback resolution — see OQ-14)
4. Archive old LFF and lfx-crowdfunding-upgrade repos (read-only)

---

## Migration Notes

- **`community` entity type:** 3 rows, all declined/submitted 2019, no active users. Discarded — not inserted into Postgres.
- **`other (travel)` entity type:** 26 rows. DynamoDB `entityType` value is `other`; mapped to `initiative_type = 'general_fund'`. Retained in DB; not shown in UI.
- **Mentorship initiatives (~1,249 rows):** Have a `mentorship_program_id` linking them to the Mentorship service. Migration must populate `mentorship_program_id` from the DynamoDB `jobspringProjectId` field. New mentorship programs arrive post-migration via the `mentorship-sync` Snowflake CronJob.
- **Old IDs and Ledger:** `initiatives.id` holds the original DynamoDB UUID and is used directly as the Ledger `project_id` lookup key. DynamoDB `projectId`/`entityId` are UUID v4 strings (generated by `satori/go.uuid`) and cast cleanly to `UUID` — same value, no bridging column needed. See OQ-15 in `03-open-questions.md`.
- **Stripe subscription continuity:** Active Stripe subscriptions must not be cancelled or recreated. The migration preserves `stripe_subscription_id` — Stripe continues charging the same plan. No Stripe API calls needed during migration.
- **Non-published records:** 639 rows are not published (submitted, declined, hidden). All must be migrated — active subscriptions or pending approvals may reference them.
