# LFF PostgreSQL Data Design and Migration

**Schema version:** 2.0.0  
**Source:** Live scan of `lff-prod-entities` (182 records) and `lff-prod-projects` (1,841 records)  
**Migration status:** Complete — 2,021 initiatives, 16,840 users, 1,598 donations, 277 subscriptions migrated (2026-05-11)  
**Schema DDL:** [`db/migrations/001_initial.up.sql`](../../db/migrations/001_initial.up.sql)

> **Note:** This document describes the normalized 20-table schema implemented in `001_initial.up.sql` and the Python migration script used for the initial data migration. The field mappings and data dictionary are the authoritative reference for DynamoDB→Postgres field transformations.

---

## Overview

Schema v2.0.0 fully normalises all DynamoDB `lff-prod-*` tables into 20 PostgreSQL tables. The two source initiative tables (`lff-prod-projects` and `lff-prod-entities`) are merged into a single `initiatives` table with 15 child tables handling all repeating groups and type-specific detail.

**Schema decisions applied vs. Lewis's original v2.0.0:**

| Decision | Change |
|---|---|
| `users` — add UUID surrogate PK | `id UUID PRIMARY KEY` added; `user_id` kept as `UNIQUE` natural key for Auth0 lookups; all FK columns in child tables changed from `VARCHAR(255)` → `UUID REFERENCES users(id)` |
| `organizations` — drop `organization_id` | Column was a pure duplicate of `id` (same UUID value); removed |
| `initiatives` — drop `initiative_id` | Column held identical value to `id` (DynamoDB IDs are already UUIDs); all 15 child table FKs changed from `VARCHAR(255) REFERENCES initiatives(initiative_id)` → `UUID REFERENCES initiatives(id)` |

Lewis's migration script must be updated to reflect these changes — see [Migration Script Changes](#migration-script-changes).

---

## Entity Relationship Diagram

```
users
  ├── organizations (owner_id → users.id)
  ├── initiatives (owner_id → users.id)
  │     ├── initiative_goals
  │     ├── initiative_beneficiaries
  │     ├── initiative_custom_websites
  │     ├── initiative_contributors        (projects only)
  │     ├── initiative_mentors             (projects only)
  │     ├── initiative_program_info_terms  (projects only)
  │     ├── initiative_program_info_skills (projects only)
  │     ├── initiative_program_info_config (projects only)
  │     ├── initiative_program_info_custom_term (projects only)
  │     ├── initiative_sponsorship_tiers   (entities only)
  │     ├── initiative_ostif_detail        (ostif entities only)
  │     ├── initiative_contacts            (ostif entities only)
  │     ├── initiative_github_stats        (projects only)
  │     ├── initiative_stats               (projects only)
  │     └── initiative_entity_details      (entities only)
  ├── donations (user_id → users.id, initiative_id → initiatives.id)
  └── subscriptions (user_id → users.id, initiative_id → initiatives.id)
```

---

## Table Inventory

| Table | Source DynamoDB table(s) | Rows (2026-05-11) |
|---|---|---:|
| `users` | `lff-prod-users` | 16,840 |
| `organizations` | `lff-prod-organizations` | 606 |
| `initiatives` | `lff-prod-projects` + `lff-prod-entities` (merged) | 2,021 |
| `initiative_goals` | both | 4,436 |
| `initiative_beneficiaries` | both | 2,004 |
| `initiative_contributors` | projects only | 37 |
| `initiative_custom_websites` | both | 1 |
| `initiative_mentors` | projects only | 2,911 |
| `initiative_program_info_skills` | projects only | 5,462 |
| `initiative_program_info_terms` | projects only | 92 |
| `initiative_program_info_config` | projects only | 1,840 |
| `initiative_program_info_custom_term` | projects only | 11 |
| `initiative_sponsorship_tiers` | entities only | 0 |
| `initiative_ostif_detail` | entities (ostif only) | 11 |
| `initiative_contacts` | entities (ostif only) | 26 |
| `initiative_github_stats` | projects only | 1,551 |
| `initiative_stats` | projects only | 1,840 |
| `initiative_entity_details` | entities only | 0 |
| `donations` | `lff-prod-donations` + `lff-prod-entity-donations` (merged) | 1,598 |
| `subscriptions` | `lff-prod-subscriptions` + `lff-prod-entity-subscriptions` (merged) | 277 |

**FK integrity:** 0 orphaned goals, 0 orphaned donations (initiative or user), 0 NULL owner IDs.

**Initiative type breakdown:**

| `initiative_type` | Source | Count |
|---|---|---:|
| `project` | projects | 1,839 |
| `general_fund` | entities | 122 |
| `other` | entities | 26 |
| `event` | entities | 20 |
| `ostif` | entities | 11 |
| `community` | entities (discarded) | 3 |

---

## Data Dictionary

### Table: `users`

Source: `lff-prod-users`

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | — | Surrogate PK. |
| `user_id` | VARCHAR(255) | NN, UNIQUE | `id` | Auth0 subject (e.g. `auth0|abc123`). Natural key used for all Auth0 lookups. |
| `email` | TEXT | | `email` | |
| `given_name` | TEXT | | `givenname` | DynamoDB key is `givenname` (no camelCase). |
| `family_name` | TEXT | | `familyName` | |
| `name` | TEXT | | `name` | Full display name. |
| `avatar_url` | TEXT | | `avatarUrl` | |
| `stripe_customer_id` | TEXT | | `stripeCustomerId` | Stripe customer ID. |
| `github_access_token` | TEXT | | *(OAuth token)* | GitHub OAuth token; plain text matching current LFF behavior. |
| `created_on` | TIMESTAMPTZ | DEFAULT NOW() | — | Set by DB default; DynamoDB users table has no `createdOn`. |
| `updated_on` | TIMESTAMPTZ | DEFAULT NOW() | — | Set by DB default. |

**Placeholder rows:** 58 synthetic rows are inserted for user IDs referenced by other tables but absent from `lff-prod-users`. All fields except `user_id` are NULL.

---

### Table: `organizations`

Source: `lff-prod-organizations`

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | `organizationId` | Deterministically derived via `_as_uuid()`. |
| `owner_id` | UUID | NN, FK → `users.id` | `ownerId` | |
| `name` | TEXT | NN | `name` | |
| `avatar_url` | TEXT | | `avatarUrl` | DynamoDB key is `avatarUrl` (capital U, lowercase rl). |
| `status` | VARCHAR(50) | | `status` | All prod rows are `"approved"`. |
| `created_on` | TIMESTAMPTZ | DEFAULT NOW() | — | Not persisted in DynamoDB organizations. |
| `updated_on` | TIMESTAMPTZ | DEFAULT NOW() | — | Not persisted in DynamoDB organizations. |

All production rows have `status = "approved"`. No `description`, `website`, `approved_at`, or `rejected_at` fields exist in DynamoDB — those columns were excluded from schema.

---

### Table: `initiatives`

Source: `lff-prod-projects` + `lff-prod-entities` (merged)

#### Identity columns

| Column | Type | Constraints | DynamoDB field | Source | Notes |
|---|---|---|---|---|---|
| `id` | UUID | PK | `projectId` / `entityId` | both | Surrogate PK. Deterministically generated via `_as_uuid(projectId\|entityId)` — stable across re-runs. Referenced by all child tables and by `donations.initiative_id` / `subscriptions.initiative_id`. |
| `initiative_type` | VARCHAR(50) | NN | `entityType` / `'project'` | both | For projects: hardcoded `'project'` initially, then reclassified to `'mentorship'` in Phase 3 if a mentee goal with `amount_in_cents > 0` exists. For entities: taken from `entityType`; DynamoDB quirk reverses `'initiative'` → `'general_fund'`. Known values: `project`, `mentorship`, `general_fund`, `event`, `ostif`. |
| `source_dynamo_table` | VARCHAR(50) | | — | *migration* | `'projects'` or `'entities'`. **Migration-only column** — drop after cutover (`ALTER TABLE initiatives DROP COLUMN source_dynamo_table`). |

#### Ownership

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `owner_id` | UUID | NN, FK → `users.id` | `ownerId` | UUID FK to `users.id` (surrogate PK). The Auth0 subject lives in `users.user_id`; resolve by looking up `users.user_id = ownerId` → `users.id`. |

#### Core display fields

| Column | Type | Constraints | DynamoDB field | Source | Notes |
|---|---|---|---|---|---|
| `name` | TEXT | NN | `name` | both | Display name. |
| `slug` | TEXT | | `slug` | both | URL-safe identifier (e.g. `my-project`). |
| `status` | VARCHAR(50) | | `status` | both | Projects: `submitted`, `published`, `declined`, `hidden`. Entities: `pending`, `published`. |
| `industry` | TEXT | | `projectDetails.industry` / `industry` | both | Free-form industry tag. |
| `description` | TEXT | | `projectDetails.description` / `description` | both | |
| `color` | VARCHAR(10) | | `projectDetails.color` / `color` | both | 6-digit hex color code (no `#`). Max 7 chars; truncated at 10 for safety. |
| `logo_url` | TEXT | | `logoUrl` | both | |
| `website_url` | TEXT | | `projectDetails.website` / `websiteURL` | both | Projects use `"website"` JSON key; entities use `"websiteURL"`. |
| `coc_url` | TEXT | | `projectDetails.codeOfConduct.link` / `cocURL` | both | Projects nest it under `codeOfConduct.link`; entities store it flat. |

#### Financial / platform IDs

| Column | Type | Constraints | DynamoDB field | Source | Notes |
|---|---|---|---|---|---|
| `stripe_plan_id` | TEXT | | `planId` / `stripePlanId` | both | Stripe recurring-payment plan ID. **Critical — must be preserved exactly.** |
| `stripe_product_id` | TEXT | | `productId` / `stripeProductId` | both | Stripe product ID. **Critical — must be preserved exactly.** |
| `amount_raised_in_cents` | BIGINT | NN, DEFAULT 0 | `amountRaised` | both | Denormalised total donations in cents raised. Updated by the `amount-raised-sync` CronJob. |
| `accept_funding` | BOOLEAN | DEFAULT false | `acceptFunding` | entities only | Whether the entity is currently accepting donations. Projects always accept funding when published. |

#### Project-only fields

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `cii_project_id` | TEXT | | `projectDetails.ciiProjectID` | CII Best Practices badge programme ID. |
| `mentorship_program_id` | TEXT | | `jobspringProjectId` | ID of the linked LFX Mentorship program. Non-NULL when `initiative_type = 'mentorship'`. Used as upsert key by `mentorship-sync` CronJob. DynamoDB field name uses lowercase `d` (`jobspringProjectId`). |
| `stacks_identifier` | TEXT | | `projectDetails.stacksIdentifier` | Identifier in the Stacks platform. |

#### Entity-only fields

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `eventbrite_id` | TEXT | | `eventbriteId` | Event-type entities only. Despite the field name, the stored value is a URL — handle accordingly at the application layer. |
| `application_url` | TEXT | | `applicationURL` | URL for applicants to apply (event entities). |
| `event_start_date` | TIMESTAMPTZ | | `eventStartDate` | Parsed from string. Event type only. |
| `event_end_date` | TIMESTAMPTZ | | `eventEndDate` | Event type only. |
| `country` | VARCHAR(100) | | `EntityLocation.country` | From embedded `EntityLocation` struct. |
| `city` | VARCHAR(100) | | `EntityLocation.city` | |
| `is_online` | BOOLEAN | DEFAULT false | `isOnline` | |

#### Timestamps

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `created_on` | TIMESTAMPTZ | DEFAULT NOW() | `createdOn` | Parsed from 8 possible timestamp formats including Go `time.String()` with nanoseconds + monotonic suffix. |
| `updated_on` | TIMESTAMPTZ | DEFAULT NOW() | `updatedOn` | Same parsing. |

---

### Table: `initiative_goals`

Source: `lff-prod-projects` (per-category budgets) + `lff-prod-entities` (entity goals array)

One row per funding category per initiative. Projects produce up to 8 fixed rows; entities produce one row per element in `entity.Goals[]`.

**⚠️ Critical:** `Budget.AmountInCents` has JSON tag `"amount"` — read from DynamoDB as `budget["amount"]`, NOT `budget["amountInCents"]`.

**⚠️ Critical (mentorship projects):** Read mentee budget data from `projectDetails.mentee`, NOT from a top-level `mentee` attribute. Reading from the wrong path silently drops all mentorship metadata — this was the bug that caused the first migration pass to miss 1,249 of 1,476 rows.

| Column | Type | Constraints | DynamoDB field | Source | Notes |
|---|---|---|---|---|---|
| `id` | UUID | PK | — | both | Deterministic: `uuid5("goal", initiative_id, name)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | — | both | |
| `name` | TEXT | NN, UQ(initiative_id, name) | — | both | Projects: `development`, `marketing`, `meetups`, `travel`, `bugBounty`, `documentation`, `other`, `mentee`. Entities: free-form name from `Goal.Name`. |
| `amount_in_cents` | BIGINT | NN, DEFAULT 0 | `budget.amount` | both | **JSON tag is `"amount"` not `"amountInCents"`.** Stored as cents integer. |
| `allocation` | TEXT | | `budget.allocation` | both | Free-form description of how the budget is allocated (e.g. `"50%"`). |
| `repo_link` | TEXT | | `development.repoLink` | projects only | Only populated for the `development` goal row. |
| `description` | TEXT | | `goals[].description` | entities only | NULL for project goals. |
| `color` | VARCHAR(10) | | `goals[].goalColor` | entities only | NULL for project goals. Truncated to 10 chars. |
| `icon` | TEXT | | `goals[].goalIcon` | entities only | NULL for project goals. |
| `sort_order` | INTEGER | DEFAULT 0 | — | both | Projects: 0=development, 1=marketing, 2=meetups, 3=travel, 4=bugBounty, 5=documentation, 6=other, 7=mentee. Entities: array index. |

---

### Table: `initiative_beneficiaries`

Source: `lff-prod-projects` + `lff-prod-entities`

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Deterministic: `uuid5("beneficiary", initiative_id, email\|name)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | | |
| `name` | TEXT | | `beneficiaries[].name` | |
| `email` | TEXT | | `beneficiaries[].email` | |

---

### Table: `initiative_custom_websites`

Source: `lff-prod-projects` + `lff-prod-entities`

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Deterministic: `uuid5("custom_website", initiative_id, url)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | | |
| `name` | TEXT | | `customWebsites[].name` | Display label for the link. |
| `url` | TEXT | NN | `customWebsites[].url` | |

---

### Table: `initiative_contributors`

Source: `lff-prod-projects` only

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Deterministic: `uuid5("contributor", initiative_id, email\|name)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | | |
| `name` | TEXT | | `contributors[].name` | |
| `email` | TEXT | | `contributors[].email` | |

---

### Table: `initiative_mentors`

Source: `lff-prod-projects` only — present only when `initiative_type = 'mentorship'`

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Deterministic: `uuid5("mentor", initiative_id, email\|name)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | | |
| `name` | TEXT | | `projectDetails.mentee.mentor[].name` | |
| `email` | TEXT | | `projectDetails.mentee.mentor[].email` | |
| `avatar_url` | TEXT | | `projectDetails.mentee.mentor[].avatarURL` | JSON tag `avatarURL`. |
| `introduction` | TEXT | | `projectDetails.mentee.mentor[].introduction` | Free-text bio. |

---

### Table: `initiative_program_info_terms`

Source: `lff-prod-projects` only — present only when `initiative_type = 'mentorship'`

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Deterministic: `uuid5("mentee_term", initiative_id, array_index)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | | |
| `term` | TEXT | NN | `projectDetails.mentee.terms[]` | Programme term label (e.g. `"Spring 2024"`). |
| `sort_order` | INTEGER | DEFAULT 0 | — | Array index from DynamoDB. |

---

### Table: `initiative_program_info_skills`

Source: `lff-prod-projects` only — present only when `initiative_type = 'mentorship'`

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Deterministic: `uuid5("mentee_skill", initiative_id, skill)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | | |
| `skill` | TEXT | NN, UQ(initiative_id, skill) | `projectDetails.mentee.skills[]` | Skill tag; values drawn from `frontend/skills.json`. |

---

### Table: `initiative_program_info_config`

Source: `lff-prod-projects` only — one row per mentorship initiative

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `initiative_id` | UUID | PK, FK → `initiatives.id` | | One-to-one with parent initiative. |
| `terms_conditions` | BOOLEAN | DEFAULT false | `projectDetails.mentee.termsConditions` | Whether the project owner has accepted the mentorship programme T&Cs. |

---

### Table: `initiative_program_info_custom_term`

Source: `lff-prod-projects` only — present only when `customTerm.termName` is non-empty

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `initiative_id` | UUID | PK, FK → `initiatives.id` | | One-to-one. |
| `term_name` | TEXT | | `projectDetails.mentee.customTerm.termName` | Human-readable term name. |
| `start_month` | VARCHAR(20) | | `projectDetails.mentee.customTerm.startMonth` | Month name or abbreviation (e.g. `"January"`). |
| `end_month` | VARCHAR(20) | | `projectDetails.mentee.customTerm.endMonth` | |
| `year` | INTEGER | | `projectDetails.mentee.customTerm.year` | 4-digit year. |

---

### Table: `initiative_sponsorship_tiers`

Source: `lff-prod-entities` only

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Deterministic: `uuid5("sponsorship_tier", initiative_id, name)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | | |
| `name` | TEXT | | `sponsorshipTiers[].name` | Display name (e.g. `"Gold Sponsor"`). |
| `description` | TEXT | | `sponsorshipTiers[].description` | |
| `color` | VARCHAR(10) | | `sponsorshipTiers[].color` | Hex color. Truncated to 10 chars. |
| `icon` | TEXT | | `sponsorshipTiers[].icon` | Icon class or URL. |
| `minimum` | INTEGER | NN, DEFAULT 0 | `sponsorshipTiers[].minimum` | Minimum donation amount in cents for this tier. |
| `sort_order` | INTEGER | DEFAULT 0 | — | Array index from DynamoDB. |

---

### Table: `initiative_ostif_detail`

Source: `lff-prod-entities` only — ostif entity type only (one row per ostif initiative)

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `initiative_id` | UUID | PK, FK → `initiatives.id` | | One-to-one. |
| `monetization_strategy` | TEXT | | `detail.monetizationStrategy` | |
| `current_security_strategy` | TEXT | | `detail.currentSecurityStrategy` | |
| `license_type` | VARCHAR(100) | | `detail.licenseType` | e.g. `"MIT"`, `"Apache 2.0"`. |
| `total_budget_in_cents` | BIGINT | DEFAULT 0 | `detail.totalBudget` | Total security audit budget in cents. Used as `FundingStatus.TotalAnnualGoalInCents` for ostif entities. |
| `terms_conditions` | BOOLEAN | DEFAULT false | `detail.termsConditions` | |

---

### Table: `initiative_contacts`

Source: `lff-prod-entities` only — ostif entity type only

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Deterministic: `uuid5("contact", initiative_id, contact_type)`. |
| `initiative_id` | UUID | NN, FK → `initiatives.id` | | |
| `contact_type` | VARCHAR(50) | NN, UQ(initiative_id, contact_type) | — | `primary`, `secondary`, or `technical_lead`. Derived from which DynamoDB key the contact appears under (`primaryContact`, `secondaryContact`, `technicalLead`). |
| `first_name` | TEXT | | `detail.{type}.firstName` | |
| `last_name` | TEXT | | `detail.{type}.lastName` | |
| `email` | TEXT | | `detail.{type}.email` | |
| `phone_number` | VARCHAR(50) | | `detail.{type}.phoneNumber` | |
| `other_contact_option` | TEXT | | `detail.{type}.otherContactOption` | Alternative contact method description. |
| `preferred_contact_method` | VARCHAR(50) | | `detail.{type}.preferredContactMethod` | e.g. `"email"`, `"phone"`. |

---

### Table: `initiative_github_stats`

Source: `lff-prod-projects` only. Updated independently via `UpdateGithubDataCache` (targeted DynamoDB `UpdateItem` on `cachedDetails.githubStats.*`).

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `initiative_id` | UUID | PK, FK → `initiatives.id` | | One-to-one. |
| `forks` | INTEGER | NN, DEFAULT 0 | `cachedDetails.githubStats.forks` | |
| `stars` | INTEGER | NN, DEFAULT 0 | `cachedDetails.githubStats.stars` | |
| `open_issues` | INTEGER | NN, DEFAULT 0 | `cachedDetails.githubStats.openIssues` | |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | — | Set by DB default on upsert. |

---

### Table: `initiative_stats`

Source: `lff-prod-projects` only. Updated independently via `UpdateProjectStats` (targeted DynamoDB `UpdateItem` on `cachedDetails.projectStats.backers ± 1`).

`totalRaised` excluded — no active write path; `UpdateProjectStats` only increments/decrements `backers`. Always 0 in production.

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `initiative_id` | UUID | PK, FK → `initiatives.id` | | One-to-one. |
| `backers` | INTEGER | NN, DEFAULT 0 | `cachedDetails.projectStats.backers` | Count of unique donors. Incremented/decremented on each donation/cancellation. |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | — | |

---

### Table: `initiative_entity_details`

Source: `lff-prod-entities` only. `entity.EntityDetails` is `map[string]string` with application-defined keys. Serialised by `dynamodbattribute.MarshalMap` under the `entityDetails` key. All production records have empty maps; table exists for completeness.

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `initiative_id` | UUID | PK, FK → `initiatives.id` | | One-to-one. |
| `details` | JSONB | NN, DEFAULT '{}' | `entityDetails` | Arbitrary `map[string]string` serialised as JSONB. |

---

### Table: `donations`

Source: `lff-prod-donations` + `lff-prod-entity-donations` (merged)

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Generated on migration. |
| `user_id` | UUID | NN, FK → `users(id)` | `userId` | Auth0 subject resolved via `users.user_id` → `users.id`. |
| `initiative_id` | UUID | FK → `initiatives(id)` | `projectId` / `entityId` | May be NULL if initiative was not found. |
| `organization_id` | UUID | FK → `organizations(id)` | `orgId` | Optional. Set when donation is made on behalf of an organisation. |
| `cached_details` | JSONB | | `cachedDetails` | Snapshot of backer metadata at donation time. |
| `category` | TEXT | | `category` | Funding category (e.g. `development`, `marketing`, `mentee`). NULL means "all needs". |
| `created_on` | TIMESTAMPTZ | DEFAULT NOW() | `createdOn` | Parsed from RFC3339 string. |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | — | Set by DB default. |
| `current_amount_in_cents` | BIGINT | NN | `currentAmountInCents` | Donation amount in US cents (e.g. `500` = $5.00). |
| `payment_method` | VARCHAR(50) | | `paymentmethod` | DynamoDB key is `paymentmethod` (all lowercase). `card` or `invoice`. |
| `po_number` | TEXT | | `ponumber` | DynamoDB field is `ponumber` (all lowercase). Invoice payment purchase order number. |
| `status` | VARCHAR(50) | | `status` | Known values: `succeeded`, `failed`, `pending`. NULL on all production rows — migrate as NULL. |
| `stripe_charge_id` | VARCHAR(255) | | `stripeChargeId` | Stripe charge object ID (e.g. `ch_abc123`). NULL for invoice payments. |

---

### Table: `subscriptions`

Source: `lff-prod-subscriptions` + `lff-prod-entity-subscriptions` (merged). Mirrors `donations` structure with additional frequency/Stripe columns.

| Column | Type | Constraints | DynamoDB field | Notes |
|---|---|---|---|---|
| `id` | UUID | PK | — | Generated on migration. |
| `user_id` | UUID | NN, FK → `users(id)` | `userId` | Auth0 subject resolved via `users.user_id` → `users.id`. |
| `initiative_id` | UUID | FK → `initiatives(id)` | `projectId` / `entityId` | May be NULL if initiative not found. |
| `organization_id` | UUID | FK → `organizations(id)` | `orgId` | Optional. Organisation on whose behalf the subscription was created. |
| `cached_details` | JSONB | | `cachedDetails` | Snapshot of backer metadata at subscription time. |
| `category` | TEXT | | `category` | Same values as `donations.category`. NULL = all needs. |
| `created_on` | TIMESTAMPTZ | DEFAULT NOW() | `createdOn` | Parsed from string. |
| `updated_at` | TIMESTAMPTZ | DEFAULT NOW() | *(absent)* | Does not exist in DynamoDB — default to `created_on` value on migration. |
| `current_amount_in_cents` | BIGINT | NN | `currentAmountInCents` | Monthly or annual recurring amount in US cents. |
| `frequency` | VARCHAR(50) | | `frequency` | `monthly` or `annual`. Note: DynamoDB may contain `yearly` — normalize to `annual` on migration. |
| `status` | VARCHAR(50) | | `status` | `active` or `inactive`. |
| `stripe_subscription_id` | VARCHAR(255) | | `stripeSubscriptionId` | Stripe subscription object ID (e.g. `sub_abc123`). |
| `stripe_subscription_item_id` | VARCHAR(255) | | `stripeSubscriptionItemId` | Stripe subscription item ID (e.g. `si_abc123`). Needed for per-item price/quantity updates. |

---

## Column Type Rationale

| Type | Used for | Reason |
|---|---|---|
| `TEXT` | All free-form string fields | No length enforced in Go; real data exceeds 255 chars (CII URLs, descriptions, skill names). |
| `VARCHAR(255)` | Natural keys (`user_id`) and Stripe/external IDs that are structurally bounded | Preserves index efficiency for FK join columns. |
| `VARCHAR(50)` | Enum-like fields: `status`, `initiative_type`, `payment_method`, `frequency`, `contact_type` | Go validation constraints (`valid:"in(...)"`) enforce short values. |
| `VARCHAR(10)` | `color` | Hex color max 7 chars (`#RRGGBB`); `valid:"hexcolor"` enforced in Go. |
| `BIGINT` | `current_amount_in_cents` (donations, subscriptions), `amount_in_cents` (goals), `amount_raised_in_cents` (initiatives), `total_budget_in_cents` (ostif) | Go uses `int64` for monetary amounts. |
| `INTEGER` | Stats counters (`backers`, `forks`, `stars`, `open_issues`) | Go uses `int`; max observed value ~9.8M (fits 32-bit). |
| `BOOLEAN` | `accept_funding`, `is_online`, `terms_conditions` | Go `bool`. |
| `TIMESTAMPTZ` | All timestamps | DynamoDB stores as strings in multiple formats; normalised to UTC on import. |
| `UUID` | Surrogate PKs, FK refs in donations/subscriptions | Stable across re-runs via deterministic `uuid5`. |
| `JSONB` | `donations.cached_details`, `subscriptions.cached_details`, `initiative_entity_details.details` | Variable structure; not queried by key in application hot path. |

---

## Fields Excluded from Schema

These fields appear in the Go domain structs and/or DynamoDB but are **not persisted** to PostgreSQL because they have no DynamoDB write path or are computed at read time.

| Field | Go type / location | Reason excluded |
|---|---|---|
| `balance` | `projects/domain.Balance` / `entities/domain.Balance` | Computed at read time by calling the Ledger microservice API (`GetEntityBalanceFromStartDate`). Never written to DynamoDB by `SaveProject` or `SaveEntity`. |
| `funding_status` | `projects/domain.FundingStatus` / `entities/domain.FundingStatus` | Computed from `balance` + subscription summary aggregates. Populated by `fillFundingStatus` at request time. |
| `entity_stats.sponsors` | `entities/domain.EntityStats.Sponsors` | `GetEntitySponsors` pushes to Elasticsearch only. `SaveEntity` is never called with `EntityStats` populated. |
| `sponsors` | `projects/domain.Sponsor[]` / `entities/domain.Sponsor[]` | `GetEntitySponsors` / `GetProjectSponsors` push to ES only. |
| `diversity` | `projects/domain.Diversity` | `GetDiversity` fetches from an external demographic API at read time and does not persist results. |
| `vulnerability_summary` | `projects/domain.VulnerabilitySummary` | `GetVulnerabilitySummary` fetches from an external security scanning service at read time. |
| `badges` | `projects/domain.Badge[]` | `convertProjectToDynamoRepresentation` never assigns `Badges`. The field is always nil/empty in DynamoDB. |
| `project_stats.total_raised` | `projects/domain.ProjectStats.TotalRaised` | No active write path. `UpdateProjectStats` only increments/decrements `backers`. Value is always 0 in DynamoDB. |
| `entity_stats.total_raised` | `entities/domain.EntityStats.TotalFundsRaised` | Same as above for entities. |
| `cii_markup` | `projects/domain.ProjectDetails.CIIMarkup` | Fetched live from `bestpractices.coreinfrastructure.org` at request time, not stored. |
| `uncategorised` | `projects/domain.Uncategorised` | "All project needs" pseudo-category; not a true budget category and has no independent representation in goals. |
| `project_funding_status` | `projects/domain.ProjectFundingStatus` | Per-category funding breakdown; computed from ledger + subscriptions at read time. |

---

## Financial Data: Balance and Funding Status

`balance` and `funding_status` are not stored in the PostgreSQL schema because they are **never persisted** — neither in DynamoDB nor anywhere else. They are computed fresh on every API call from two live sources:

1. **Ledger API** (`balanceData` / `ledgerService`) — the external service that holds the canonical transaction ledger for each initiative. All credits (donations, subscriptions) and debits (payouts to beneficiaries) live there.
2. **Subscription summaries** (`subscriptionRepository.GetSubscriptionSummary`) — aggregated directly from the `subscriptions` table in PostgreSQL.

The DynamoDB records for `balance` and `funding_status` were snapshots written by background jobs (`updateEntityAmountRaised`, `seed_amountraised_*`) and are **stale by design** — they existed solely as a read-cache for list views and are now superseded by the PostgreSQL `amount_raised_in_cents` column and real-time derivation.

> **Note:** `balance.debit_in_cents` cannot be populated until `lff-prod-transactions` is migrated to a `transactions` table. Until then, the Ledger API remains the authoritative source for payout data and `balance_in_cents`.

### Deriving funding status from PostgreSQL

```sql
-- credit_in_cents: all-time credited amount for an initiative
SELECT
    i.id,
    i.name,
    COALESCE(SUM(d.current_amount_in_cents), 0) AS credit_in_cents
FROM initiatives i
LEFT JOIN donations d ON d.initiative_id = i.id
                     AND d.status = 'completed'
GROUP BY i.id, i.name;

-- total_subscription_count and annual_subscription_amount_in_cents
SELECT
    i.id                                               AS initiative_id,
    COUNT(s.id)                                        AS total_subscription_count,
    COALESCE(SUM(s.current_amount_in_cents), 0)        AS annual_subscription_amount_in_cents
FROM initiatives i
LEFT JOIN subscriptions s ON s.initiative_id = i.id
                         AND s.status = 'active'
GROUP BY i.id;
```

### Recommended materialized view

```sql
CREATE MATERIALIZED VIEW initiative_funding_summary AS
SELECT
    i.id,
    i.name,
    COALESCE(SUM(d.current_amount_in_cents)
             FILTER (WHERE d.status = 'completed'), 0)        AS total_donations_in_cents,
    COUNT(s.id) FILTER (WHERE s.status = 'active')            AS total_subscription_count,
    COALESCE(SUM(s.current_amount_in_cents)
             FILTER (WHERE s.status = 'active'), 0)           AS annual_subscription_amount_in_cents
FROM initiatives i
LEFT JOIN donations     d ON d.initiative_id = i.id
LEFT JOIN subscriptions s ON s.initiative_id = i.id
GROUP BY i.id, i.name;

-- Refresh after each donation/subscription write:
-- REFRESH MATERIALIZED VIEW CONCURRENTLY crowdfunding.initiative_funding_summary;
```

---

## Source → Target Field Mapping

### `initiatives` (core columns)

| PostgreSQL column | Project source (`lff-prod-projects`) | Entity source (`lff-prod-entities`) | Go type |
|---|---|---|---|
| `id` (UUID FK target) | `projectId` → `_as_uuid()` | `entityId` → `_as_uuid()` | `string` |
| `initiative_type` | hardcoded `'project'` | `entityType` (see quirk below) | `string` |
| `source_dynamo_table` | `'projects'` | `'entities'` | migration-only |
| `owner_id` | `ownerId` | `ownerId` | `string` |
| `name` | `name` | `name` | `string` |
| `slug` | `slug` | `slug` | `string` |
| `status` | `status` | `status` — entity: `pending\|published` | `string` |
| `industry` | `projectDetails.industry` | `industry` | `string` |
| `description` | `projectDetails.description` | `description` | `string` |
| `color` | `projectDetails.color` | `color` | `string` — hex, max 7 chars |
| `logo_url` | `logoUrl` | `logoUrl` | `string` |
| `website_url` | `projectDetails.website` | `websiteURL` | `string` |
| `coc_url` | `projectDetails.codeOfConduct.link` | `cocURL` | `string` |
| `cii_project_id` | `projectDetails.ciiProjectID` | — | `string` (full badge URLs observed) |
| `stripe_plan_id` | `planId` | `stripePlanId` | `string` |
| `stripe_product_id` | `productId` | `stripeProductId` | `string` |
| `mentorship_program_id` | `jobspringProjectId` | — | `string` |
| `stacks_identifier` | `projectDetails.stacksIdentifier` | — | `string` |
| `eventbrite_id` | — | `eventbriteId` | `string` — despite field name, value is a URL |
| `application_url` | — | `applicationURL` | `string` |
| `amount_raised_in_cents` | `amountRaised` | `amountRaised` | `int` |
| `accept_funding` | `false` (hardcoded) | `acceptFunding` | `bool` |
| `event_start_date` | — | `eventStartDate` | `string` → `TIMESTAMPTZ` |
| `event_end_date` | — | `eventEndDate` | `string` → `TIMESTAMPTZ` |
| `country` | — | `EntityLocation.country` | `string` |
| `city` | — | `EntityLocation.city` | `string` |
| `is_online` | `false` (hardcoded) | `isOnline` | `bool` |
| `created_on` | `createdOn` | `createdOn` | `string` → `TIMESTAMPTZ` |
| `updated_on` | `updatedOn` | `updatedOn` | `string` → `TIMESTAMPTZ` |

**entityType quirk:** `SaveEntity` rewrites `'general fund'` → `'initiative'` before every `PutItem`. Migration reverses this: `'initiative'` → `'general fund'`.

### `donations`

| Column | `lff-prod-donations` | `lff-prod-entity-donations` |
|---|---|---|
| `user_id` | `userId` (Auth0 subject, direct) | `userId` (Auth0 subject, direct) |
| `initiative_id` | `projectId` → new Postgres UUID via in-memory map | `entityId` → new Postgres UUID via in-memory map |
| `organization_id` | `orgId` → Postgres UUID via in-memory map or NULL | same |
| `category` | `category` | `category` |
| `created_on` | `createdOn` | `createdOn` |
| `current_amount_in_cents` | `currentAmountInCents` | `currentAmountInCents` |
| `payment_method` | `paymentmethod` (all lowercase) | same |
| `po_number` | `ponumber` (all lowercase) | same |
| `status` | `status` | `status` |
| `stripe_charge_id` | `stripeChargeId` | `stripeChargeId` |

### `subscriptions`

Mirrors `donations`. Additional columns: `frequency`, `stripe_subscription_id`, `stripe_subscription_item_id`.

---

## Migration Script

**File:** [`tools/migrate-cf/migrate_dynamo_to_postgres.py`](../../tools/migrate-cf/migrate_dynamo_to_postgres.py) — see `05-migration-plan.md` for operational instructions.

### Usage

```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...        # for STS/temporary credentials
export AWS_REGION=us-east-1
export PG_DSN="host=localhost port=5432 dbname=lff user=postgres password=..."
python3 migrate_dynamo_to_postgres.py
```

All INSERTs use `ON CONFLICT ... DO UPDATE` — idempotent and safe to re-run.

### Execution order

```
1. users          — no FK dependencies; placeholder rows for referenced-but-absent user IDs
2. organizations  — FK → users
3. initiatives    — FK → users  (merged entities + projects + all 15 child tables)
4. donations      — FK → users, initiatives, organizations
5. subscriptions  — FK → users, initiatives, organizations
```

### Key implementation notes

- **Deterministic UUIDs:** All PKs generated via `uuid5` — re-runs produce identical IDs, `ON CONFLICT` upserts are stable.
- **`_as_uuid(value)`:** Parses UUID strings; falls back to `uuid5("coerce", s)` for non-UUID strings.
- **`_uuid5(scope, *parts)`:** `str(uuid.uuid5(NS, f"{scope}:{key}"))` where `key = "|".join(parts)` and namespace is `6ba7b810-9dad-11d1-80b4-00c04fd430c8`.
- **Placeholder users:** User IDs referenced in organizations, initiatives, donations, or subscriptions but absent from `lff-prod-users` are inserted as placeholder rows (all fields NULL except `user_id`).
- **entityType quirk:** `SaveEntity` rewrites `'general fund'` → `'initiative'` on every DynamoDB write. Migration reverses: if `entityType == 'initiative'` → restore to `'general fund'`.
- **Budget `amount` JSON tag:** `Budget.AmountInCents` is serialised with JSON tag `"amount"` — access as `budget.get("amount")`, NOT `budget.get("amountInCents")`.
- **Mentorship reclassification (Phase 3):** After all rows inserted, an UPDATE sets `initiative_type = 'mentorship'` for any initiative with a `mentee` goal where `amount_in_cents > 0`.
- **`community` entities:** 3 rows (all declined/submitted 2019, no active users) — discarded, not inserted.
- **Status normalization:** `'hide'` → `'hidden'` applied to all rows.

### Migration script changes needed

The Python script was written for Lewis's original normalized v2.0.0 schema (20 tables, `initiative_id VARCHAR(255)` as natural key, `users.user_id` as PK). That original schema and the full script change details are preserved in `docs/rewrite/archived/data-design_and_migration.archived.pdf`.

The current target schema (`001_initial.up.sql`) uses:
- `users.id UUID PRIMARY KEY` (surrogate PK, `gen_random_uuid()`); Auth0 subject stored in `user_id VARCHAR(255) UNIQUE`
- `organizations.id UUID` (no separate `organization_id` column)
- `initiatives.id UUID` (no separate `initiative_id` column); all repeating groups in normalized child tables

Any migration script re-run must target the schema in `001_initial.up.sql`. See `05-migration-plan.md` for the complete field mapping reference.

**No `legacy_id` column is needed.** DynamoDB `projectId` and `entityId` are UUID v4 strings (confirmed in source: `satori/go.uuid`, `projects/usecases/projects.go`). They cast directly to `initiatives.id UUID` — same value, no bridging column. Ledger's `project_id` column already matches `initiatives.id` for all migrated rows because LFF always placed the DynamoDB ID in Stripe metadata, which Ledger stored verbatim. See OQ-15 in `03-open-questions.md`.

---

## Post-cutover Cleanup

Once the application no longer writes to DynamoDB, drop the migration scaffolding:

```sql
ALTER TABLE initiatives DROP COLUMN source_dynamo_table;
```
