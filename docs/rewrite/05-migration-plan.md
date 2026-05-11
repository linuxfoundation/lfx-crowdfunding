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
3. **Validation** — reconcile record counts, spot-checks, Stripe cross-check
4. **Cutover** — switch DNS/Ingress from old Lambda API Gateway to new K8s service
5. **Decommission** — tear down old Lambda stack, DynamoDB, OpenSearch

These phases are sequential and gated by human review. Do not proceed to the next phase without explicit sign-off.

---

## Phase 1: Schema Review (no data touched)

**Goal:** finalize the Postgres schema before writing migration code.

Steps:
1. Write initial SQL migration files (`db/migrations/001_initial.up.sql`)
2. Review schema with team — specifically:
   - JSONB columns for budgets (confirmed decision)
   - Unified `initiatives` table with `initiative_type` discriminator (replaces old `projects` + `entities` split)
   - `stripe_plan_id` / `stripe_product_id` preserved on initiatives (must survive migration)
   - `stripe_subscription_id` UNIQUE constraint (active subscriptions — must survive)
   - `stripe_charge_id` UNIQUE constraint (donations — idempotency)
   - `category` CHECK constraint on subscriptions and donations
3. Run migrations against a local dev Postgres instance
4. **Sign-off required before Phase 2**

---

## Phase 2: Data Migration (`migrate-cf` CLI)

### Tool: Dedicated Go CLI

Location: `cmd/migrate-cf/` in the new CF repo. (`cmd/migrate/` is reserved for the golang-migrate schema runner — these are two distinct tools.)

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
| `lff-prod-projects` | `crowdfunding.initiatives` | Set `initiative_type` (`project` or `mentorship`); branch budget path by type |
| `lff-prod-entities` | `crowdfunding.initiatives` | Reclassify DynamoDB `entityType` → `initiative_type` per type mapping table below |
| `lff-prod-organizations` | `crowdfunding.organizations` | Straightforward |
| `lff-prod-subscriptions` | `crowdfunding.subscriptions` | Preserve `stripe_subscription_id`; resolve old `projectId` → `initiative_id` |
| `lff-prod-entity-subscriptions` | `crowdfunding.subscriptions` | Merge with subscriptions; resolve old `entityId` → `initiative_id` |
| `lff-prod-donations` | `crowdfunding.donations` | Preserve `stripe_charge_id`; resolve old `projectId` → `initiative_id` |
| `lff-prod-entity-donations` | `crowdfunding.donations` | Merge with donations; resolve old `entityId` → `initiative_id` |
| `lff-prod-users` | `crowdfunding.users` | Minimal — just `stripe_customer_id` |

**Not migrated:** Ledger transactions remain in the Ledger Service Postgres DB unchanged.

### Field Mapping: Projects

**Critical:** DynamoDB project rows contain two distinct data shapes depending on origin.
The migration tool MUST detect the type and branch accordingly.

**Detecting initiative_type:**
- If `jobspringProjectId` field is present and non-empty → `initiative_type = 'mentorship'`
- Otherwise → `initiative_type = 'project'`

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
| `projectDetails.website` | `website` | Nested under `projectDetails` |
| `projectDetails.description` | `description` | Nested under `projectDetails` |
| `projectDetails.color` | `color` | Nested under `projectDetails` |
| `logoUrl` | `logo_url` | Direct (top-level) |
| `projectDetails.codeOfConduct.link` | `code_of_conduct` | Nested; extract `.link` string |
| `projectDetails.ciiProjectID` | `cii_project_id` | Nested under `projectDetails` |
| `projectDetails.stacksIdentifier` | `stacks_id` | Nested under `projectDetails` |
| `projectDetails.industry` | `industry` | Nested under `projectDetails`; comma-separated topic tags string |
| `jobspringProjectId` | `mentorship_program_id` | Direct (renamed column); field name uses lowercase `d` |
| `planId` | `stripe_plan_id` | **Critical — must be preserved exactly** |
| `productId` | `stripe_product_id` | **Critical — must be preserved exactly** |
| `projectDetails.contributors` | `contributors` | Marshal to JSONB array |
| `projectDetails.beneficiaries` | `beneficiaries` | Marshal to JSONB array |
| `projectDetails.customWebsites` | `custom_websites` | Marshal to JSONB array |
| `projectDetails.sponsors` | `sponsors` | Marshal to JSONB array |
| `cachedDetails.GithubStats` | `github_stats` | Marshal to JSONB |
| `amountRaised` | — | **Drop** — replaced by Ledger view |
| `createdOn` | `created_at` | Parse timestamp |
| `updatedOn` | `updated_at` | Parse timestamp |

**Budget mapping — branches by initiative_type:**

For `initiative_type = 'project'` (read from `data.projectDetails.*`):
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

For `initiative_type = 'mentorship'` (read from `data.projectDetails.mentee`):
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
- `amountRaised` — computed from Ledger
- `cachedDetails.ProjectStats` — recomputed
- `details.Diversity` — deferred feature
- `details.VulnerabilitySummary` — deferred feature

### Field Mapping: Initiatives from entities (formerly `lff-prod-entities`)

These rows migrate into the same `crowdfunding.initiatives` table as projects. The DynamoDB `entityType` value maps to `initiative_type` per the table below.

**Type reclassification (applied during migration):**

| DynamoDB `entityType` value | Postgres `initiative_type` | Row count | Notes |
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
| `entityType` | `initiative_type` | Reclassify per table above (DynamoDB field is `entityType`, not `type`) |
| `status` | `status` | Normalize `'hide'` → `'hidden'` |
| `description` | `description` | Direct |
| `websiteURL` | `website` | Direct (DynamoDB field is `websiteURL`) |
| `logoUrl` | `logo_url` | Direct |
| `city` | `city` | Direct; nullable |
| `country` | `country` | Direct; nullable |
| `isOnline` | `is_online` | Direct boolean; default false if missing |
| `acceptFunding` | `accept_funding` | Direct boolean; default true if missing |
| `applicationURL` | `application_url` | Direct; present on travel fund / scholarship types |
| `eventStartDate` | `event_start_date` | Parse date; event type only |
| `eventEndDate` | `event_end_date` | Parse date; event type only |
| `eventbriteId` | `eventbrite_url` | Direct (despite the name, contains a URL); event type only |
| `industry` | `industry` | Direct; comma-separated topic tags string |
| `details.Beneficiaries` | `beneficiaries` | Marshal to JSONB array |
| `goals` | `budgets` | Marshal to JSONB array (DynamoDB field is `goals`, a top-level array — not `details.Goals`) |
| `amountRaised` | — | **Drop** — computed from Ledger |
| `approvedOn` | `approved_at` | Parse timestamp; nullable. Note: not observed in any production sample — may be absent on all entity rows. Migrate if present, leave null otherwise. |
| `createdOn` | `created_at` | Parse timestamp |
| `updatedOn` | `updated_at` | Parse timestamp |

### Field Mapping: Subscriptions

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `stripeSubscriptionId` | `stripe_subscription_id` | **Critical unique constraint** |
| `stripeSubscriptionItemId` | `stripe_subscription_item_id` | Nullable; needed for Stripe price/quantity updates |
| `projectId` | `initiative_id` | Resolve old `projectId` → new Postgres UUID via `_id_migration_map` |
| `userId` | `user_id` | Direct (Auth0 subject) |
| `frequency` | `frequency` | Direct (`monthly` \| `annual`) |
| `currentAmountInCents` | `amount_in_cents` | Same field name as donations (`currentAmountInCents`, not `amountInCents`) |
| *(absent)* | `payment_method` | Field does not exist in DynamoDB subscriptions — column is nullable; set `NULL` on migration |
| `status` | `status` | Actual prod values: `"active"` \| `"inactive"` (not `"cancelled"` or `"past_due"`) |
| `createdOn` | `created_at` | ISO 8601 `T`-separator format (e.g. `2020-01-22T12:51:06Z`) |
| *(absent)* | `updated_at` | Field does not exist in DynamoDB subscriptions — default to `created_at` value on migration |

For `lff-prod-entity-subscriptions`: same mapping, but resolve old `entityId` → new Postgres UUID via `_id_migration_map` and write it to `initiative_id`.

### Field Mapping: Donations

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `stripeChargeId` | `stripe_charge_id` | **Critical unique constraint**; null for invoice payments — Postgres UNIQUE allows multiple NULLs |
| `projectId` | `initiative_id` | Resolve old `projectId` → new Postgres UUID via `_id_migration_map` |
| `userId` | `user_id` | Direct |
| `orgId` | `org_id` | Resolve to Postgres UUID via `_id_migration_map` if set |
| `cachedDetails.backerDetails.name` | `name` | Nested path |
| `cachedDetails.backerDetails.avatarURL` | `avatar_url` | Nested path; DynamoDB key is `avatarURL` (uppercase URL) |
| `currentAmountInCents` | `amount_in_cents` | DynamoDB field is `currentAmountInCents`, not `amountInCents` |
| `category` | `category` | Direct |
| `paymentmethod` | `payment_method` | DynamoDB field is `paymentmethod` (all lowercase) |
| `ponumber` | `po_number` | DynamoDB field is `ponumber` (all lowercase) |
| `status` | `status` | Direct; null on all production rows — migrate as null |
| `createdOn` | `created_at` | Parse timestamp |

For `lff-prod-entity-donations`: same mapping, but resolve old `entityId` → new Postgres UUID via `_id_migration_map` and write it to `initiative_id`.

### Field Mapping: Organizations

Organization IDs in DynamoDB are already UUIDs — migrate `organizationId` directly to `id` (no new UUID needed, no `_id_migration_map` entry required).

All production rows have `status = "approved"`. No `description`, `website`, `approved_at`, or `rejected_at` fields exist in DynamoDB — those columns were dropped from the schema.

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `organizationId` | `id` | Already UUID — preserve as-is |
| `ownerId` | `owner_id` | Direct (Auth0 subject) |
| `name` | `name` | Direct |
| `status` | `status` | Direct; all prod rows are `"approved"` |
| `avatarUrl` | `avatar_url` | DynamoDB field is `avatarUrl` (capital U, lowercase rl) |

### ID Mapping Problem

DynamoDB uses string IDs (e.g., `projectId: "abc-123"`). Postgres uses UUIDs. The new system generates new UUIDs for all records. During migration, a mapping table is needed:

```sql
CREATE TABLE crowdfunding._id_migration_map (
  old_id      text NOT NULL,
  source      text NOT NULL,  -- 'project' | 'entity' (organizations are already UUIDs — not needed)
  new_id      uuid NOT NULL,
  PRIMARY KEY (old_id, source)
);
```

This table is used during migration to resolve `projectId` and `entityId` references in subscriptions and donations to the new `initiative_id` UUID. It can be dropped after migration is validated.

### Data Quality Checks (run in `--validate` mode)

- Subscriptions referencing non-existent project or entity IDs → log as warnings, skip or null-out
- Donations referencing non-existent project or entity IDs → same
- Duplicate `stripe_subscription_id` values → error (must not exist)
- Duplicate `stripe_charge_id` values → error (must not exist)
- Missing `slug` on initiatives → error (slug is UNIQUE NOT NULL)
- Initiatives with same slug → error
- Null `owner_id` → error
- Invalid `status` values → log and map to closest valid status
- `amountInCents` outside valid range → log as warning

### Migration Execution Order

Run tables in this order (respect FK dependencies):

1. `organizations`
2. `initiatives` (from `lff-prod-projects` — projects and mentorship)
3. `initiatives` (from `lff-prod-entities` — general funds, events, OSTIF; appended to same table)
4. `users`
5. `subscriptions` (from `lff-prod-subscriptions` + `lff-prod-entity-subscriptions`, merged)
6. `donations` (from `lff-prod-donations` + `lff-prod-entity-donations`, merged)
7. Drop `_id_migration_map` after validation

---

## Phase 3: Validation

Before cutover, run a reconciliation pass:

1. **Record counts:** DynamoDB item count == Postgres row count for each table
2. **Spot checks:** Sample 20 records per table, compare DynamoDB vs Postgres field by field
3. **Active subscriptions:** Every `stripeSubscriptionId` in DynamoDB is present in Postgres with `status = active`
4. **Stripe cross-check:** Query Stripe API for active subscriptions; verify each one has a matching Postgres record
5. **Financial totals:** Sum of `amountInCents` in DynamoDB donations == sum of `amount_in_cents` in Postgres donations (per initiative)

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
- [ ] OQ-15 resolved — Ledger balance lookup mechanism for post-cutover initiatives confirmed

### Cutover Steps

1. Put old system in read-only mode (disable writes) — or accept brief dual-write window
2. Run final incremental migration (any records created since the last full migration)
3. **Run `amount_raised_cents` pre-population** — execute the reconciliation CronJob manually against prod Ledger API to populate `amount_raised_cents` for all migrated initiatives before DNS switches. This ensures no published initiative card shows `$0 raised` incorrectly on day one.
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
3. Decommission OpenSearch (after Reimbursement Service migrated off it)
4. Archive old LFF and lfx-crowdfunding-upgrade repos (read-only)

---

## Production Data Size (as of May 2026)

2,013 total rows: 1,832 from `lff-prod-projects` + 181 from `lff-prod-entities`, all landing in `crowdfunding.initiatives`. At this volume, a full DynamoDB scan is fast (seconds, not minutes) and a full Postgres load is trivial. No need for chunked/streaming migration — a single-pass load is fine.

Migration record counts to verify after execute:

| Table | Expected rows | Source |
|---|---|---|
| `crowdfunding.initiatives` | 2,013 (minus 3 discarded `community` rows = 2,010) | `lff-prod-projects` + `lff-prod-entities` |
| `crowdfunding.organizations` | TBD (query DynamoDB) | `lff-prod-organizations` |
| `crowdfunding.subscriptions` | TBD (query DynamoDB) | `lff-prod-subscriptions` + `lff-prod-entity-subscriptions` |
| `crowdfunding.donations` | TBD (query DynamoDB) | `lff-prod-donations` + `lff-prod-entity-donations` |

---

## Migration Notes

- **`community` entity type:** Resolved — 3 rows, all declined/submitted in 2019 with no active users. Discarded during migration (not inserted into `crowdfunding.initiatives`). No action needed.
- **`other (travel)` entity type:** Resolved — 26 rows. DynamoDB `entityType` value is `other`; mapped to `initiative_type = 'general_fund'` per the type reclassification table above.
- **Mentorship initiatives (1,476 rows):** Have a `mentorship_program_id` linking them to the Mentorship service. Migration must populate `mentorship_program_id` from the DynamoDB `jobspringProjectId` field (see Projects field mapping above). New mentorship programs arrive post-migration via the `mentorship-sync` Snowflake CronJob — SNS/SQS is not used.
- **Old IDs and Ledger:** Ledger records use the old DynamoDB string ID as `project_id`. The `legacy_id` column on `crowdfunding.initiatives` bridges old string IDs to the new Postgres UUIDs when calling `GET /balance/{legacy_id}` on the Ledger API. `legacy_id` is populated during migration from `projectId` (for projects) and `entityId` (for entities). Never exposed in the public API.
- **Stripe subscription continuity:** Active Stripe subscriptions must not be cancelled or recreated. The migration preserves `stripe_subscription_id` — Stripe continues charging the same plan. No Stripe API calls needed during migration.
- **Non-published records:** 639 rows are not published (submitted, declined, hidden). All must be migrated — active subscriptions or pending approvals may reference them. Never filter to published-only during migration.
