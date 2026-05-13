<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

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
2. **Data migration** — one-time copy from DynamoDB to Postgres using `migrate_dynamo_to_postgres.py`
3. **Validation** — reconcile record counts, spot-checks, Stripe cross-check
4. **Cutover** — switch DNS/Ingress from old Lambda API Gateway to new K8s service
5. **Decommission** — tear down old Lambda stack, DynamoDB; OpenSearch decommission is a separate later phase (blocked on RS moving to K8s, see OQ-7)

These phases are sequential and gated by human review. Do not proceed to the next phase without explicit sign-off.

---

## Phase 1: Schema Review (no data touched)

**Goal:** finalize the Postgres schema before writing migration code.

Steps:
1. Write initial SQL migration files (`db/migrations/001_initial.up.sql`)
2. Review schema with team — specifically:
   - Normalized `initiative_goals` child table for budget categories (one row per category per initiative; replaces earlier JSONB budgets design)
   - Unified `initiatives` table with `initiative_type` discriminator (replaces old `projects` + `entities` split)
   - `stripe_plan_id` / `stripe_product_id` preserved on initiatives (must survive migration)
   - `stripe_subscription_id` / `stripe_charge_id` — nullable `VARCHAR(255)`; no UNIQUE constraint; uniqueness enforced by Stripe, not the DB
   - `category` — free-form `TEXT` on subscriptions and donations; validated at API layer, no DB CHECK constraint
3. Run migrations against a local dev Postgres instance
4. **Sign-off required before Phase 2**

---

## Phase 2: Data Migration (`migrate_dynamo_to_postgres.py`)

### Tool: Python script

Location: `db/scripts/migrate_dynamo_to_postgres.py` in `linuxfoundation/lfx-crowdfunding` (this repo). Migration status: **complete** — exit 0 against production data (2026-05-12).

Uses `boto3` for DynamoDB scans and `psycopg2` for Postgres upserts. The script is idempotent: all inserts use `ON CONFLICT DO UPDATE`, so it can be re-run safely against the same target DB.

### UUID strategy — deterministic UUID5

All surrogate PKs are generated deterministically via `uuid5(UUID_NS, "{scope}:{natural_key}")` where `UUID_NS = 6ba7b810-9dad-11d1-80b4-00c04fd430c8`. This ensures re-runs produce the same IDs and FK relationships remain stable. Subscriptions and donations resolve old DynamoDB string IDs to Postgres UUIDs by recomputing the same `uuid5` of the old ID.

### Migration phases

```
Phase 1 — users
Phase 2 — organizations
Phase 3 — initiatives + 15 child tables
Phase 4 — mentorship reclassification (UPDATE initiative_type = 'mentorship')
Phase 5 — donations
Phase 6 — subscriptions
```

### DynamoDB Tables to Migrate

| DynamoDB Table | Target Postgres Table | Notes |
|---|---|---|
| `lff-prod-projects` | `crowdfunding.initiatives` | Set `initiative_type` (`project` or `mentorship` — reclassify in Phase 4); populate 15 child tables |
| `lff-prod-entities` | `crowdfunding.initiatives` | Reclassify DynamoDB `entityType` → `initiative_type` per type mapping table below; populate child tables |
| `lff-prod-organizations` | `crowdfunding.organizations` | Straightforward |
| `lff-prod-subscriptions` | `crowdfunding.subscriptions` | Preserve `stripe_subscription_id`; resolve old `projectId` → `initiative_id` via uuid5 |
| `lff-prod-entity-subscriptions` | `crowdfunding.subscriptions` | Merge with subscriptions; resolve old `entityId` → `initiative_id` via uuid5 |
| `lff-prod-donations` | `crowdfunding.donations` | Preserve `stripe_charge_id`; resolve old `projectId` → `initiative_id` via uuid5 |
| `lff-prod-entity-donations` | `crowdfunding.donations` | Merge with donations; resolve old `entityId` → `initiative_id` via uuid5 |
| `lff-prod-users` | `crowdfunding.users` | User profile (email, given_name, family_name, name, avatar_url) |

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
| `projectId` | — | Source natural key; deterministic UUID generated via `_as_uuid(projectId)` (preserves UUID-form IDs; otherwise `uuid5("coerce", projectId)`); stable across re-runs |
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
| `jobspringProjectId` | `jobspring_project_id` | Direct; field name uses lowercase `d` |
| `planId` | `stripe_plan_id` | **Critical — must be preserved exactly** |
| `productId` | `stripe_product_id` | **Critical — must be preserved exactly** |
| `amountRaised` | `amount_raised_in_cents` | Convert to cents if needed |
| `createdOn` | `created_on` | Parse timestamp |
| `updatedOn` | `updated_on` | Parse timestamp |

**Budget mapping — inserted into `initiative_goals` child table:**

For `initiative_type = 'project'` (read from `projectDetails.*`):
```text
projectDetails.development  → initiative_goals row  name='development'  amount_in_cents, allocation, repo_link
projectDetails.marketing    → initiative_goals row  name='marketing'    amount_in_cents, allocation
projectDetails.meetups      → initiative_goals row  name='meetups'      amount_in_cents, allocation
projectDetails.travel       → initiative_goals row  name='travel'       amount_in_cents, allocation
projectDetails.bugBounty    → initiative_goals row  name='bugBounty'    amount_in_cents, allocation
projectDetails.documentation → initiative_goals row name='documentation' amount_in_cents, allocation
projectDetails.mentee       → initiative_goals row  name='mentee'       amount_in_cents (simple: amount + description only)
projectDetails.other        → initiative_goals row  name='other'        amount_in_cents, allocation
```

For `initiative_type = 'mentorship'` (read from `projectDetails.mentee`):
```text
projectDetails.mentee.budget.amountInCents → initiative_goals row name='mentee' amount_in_cents
projectDetails.mentee.skills               → initiative_program_info_skills rows
projectDetails.mentee.terms                → initiative_program_info_terms rows
projectDetails.mentee.mentor               → initiative_mentors rows
projectDetails.mentee.customTerm           → initiative_program_info_custom_term row
```

**Child table insertions (all initiative types):**
```text
projectDetails.contributors  → initiative_contributors rows
projectDetails.beneficiaries → initiative_beneficiaries rows
projectDetails.customWebsites → initiative_custom_websites rows
cachedDetails.githubStats    → initiative_github_stats row
cachedDetails.projectStats.backers → initiative_stats row
```

**⚠️ Do NOT read the top-level `mentee` attribute.** The actual data is always nested under `projectDetails.mentee`. Reading from the wrong path silently drops all mentorship metadata — this was the bug that caused the first SQL pass to miss all 1,486 reclassified rows.

**Fields excluded from migration (no Postgres column):**
- `cachedDetails.ProjectStats.totalRaised` — no active write path; always 0 in production
- `details.Diversity` — deferred feature
- `details.VulnerabilitySummary` — deferred feature
- `projectDetails.sponsors` — read-time only via `GetEntitySponsors`; no write path in `SaveProject`

### Field Mapping: Initiatives from entities (formerly `lff-prod-entities`)

These rows migrate into the same `crowdfunding.initiatives` table as projects. The DynamoDB `entityType` value maps to `initiative_type` per the table below.

**Type reclassification (applied during migration):**

| DynamoDB `entityType` value | Postgres `initiative_type` | Row count | Notes |
|---|---|---|---|
| `initiative` | `general fund` | 121 | UI label was already "General Fund" |
| `general-fund` | `general fund` | (subset of above) | Normalize hyphen |
| `other` | `other` | 26 | DynamoDB `entityType = 'other'`; migrated as-is |
| `event` | `event` | 20 | Unchanged |
| `ostif` | `ostif` | 11 | Unchanged |
| `community` | `community` | 3 | Migrated as-is; all declined/submitted 2019; no active users |

The old DynamoDB `entityId` maps to the Postgres `id` via `_as_uuid(entityId)` (preserves UUID-form IDs; otherwise `uuid5("coerce", entityId)` — same as projects).
Status normalization applies here too: `'hide'` → `'hidden'`.

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `entityId` | — (UUID via _as_uuid) | New `id` is deterministic: UUID-form IDs preserved; non-UUID strings → `uuid5("coerce", entityId)` |
| `ownerId` | `owner_id` | Direct |
| `name` | `name` | Direct |
| `slug` | `slug` | Direct |
| `entityType` | `initiative_type` | Reclassify per table above (DynamoDB field is `entityType`, not `type`) |
| `status` | `status` | Normalize `'hide'` → `'hidden'` |
| `description` | `description` | Direct |
| `websiteURL` | `website_url` | Direct |
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
| `amountRaised` | `amount_raised_in_cents` | Convert to cents if needed |
| `createdOn` | `created_on` | Parse timestamp |
| `updatedOn` | `updated_on` | Parse timestamp |

**Child table insertions:**
```text
entity.Beneficiary[]    → initiative_beneficiaries rows
entity.Goals[]          → initiative_goals rows (free-form name from goal.Name)
entity.SponsorshipTiers[] → initiative_sponsorship_tiers rows
entity.Detail (ostif)   → initiative_ostif_detail row + initiative_contacts rows
entity.EntityDetails    → initiative_entity_details row (JSONB)
entity.CustomWebsites[] → initiative_custom_websites rows
```

### Field Mapping: Subscriptions

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `stripeSubscriptionId` | `stripe_subscription_id` | Nullable `VARCHAR(255)`; no UNIQUE constraint |
| `stripeSubscriptionItemId` | `stripe_subscription_item_id` | Nullable; needed for Stripe price/quantity updates |
| `projectId` | `initiative_id` | Resolve old `projectId` → new Postgres UUID via `uuid5` of `projectId` |
| `userId` | `user_id` | Direct (Auth0 subject) |
| `frequency` | `frequency` | Only `monthly` observed in all 270/270 production records; `yearly` is valid but unused |
| `currentAmountInCents` | `current_amount_in_cents` | Same field name as donations (`currentAmountInCents`, not `amountInCents`) |
| `status` | `status` | Actual prod values: `"active"` \| `"inactive"` (not `"cancelled"` or `"past_due"`) |
| `createdOn` | `created_on` | ISO 8601 `T`-separator format (e.g. `2020-01-22T12:51:06Z`) |
| *(absent)* | `updated_on` | Field does not exist in DynamoDB subscriptions — default to `created_on` value on migration |

For `lff-prod-entity-subscriptions`: same mapping, but resolve old `entityId` → new Postgres UUID via `uuid5` of `entityId` and write it to `initiative_id`.

### Field Mapping: Donations

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `stripeChargeId` | `stripe_charge_id` | Nullable `VARCHAR(255)`; no UNIQUE constraint; null for invoice payments |
| `projectId` | `initiative_id` | Resolve old `projectId` → new Postgres UUID via `uuid5` of `projectId` |
| `userId` | `user_id` | Direct |
| `orgId` | `organization_id` | Resolve to Postgres UUID via `uuid5` of `orgId` if set |
| `cachedDetails.backerDetails.name` | `cached_details` | Preserved as JSONB; nested path |
| `currentAmountInCents` | `current_amount_in_cents` | DynamoDB field is `currentAmountInCents`, not `amountInCents` |
| `category` | `category` | Direct |
| `paymentmethod` | `payment_method` | DynamoDB field is `paymentmethod` (all lowercase) |
| `ponumber` | `po_number` | DynamoDB field is `ponumber` (all lowercase) |
| `status` | `status` | Direct; null on all production rows — migrate as null |
| `createdOn` | `created_on` | Parse timestamp |

For `lff-prod-entity-donations`: same mapping, but resolve old `entityId` → new Postgres UUID via `uuid5` of `entityId` and write it to `initiative_id`.

### Field Mapping: Organizations

Organization IDs in DynamoDB are already UUIDs — migrate `organizationId` directly to `id` (no new UUID needed, no separate mapping required).

All production rows have `status = "approved"`. No `description`, `website`, or `rejected_at` fields exist in DynamoDB — those columns were dropped from the schema.

| DynamoDB attribute | Postgres column | Notes |
|---|---|---|
| `organizationId` | `id` | Already UUID — preserve as-is |
| `ownerId` | `owner_id` | Direct (Auth0 subject) |
| `name` | `name` | Direct |
| `status` | `status` | Direct; all prod rows are `"approved"` |
| `avatarUrl` | `avatar_url` | DynamoDB field is `avatarUrl` (capital U, lowercase rl) |

### ID Strategy — Deterministic UUID5

DynamoDB uses string IDs (e.g., `projectId: "abc-123"`). Postgres uses UUIDs. All PKs are generated deterministically via `uuid5(UUID_NS, "{scope}:{natural_key}")` where `UUID_NS = 6ba7b810-9dad-11d1-80b4-00c04fd430c8`. When migrating subscriptions and donations, the script recomputes `uuid5` of the old `projectId`/`entityId` to resolve the correct `initiative_id` in Postgres. No separate mapping table is needed.

### Data Quality Checks

- Subscriptions referencing non-existent project or entity IDs → log as warnings, skip or null-out
- Donations referencing non-existent project or entity IDs → same
- `stripe_subscription_id` / `stripe_charge_id` — no UNIQUE constraint in schema; duplicates are not an error at the DB level
- Missing `slug` on initiatives → log as warning (`slug` is nullable TEXT with an index, not NOT NULL UNIQUE)
- Initiatives with same slug → log as warning (no UNIQUE constraint on slug)
- Null `owner_id` → error
- Unexpected `status` values → log as warning; normalize `'hide'` → `'hidden'`; otherwise preserve the source value unchanged
- `current_amount_in_cents` outside valid range → log as warning

### Migration Execution Order

Run tables in this order (respect FK dependencies):

1. `users` (all tables have FK to `users.user_id`)
2. `organizations`
3. `initiatives` (from `lff-prod-entities` — general funds, events, OSTIF, other, community; inserted first)
4. `initiatives` (from `lff-prod-projects` — projects and mentorship; appended to same table)
5. Mentorship reclassification (`UPDATE initiative_type = 'mentorship'` for rows with `mentee` goal)
6. All 15 `initiative_*` child tables (goals, beneficiaries, mentors, skills, terms, etc.)
7. `donations` (from `lff-prod-donations` + `lff-prod-entity-donations`, merged)
8. `subscriptions` (from `lff-prod-subscriptions` + `lff-prod-entity-subscriptions`, merged)
9. Verify all subscriptions and donations resolved correctly (no nulled-out `initiative_id` rows in validation report)

---

## Phase 3: Validation

Before cutover, run a reconciliation pass:

1. **Record counts:** DynamoDB item count == Postgres row count for each table
2. **Spot checks:** Sample 20 records per table, compare DynamoDB vs Postgres field by field
3. **Active subscriptions:** Every `stripeSubscriptionId` in DynamoDB is present in Postgres with `status = 'active'`
4. **Stripe cross-check:** Query Stripe API for active subscriptions; verify each one has a matching Postgres record
5. **Financial totals:** Sum of `currentAmountInCents` in DynamoDB donations == sum of `current_amount_in_cents` in Postgres donations (per initiative)

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
- [ ] Ledger Service updated and deployed — auth headers fixed in `fundspring.go`: `x-ledger-auth` → `Authorization: Bearer` for `GetProject`/`GetUserName`; `Authorization: Bearer` added to `GetOrganizationName` (currently sends no auth — a Ledger bug); must be live before DNS cutover or donation confirmation emails break immediately
- [ ] `STRIPE_WEBHOOK_SIGNING_SECRET` provisioned in new CF service (AWS Secrets Manager / ESO) — required to verify the `Stripe-Signature` HMAC on incoming `POST /v1/hooks/stripe` events via `webhook.ConstructEvent()`; this is a separate credential from the Stripe API key; the webhook URL itself does not change (same domain, DNS cutover swaps what is behind it)

### Cutover Steps

1. Put old system in read-only mode (disable writes) — or accept brief dual-write window
2. Run final incremental migration (any records created since the last full migration)
3. **Run `amount_raised_in_cents` pre-population** — execute the `amount-raised-sync` CronJob manually against prod Ledger API to populate `amount_raised_in_cents` for all migrated initiatives before DNS switches. This ensures no published initiative card shows `$0 raised` incorrectly on day one.
4. Switch DNS / K8s ingress from old Lambda API Gateway to new K8s service — Stripe webhooks now land on the new service automatically (same URL, same domain)
5. Smoke test: login, view projects, make a test donation (test card), check subscription list
6. Monitor for errors (Go service logs, Postgres errors)
7. Keep old Lambda running for 2 weeks minimum (rollback window)

**Stripe webhook rollback caveat:** If DNS is switched back to the old Lambda, any `customer.subscription.deleted` events that Stripe delivered to the new service during the forward window will not be re-delivered to the old Lambda — Stripe delivers each event once. A subscription cancelled during the forward window may appear active in DynamoDB after rollback. Before decommission, cross-check active subscriptions against Stripe API to catch any such gaps (already a decommission step).

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

## Production Data Size (as of May 2026)

2,023 initiative rows (1,841 from `lff-prod-projects` + 182 from `lff-prod-entities`), 16,847 users (16,789 from DynamoDB + 58 placeholders for referenced-but-absent users), 1,598 donations, 277 subscriptions. At this volume, a full DynamoDB scan is fast (seconds, not minutes) and a full Postgres load is trivial. No need for chunked/streaming migration — a single-pass load is fine.

Migration record counts to verify after execute:

| Table | Expected rows | Source |
|---|---|---|
| `crowdfunding.users` | 16,847 (16,789 from DynamoDB + 58 placeholders) | `lff-prod-users` |
| `crowdfunding.organizations` | 606 | `lff-prod-organizations` |
| `crowdfunding.initiatives` | 2,023 (1,841 projects + 182 entities; community rows included) | `lff-prod-projects` + `lff-prod-entities` |
| `crowdfunding.initiative_goals` | 4,438 | projects + entities |
| `crowdfunding.initiative_beneficiaries` | 2,004 | both |
| `crowdfunding.initiative_contributors` | 37 | projects |
| `crowdfunding.initiative_custom_websites` | 1 | both |
| `crowdfunding.initiative_mentors` | 2,914 | projects |
| `crowdfunding.initiative_program_info_skills` | 5,472 | projects |
| `crowdfunding.initiative_program_info_terms` | 92 | projects |
| `crowdfunding.initiative_program_info_config` | 1,841 | projects |
| `crowdfunding.initiative_program_info_custom_term` | 11 | projects |
| `crowdfunding.initiative_sponsorship_tiers` | 0 | entities |
| `crowdfunding.initiative_github_stats` | 1,552 | projects |
| `crowdfunding.initiative_stats` | 1,841 | projects |
| `crowdfunding.initiative_ostif_detail` | 11 | entities |
| `crowdfunding.initiative_contacts` | 26 | entities |
| `crowdfunding.initiative_entity_details` | 0 | entities |
| `crowdfunding.donations` | 1,598 (1 skipped — orphaned initiative FK) | `lff-prod-donations` + `lff-prod-entity-donations` |
| `crowdfunding.subscriptions` | 277 | `lff-prod-subscriptions` + `lff-prod-entity-subscriptions` |

---

## Migration Notes

- **`community` entity type:** Resolved — 3 rows, all declined/submitted in 2019 with no active users. Migrated as `initiative_type = 'community'` (not discarded). No new rows expected.
- **`other` entity type:** Resolved — 26 rows with DynamoDB `entityType = 'other'`. Migrated as `initiative_type = 'other'` (not merged into `general fund`).
- **Mentorship initiatives (1,486 rows):** Detection: rows where `initiative_goals.name = 'mentee'` and `amount_in_cents > 0` are reclassified to `initiative_type = 'mentorship'` in Phase 4. The `jobspring_project_id` column is populated from the DynamoDB `jobspringProjectId` field. New mentorship programs arrive post-migration via the `mentorship-sync` Snowflake CronJob — SNS/SQS is not used.
- **Old IDs and Ledger:** Ledger records use the old DynamoDB string ID as `project_id`. The migration script uses `_as_uuid()` to generate Postgres UUIDs: if the DynamoDB ID is already a valid UUID, it is preserved unchanged as `initiatives.id`; otherwise a deterministic `uuid5("coerce", id)` is generated. For UUID-form DynamoDB IDs (the vast majority), the Postgres UUID matches the original ID exactly, so the CF Go API can use `initiatives.id` directly when calling Ledger. For the small number of non-UUID legacy IDs, the service must store/maintain a mapping if Ledger lookups are required. `source_dynamo_table` (migration-only column, dropped post-cutover) records origin for auditability.
- **Stripe subscription continuity:** Active Stripe subscriptions must not be cancelled or recreated. The migration preserves `stripe_subscription_id` — Stripe continues charging the same plan. No Stripe API calls needed during migration.
- **Non-published records:** 639 rows are not published (submitted, declined, hidden). All must be migrated — active subscriptions or pending approvals may reference them. Never filter to published-only during migration.
