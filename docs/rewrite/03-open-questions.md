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

Namespace is `crowdfunding` with ArgoCD entry in `lfx-v2-applications.yaml` and Helm chart will live at `charts/lfx-crowdfunding/` (not yet created). See 02-decisions.md for details.

---

## Data & Stripe

### OQ-6: Hardcoded Stripe Plan IDs / Product IDs outside DynamoDB?

**Status:** Resolved

356 projects have Stripe plan/product IDs; 104 active subscriptions exist. All must be migrated as-is to Postgres with UNIQUE constraint on `stripe_subscription_id`. See 02-decisions.md for migration strategy.

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
| `lff-users` | `GET /internal/v1/users/{owner_id}` | Owner email lookup — returns `{id, email}` where `id` is the Auth0 subject and `email` is from the CF `users` table |

**Why the bulk endpoint is required:** Once CF DNS cuts over, the new CF service writes exclusively to Postgres — OpenSearch receives no new writes. From cutover day, OpenSearch is a stale snapshot. `RefreshTags()` runs every 3 hours and bulk-reads all published initiatives to rebuild Expensify GL code tags. If it keeps reading from stale OpenSearch, new projects created after cutover will never appear as Expensify tags, and beneficiaries cannot submit expenses against them. This is a silent failure with real financial impact.

The bulk endpoint returns `[{legacy_id, name}]` for all published initiatives. RS uses `legacy_id` as the Expensify GL code (matching what is already stored in Expensify from the old system).

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

### OQ-15: Ledger balance lookup — `initiatives.id` IS the Ledger `project_id`

**Status:** Resolved
**Owner:** Michal / Lewis

**Resolution (2026-05-11):** No `legacy_id` column is needed. Source code confirmed that DynamoDB `projectId` (projects table) and `entityId` (entities table) are UUID v4 strings generated by `satori/go.uuid` at creation time. They migrate directly to `initiatives.id UUID` in Postgres — same value, no transformation. Since Ledger stores `project_id text` verbatim from Stripe metadata, and LFF always put `projectId`/`entityId` in that metadata, the value already in the Ledger DB matches `initiatives.id` for all migrated initiatives.

**How the ID flows end-to-end:**

1. LFF generates `projectId = uuid.NewV4().String()` at project creation (`projects/usecases/projects.go`)
2. LFF stores it in Stripe charge/plan metadata as `"projectID"` (or `"entityID"` for entities)
3. Ledger webhook (`stripehook/hook.go:92`) reads `ch.Metadata["projectID"]` and stores it verbatim as `ledger.project_id text`
4. Migration writes the same value directly to `initiatives.id UUID` (standard RFC4122 format casts cleanly)
5. `GET /balance/{id}` on Ledger uses `WHERE project_id = $1` — passing `initiatives.id::text` returns the correct balance with no bridging column

**For post-cutover initiatives** (no DynamoDB origin): CF puts `initiatives.id` (Postgres UUID) in Stripe metadata at charge-creation time. Ledger stores it verbatim. `GET /balance/{initiatives.id}` finds it correctly. Ledger's regex (`^[0-9a-zA-Z\_\-]+$`) accepts UUIDs. No Ledger code changes required.

**`SendNotifications()` calls CF:** Ledger calls `GET /v1/projects/{project_id}` using the stored `project_id`. For migrated initiatives that value is the Postgres UUID (same as `initiatives.id`). The CF Go API resolves by UUID — no special handling needed.

**`legacy_id` column:** Not needed. Remove any references to `legacy_id` from schema, migration plan, or application code. `initiatives.id` serves as both the Postgres PK and the Ledger lookup key for all initiatives, migrated or post-cutover.

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

### OQ-11: Full scope of CF data needed in LFX One

**Status:** Open — blocked on PM + design
**Owner:** Michal / PM

**Question:** The PM has requested "My Donations" and "My Initiatives" in LFX One, but the full list of CF data surfaces is not confirmed. Before any LFX One integration code is written, this must be clarified:

- What exactly is shown per widget — count, amount total, list of items, grouped by project?
- Which data types: donations only, or also subscriptions, owned projects, owned funds?
- Is there an empty state design?
- Are cancelled subscriptions / declined projects shown or hidden?

**Blocked on:** UI design for the LFX One CF widgets. No implementation until design is delivered and the full data list is confirmed.

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
