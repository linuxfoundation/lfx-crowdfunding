<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Open Questions

Questions that must be answered before or during implementation. Update status as questions are resolved.

---

## Open

### OQ-20: GitHub URL storage — where does it live?

**Status:** Open — awaiting PM/Lewis input.
**Owner:** Michal / PM

**Question:** The initiative overview header shows a "GitHub" button linking to the project's GitHub org/repo. Currently the GitHub URL is stored in `initiative_custom_websites` (name = 'GitHub'), which is a freeform table not surfaced by the API. The `initiatives` table has no dedicated `github_url` column.

Options:
1. **Add `github_url` column to `initiatives`** — clean, typed, indexed, matches the prominence of the field in the UI.
2. **Query `initiative_custom_websites` by name = 'GitHub'** — no schema change, but fragile (case-sensitive name match, no uniqueness constraint).

The old DynamoDB model stored it under `projectDetails.GithubURL` (projects) and had no equivalent on entities.

**Blocking:** GitHub button in the initiative header is always hidden.

**Draft message for Lewis:**

> Hi Lewis — for the initiative overview page, there's a "GitHub" button in the header that links to the project's GitHub repo. In the old DynamoDB model this was `projectDetails.GithubURL` on project-type initiatives.
>
> In the new schema, `initiatives` has no `github_url` column — it would need to come from `initiative_custom_websites` (name = 'GitHub') or a new dedicated column.
>
> Two questions:
> 1. Do you know if GitHub URL applies to project-type initiatives only, or also to funds/events?
> 2. Any preference on schema approach — dedicated `github_url` column on `initiatives`, or keep it in `initiative_custom_websites`?
>
> Happy to just add a `github_url` column if that's the simplest path.

---


### OQ-7: Reimbursement Service OpenSearch dependency — long-term plan

**Status:** Partially resolved — RS continues on OpenSearch after initial CF release; migration deferred to when RS moves to K8s.
**Owner:** Michal

**Decision for initial CF release:** RS continues reading CF-owned data from OpenSearch as it does today. The old LFF Lambda keeps the `projects`/`entities`/`lff-users` OpenSearch indices populated during the parallel-run window. No Phase 1 internal endpoints are required for the initial release.

CF → RS auth uses a static API token (`X-API-KEY` header, `REIMBURSEMENTS_API_KEY` env var). This is the implemented pattern in `internal/infrastructure/clients/reimbursement*.go`.

**When RS moves to Kubernetes (timeline TBD):**
- RS migrates its three OpenSearch indices (`lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`) to its own Postgres DB on the shared RDS
- RS switches CF data reads to the CF HTTP API (already available: `GET /v1/initiatives/{slug}/owner-info` with Auth0 M2M `access:manage`)
- OpenSearch decommissions at this point

**Notes:**
- OpenSearch must NOT be decommissioned before RS moves to K8s — RS still owns three live indices there
- Existing `lfx-expense-log` data must be migrated to Postgres before OpenSearch cutover to preserve email deduplication history

---

### OQ-11: Full scope of CF data needed in LFX Self Serve

**Status:** Open — UI design exists, needs review
**Owner:** Michal / PM

**Question:** The PM has requested "My Donations" and "My Initiatives" in LFX Self Serve, but the full list of CF data surfaces and their Snowflake data requirements are not confirmed.

**Action:** Review the existing LFX Self Serve UI design and extract the full list of CF fields and data types required. This determines which columns must be in the Fivetran CF→Snowflake sync and what Snowflake views LFX Self Serve needs. No integration code is written until this is confirmed.

---

### OQ-14: Ledger Expensify fallback — OpenSearch dependency

**Status:** Open — awaiting Lewis input.
**Owner:** Lewis

**Question:** Ledger's Expensify webhook handler (`expensify/main.go`) has a fallback path: when an incoming Expensify expense has no `projectID` field, it calls `getProjectIDByReport()` which queries four OpenSearch indices to resolve the project ID via slug lookup (`lfx-expense-log`, `projects`, `entities`, `spring-projects`).

The `spring-projects` index is owned and written by the Mentorship service (jobspring), not CF. When OpenSearch is decommissioned, this fallback breaks for all three populations. OpenSearch decommission is therefore gated on Mentorship's K8s migration, independent of CF's timeline.

**Blocking:** OpenSearch decommission.

**Draft message for Lewis:**

> Hi Lewis — quick question about the Expensify webhook in the Ledger service. There's a fallback path in `expensify/main.go`: when an incoming expense has no `projectID`, `getProjectIDByReport()` falls back to querying OpenSearch (`lfx-expense-log`, `projects`, `entities`, `spring-projects`) to resolve the project ID by slug.
>
> Two questions:
> 1. How often does this fallback actually trigger in production? Is `expense.ProjectID` reliably set by Expensify, or does it regularly fall back to the OpenSearch lookup?
> 2. If it does trigger regularly — what's your preferred fix when OpenSearch is eventually decommissioned? Options: (a) confirm Expensify always sends projectID so the fallback is dead code, (b) update the fallback to call the CF API + Mentorship API instead, or (c) accept it as low-frequency and add a Slack alert.
>
> Not blocking anything right now since OpenSearch decommission is gated on RS moving to K8s, but want to understand the scope before we get there.

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
| OQ-21 | Ledger txnCategory ↔ CF goal name mismatch | Not a release blocker. Deferred post-release pending finance team conversation on canonical category mapping. Tracked in [LFXV2-2202](https://linuxfoundation.atlassian.net/browse/LFXV2-2202). |
| OQ-23 | Auth0 sub → LFID username migration | Complete. New CF uses LFID username everywhere from day one. Migration script bulk-resolves Auth0 subs to LFID usernames via Auth0 Management API. `users.legacy_user_id` stores the Auth0 `sub` — set on every profile sync (`PATCH /v1/me`) and used for Ledger user lookups; not migration-only. Tracked in [LFXV2-2025](https://linuxfoundation.atlassian.net/browse/LFXV2-2025). |
| OQ-19 | Ledger API shape for stats-sync | Resolved. `ledger-stats-sync` is fully implemented: uses bulk `GET /balance` (single HTTP call for all projects), fields confirmed (`totalCredit`, `totalDebit`, `totalBalance`, `availableBalance`, `feeBalance`, `backers`, `sponsors`). See `cmd/ledger-stats-sync/` and `internal/infrastructure/clients/ledger.go`. |
| OQ-15 | Ledger balance lookup for post-cutover initiatives | Resolved. Ledger's `project_id` validation regex (`^[0-9a-zA-Z\_\-]+$`) accepts UUIDs. CF already passes `initiative.ID` (Postgres UUID) directly to Ledger API calls. No Ledger code changes required. |
| OQ-11 | Full scope of CF data in LFX Self Serve | Resolved. LFX Self Serve calls the CF Go API directly using a user-issued access token (`access:me`). No Fivetran CF→Snowflake sync required for SS integration. See `docs/authentication-architecture.md` Flow 2. |
| OQ-22 | Feature-branch deployment pattern | Not needed at this time. |
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
