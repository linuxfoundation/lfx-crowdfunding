# Migration Plan

DynamoDB → PostgreSQL data migration and cutover plan.

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

1. **Schema** — define and review Postgres tables before touching any data
2. **Data migration** — one-time copy from DynamoDB to Postgres using `migrate-cf` CLI
3. **Cutover** — switch DNS/Ingress from old Lambda API Gateway to new K8s service

These phases are sequential and gated by human review. Do not proceed to the next phase without explicit sign-off.

---

## Phase 1: Schema Review (no data touched)

**Goal:** finalize the Postgres schema before writing migration code.

Steps:
1. Write initial SQL migration files (`db/migrations/001_initial.up.sql`)
2. Review schema with team — specifically:
   - JSONB columns for budgets (confirmed decision)
   - `stripe_plan_id` / `stripe_product_id` preserved on projects (must survive migration)
   - `stripe_subscription_id` UNIQUE constraint (active subscriptions — must survive)
   - `stripe_charge_id` UNIQUE constraint (donations — idempotency)
   - `project_or_entity` CHECK constraint on subscriptions and donations
3. Run migrations against a local dev Postgres instance
4. **Sign-off required before Phase 2**

---

## Phase 2: Data Migration (`migrate-cf` CLI)

### Tool: Dedicated Go CLI

Location: `cmd/migrate/` in the new CF repo (or a standalone tool — TBD).

Not `lfx-v1-sync-helper`. Reason: lfx-v1-sync-helper is a NATS KV sync service with no knowledge of Crowdfunding data shapes. A purpose-built CLI is 200–300 lines, fully auditable, and disposable after the migration.

### Two modes

```
migrate-cf --mode=validate   # Read DynamoDB, report what would be written. No writes.
migrate-cf --mode=execute    # Read DynamoDB, write to Postgres.
```

Always run `--validate` first against prod data. Fix any issues. Then run `--execute`.

### DynamoDB Tables to Migrate

| DynamoDB Table | Target Postgres Table | Notes |
|---|---|---|
| `lff-prod-projects` | `crowdfunding.projects` | Set `campaign_type`; branch budget path by type |
| `lff-prod-entities` | `crowdfunding.funds` | Reclassify types; rename column `entity_type` → `fund_type` |
| `lff-prod-organizations` | `crowdfunding.organizations` | Straightforward |
| `lff-prod-subscriptions` | `crowdfunding.subscriptions` | Preserve `stripe_subscription_id`; FK to `project_id` |
| `lff-prod-entity-subscriptions` | `crowdfunding.subscriptions` | Merge with subscriptions; FK to `fund_id` |
| `lff-prod-donations` | `crowdfunding.donations` | Preserve `stripe_charge_id`; FK to `project_id` |
| `lff-prod-entity-donations` | `crowdfunding.donations` | Merge with donations; FK to `fund_id` |
| `lff-prod-users` | `crowdfunding.users` | Minimal — just `stripe_customer_id` |

**Not migrated:** Ledger transactions remain in the Ledger Service Postgres DB unchanged.

### Field Mapping: Projects

**Critical:** DynamoDB project rows contain two distinct data shapes depending on origin.
The migration tool MUST detect the type and branch accordingly.

**Detecting campaign_type:**
- If `jobspringProjectID` field is present and non-empty → `campaign_type = 'mentorship'`
- Otherwise → `campaign_type = 'project'`

**Status normalization (apply to all rows):**
- `'hide'` → `'hidden'` (13 rows in prod have this dirty value)
- All other status values pass through as-is

| DynamoDB attribute | Postgres column | Transform |
|---|---|---|
| `projectId` | `legacy_id` | **Store old string ID here** — needed for Ledger API calls |
| `projectId` | — (new UUID generated) | New `id` is a fresh UUID |
| `ownerId` | `owner_id` | Direct |
| `name` | `name` | Direct |
| `slug` | `slug` | Direct |
| `status` | `status` | Normalize: `'hide'` → `'hidden'` |
| `website` | `website` | From `details.Website` (nested) |
| `description` | `description` | From `details.Description` (nested) |
| `color` | `color` | Direct |
| `logoUrl` | `logo_url` | Direct |
| `details.CodeOfConduct` | `code_of_conduct` | Direct |
| `details.CIIProjectID` | `cii_project_id` | Direct |
| `details.StacksIdentifier` | `stacks_id` | Direct |
| `jobspringProjectID` | `mentorship_program_id` | Direct (renamed column) |
| `planId` | `stripe_plan_id` | **Critical — must be preserved exactly** |
| `productId` | `stripe_product_id` | **Critical — must be preserved exactly** |
| `details.Contributors` | `contributors` | Marshal to JSONB array |
| `details.Beneficiaries` | `beneficiaries` | Marshal to JSONB array |
| `details.CustomWebsites` | `custom_websites` | Marshal to JSONB array |
| `cachedDetails.GithubStats` | `github_stats` | Marshal to JSONB |
| `amountRaised` | — | **Drop** — replaced by Ledger view |
| `createdOn` | `created_at` | Parse timestamp |
| `updatedOn` | `updated_at` | Parse timestamp |

**Budget mapping — branches by campaign_type:**

For `campaign_type = 'project'` (read from `data.projectDetails.*`):
```
data.projectDetails.development  → budgets.development  {amount_in_cents, description, goals, is_active}
data.projectDetails.marketing    → budgets.marketing
data.projectDetails.meetups      → budgets.meetups
data.projectDetails.travel       → budgets.travel
data.projectDetails.bugBounty    → budgets.bug_bounty
data.projectDetails.documentation → budgets.documentation
data.projectDetails.mentee       → budgets.mentee       (simple: amount + description only)
data.projectDetails.other        → budgets.other
```

For `campaign_type = 'mentorship'` (read from `data.projectDetails.mentee`):
```
data.projectDetails.mentee.budget.amountInCents → budgets.mentee.amount_in_cents
data.projectDetails.mentee.isActive             → budgets.mentee.is_active
data.projectDetails.mentee.skills               → budgets.mentee.skills
data.projectDetails.mentee.terms                → budgets.mentee.terms
data.projectDetails.mentee.mentor               → budgets.mentee.mentors
data.projectDetails.mentee.customTerm           → budgets.mentee.custom_term
```

**⚠️ Do NOT read `data.mentee` (top-level).** The actual data is always nested under `data.projectDetails.mentee`. Reading from the wrong path silently drops all mentorship metadata — this was the bug that caused the first SQL pass to miss 1,249 of 1,476 rows.

**Drop from migration:**
- `industry` — not in target schema
- `amountRaised` — computed from Ledger
- `cachedDetails.ProjectStats` — recomputed
- `details.Diversity` — deferred feature
- `details.VulnerabilitySummary` — deferred feature

### Field Mapping: Funds (formerly entities)

**Type reclassification (applied during migration):**

| DynamoDB `type` value | Postgres `fund_type` | Row count | Notes |
|---|---|---|---|
| `initiative` | `general_fund` | 121 | UI label was already "General Fund" |
| `general-fund` | `general_fund` | (subset of above) | Normalize hyphen |
| `other` (travel) | `general_fund` | 26 | UI commented out; functionally identical |
| `event` | `event` | 20 | Unchanged |
| `ostif` | `ostif` | 11 | Unchanged |
| `community` | **discard** | 3 | All declined/submitted 2019; no active users |

The old DynamoDB `entityId` maps to `legacy_id` (same as projects — needed for Ledger API calls).
Status normalization applies here too: `'hide'` → `'hidden'`.

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `entityId` | `legacy_id` | Store old string ID for Ledger calls |
| `entityId` | — (new UUID generated) | New `id` is a fresh UUID |
| `ownerId` | `owner_id` | Direct |
| `name` | `name` | Direct |
| `slug` | `slug` | Direct |
| `type` | `fund_type` | Reclassify per table above |
| `status` | `status` | Normalize `'hide'` → `'hidden'` |
| `description` | `description` | Direct |
| `website` | `website` | Direct |
| `logoUrl` | `logo_url` | Direct |
| `details.Beneficiaries` | `beneficiaries` | Marshal to JSONB array |
| `details.Goals` | `budgets` | Marshal to JSONB |
| `createdOn` | `created_at` | Parse timestamp |
| `updatedOn` | `updated_at` | Parse timestamp |

### Field Mapping: Subscriptions

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `stripeSubscriptionId` | `stripe_subscription_id` | **Critical unique constraint** |
| `projectId` | `project_id` | Resolve to Postgres UUID via `_id_migration_map` |
| `userId` | `user_id` | Direct (Auth0 subject) |
| `frequency` | `frequency` | Direct (`monthly` \| `annual`) |
| `amountInCents` | `amount_in_cents` | Direct |
| `status` | `status` | Direct |
| `createdOn` | `created_at` | Parse timestamp |
| `updatedOn` | `updated_at` | Parse timestamp |

For `lff-prod-entity-subscriptions`: same mapping with `fund_id` instead of `project_id`.
Resolve old `entityId` to Postgres `funds.id` via `_id_migration_map`.

### Field Mapping: Donations

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `stripeChargeId` | `stripe_charge_id` | **Critical unique constraint** |
| `projectId` | `project_id` | Resolve to Postgres UUID via `_id_migration_map` |
| `userId` | `user_id` | Direct |
| `orgId` | `org_id` | Resolve to Postgres UUID via `_id_migration_map` if set |
| `name` | `name` | Direct |
| `avatarUrl` | `avatar_url` | Direct |
| `amountInCents` | `amount_in_cents` | Direct |
| `category` | `category` | Direct |
| `paymentMethod` | `payment_method` | Direct |
| `poNumber` | `po_number` | Direct |
| `status` | `status` | Direct |
| `createdOn` | `created_at` | Parse timestamp |

For `lff-prod-entity-donations`: same mapping with `fund_id` instead of `project_id`.

### ID Mapping Problem

DynamoDB uses string IDs (e.g., `projectId: "abc-123"`). Postgres uses UUIDs. The new system generates new UUIDs for all records. During migration, a mapping table is needed:

```sql
CREATE TABLE crowdfunding._id_migration_map (
  old_id      text NOT NULL,
  entity_type text NOT NULL,  -- project|entity|organization
  new_id      uuid NOT NULL,
  PRIMARY KEY (old_id, entity_type)
);
```

This table is used during migration to resolve `projectId` references in subscriptions and donations. It can be dropped after migration is validated.

### Data Quality Checks (run in `--validate` mode)

- Subscriptions referencing non-existent project IDs → log as warnings, skip or null-out
- Donations referencing non-existent project IDs → same
- Duplicate `stripe_subscription_id` values → error (must not exist)
- Duplicate `stripe_charge_id` values → error (must not exist)
- Missing `slug` on projects → error (slug is UNIQUE NOT NULL)
- Projects with same slug → error
- Null `owner_id` → error
- Invalid `status` values → log and map to closest valid status
- `amountInCents` outside valid range → log as warning

### Migration Execution Order

Run tables in this order (respect FK dependencies):

1. `organizations`
2. `projects`
3. `entities`
4. `users`
5. `subscriptions` (project + entity, merged)
6. `donations` (project + entity, merged)
7. Drop `_id_migration_map` after validation

---

## Phase 3: Validation

Before cutover, run a reconciliation pass:

1. **Record counts:** DynamoDB item count == Postgres row count for each table
2. **Spot checks:** Sample 20 records per table, compare DynamoDB vs Postgres field by field
3. **Active subscriptions:** Every `stripeSubscriptionId` in DynamoDB is present in Postgres with `status = active`
4. **Stripe cross-check:** Query Stripe API for active subscriptions; verify each one has a matching Postgres record
5. **Financial totals:** Sum of `amountInCents` in DynamoDB donations == sum of `amount_in_cents` in Postgres donations (per project)

Document results of validation. Keep the validation report alongside migration logs.

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

### Cutover Steps

1. Put old system in read-only mode (disable writes) — or accept brief dual-write window
2. Run final incremental migration (any records created since the last full migration)
3. Switch DNS / K8s ingress from old Lambda API Gateway to new K8s service
4. Smoke test: login, view projects, make a test donation (test card), check subscription list
5. Monitor for errors (Go service logs, Postgres errors)
6. Keep old Lambda running for 2 weeks minimum (rollback window)

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
3. Decommission OpenSearch (after Reimbursement Service migrated off it)
4. Archive old LFF and lfx-crowdfunding-upgrade repos (read-only)

---

## Production Data Size (as of May 2026)

2,013 total rows: 1,832 projects + 181 entities. At this volume, a full DynamoDB scan is fast
(seconds, not minutes) and a full Postgres load is trivial. No need for chunked/streaming
migration — a single-pass load is fine.

Migration record counts to verify after execute:

| Table | Expected rows |
|---|---|
| `crowdfunding.projects` | 1,832 |
| `crowdfunding.entities` | 181 |
| `crowdfunding.organizations` | TBD (query DynamoDB) |
| `crowdfunding.subscriptions` | TBD (query DynamoDB) |
| `crowdfunding.donations` | TBD (query DynamoDB) |

---

## Outstanding Migration Questions

- **`community` entity type:** 3 entity rows have type `community`, which does not appear in the Go codebase entity type enum (`project`, `initiative`, `general-fund`). Must decide: add `community` to the enum, or map to `general-fund`. Needs clarification from product owner before migration code is written.
- **`other (travel)` entity type:** 26 rows. Likely `general-fund` subtype used for travel funds. Confirm exact DynamoDB type value before migration.
- **Mentorship projects (1,476 rows):** These have a `jobspring_id` linking them to the Mentorship service. The SQS consumer must keep working post-migration so new Mentorship projects continue arriving. The `jobspring_id` column is how the new system will match incoming SQS update events to existing Postgres rows (instead of the old DynamoDB `projectId` string). Migration must populate `jobspring_id` from the DynamoDB `projectId` for all mentorship-type projects.
- **Old project IDs and Ledger:** Ledger records use `project_id` as plain text (the old DynamoDB string ID). The Ledger Service is not migrated in the initial release. The `_id_migration_map` table (or a `legacy_id` column on projects) bridges old string IDs to new Postgres UUIDs when calling `GET /balance/{projectID}` on the Ledger API. Recommended: add a `legacy_id text` column to `projects` and `entities` tables, populated during migration, never exposed in the public API.
- **Stripe subscription continuity:** Active Stripe subscriptions must not be cancelled or recreated. The migration preserves `stripe_subscription_id` — Stripe continues charging the same plan. No Stripe API calls needed during migration.
- **Non-published records:** 639 rows are not published (submitted, declined, hidden). All must be migrated — active subscriptions or pending approvals may reference them. Never filter to published-only during migration.
