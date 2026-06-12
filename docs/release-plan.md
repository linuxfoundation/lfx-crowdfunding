<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Release Plan — CF Go-Live (Monday 2026-06-16)

**Today:** Friday 2026-06-13. No staging. Two working days.

**Jira:** [LFXV2-1690](https://linuxfoundation.atlassian.net/browse/LFXV2-1690)

**Old CF:** `https://crowdfunding.lfx.linuxfoundation.org` → forwarded to new CF on release day

**New CF:** `https://crowdfunding.linuxfoundation.org` (K8s + Postgres)

---

## Friday 2026-06-13 — Code, Infra, Stripe

### Michal

| # | Task | Est. | Status |
|---|---|---|---|
| F-M1 | Enable CF Stripe webhook in prod Dashboard (already registered at `https://crowdfunding-api.linuxfoundation.org/v1/hooks/stripe`, secret already in SM — just needs to be enabled) | 15 min | ❌ Pending |
| F-M2 | Stripe Dashboard: confirm subscription retry → "Cancel the subscription" (not "Mark as unpaid") under Settings → Billing → Subscriptions and emails | 15 min | ❓ Unverified |
| F-M3 | Add `mentorship-sync-secrets` to AWS Secrets Manager prod with keys: `snowflake-account`, `snowflake-user`, `snowflake-warehouse`, `snowflake-role`, `snowflake-private-key` | 30 min | ❌ Pending |
| F-M4 | Create `mentorship-sync-secrets` K8s Secret in prod namespace from SM | 15 min | ❌ Pending |
| F-M5 | Merge `feat/mentorship-sync` PR (self-approve — code complete, Dockerfile, tests all present) | 15 min | ❌ Pending |
| F-M6 | PR to `lfx-v2-argocd`: add `lfx-crowdfunding-mentorship-sync` to `apps/prod/lfx-v2-applications.yaml` (values file already exists at `values/prod/lfx-crowdfunding-mentorship-sync.yaml`) | 30 min | ❌ Pending |
| F-M7 | Confirm prod pods are running (`kubectl get pods -n crowdfunding-backend` and `-n crowdfunding-frontend`) | 10 min | ❓ Self-verify |
| F-M8 | Verify `ledger-stats-sync` CronJob is deployed and has run at least once (`kubectl get cronjob -n crowdfunding-backend`, `kubectl get jobs -n crowdfunding-backend`) | 10 min | ❓ Self-verify |
| F-M9 | Update Mandrill API key in 1Password prod to the real key (currently intentionally wrong to prevent accidental sends) | 10 min | ❌ Pending — do this on Monday morning, not before |
| F-M10 | Write migration validation script (`db/scripts/validate_migration.py`) — scans DynamoDB tables, counts rows, compares to Postgres. ~1 h. See note below. | 1 h | ❌ Pending |

### Efren

| # | Task | Est. | Status |
|---|---|---|---|
| F-E1 | Create `/payment/complete` page — reads `redirect_status` param from URL, shows success if `succeeded`, error + retry link otherwise. Required for the 3DS redirect fallback path (`STRIPE_RETURN_URL` points here). | 1–2 h | ❌ Missing |

### Slack to DevOps (Robert / Alan) — send today

Send one Slack message now so they can plan their Monday. Ask for:

1. **LFF maintenance mode on Monday morning** — put `crowdfunding.lfx.linuxfoundation.org` into maintenance mode (display maintenance page, block donations) while we run the data migration. Window: ~45–60 min.
2. **URL forward on Monday** — once we confirm migration + smoke test pass, set a forward from `crowdfunding.lfx.linuxfoundation.org` to `https://crowdfunding.linuxfoundation.org`. Old URL no longer needs to be live after that.
3. **Reimbursement Service + Ledger deploy on Monday** — we need both deployed to prod on Monday as part of go-live. Confirm they are available and can do this.

---

## Migration Validation Script (F-M10)

The migration script already logs per-table upsert counts, but there's no post-run reconciliation against DynamoDB. A validation script is worth ~1 h to write — it reuses the existing `boto3` + `psycopg2` setup in `migrate_dynamo_to_postgres.py`.

**What it does:** scans each DynamoDB source table, counts items, queries Postgres row counts, prints a comparison table, exits non-zero if any count is off.

**Tables to compare:**

| DynamoDB table | Postgres table | Notes |
|---|---|---|
| `lff-prod-users` | `crowdfunding.users` | +58 placeholders expected |
| `lff-prod-organizations` | `crowdfunding.organizations` | |
| `lff-prod-projects` + `lff-prod-entities` | `crowdfunding.initiatives` | Combined; 1 donation skipped (orphaned FK) |
| `lff-prod-donations` + `lff-prod-entity-donations` | `crowdfunding.donations` | |
| `lff-prod-subscriptions` + `lff-prod-entity-subscriptions` | `crowdfunding.subscriptions` | |

Expected counts are in [`backend/docs/rewrite/05-migration-plan.md`](../backend/docs/rewrite/05-migration-plan.md).

---

## Monday 2026-06-16 — Go-Live Sequence

**Suggested window: 8 AM Pacific. Budget ~2 hours.**

| # | Step | Who | Est. |
|---|---|---|---|
| GO1 | Update Mandrill API key in 1Password prod to the real key (F-M9) | Michal | 10 min |
| GO2 | Ask DevOps to put LFF into maintenance mode (`crowdfunding.lfx.linuxfoundation.org`) | Robert/Alan | 15 min |
| GO3 | Deploy Reimbursement Service to prod | Lewis / DevOps | 20 min |
| GO4 | Deploy Ledger to prod | Lewis / DevOps | 20 min |
| GO5 | Run data migration script (`migrate_dynamo_to_postgres.py`) against prod DynamoDB → prod Postgres | Lewis | 30 min |
| GO6 | Run validation script (`validate_migration.py`) — confirm counts match | Lewis + Michal | 10 min |
| GO7 | Manually trigger `ledger-stats-sync` CronJob, verify `amount_raised_in_cents` is populated for a sample of published initiatives | Michal | 15 min |
| GO8 | Ask DevOps to set URL forward from old CF to `https://crowdfunding.linuxfoundation.org` | Robert/Alan | 10 min |
| GO9 | Run manual smoke test (see below) | Efren + Michal | 45 min |
| GO10 | Watch logs for 1 hour: `kubectl logs -n crowdfunding-backend` + Datadog | Michal | 1 h |
| GO11 | **Rollback trigger:** if critical errors — ask DevOps to remove forward and restore old CF. Old DynamoDB untouched. | Robert/Alan | 5 min |

---

## Manual Smoke Test (run after GO8 — forward is live)

The existing Playwright e2e tests use a mock auth bypass which must never be enabled in prod. The smoke test is a **manual checklist**.

| # | Test | How |
|---|---|---|
| SM1 | Browse initiatives list | Open `https://crowdfunding.linuxfoundation.org` — verify list loads, amounts show (not $0) |
| SM2 | Login | Click login, complete Auth0 flow, verify user menu appears |
| SM3 | Create a test initiative | Create a General Fund, fill details, submit — verify approval email arrives in Mandrill |
| SM4 | Approve initiative via email link | Click HMAC link in the approval email — verify initiative status changes to published |
| SM5 | Make a real donation (small amount) | Donate $1 to a test initiative — verify success state in UI and confirmation email arrives |
| SM6 | Subscription creation | Subscribe $1/month — verify Stripe subscription created, DB row shows `active` |
| SM7 | `/payment/complete` page exists | Visit `https://crowdfunding.linuxfoundation.org/payment/complete?redirect_status=succeeded` — verify page renders success (not 404) |
| SM8 | Mentorship initiative visible | Navigate to a known mentorship initiative — verify it loads | 
| SM9 | Financial tab | Open an initiative's financial tab — verify transactions load from Ledger |

**Note on 3DS:** The inline 3DS pop-up path (`confirmCardPayment`) will be exercised by the real $1 donation in SM5. The redirect fallback path is covered by SM7. Real Stripe test cards only work in test mode — not usable in prod.

---

## E2E Tests — Current Coverage and Gaps

Tests run locally against a live dev server with mock auth (`NUXT_E2E_TEST_MODE=true`). **Cannot run against prod** — mock auth bypass must never be enabled there.

### What exists

| File | Coverage |
|---|---|
| `initiatives.spec.ts` | List page loads, search visible, cards/empty state |
| `initiative-crud.spec.ts` | Detail page loads, fundraise page loads, non-owner can't see edit controls |
| `donate.spec.ts` | Donate button visible, donation form opens |
| `subscribe.spec.ts` | Subscribe button visible, my subscriptions page loads |
| `statistics.spec.ts` | Statistics page loads, labels/containers visible |

### Gaps (post-release backlog)

- No test for actual payment submission (requires Stripe test mode — needs a separate test-mode deployment or Stripe mock)
- No form validation / error state tests
- No initiative creation flow (submit → approval email)
- No search/filter interaction tests
- No logged-out user flow (redirects, locked actions)
- No subscription cancellation or payment method update

These are all post-launch backlog items — not blocking Monday.

---

## Post-Launch Backlog (not blocking go-live)

### Intercom integration (stretch)

Not integrated yet. Zero Intercom references in the codebase.

The `lfx-intercom` skill covers the canonical LFX pattern (JWT pre-set, `boot()`/`shutdown()`, dev app ID `mxl90k6y`, prod `w29sqomy`). However the skill and all existing implementations are Angular-based. For Nuxt 4 this needs a composable-based rewrite — no existing template.

**What's needed:**
1. Auth0 `custom_claims.js` update in `auth0-terraform` to add the `http://lfx.dev/claims/intercom` claim (shared work across LFX apps)
2. A `useIntercom` composable in the CF frontend (boot on mount, shutdown on logout, identity on login)
3. Intercom app IDs in the Helm values (`NUXT_PUBLIC_INTERCOM_APP_ID`)

Estimate: ~1 day. Use `/lfx-intercom` skill when picking this up.

---

### AI-generated in-app docs

Same feature as in LFX Self Serve. The LFX AI service (`LFX_AI_BASE_URL`) exposes streaming agent runs via SSE (`POST /v1/agents/{agentId}/runs`). The PCC implementation is backend-driven (NestJS), not frontend.

**What's needed for CF:**
1. Understand which agent ID covers CF documentation (ask the AI platform team)
2. A Nuxt server route that proxies the SSE stream with an M2M token (`LFX_AI_AUTH_TOKEN`)
3. A UI component to trigger and display the streamed output

Not well-defined yet — needs a conversation with the AI platform team before scoping. Do not start implementation without that.

---

## Confirmed — No Action Needed

- ✅ AWS Secrets Manager (`lfx-crowdfunding-backend-secrets`) — provisioned and verified
- ✅ DNS — `crowdfunding.linuxfoundation.org` + `crowdfunding-api.linuxfoundation.org` verified
- ✅ `DISABLED_MOCK_LOCAL_PRINCIPAL` — not set in any CF ArgoCD values
- ✅ CF Stripe webhook endpoint registered in prod Dashboard (needs enabling — F-M1)
- ✅ `STRIPE_RETURN_URL` set to `https://crowdfunding.linuxfoundation.org/payment/complete` in prod values
- ✅ ArgoCD values files exist for mentorship-sync in dev and prod
- ✅ Ledger auth fix deployed
- ✅ Ledger balance lookup for post-cutover initiatives confirmed (OQ-15 resolved)

---

## Known Gaps at Launch

- **OQ-20 (GitHub URL):** GitHub button in initiative header is permanently hidden. No `github_url` column in schema — data lives in `initiative_custom_websites`. Not a blocker — tracked in open questions.

---

## Rollback

Remove the URL forward — old LFF Lambda is still running. Old DynamoDB is untouched (migration is read-only from DynamoDB). Safe at any point. Keep old Lambda running for minimum 2 weeks before decommission.
