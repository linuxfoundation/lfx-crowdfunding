<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Open Questions

Questions that must be answered before or during implementation. Update status as questions are resolved.

Format: **Question** | Owner | Status | Resolution

---

## Infrastructure & Networking

### OQ-1: Can K8s cluster reach Lambda API Gateway endpoints?

**Status:** Resolved

Both Ledger and Reimbursement Service APIs are reachable over public HTTPS from K8s. See decision in 02-decisions.md for implications on Reimbursement Service integration.

---

### OQ-2: Is the Ledger API URL public HTTPS or private VPC?

**Status:** Resolved

Ledger API is public HTTPS. CF K8s service can call Ledger API directly over HTTPS.

---

### OQ-3: Mentorship → CF data sync mechanism

**Status:** Resolved

SNS/SQS dropped entirely. CF syncs Mentorship program data from Snowflake via K8s CronJob. Full rationale and implementation in 02-decisions.md.

---

### OQ-4: GitHub repo `linuxfoundation/lfx-crowdfunding` — created and visibility confirmed?

**Status:** Resolved
**Owner:** DevOps

**Resolution:** Repo is `linuxfoundation/lfx-crowdfunding`. This is the repo where implementation lives and where these docs reside.

---

### OQ-5: ArgoCD app for Crowdfunding K8s deployment

**Status:** Resolved

Namespace is `crowdfunding` with ArgoCD entry in `lfx-v2-applications.yaml` and Helm chart at `charts/lfx-crowdfunding/`. See 02-decisions.md for details.

---

## Data & Stripe

### OQ-6: Hardcoded Stripe Plan IDs / Product IDs outside DynamoDB?

**Status:** Resolved

356 projects have Stripe plan/product IDs; 104 active subscriptions exist. All must be migrated as-is to Postgres. Note: `stripe_subscription_id` is a nullable `VARCHAR(255)` with no UNIQUE constraint in the schema. See 02-decisions.md for migration strategy.

---

## Dependencies

### OQ-7: Reimbursement Service OpenSearch dependency — long-term plan

**Status:** Partially resolved — Phase 1 plan updated; Phase 2 blocked on RS moving to K8s (see OQ-12).
**Owner:** Michal

**Phase 1 — on CF release day:**
RS switches Category 1 reads (CF-owned data) from OpenSearch to three narrow internal HTTP endpoints on the CF Go API:

| OpenSearch index replaced | CF internal endpoint | Used by RS for |
|---|---|---|
| `projects` + `entities` (per-slug) | `GET /internal/v1/initiatives?slug={slug}` | Project/entity owner lookup (`getEmailBySlug`) |
| `projects` + `entities` (bulk) | `GET /internal/v1/initiatives?status=published` | Bulk tag rebuild (`RefreshTags` cron, runs every 3h) |
| `lff-users` | `GET /internal/v1/users/{owner_id}` | Owner email lookup |

**Why the bulk endpoint is required:** Once CF DNS cuts over, the new CF service writes exclusively to Postgres — OpenSearch receives no new writes. From cutover day, OpenSearch is a stale snapshot. `RefreshTags()` runs every 3 hours and bulk-reads all published initiatives to rebuild Expensify GL code tags. If it keeps reading from stale OpenSearch, new projects created after cutover will never appear as Expensify tags, and beneficiaries cannot submit expenses against them. This is a silent failure with real financial impact.

The bulk endpoint returns `[{id, name}]` for all published initiatives, where `id` is `initiatives.id` (Postgres UUID). For UUID-form DynamoDB IDs (the vast majority), this equals the original DynamoDB string ID — RS can use it directly as the Expensify GL code. For the small number of non-UUID legacy IDs, `initiatives.id` differs from the original DynamoDB string ID and RS must maintain a mapping.

These endpoints are on the CF public HTTPS ingress, authenticated via a shared secret (`X-Internal-Token` header). RS Lambda can reach them over public HTTPS — same network path as all other Lambda→K8s calls today (confirmed reachable, OQ-1/OQ-2).

No direct Postgres access from RS Lambda. The shared LFX v2 RDS is `publicly_accessible = false` and unreachable from the RS Lambda VPC (separate AWS account). See OQ-12.

**Phase 2 — when RS moves to Kubernetes (timeline TBD):**
Once RS is on K8s in the same cluster, it can reach the shared RDS directly. At that point:
- RS gets its own database on the shared RDS via the `lfx-v2-opentofu` pattern (one entry in `postgres.tf`)
- RS migrates its three OpenSearch indices to its own Postgres DB:

| OpenSearch index | Replacement |
|---|---|
| `lfx-expense-log` | `reimbursement.expense_log` table |
| `beneficiary-actions` | `reimbursement.beneficiary_actions` table |
| `travel-funds-tickets` | `reimbursement.travel_fund_tickets` table |

- RS switches CF data reads from the Phase 1 HTTP endpoints to a read-only Postgres role on `crowdfunding` schema (direct SQL, no HTTP layer)
- OpenSearch decommissions at this point

**Notes:**
- The "CF release + 2 weeks" hard deadline for Phase 2 is removed — it was premature given OQ-12
- OpenSearch must NOT be decommissioned before Phase 2 — RS still owns three live indices there (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`)
- The old CF Lambda keeps `projects`/`entities` OpenSearch indices populated during the parallel-run window only — once it is decommissioned, those indices go stale. Phase 1 HTTP endpoints must be live before the old Lambda is shut down
- `beneficiary-actions` remove path is fully commented out in RS (`PolicyEmployeesRemove` at service.go:919) — low risk
- Existing `lfx-expense-log` data must be migrated to Postgres before OpenSearch cutover to preserve email deduplication history

---

### OQ-15: Ledger balance lookup for post-cutover initiatives (no original DynamoDB string ID)

**Status:** Open
**Owner:** Michal / Lewis

**Question:** The `amount_raised_in_cents` cache column is kept fresh by calling `GET /balance/{id}` on the Ledger API. The migration script uses `_as_uuid()` to generate Postgres UUIDs: if the DynamoDB ID is already a valid UUID (the vast majority), it is preserved unchanged as `initiatives.id`; otherwise a deterministic `uuid5("coerce", id)` is generated. For UUID-form DynamoDB IDs, the Postgres UUID matches the original ID exactly, so the CF Go API can use `initiatives.id` directly when calling Ledger.

New initiatives created after DNS cutover have no DynamoDB origin and therefore no original string ID. The Ledger API has no key to look up their balance. Additionally, when CF creates a Stripe charge or subscription for a new post-cutover initiative, it must put an ID in the Stripe object metadata fields `projectID` / `entityID` — Ledger reads this to associate transactions with initiatives. If the new Postgres UUID is used instead of the original DynamoDB ID, `GET /balance/{id}` will not find those transactions and the balance will always be `0`.

**How Ledger stores and looks up `project_id`:**
- Stripe webhook handler (`stripehook/hook.go:92`) reads `ch.Metadata["projectID"]` and stores it verbatim as `project_id text` in the `ledger` table — no validation, no transformation
- `GET /balance/{projectID}` does a direct `WHERE project_id = $1` — returns `$0` for any ID not already in the table
- Regex validation on the balance endpoint (`^[0-9a-zA-Z\_\-]+$`) **accepts UUIDs** — hyphens and hex chars all pass
- There is no project registration mechanism in Ledger; it is purely passive

**Questions for Lewis:**
- Should the new Postgres UUID be used directly as the project ID in Stripe metadata and `GET /balance/{id}` for post-cutover initiatives?
- Or is there a different mechanism Lewis prefers?

**Blocking:** Stripe webhook refresh and reconciliation CronJob for any initiative created after cutover. Also blocks correct Stripe metadata on new donations/subscriptions. Also affects the backer list — `GET /v1/initiatives/{id}/backers` for org-only backers calls `GET /transactions/?projectId={id}` on the Ledger API using the same ID; same resolution applies.

**Action:** Lewis to confirm approach.

---

**AI Recommendation: use the Postgres UUID directly for post-cutover initiatives**

No Ledger code changes are required. Here is why this works end-to-end:

1. **Stripe metadata:** CF puts the Postgres UUID in `ch.Metadata["projectID"]` at charge-creation time for new initiatives. Ledger stores it verbatim as `project_id`.

2. **Balance lookup:** CF calls `GET /balance/{uuid}` for post-cutover initiatives. Ledger's regex (`^[0-9a-zA-Z\_\-]+$`) accepts UUIDs. The query `WHERE project_id = $1` finds the rows because CF put the UUID in Stripe metadata in step 1. This works correctly with no Ledger changes.

3. **`SendNotifications()` calls CF:** Ledger calls `GET /v1/projects/{project_id}` using the stored `project_id`. For post-cutover rows that value is a UUID. The new CF Go API already needs to support UUID lookups on these endpoints for its own use — so this costs nothing extra.

4. **Ledger ID for migrated vs post-cutover initiatives:** For migrated initiatives with UUID-form DynamoDB IDs (the vast majority), the Postgres UUID matches the original ID exactly — CF uses `initiatives.id` directly in Stripe metadata and Ledger lookups. For the small number of non-UUID legacy IDs, the service must store/maintain a mapping if Ledger lookups are required. For post-cutover initiatives with no DynamoDB origin, CF uses the Postgres UUID directly.

5. **Reconciliation CronJob:** Calls `GET /balance/{id}` for all published initiatives using `initiatives.id` (which matches the original DynamoDB ID for UUID-form IDs).

**The only risk:** If any other Ledger code path assumes `project_id` is in a non-UUID format (e.g. a slug or a DynamoDB-style opaque string), it would break silently. Lewis should verify no such assumption exists before approving this approach. Based on code review, no such assumption was found — `project_id` is treated as an opaque text key throughout Ledger.

---

### OQ-18: Architect approval — mirror `ledger` table into CF DB

**Status:** Resolved — stats-sync via Ledger HTTP API confirmed (May 2026)
**Owner:** Eric (architect) / Michal

**Question:** Should CF mirror the raw Ledger `ledger` table into CF DB via cross-account DB access, or sync pre-aggregated stats via the Ledger HTTP API?

**Decision:** Eric rejected cross-account DB mirroring. Rationale: coupling two services against the same database schema is not the right approach — an API is the correct way to expose a defined contract. Since CF owns both services, the coordination cost of adding Ledger API endpoints alongside CF UI changes is low.

**Confirmed approach:** stats-sync via Ledger HTTP API. A CronJob calls Ledger HTTP endpoints to sync pre-aggregated financial stats as cached columns on `crowdfunding.initiatives`. See `02-decisions.md` for implementation details. See OQ-19 for the follow-on open question on Ledger API shape.

---

### OQ-19: Ledger API shape for stats-sync

**Status:** Open — pending UI field review
**Owner:** Michal / Lewis

**Question:** The confirmed approach (OQ-18) is to sync pre-aggregated financial stats from the Ledger HTTP API into cached columns on `crowdfunding.initiatives`. The initial release uses the existing `GET /balance/{id}` endpoint (via `amount-raised-sync`). As the UI evolves, additional fields will be needed — backer count, subscription totals, "close to goal" indicator, etc.

Before the `ledger-stats-sync` CronJob is designed, two things must be confirmed:

1. **Which fields does the UI actually need?** Review the initiative card and discovery page designs to produce a complete list. This determines how many Ledger API calls are required and whether a single combined endpoint is worthwhile.

2. **What is the preferred Ledger API shape?** Options:
   - Extend `GET /balance/{id}` to return additional fields alongside the current balance
   - Add a new `GET /stats/{id}` endpoint returning all financial stats in one call (preferred if multiple fields are needed — one HTTP call per initiative per cron run)
   - Keep separate endpoints per field (simpler Ledger changes but more cron complexity)

**Blocking:** Design of `ledger-stats-sync` CronJob and any new Ledger API endpoints.

**Action:** Michal to review UI designs and confirm required fields. Then align with Lewis on the Ledger API shape before implementation begins.

---

### OQ-16: Are Ledger transactions append-only? How are refunds recorded?

**Status:** Superseded — no longer relevant (OQ-18 resolved; raw table mirroring rejected)

This question was relevant to the `ledger-sync` raw table mirroring approach (Plan A). Since OQ-18 resolved in favour of stats-sync via Ledger HTTP API, the sync strategy does not involve direct access to the Ledger `ledger` table and this question does not need to be answered.

---

### OQ-17: Is "amount raised" gross or net of fees?

**Status:** Superseded — answered implicitly by confirmed approach (OQ-18 resolved)

The `GET /balance/{id}` Ledger API endpoint already encodes the gross vs net choice — whatever it returns is what `amount-raised-sync` stores as `amount_raised_in_cents`, and that behaviour is unchanged. This question was relevant to writing a raw SQL aggregation query over the `ledger` table (Plan A), which is no longer the approach.

---

### OQ-14: Ledger Expensify fallback — OpenSearch dependency

**Status:** Open
**Owner:** Lewis

**Question:** Ledger's Expensify webhook handler (`expensify/main.go`) has a fallback path: when an incoming Expensify expense has no `projectID` field, it calls `getProjectIDByReport()` which queries four OpenSearch indices to resolve the project ID from the report ID via slug lookup:

- `lfx-expense-log` — fetches the expense record to extract the first tag (used as slug)
- `projects` (LFF CF projects) — slug lookup
- `entities` (LFF CF entities) — slug lookup
- `spring-projects` (Mentorship/jobspring programs) — slug lookup

The `spring-projects` index is **owned and written by the Mentorship service (jobspring)**, not by CF. It is a separate index population from `projects`/`entities`. When OpenSearch is decommissioned, this fallback breaks for **all three populations** — not just CF projects.

**Implication:** OpenSearch cannot be decommissioned until both CF and Mentorship have migrated off it. CF's migration replaces `projects`/`entities` reads with CF internal HTTP endpoints. Mentorship's `spring-projects` index has no replacement until Mentorship moves to Kubernetes. This means OpenSearch decommission is gated on Mentorship's K8s migration, independent of CF's timeline.

**Questions for Lewis:**
- How frequently does this fallback trigger in practice? Is `expense.ProjectID` reliably populated by Expensify, or does it regularly fall back to the OpenSearch lookup?
- If it triggers regularly, what is the fix? Options: (a) Expensify is configured to always include projectID — confirm this is the case, (b) Ledger fallback is updated to call CF internal endpoint (covers CF projects) + Mentorship endpoint (covers spring-projects), (c) accept the data loss as low-frequency and monitor.

**Blocking:** OpenSearch decommission. Must be assessed before OpenSearch is shut down.

**Action:** Lewis to check Ledger logs/metrics for how often `getProjectIDByReport()` is actually called in production.

---

### OQ-13: Reimbursement Service expense approval — how does auth work?

**Status:** Resolved

Auth flow traced. Implementation requirements: Env vars `REIMBURSEMENTS_API_URL`, `REIMBURSEMENTS_API_SECRET`, Auth0 M2M credentials, plus `/expense-email/approve|reject` routes in Nuxt frontend. See 04-target-architecture.md.

---

### OQ-8: New Auth0 application for rewritten Crowdfunding

**Status:** Resolved

Auth0 app `lfx_crowdfunding` created as `regular_web` in all tenants (PR: linuxfoundation/auth0-terraform#299). Pending: DevOps to share new `client_id` values per tenant for ESO secrets.

---

### OQ-9: Mentorship → CF direct HTTP calls — will they work post-cutover?

**Status:** Resolved

All five direct HTTP calls from Mentorship to CF eliminated. Data flows through Snowflake. See OQ-3 for details.

---

## Design

### OQ-10: UI prototype — final design or rough reference?

**Status:** Resolved

Prototype is a rough reference only. Implement functionally with PrimeVue; UI will update once final designs are delivered.

---

### OQ-11: Full scope of CF data needed in LFX Self Serve

**Status:** Open — blocked on PM + design
**Owner:** Michal / PM

**Question:** The PM has requested "My Donations" and "My Initiatives" in LFX Self Serve, but the full list of CF data surfaces is not confirmed. Before any LFX Self Serve integration code is written, this must be clarified:

- What exactly is shown per widget — count, amount total, list of items, grouped by project?
- Which data types: donations only, or also subscriptions, owned projects, owned funds?
- Is there an empty state design?
- Are cancelled subscriptions / declined projects shown or hidden?

**Blocked on:** UI design for the LFX Self Serve CF widgets. No implementation until design is delivered and the full data list is confirmed.

**Integration approach:** Snowflake (Option B) via Fivetran CF→Snowflake sync. See decision in `02-decisions.md`.

---

## Resolved Questions (for reference)

| # | Question | Resolution |
|---|---|---|
| R-1 | Does LFF write directly to Ledger DB? | No — LFF calls Ledger HTTP API read-only. Ledger writes come from its own Stripe/Expensify webhooks. |
| OQ-4 | GitHub repo created? | Yes — `linuxfoundation/lfx-crowdfunding`. |
| OQ-12 | Can RS Lambda reach CF Postgres? | No — shared RDS is private (`publicly_accessible = false`), K8s-only. RS Lambda is in a separate AWS account/VPC. RS reads CF data via CF public HTTPS API instead. Direct DB access deferred until RS moves to K8s. |
| R-2 | Does Reimbursement Service query Crowdfunding OpenSearch? | Yes — reads `projects`, `entities`, `lff-users`, `spring-projects`, `spring-users`, `beneficiary-actions`, `travel-funds-tickets`. Writes `lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`. Migration plan in OQ-7. |
| R-3 | Who owns the Mentorship SNS topic? | Mentorship (jobspring) owns it. CF is a subscriber. Topic: `lfx-topic-{stage}-project`. CF queue: `fundspring-lfx-queues-{stage}-project`. |
| R-4 | Is there a separate admin UI for project approvals? | No. Approvals are done via HMAC-signed token links in emails sent to the CF approver (Sriji, LFID: `shubhrakar`). The token encodes the initiative ID and action — no Auth0 login required to click the link. |
| R-5 | Should Expensify sync be rewritten for initial release? | No. Keep old Lambda running it. Not end-user visible. Reimbursement Service unchanged. |
| R-6 | Is lfx-v1-sync-helper useful for CF DB migration? | No. It syncs project/committee metadata via NATS KV. Does not touch CF donations, subscriptions, or Ledger data. |
