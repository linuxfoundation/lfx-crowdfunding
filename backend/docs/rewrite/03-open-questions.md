<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Open Questions

Questions that must be answered before or during implementation. Update status as questions are resolved.

---

## Open

### OQ-23: Auth0 sub → LFID username migration

**Status:** Open — action required before launch
**Owner:** Michal
**Source:** Architecture call with Eric Searcy, May 28 2026

**Context:** The LFX v2 platform is migrating from Auth0 `sub` identifiers (e.g. `auth0|elim`) to LFID usernames across all services. Eric announced this is happening in approximately two weeks (from the call date). The motivation: Auth0 `sub` is an internal Auth0 identifier, not a stable LFID identity. Applications should never expose it externally or use it as a join key — the trailing part after `auth0|` looks like a username but is not guaranteed to be one.

**Impact on CF:**

1. **Old DynamoDB data uses Auth0 subs.** The existing CF data stores user identifiers as Auth0 pipe subs (`auth0|...`). These must be converted to LFID usernames during the DynamoDB → Postgres migration.

2. **Conversion is one-directional at migration time.** Eric's guidance: there is an algorithm to go *from* LFID username *to* Auth0 pipe sub (for all users that exist today), but not reliably in reverse (the part after `auth0|` can be a SHA sub, not always a username). The correct approach is to call Auth0 at migration time to look up the username for each stored sub.

3. **New CF Postgres schema must use LFID username as the user identifier.** Columns that currently map to `users.user_id` (Auth0 sub) need to store LFID usernames instead. This affects the `donations`, `subscriptions`, and `stripe_payment_accounts` tables (and any others storing per-user data).

4. **Stripe customer keys.** CF currently keys Stripe customers by Auth0 sub. After migration these must be keyed by LFID username — requires a one-time Stripe customer metadata update alongside the Postgres migration.

5. **M2M identity header.** The `X-Username` header in `08-self-serve-auth.md` already reflects this decision — it carries LFID username, not sub.

6. **EasyCLA may be affected.** David Deal raised this on the call — needs a separate check.

**Required actions:**
- Determine whether the Auth0 Management API is the right tool for the bulk sub → username lookup at migration time, or whether user-service already provides this.
- Add the username conversion step to the DynamoDB → Postgres migration script.
- Update CF Postgres schema columns that store user identity to use LFID username.
- Check EasyCLA for the same issue.

**Blocking:** DynamoDB → Postgres data migration; M2M middleware implementation in CF backend.

---

### OQ-20: GitHub URL storage — where does it live?

**Status:** Open
**Owner:** Michal / PM

**Question:** The initiative overview header shows a "GitHub" button linking to the project's GitHub org/repo. Currently the GitHub URL is stored in `initiative_custom_websites` (name = 'GitHub'), which is a freeform table not surfaced by the API. The `initiatives` table has no dedicated `github_url` column.

Options:
1. **Add `github_url` column to `initiatives`** — clean, typed, indexed, matches the prominence of the field in the UI.
2. **Query `initiative_custom_websites` by name = 'GitHub'** — no schema change, but fragile (case-sensitive name match, no uniqueness constraint).
3. **Keep it in `initiative_custom_websites` and add a unique partial index** — compromise; still requires a JOIN on every initiative detail fetch.

The old DynamoDB model stored it under `projectDetails.GithubURL` (projects) and had no equivalent on entities. A dedicated column likely makes the most sense for a field this prominent, but needs PM confirmation on whether entities (events, mentorship programs) also have GitHub URLs.

**Action:** PM to confirm whether GitHub URL applies to all initiative types or projects only. Then decide schema approach.

**Blocking:** GitHub button in the initiative header is always hidden.

---

### OQ-21: Ledger txnCategory ↔ CF goal name — case and spelling mismatch

**Status:** Open
**Owner:** Michal / Lewis

**Question:** The Ledger `/balance/{id}` response returns `subTotals` keyed by PascalCase category names (`"Mentorship"`, `"Development"`, `"BugBounty"`, `"Uncategorised"`). The Ledger `/transactions/` response returns `txnCategory` in lowercase (`"mentorship"`, `"other"`). Our `initiative_goals` table uses the `name` column to identify goals (e.g. `"mentorship"`, `"development"`, `"travel"`).

The matching logic needed to populate per-goal `donated_cents`/`spent_cents` from Ledger `subTotals` must normalise casing. This is implemented with a case-insensitive match (lowercasing both sides) — but there are still potential mismatches:

- Ledger uses `"BugBounty"` (no space); CF uses `"bug_bounty"` or `"bugbounty"` depending on how goals were created
- Ledger uses `"Uncategorised"` for transactions with no category set; these cannot map to any CF goal
- Ledger uses `"Mentee"` in some contexts, CF uses `"mentee"` as goal name — needs verification
- Future goal names added via the UI must match exactly one Ledger category or their subTotal will always be zero

**Action:** Lewis to confirm the canonical list of Ledger `txnCategory` values and whether they are stable. CF to document the mapping table and enforce it at goal creation time.

**Blocking:** Per-goal donated/spent donut charts on the initiative overview screen.

---

### OQ-7: Reimbursement Service OpenSearch dependency — long-term plan

**Status:** Partially resolved — Phase 1 plan confirmed; Phase 2 blocked on RS moving to K8s.
**Owner:** Michal

**Phase 1 — on CF release day:**
RS switches Category 1 reads (CF-owned data) from OpenSearch to three narrow internal HTTP endpoints on the CF Go API:

| OpenSearch index replaced | CF internal endpoint | Used by RS for |
|---|---|---|
| `projects` + `entities` (per-slug) | `GET /internal/v1/initiatives?slug={slug}` | Project/entity owner lookup (`getEmailBySlug`) |
| `projects` + `entities` (bulk) | `GET /internal/v1/initiatives?status=published` | Bulk tag rebuild (`RefreshTags` cron, runs every 3h) |
| `lff-users` | `GET /internal/v1/users/{owner_id}` | Owner email lookup |

**Why the bulk endpoint is required:** Once CF DNS cuts over, the new CF service writes exclusively to
Postgres — OpenSearch receives no new writes and becomes a stale snapshot. `RefreshTags()` runs every
3 hours and bulk-reads all published initiatives to rebuild Expensify GL code tags. If it keeps reading
from stale OpenSearch, new projects created after cutover will never appear as Expensify tags and
beneficiaries cannot submit expenses against them — silent failure with real financial impact.

These endpoints are on the CF public HTTPS ingress, authenticated via a shared secret (`X-Internal-Token` header). RS Lambda can reach them over public HTTPS (OQ-1/OQ-2 confirmed reachable). No direct Postgres access from RS Lambda — the shared LFX v2 RDS is `publicly_accessible = false` and unreachable from the RS Lambda VPC.

**Phase 2 — when RS moves to Kubernetes (timeline TBD):**
- RS gets its own database on the shared RDS via the `lfx-v2-opentofu` pattern
- RS migrates its three OpenSearch indices (`lfx-expense-log` → `reimbursement.expense_log`, `beneficiary-actions` → `reimbursement.beneficiary_actions`, `travel-funds-tickets` → `reimbursement.travel_fund_tickets`) to its own Postgres DB
- RS switches CF data reads from Phase 1 HTTP endpoints to a read-only Postgres role on `crowdfunding` schema
- OpenSearch decommissions at this point

**Notes:**
- OpenSearch must NOT be decommissioned before Phase 2 — RS still owns three live indices there
- The old CF Lambda keeps `projects`/`entities` OpenSearch indices populated during the parallel-run window only. Phase 1 HTTP endpoints must be live before the old Lambda is shut down
- Existing `lfx-expense-log` data must be migrated to Postgres before OpenSearch cutover to preserve email deduplication history

---

### OQ-11: Full scope of CF data needed in LFX Self Serve

**Status:** Open — UI design exists, needs review
**Owner:** Michal / PM

**Question:** The PM has requested "My Donations" and "My Initiatives" in LFX Self Serve, but the full list of CF data surfaces and their Snowflake data requirements are not confirmed.

**Action:** Review the existing LFX Self Serve UI design and extract the full list of CF fields and data types required. This determines which columns must be in the Fivetran CF→Snowflake sync and what Snowflake views LFX Self Serve needs. No integration code is written until this is confirmed.

---

### OQ-14: Ledger Expensify fallback — OpenSearch dependency

**Status:** Open
**Owner:** Lewis

**Question:** Ledger's Expensify webhook handler (`expensify/main.go`) has a fallback path: when an incoming Expensify expense has no `projectID` field, it calls `getProjectIDByReport()` which queries four OpenSearch indices to resolve the project ID via slug lookup (`lfx-expense-log`, `projects`, `entities`, `spring-projects`).

The `spring-projects` index is owned and written by the Mentorship service (jobspring), not CF. When OpenSearch is decommissioned, this fallback breaks for all three populations. OpenSearch decommission is therefore gated on Mentorship's K8s migration, independent of CF's timeline.

**Questions for Lewis:**
- How frequently does this fallback trigger? Is `expense.ProjectID` reliably populated by Expensify, or does it regularly fall back to the OpenSearch lookup?
- If it triggers regularly: (a) confirm Expensify always includes projectID, (b) update Ledger fallback to call CF internal endpoint + Mentorship endpoint, or (c) accept data loss as low-frequency and monitor.

**Blocking:** OpenSearch decommission.

**Action:** Lewis to check Ledger logs/metrics for how often `getProjectIDByReport()` is called in production.

---

### OQ-15: Ledger balance lookup for post-cutover initiatives

**Status:** Open — pending Lewis confirmation
**Owner:** Michal / Lewis

**Question:** New initiatives created after DNS cutover have no DynamoDB origin. The recommended approach is to use the Postgres UUID directly as the project ID in Stripe metadata and `GET /balance/{id}` calls — Ledger's regex accepts UUIDs and no Ledger code changes are required. See `02-decisions.md` for the full rationale.

**Action:** Lewis to confirm no Ledger code path assumes `project_id` is in a non-UUID format before this approach is adopted.

**Blocking:** Stripe integration for post-cutover initiatives; `ledger-stats-sync` CronJob; backer list (`GET /v1/initiatives/{id}/backers`).

---

### OQ-19: Ledger API shape for stats-sync

**Status:** Open — pending UI field review
**Owner:** Michal / Lewis

**Question:** Before the `ledger-stats-sync` CronJob is designed, two things must be confirmed:

1. **Which fields does the UI need?** Review the initiative card and discovery page designs. This determines how many Ledger API calls are required and whether a combined endpoint is worthwhile.
2. **Ledger API shape:** extend `GET /balance/{id}`, add a new `GET /stats/{id}` endpoint (preferred if multiple fields), or keep separate endpoints per field.

**Action:** Michal to review UI designs and confirm required fields. Then align with Lewis on Ledger API shape before implementation begins.

**Blocking:** Design of `ledger-stats-sync` CronJob and any new Ledger API endpoints.

---

### OQ-22: Feature-branch deployment — pattern and infrastructure

**Status:** Open
**Owner:** DevOps / Michal

**Question:** Like LFX Self Serve, should Crowdfunding support per-feature-branch deployments in dev? This would let engineers test frontend and backend changes end-to-end before merging, without sharing a single dev environment.

Key decisions:
1. **ArgoCD ApplicationSet pattern** — does the existing LFX Self Serve ApplicationSet use a Git or PR generator to auto-create per-branch apps? What namespace isolation model does it use?
2. **DNS / ingress** — are feature branches served at `<branch>.crowdfunding.dev.platform.linuxfoundation.org` or a different scheme?
3. **Secret provisioning** — do feature-branch deployments share the dev ESO SecretStore and tag-based secrets, or do they need separate 1Password items?
4. **Teardown** — are per-branch environments torn down automatically on PR merge/close, or manually?

**Action:** Check `lfx-v2-argocd` for the Self Serve ApplicationSet definition and any PR/branch generator config. This is the fastest path to understanding the pattern already in use.

**Blocking:** Feature-branch testing workflow.

---

## Resolved

| # | Question | Resolution |
|---|---|---|
| OQ-18 | Cross-account DB mirroring — architect decision | Rejected by Eric (May 2026): coupling services via shared DB schema is wrong; API is the correct contract. Confirmed approach: stats-sync via Ledger HTTP API. See `02-decisions.md` and OQ-19. |
| OQ-1 | Can K8s cluster reach Lambda API Gateway endpoints? | Yes — both Ledger and RS APIs are reachable over public HTTPS from K8s. |
| OQ-2 | Is the Ledger API URL public HTTPS or private VPC? | Public HTTPS — CF K8s can call Ledger API directly. |
| OQ-3 | Mentorship → CF data sync mechanism | SNS/SQS dropped. CF syncs from Snowflake via K8s CronJob. See `02-decisions.md`. |
| OQ-4 | GitHub repo created and visibility confirmed? | Yes — `linuxfoundation/lfx-crowdfunding`. |
| OQ-5 | ArgoCD app for CF K8s deployment | Namespace `crowdfunding`, ArgoCD entry in `lfx-v2-applications.yaml`, Helm chart at `charts/lfx-crowdfunding/`. |
| OQ-6 | Hardcoded Stripe Plan/Product IDs outside DynamoDB? | 356 projects have Stripe plan/product IDs; 104 active subscriptions. All migrated as-is to Postgres. |
| OQ-8 | New Auth0 application for rewritten CF | `lfx_crowdfunding` created as `regular_web` in all tenants (auth0-terraform#299). Pending: DevOps to share `client_id` values per tenant for ESO secrets. |
| OQ-9 | Mentorship → CF direct HTTP calls post-cutover? | All five calls eliminated. Data flows through Snowflake. See OQ-3. |
| OQ-10 | UI prototype — final design or rough reference? | Rough reference only. Implement functionally with PrimeVue; final designs TBD. |
| OQ-12 | Can RS Lambda reach CF Postgres? | No — shared RDS is private, K8s-only. RS reads CF data via public HTTPS API. Direct DB access deferred to RS K8s migration. |
| OQ-13 | RS expense approval — how does auth work? | Auth flow traced. See `04-target-architecture.md` for env vars and routes required. |
| OQ-16 | Are Ledger transactions append-only? How are refunds recorded? | Superseded — irrelevant now that raw table mirroring is rejected (OQ-18). |
| OQ-17 | Is "amount raised" gross or net of fees? | Superseded — `GET /balance/{id}` encodes this; CF uses whatever it returns. |
| R-1 | Does LFF write directly to Ledger DB? | No — LFF calls Ledger HTTP API read-only. Ledger writes come from its own Stripe/Expensify webhooks. |
| R-2 | Does RS query CF OpenSearch? | Yes — reads `projects`, `entities`, `lff-users`, `spring-projects`, `spring-users`, `beneficiary-actions`, `travel-funds-tickets`. Migration plan in OQ-7. |
| R-3 | Who owns the Mentorship SNS topic? | Mentorship (jobspring). Topic: `lfx-topic-{stage}-project`. CF queue: `fundspring-lfx-queues-{stage}-project`. |
| R-4 | Separate admin UI for project approvals? | No — HMAC-signed token links in emails to CF approver (Sriji, `shubhrakar`). No Auth0 login required. |
| R-5 | Should Expensify sync be rewritten for initial release? | No — keep old Lambda. Not end-user visible; RS unchanged. |
| R-6 | Is lfx-v1-sync-helper useful for CF DB migration? | No — syncs project/committee metadata via NATS KV; does not touch CF donations, subscriptions, or Ledger data. |
