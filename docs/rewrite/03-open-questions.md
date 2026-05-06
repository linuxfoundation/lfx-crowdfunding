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
- CF runs a K8s CronJob (`mentorship-sync`) that queries Snowflake for mentorship programs and creates/updates `campaign_type = mentorship` rows in the CF Postgres DB.
- Mentorship continues calling CF API endpoints directly (slug sync, funding status, title-check, add/remove beneficiary). CF returns 404 gracefully when a program is not yet synced — Mentorship must handle this gracefully.
- CF no longer publishes SNS events back to Mentorship (those were only needed to keep the old bidirectional sync working).

**Mentorship changes required:**
- Handle 404 responses gracefully on: `GET /v1/projects/{id}/funding`, `POST /v1/entities/{id}/addbeneficiary`, `POST /v1/entities/{id}/removebeneficiary`. Treat 404 as "not synced yet" — retry or queue for later rather than failing hard.

---

### OQ-4: GitHub repo `linuxfoundation/lfx-crowdfunding` — created and visibility confirmed?

**Question:** Jira task MENV2-1622 was to create the GitHub repo. Has it been created? Is visibility public or private?

**Owner:** DevOps
**Status:** Open (as of May 2026)
**Notes:** Repo is needed before any code can be pushed. Branch protection on `main` is expected — first PR will need to be from a feature branch.

---

### OQ-5: ArgoCD app for Crowdfunding K8s deployment

**Question:** A new ArgoCD application needs to be created in `linuxfoundation/lfx-v2-argocd` for the Crowdfunding K8s service. What namespace, cluster, and values structure should be used? Who creates it?

**Owner:** DevOps
**Status:** Open
**Notes:** Check existing ArgoCD apps (e.g., Insights) for the expected structure. The `lfx-v2-argocd/apps/dev`, `apps/staging`, `apps/prod` directories are the target.

---

## Data & Stripe

### OQ-6: Hardcoded Stripe Plan IDs / Product IDs outside DynamoDB?

**Question:** Are there any Stripe Plan IDs or Product IDs hardcoded in configuration files, SSM parameters, Terraform, or other systems outside the DynamoDB tables?

**Owner:** Michal
**Status:** Open — needs DynamoDB query
**Resolution steps:** Run the following queries against prod DynamoDB to find all Stripe IDs stored in the live data:

```bash
# Find all projects with Stripe Plan or Product IDs
aws dynamodb scan \
  --table-name lff-prod-projects \
  --filter-expression "attribute_exists(planId) OR attribute_exists(productId)" \
  --projection-expression "projectId, #n, planId, productId" \
  --expression-attribute-names '{"#n":"name"}' \
  --region us-east-1 \
  --output json | jq '.Items[] | {projectId: .projectId.S, name: .name.S, planId: .planId.S, productId: .productId.S}'

# Count active subscriptions with Stripe subscription IDs
aws dynamodb scan \
  --table-name lff-prod-subscriptions \
  --filter-expression "attribute_exists(stripeSubscriptionId)" \
  --projection-expression "projectId, userId, stripeSubscriptionId, #s" \
  --expression-attribute-names '{"#s":"status"}' \
  --region us-east-1 \
  --output json | jq '.Items[] | {projectId: .projectId.S, userId: .userId.S, stripeSubscriptionId: .stripeSubscriptionId.S, status: .status.S}'

# Count total donations
aws dynamodb scan \
  --table-name lff-prod-donations \
  --filter-expression "attribute_exists(stripeChargeId)" \
  --projection-expression "projectId, stripeChargeId" \
  --region us-east-1 \
  --output json | jq '.Count'
```

**Notes:** Stripe Plan IDs and Product IDs on live subscriptions must be preserved during migration — active Stripe subscriptions continue charging against the original plan. These IDs must be migrated as-is to the new Postgres `subscriptions` table.

---

## Dependencies

### OQ-7: Reimbursement Service OpenSearch dependency — long-term plan

**Status:** Resolved
**Owner:** Michal

**Resolution:** Two-phase migration with hard decommission deadline of CF release + 2 weeks.

**Phase 1 — on CF release day:**
RS switches Category 1 reads (CF-owned data) from OpenSearch to direct SQL on CF Postgres:
- `projects` OpenSearch index → `SELECT id, name, owner_id FROM crowdfunding.projects WHERE status = 'published'`
- `entities` OpenSearch index → `SELECT id, name, owner_id FROM crowdfunding.funds WHERE status = 'published'`
- `lff-users` OpenSearch index → `SELECT owner_id, email FROM crowdfunding.users`

RS gets a read-only Postgres role on `crowdfunding` schema. No HTTP API layer between RS and CF — direct DB connection. These are the reads used by `RefreshTags()` (Expensify GL code sync) and `getEmailBySlug()` (project owner email lookup).

**Phase 2 — CF release + 2 weeks (hard deadline):**
RS migrates Category 2 indices (RS-owned data) from OpenSearch to the `reimbursement` schema on the CF Postgres instance:

| OpenSearch index | Replacement | Data |
|---|---|---|
| `lfx-expense-log` | `reimbursement.expense_log` table | Email send idempotency records |
| `beneficiary-actions` | `reimbursement.beneficiary_actions` table | Pending add/remove work queue |
| `travel-funds-tickets` | `reimbursement.travel_fund_tickets` table | Ticket→beneficiary mapping |

All three are simple key-value access patterns (get by ID, get by action type) — no full-text search. Schema is ~65 lines SQL, implementation ~100 lines Go replacing `ElasticRepository` calls with SQL queries.

OpenSearch decommissions on CF release + 2 weeks. Not "when we get to it."

**Notes:**
- RS tables live on the CF Postgres instance under the `reimbursement` schema — same connection, separate schema, separate role
- `beneficiary-actions` remove path is fully commented out in RS (`PolicyEmployeesRemove` at service.go:919) — migration of this index is low risk
- Existing `lfx-expense-log` data must be migrated before cutover to preserve email deduplication history
- Old CF Lambda has zero OpenSearch writes — verify who populates `projects`/`entities` OpenSearch indices today before Phase 1 cutover

---

### OQ-8: Auth0 allowed origins/callbacks for new service URLs

**Question:** The new Crowdfunding Nuxt frontend and Go API will have new internal/staging URLs before cutover to the production domain. Auth0 must be configured with:
- New allowed callback URLs (for PKCE flow)
- New allowed CORS origins
- New allowed logout URLs

Who manages Auth0 tenant configuration? Does it require a Terraform change or manual update?

**Owner:** DevOps / Auth0 owner
**Status:** Open
**Notes:** Must be resolved before the frontend can authenticate in dev/staging environments.

---

### OQ-9: Mentorship → CF direct HTTP calls — will they work post-cutover?

**Question:** Mentorship (Lambda) calls the CF API directly via `FUNDING_API_URL` for five endpoints (slug sync, funding status, title-check, addbeneficiary, removebeneficiary). Today that URL points at the Lambda API Gateway. After cutover it must point at the new K8s Ingress.

Two sub-questions:
1. **DNS cutover is sufficient:** The plan is to keep the same public URLs (`https://api.crowdfunding.lfx.linuxfoundation.org/`) and flip DNS from Lambda API Gateway to K8s Ingress. If Mentorship uses this URL, no config change is needed on their side. **Confirm Mentorship's `FUNDING_API_URL` uses the production domain, not a Lambda-specific URL.**
2. **Network path:** Can a Mentorship Lambda reach the K8s Ingress over public HTTPS? If the K8s Ingress is public (same as the frontend URL), this should work. If it is internal-only, Mentorship Lambda cannot reach it and a separate public endpoint or VPC peering is needed.

**Owner:** Architect / Mentorship team
**Status:** Open
**Notes:** If both above are confirmed (Mentorship uses prod domain URL + K8s Ingress is publicly reachable), this question resolves itself at DNS cutover with no extra work. If not, it must be resolved before cutover or Mentorship's addbeneficiary, removebeneficiary, and funding status calls will break silently.

---

## Design

### OQ-10: UI prototype — final design or rough reference?

**Status:** Resolved
**Owner:** Michal / Design

**Resolution:** The prototype is a rough reference only. The UI designer is still finalizing the design. Implement functionally using PrimeVue components — do not spend time on pixel-perfect matching against the prototype. UI will be updated once final designs are delivered.

---

## Resolved Questions (for reference)

| # | Question | Resolution |
|---|---|---|
| R-1 | Does LFF write directly to Ledger DB? | No — LFF calls Ledger HTTP API read-only. Ledger writes come from its own Stripe/Expensify webhooks. |
| OQ-1 | Can K8s reach Lambda API Gateway endpoints? | Yes — both reachable over public HTTPS. CF only calls RS for expense approval/rejection; project sync stays on old Lambda. |
| OQ-3 | Mentorship → CF sync mechanism | SNS/SQS dropped. CF pulls from Snowflake via CronJob (24h delay acceptable). Mentorship handles CF 404s gracefully. |
| OQ-4 | GitHub repo created? | Yes — `linuxfoundation/lfx-crowdfunding` created (private, going public soon). |
| OQ-5 | ArgoCD namespace for CF K8s deployment | `crowdfunding` namespace. Helm chart in `charts/lfx-crowdfunding/` in the CF repo; ArgoCD entry in `lfx-v2-applications.yaml`. |
| OQ-10 | UI prototype fidelity | Rough reference only. Implement functionally with PrimeVue; update once designer delivers final designs. |
| R-2 | Does Reimbursement Service query Crowdfunding OpenSearch? | Yes — reads `projects`, `entities`, `lff-users`, `spring-projects`, `spring-users`, `beneficiary-actions`, `travel-funds-tickets`. Writes `lfx-expense-log`, `beneficiary-actions`, `travel-funds-tickets`. Migration plan in OQ-7. |
| R-3 | Who owns the Mentorship SNS topic? | Mentorship (jobspring) owns it. CF is a subscriber. Topic: `lfx-topic-{stage}-project`. CF queue: `fundspring-lfx-queues-{stage}-project`. |
| R-4 | Is there a separate admin UI for project approvals? | No. Approvals are done via JWT links in emails sent to admin (Sriji). |
| R-5 | Should Expensify sync be rewritten for initial release? | No. Keep old Lambda running it. Not end-user visible. Reimbursement Service unchanged. |
| R-6 | Is lfx-v1-sync-helper useful for CF DB migration? | No. It syncs project/committee metadata via NATS KV. Does not touch CF donations, subscriptions, or Ledger data. |
