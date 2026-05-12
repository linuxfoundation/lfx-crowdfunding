# Migration Plan

DynamoDB → PostgreSQL data migration and cutover plan.

**Owner:** Lewis  
**Schema version:** 2.0.0 (2026-05-07)  
**Source document:** [`docs/rewrite/data-design_and_migration.md`](data-design_and_migration.md) — canonical reference for schema DDL, migration script, and data dictionary.  
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

Ledger DB migration is a separate post-release project.

### Phases

1. **Schema** — apply schema changes in `001_initial.up.sql`; Lewis updates migration script to match (see below)
2. **Data migration** — Python script `migrate_dynamo_to_postgres.py` reads all DynamoDB tables and upserts into Postgres
3. **Validation** — reconcile record counts, spot-checks, Stripe cross-check
4. **Cutover** — switch DNS/Ingress from old Lambda API Gateway to new K8s service
5. **Decommission** — tear down old Lambda stack, DynamoDB; OpenSearch decommission is a separate later phase

These phases are sequential and gated by human review.

---

## Schema Changes (action items for Lewis)

The schema DDL below reflects the agreed target. Lewis needs to update `migrate_dynamo_to_postgres.py` to match each change.

### 1. `users` — add UUID surrogate PK

**Why:** `auth0|username` leaks the auth provider into the data model. All four child tables carry VARCHAR FK columns pointing at Auth0 IDs. If Auth0 is ever replaced, FK values across all those tables would need to be updated in place. Adding a UUID PK now is a small upfront cost.

**Target DDL:**
```sql
CREATE TABLE users (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    VARCHAR(255) NOT NULL UNIQUE,  -- Auth0 subject; used for auth lookups
  email      TEXT,
  given_name TEXT,
  family_name TEXT,
  name       TEXT,
  avatar_url TEXT,
  created_on TIMESTAMPTZ DEFAULT NOW(),
  updated_on TIMESTAMPTZ DEFAULT NOW()
);
```

All FK columns in `donations`, `subscriptions`, `organizations`, and `initiatives` change from `VARCHAR(255) REFERENCES users(user_id)` → `UUID REFERENCES users(id)`.

**Migration script changes needed:**
- `migrate_users`: generate a UUID `id` per user row; update INSERT to include `id`; conflict target stays `user_id`
- `migrate_organizations`, `migrate_initiatives`, `migrate_donations`, `migrate_subscriptions`: resolve `ownerId`/`userId` to the Postgres `users.id` UUID (via an in-memory `user_id → id` map built after migrating users) instead of passing the Auth0 string directly

---

### 2. `organizations` — drop `organization_id`

**Why:** The data dictionary confirms it is a pure duplicate of `id` ("Duplicate of `id` for backward compatibility"). Both columns receive the same UUID value from `_as_uuid(organizationId)`. Keeping it confuses future developers.

**Target DDL:**
```sql
CREATE TABLE organizations (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id   UUID         NOT NULL REFERENCES users(id),
  name       TEXT         NOT NULL,
  avatar_url TEXT,
  status     VARCHAR(50),
  created_on TIMESTAMPTZ  DEFAULT NOW(),
  updated_on TIMESTAMPTZ  DEFAULT NOW()
);
```

**Migration script changes needed:**
- Remove `organization_id` from the INSERT column list and `ON CONFLICT (id) DO UPDATE SET` clause in `migrate_organizations`

---

### 3. `initiatives` — drop `initiative_id`, normalize all child table FKs to UUID

**Why:** In practice `id` and `initiative_id` hold the same value — DynamoDB `projectId` and `entityId` are already UUIDs, so `_as_uuid(projectId)` is a no-op. The only difference is column type (UUID vs VARCHAR). The current design creates an inconsistent FK split: 15 child tables use VARCHAR FKs, while `donations`/`subscriptions` use UUID FKs. Normalizing now costs one migration script update; doing it post-launch requires a FK migration on live data.

**Target DDL (initiatives + example child table):**
```sql
CREATE TABLE initiatives (
  id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  -- initiative_id dropped; id IS the natural key (DynamoDB IDs are already UUIDs)
  initiative_type VARCHAR(50)  NOT NULL,
  source_dynamo_table VARCHAR(50),  -- migration-only; drop post-cutover
  owner_id        UUID         NOT NULL REFERENCES users(id),
  ...
);

-- All 15 child tables: FK changes from VARCHAR(255) → UUID
CREATE TABLE initiative_goals (
  id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id  UUID         NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  ...
);
```

**Migration script changes needed:**
- Remove `initiative_id` column from `initiatives` INSERT
- Change upsert conflict target from `ON CONFLICT (initiative_id)` → `ON CONFLICT (id)`; since `id = _as_uuid(projectId/entityId)` is deterministic, idempotency is preserved
- Change `known_initiative_ids` set to store UUIDs (already the case — `pg_id` is a UUID)
- All child table INSERTs: `initiative_id` column values are already the UUID `raw_initiative_id` coerced via `_as_uuid` — no value change needed, only the column type in DDL changes
- `donations` and `subscriptions`: FK resolution unchanged — already using UUID `initiative_id`

---

### 4. `donations.id` — fix determinism formula

The migration script uses a 2-field formula:
```python
pg_id = _uuid5("proj_donation", str(user_id), str(d.get("projectId")))
```

The data dictionary specifies a 3-field formula:
```
uuid5("donation", user_id, initiative_id, created_on)
```

The 2-field formula has a collision risk: the same user donating to the same initiative twice produces the same UUID. The 3-field formula avoids this.

**Since the migration has already run** (1,598 rows exist), the script's formula is the de facto standard for re-runs. The data dictionary must be corrected to match, **or** the script must be fixed and a full re-migration run.

**Migration script change needed:**
- Decide which formula is canonical and make script and data dictionary consistent
- If switching to 3-field formula: re-run migration (idempotent — existing rows will be updated via `ON CONFLICT`)

---

### 5. Post-migration validation

The migration script has no embedded validation. Add a post-run validation pass covering:

- Record counts: DynamoDB item count == Postgres row count per table
- Zero NULL `initiative_id` rows in `donations` and `subscriptions` (failed FK resolution)
- Every `stripeSubscriptionId` in DynamoDB present in Postgres with `status = 'active'`
- Stripe API cross-check: every active Stripe subscription has a matching Postgres record
- Financial totals: sum of `currentAmountInCents` in DynamoDB donations == sum of `current_amount_in_cents` in Postgres per initiative

**Migration script change needed:** Add `--validate-post` mode (or extend existing `--validate`) to run these checks against the live Postgres DB after execute.

---

## Schema Summary (v2.0.0)

20 PostgreSQL tables. Full DDL in `data-design_and_migration.pdf`.

### Table inventory

| Table | Source | Notes |
|---|---|---|
| `users` | `lff-prod-users` | UUID `id` PK; `user_id` unique index for auth lookups |
| `organizations` | `lff-prod-organizations` | `organization_id` dropped; `owner_id` is UUID FK → `users(id)` |
| `initiatives` | `lff-prod-projects` + `lff-prod-entities` (merged) | `initiative_id` dropped; all FKs on `id` UUID |
| `initiative_goals` | projects (budget categories) + entities (goals array) | FK → `initiatives(id)` |
| `initiative_beneficiaries` | both | FK → `initiatives(id)` |
| `initiative_custom_websites` | both | FK → `initiatives(id)` |
| `initiative_contributors` | projects only | FK → `initiatives(id)` |
| `initiative_mentors` | projects only (mentorship type) | FK → `initiatives(id)` |
| `initiative_program_info_terms` | projects only (mentorship type) | FK → `initiatives(id)` |
| `initiative_program_info_skills` | projects only (mentorship type) | FK → `initiatives(id)` |
| `initiative_program_info_config` | projects only (mentorship type) | FK → `initiatives(id)` |
| `initiative_program_info_custom_term` | projects only (mentorship type) | FK → `initiatives(id)` |
| `initiative_sponsorship_tiers` | entities only | FK → `initiatives(id)` |
| `initiative_ostif_detail` | entities (ostif type only) | FK → `initiatives(id)` |
| `initiative_contacts` | entities (ostif type only) | FK → `initiatives(id)` |
| `initiative_github_stats` | projects only | FK → `initiatives(id)` |
| `initiative_stats` | projects only | FK → `initiatives(id)` |
| `initiative_entity_details` | entities only | FK → `initiatives(id)` |
| `donations` | `lff-prod-donations` + `lff-prod-entity-donations` (merged) | FK → `initiatives(id)` |
| `subscriptions` | `lff-prod-subscriptions` + `lff-prod-entity-subscriptions` (merged) | FK → `initiatives(id)` |

### Post-cutover cleanup

```sql
ALTER TABLE initiatives DROP COLUMN source_dynamo_table;
```

---

## Migration Script

**File:** `tools/migrate-cf/migrate_dynamo_to_postgres.py`  
**Language:** Python 3  
**Dependencies:** `boto3`, `psycopg2-binary`

### Usage

```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...   # for STS credentials
export AWS_REGION=us-east-1
export PG_DSN="host=localhost port=5432 dbname=lff user=postgres password=..."
python3 migrate_dynamo_to_postgres.py
```

The script is a single-pass, idempotent upsert — safe to re-run. All INSERTs use `ON CONFLICT ... DO UPDATE`.

### Execution order (FK dependency order)

1. `users` — no FK dependencies; placeholder rows inserted for any user ID referenced elsewhere but absent from `lff-prod-users`
2. `organizations` — FK → `users`
3. `initiatives` (projects + entities merged) + all 15 child tables — FK → `users`
4. `donations` — FK → `users`, `initiatives(id)`, `organizations(id)`
5. `subscriptions` — FK → `users`, `initiatives(id)`, `organizations(id)`

### Key implementation notes

- **Deterministic UUIDs:** `initiatives.id` and all child table PKs are generated via `uuid5` so re-runs produce the same IDs and the `ON CONFLICT` upserts are stable.
- **`_as_uuid()`:** Coerces UUID strings; falls back to `uuid5("coerce", s)` for non-UUID strings.
- **Placeholder users:** 58 synthetic user rows are inserted for user IDs referenced in other tables but absent from `lff-prod-users`. All fields except `user_id` are NULL.
- **entityType quirk:** DynamoDB's `SaveEntity` rewrites `'general fund'` → `'initiative'` before every PutItem. Migration reverses this: `'initiative'` → `'general fund'`.
- **Mentorship reclassification (Phase 3):** After all rows are inserted, an UPDATE reclassifies initiatives with a `mentee` goal where `amount_in_cents > 0` to `initiative_type = 'mentorship'`.
- **`community` entities:** 3 rows (all declined/submitted 2019) are discarded — not inserted.
- **`status` normalization:** `'hide'` → `'hidden'` applied to all rows.

### DynamoDB tables scanned

| DynamoDB table | Target Postgres table(s) |
|---|---|
| `lff-prod-users` | `users` |
| `lff-prod-organizations` | `organizations` |
| `lff-prod-projects` | `initiatives` + 15 child tables |
| `lff-prod-entities` | `initiatives` + child tables (merged) |
| `lff-prod-donations` | `donations` |
| `lff-prod-entity-donations` | `donations` (merged) |
| `lff-prod-subscriptions` | `subscriptions` |
| `lff-prod-entity-subscriptions` | `subscriptions` (merged) |

---

## Production Data Size (as of 2026-05-11)

| Table | Rows |
|---|---:|
| `users` | 16,840 |
| `organizations` | 606 |
| `initiatives` | 2,021 |
| `initiative_goals` | 4,436 |
| `initiative_beneficiaries` | 2,004 |
| `initiative_contributors` | 37 |
| `initiative_custom_websites` | 1 |
| `initiative_mentors` | 2,911 |
| `initiative_program_info_skills` | 5,462 |
| `initiative_program_info_terms` | 92 |
| `initiative_program_info_config` | 1,840 |
| `initiative_program_info_custom_term` | 11 |
| `initiative_sponsorship_tiers` | 0 |
| `initiative_ostif_detail` | 11 |
| `initiative_contacts` | 26 |
| `initiative_github_stats` | 1,551 |
| `initiative_stats` | 1,840 |
| `initiative_entity_details` | 0 |
| `donations` | 1,598 |
| `subscriptions` | 277 |

**Initiative type breakdown:**

| `initiative_type` | Count |
|---|---:|
| `project` | 1,839 |
| `general_fund` | 122 |
| `other` | 26 |
| `event` | 20 |
| `ostif` | 11 |
| `community` (discarded) | 3 |

---

## Phase 3: Validation

Before cutover, run a reconciliation pass:

1. **Record counts:** DynamoDB item count == Postgres row count per table
2. **Spot checks:** Sample 20 records per table, compare DynamoDB vs Postgres field by field
3. **Active subscriptions:** Every `stripeSubscriptionId` in DynamoDB is present in Postgres with `status = 'active'`
4. **Stripe cross-check:** Query Stripe API for active subscriptions; verify each one has a matching Postgres record
5. **Financial totals:** Sum of `currentAmountInCents` in DynamoDB donations == sum of `current_amount_in_cents` in Postgres per initiative

Document results. Keep the validation report alongside migration logs.

---

## Phase 4: Cutover

### Prerequisites

- [ ] Open schema questions (above) resolved and schema finalised
- [ ] New CF service fully deployed and tested in dev and staging
- [ ] Migration executed and validated in staging (against a staging DynamoDB copy)
- [ ] Migration executed and validated in prod (validate mode first, then execute)
- [ ] Auth0 callback/CORS URLs updated for new service
- [ ] Reimbursement Service OpenSearch dependency acknowledged (old Lambda keeps running)
- [ ] Rollback procedure tested in staging
- [ ] Ledger Service auth header fix deployed (`x-ledger-auth` → `Authorization: Bearer` for `GetProject`/`GetUserName`; auth header added to `GetOrganizationName`)
- [ ] `amount_raised_cents` pre-populated via `amount-raised-sync` CronJob before DNS switch

### Cutover Steps

1. Put old system in read-only mode (disable writes) — or accept brief dual-write window
2. Run final incremental migration (any records created since the last full migration)
3. **Run `amount_raised_cents` pre-population** — execute the `amount-raised-sync` CronJob manually against prod Ledger API to populate `amount_raised_cents` for all migrated initiatives. This ensures no published initiative card shows `$0 raised` incorrectly on day one.
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
- Reimbursement Service OpenSearch dependency is resolved
- All active Stripe subscriptions verified working under new system

Steps:
1. Drop migration scaffolding: `ALTER TABLE initiatives DROP COLUMN source_dynamo_table;`
2. Decommission old LFF Lambda functions
3. Decommission DynamoDB tables (after backup)
4. Decommission OpenSearch (after Reimbursement Service migrated off it)
5. Archive old LFF and lfx-crowdfunding-upgrade repos (read-only)
