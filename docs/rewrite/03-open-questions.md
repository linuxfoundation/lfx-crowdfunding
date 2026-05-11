# Open Questions

Questions that must be answered before or during implementation. Update status as questions are resolved.

Format: **Question** | Owner | Status | Resolution

---

## Infrastructure & Networking

### OQ-1: Can K8s cluster reach Lambda API Gateway endpoints?

**Status:** Resolved
**Owner:** Architect

**Resolution:** Both Ledger and Reimbursement Service APIs are reachable over public HTTPS from K8s.

The new CF service only calls the Reimbursement Service for one purpose: expense approval/rejection (`POST /expense/{action}/{reportID}`) triggered via JWT email links. Project/entity sync to Expensify (create/update policy) stays on the old Lambda for the initial release and is not ported.

---

### OQ-2: Is the Ledger API URL public HTTPS or private VPC?

**Status:** Resolved
**Resolution:** Ledger API is public HTTPS. Confirmed by Lewis. No VPC peering or additional network configuration needed. CF K8s service can call Ledger API directly over HTTPS. OQ-1 is also resolved for the Ledger dependency.

---

### OQ-3: Mentorship → CF data sync mechanism

**Status:** Resolved
**Owner:** Michal / Architect

**Resolution:** SNS/SQS is dropped entirely. CF syncs Mentorship program data from Snowflake via a periodic K8s CronJob.

**Rationale:**
- Mentorship and CF run in separate AWS accounts. Cross-account SNS/SQS subscriptions require coordination with the Mentorship team and their AWS account — not something CF can do unilaterally.
- Mentorship is moving to Kubernetes in the coming months, making Lambda-era SNS/SQS infrastructure a poor long-term investment.
- Both Mentorship and CF mirror their data into Snowflake. CF can pull mentorship program data from Snowflake directly.
- A 24h sync delay is acceptable: new mentorship programs are not immediately donation-ready by business requirement, and mentees/beneficiaries don't access funds until mid-term (months after program creation).

**How it works:**
- CF runs a K8s CronJob (`mentorship-sync`) that queries Snowflake for mentorship programs and creates/updates `initiative_type = mentorship` rows in the CF Postgres DB.
- CF no longer publishes SNS events back to Mentorship — those were only needed for the old bidirectional sync.
- The five direct HTTP calls from the old system (slug sync, funding status, title-check, addbeneficiary, removebeneficiary) are all eliminated — either because the data is available in Snowflake or because the use case no longer applies.

**No Mentorship code changes required for the integration.** All data flows through Snowflake. Mentorship does not call CF APIs directly in the new system.

---

### OQ-4: GitHub repo `linuxfoundation/lfx-crowdfunding` — created and visibility confirmed?

**Status:** Resolved
**Owner:** DevOps

**Resolution:** Repo is `linuxfoundation/lfx-crowdfunding`. This is the repo where implementation lives and where these docs reside.

---

### OQ-5: ArgoCD app for Crowdfunding K8s deployment

**Status:** Resolved
**Owner:** DevOps

**Resolution:** Namespace is `crowdfunding` — consistent with the LFX ArgoCD pattern (logical service name, no `lfx-v2-` prefix; e.g. `query-service`, `committee-service`, `ui`). ArgoCD entry goes in `lfx-v2-applications.yaml`; Helm chart at `charts/lfx-crowdfunding/` in this repo.

---

## Data & Stripe

### OQ-6: Hardcoded Stripe Plan IDs / Product IDs outside DynamoDB?

**Status:** Resolved
**Owner:** Michal

**Resolution:** Queried production data via Snowflake (`FIVETRAN_INGEST.DYNAMODB_PRODUCT_US_EAST_1.LFF_PROD_PROJECTS`). Findings:

- **356 projects** have `stripe_plan_id` and `stripe_product_id` populated — the majority are mentorship-type programs.
- **104 active subscriptions** have a `stripeSubscriptionId` — these are live recurring charges in Stripe.
- No Stripe Plan IDs or Product IDs found hardcoded outside DynamoDB (no SSM parameters, Terraform, or config files reference specific plan/product IDs).

**Migration requirement:** All Stripe Plan IDs, Product IDs, and subscription IDs must be migrated as-is to the new Postgres tables. Active subscriptions continue charging against the original Stripe plan — changing or omitting these IDs would break recurring billing. The `subscriptions` table must have a UNIQUE constraint on `stripe_subscription_id` to preserve data integrity.

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

### OQ-15: Ledger balance lookup for post-cutover initiatives (no `legacy_id`)

**Status:** Open
**Owner:** Michal / Lewis

**Question:** The `amount_raised_cents` cache column is kept fresh by calling `GET /balance/{legacy_id}` on the Ledger API. `legacy_id` is the old DynamoDB string ID, preserved during migration for all existing initiatives.

New initiatives created after DNS cutover have no DynamoDB origin and therefore no `legacy_id`. The Ledger API has no key to look up their balance. Additionally, when CF creates a Stripe charge or subscription for a new post-cutover initiative, it must put an ID in the Stripe object metadata fields `projectID` / `entityID` — Ledger reads this to associate transactions with initiatives. If the new Postgres UUID is used instead of `legacy_id`, `GET /balance/{legacy_id}` will not find those transactions and the balance will always be `0`.

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

4. **`legacy_id` column:** For migrated initiatives, `legacy_id` holds the old DynamoDB string ID and CF uses that in Stripe metadata (as today). For post-cutover initiatives, `legacy_id` is `NULL`. CF branches on `legacy_id IS NULL` to decide which ID to put in Stripe metadata: `legacy_id` if set, Postgres UUID otherwise.

5. **Reconciliation CronJob:** Calls `GET /balance/{id}` for all published initiatives. For migrated rows: uses `legacy_id`. For post-cutover rows: uses `id` (UUID). Same logic as step 4.

**The only risk:** If any other Ledger code path assumes `project_id` is in a non-UUID format (e.g. a slug or a DynamoDB-style opaque string), it would break silently. Lewis should verify no such assumption exists before approving this approach. Based on code review, no such assumption was found — `project_id` is treated as an opaque text key throughout Ledger.

---

### OQ-14: Ledger Expensify fallback — OpenSearch dependency

**Status:** Open
**Owner:** Lewis

**Question:** Ledger's Expensify webhook handler (`expensify/main.go`) has a fallback path: when an incoming Expensify expense has no `projectID` field, it calls `getProjectIDByReport()` which queries three OpenSearch indices to resolve the project ID from the report ID via slug lookup:

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

**Status:** Resolved — traced from source code
**Owner:** Michal

**Resolution:** The full flow is:

1. RS sends email with link `https://{FUNDING_URL}/expense-email/approve/{reportID}`
2. Link hits the CF **frontend** at `/expense-email/approve/:reportId` — requires Auth0 login (`AuthGuardService`)
3. Frontend calls `POST /v1/projects/approvals/approve/{reportId}` (or `/reject/`) on the CF API (Auth0 JWT)
4. CF API calls RS at `POST {REIMBURSEMENTS_API_URL}/expense/{action}/{reportID}` with two headers:
   - `X-API-KEY: {REIMBURSEMENTS_API_SECRET}` — RS validates this against its own `RS_API_KEY`
   - `Authorization: Bearer {M2M client token}` — obtained via Auth0 client credentials grant

**Implementation requirements for new CF Go API:**
- Env vars needed: `REIMBURSEMENTS_API_URL`, `REIMBURSEMENTS_API_SECRET`, Auth0 M2M client credentials (`AUTH_CLIENT_ID`, `AUTH_SECRET`, `AUTH_AUDIENCE`, `AUTH_URL`) — these are already listed in the current LFF config
- The `/expense-email/approve/:reportId` and `/expense-email/reject/:reportId` routes **must be implemented in the new Nuxt frontend** — they are distinct from the initiative approval flow (`/email/approve-initiative`) and currently missing from the target architecture pages list. See `04-target-architecture.md`.

---

### OQ-8: New Auth0 application for rewritten Crowdfunding

**Status:** Resolved — PR merged at [linuxfoundation/auth0-terraform#299](https://github.com/linuxfoundation/auth0-terraform/pull/299)
**Owner:** Michal

**Resolution:** `auth0_client.lfx_crowdfunding` created as `regular_web` app in all three tenants (dev, staging, prod) with `authorization_code` + `refresh_token` grants, RS256 JWT, rotating refresh tokens, and PKCE-compatible server-side token exchange. Callback path: `/api/auth/callback`.

Pending: DevOps to share the new `client_id` values per tenant so they can be set in `NUXT_PUBLIC_AUTH0_CLIENT_ID` env vars (via AWS Secrets Manager / ESO). The old Auth0 app (`CB Funding`) stays active until the old Lambda stack is decommissioned.

---

### OQ-9: Mentorship → CF direct HTTP calls — will they work post-cutover?

**Status:** Resolved — moot
**Owner:** Architect / Mentorship team

**Resolution:** All five direct HTTP calls from Mentorship to CF have been eliminated. Mentorship no longer calls CF APIs directly — all data flows through Snowflake. The `FUNDING_API_URL` config in Mentorship can be removed when Mentorship moves to Kubernetes. No DNS or network path coordination needed.

---

## Design

### OQ-10: UI prototype — final design or rough reference?

**Status:** Resolved
**Owner:** Michal / Design

**Resolution:** The prototype is a rough reference only. The UI designer is still finalizing the design. Implement functionally using PrimeVue components — do not spend time on pixel-perfect matching against the prototype. UI will be updated once final designs are delivered.

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
| OQ-1 | Can K8s reach Lambda API Gateway endpoints? | Yes — both reachable over public HTTPS. CF only calls RS for expense approval/rejection; project sync stays on old Lambda. |
| OQ-3 | Mentorship → CF sync mechanism | SNS/SQS dropped. All data (programs + beneficiaries) via Snowflake CronJob. Zero direct HTTP calls between Mentorship and CF. |
| OQ-4 | GitHub repo created? | Yes — `linuxfoundation/lfx-crowdfunding`. |
| OQ-5 | ArgoCD namespace for CF K8s deployment | `crowdfunding` namespace. Helm chart in `charts/lfx-crowdfunding/` in the CF repo; ArgoCD entry in `lfx-v2-applications.yaml`. |
| OQ-6 | Stripe Plan/Product IDs outside DynamoDB? | 356 projects have Stripe plan/product IDs (mostly mentorship programs); 104 active subscriptions. All must be migrated as-is. No IDs hardcoded outside DynamoDB. |
| OQ-7 | RS OpenSearch migration plan | Two-phase. Phase 1 (CF release day): RS reads CF data via three internal HTTPS endpoints on CF API — slug lookup, bulk published list (required for RefreshTags cron), and user lookup. Phase 1 endpoints must be live before old Lambda is decommissioned. Phase 2 (when RS moves to K8s): RS gets own DB on shared RDS, migrates its three OpenSearch indices to Postgres, switches CF reads to direct SQL. OpenSearch decommissions at Phase 2. |
| OQ-8 | New Auth0 app for rewritten CF | Merged — `auth0_client.lfx_crowdfunding` created in all 3 tenants as `regular_web` with PKCE. Pending: DevOps to share new `client_id` values per tenant for ESO secrets. Old app stays active until Lambda decommission. |
| OQ-9 | Mentorship → CF direct HTTP calls post-cutover | Moot — all five calls eliminated. Mentorship no longer calls CF. Data flows through Snowflake. |
| OQ-10 | UI prototype fidelity | Rough reference only. Implement functionally with PrimeVue; update once designer delivers final designs. |
| OQ-12 | Can RS Lambda reach CF Postgres? | No — shared RDS is private (`publicly_accessible = false`), K8s-only. RS Lambda is in a separate AWS account/VPC. RS reads CF data via CF public HTTPS API instead. Direct DB access deferred until RS moves to K8s. |
| R-2 | Does Reimbursement Service query Crowdfunding OpenSearch? | Yes — reads `projects`, `entities`, `lff-users`, `spring-projects`, `spring-users`, `beneficiary-actions`, `travel-funds-tickets`. Writes `lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`. Migration plan in OQ-7. |
| R-3 | Who owns the Mentorship SNS topic? | Mentorship (jobspring) owns it. CF is a subscriber. Topic: `lfx-topic-{stage}-project`. CF queue: `fundspring-lfx-queues-{stage}-project`. |
| R-4 | Is there a separate admin UI for project approvals? | No. Approvals are done via HMAC-signed token links in emails sent to the CF approver (Sriji, LFID: `shubhrakar`). The token encodes the initiative ID and action — no Auth0 login required to click the link. |
| R-5 | Should Expensify sync be rewritten for initial release? | No. Keep old Lambda running it. Not end-user visible. Reimbursement Service unchanged. |
| R-6 | Is lfx-v1-sync-helper useful for CF DB migration? | No. It syncs project/committee metadata via NATS KV. Does not touch CF donations, subscriptions, or Ledger data. |
