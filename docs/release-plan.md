<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Release Plan — CF Go-Live (Monday 2026-06-15)

**Jira:** [LFXV2-1690](https://linuxfoundation.atlassian.net/browse/LFXV2-1690)

**Old CF:** `https://crowdfunding.lfx.linuxfoundation.org` → forwarded to new CF on release day

**New CF:** `https://crowdfunding.linuxfoundation.org` (K8s + Postgres)

---

## Friday 2026-06-12 — Code, Infra, Stripe

### Michal

| # | Task | Est. | Status |
|---|---|---|---|
| F-M1 | Enable CF Stripe webhook in prod Dashboard (already registered at `https://crowdfunding-api.linuxfoundation.org/v1/hooks/stripe`, secret already in SM — just needs to be enabled) | 15 min | ❌ Pending |
| F-M2 | Stripe Dashboard: confirm subscription retry → "Cancel the subscription" (not "Mark as unpaid") under Settings → Billing → Subscriptions and emails | 15 min | ❓ Unverified |
| F-M3 | Add `mentorship-sync-secrets` to AWS Secrets Manager prod (`snowflake-account`, `snowflake-user`, `snowflake-warehouse`, `snowflake-role`, `snowflake-private-key`) and create the K8s Secret in prod namespace from SM. Then PR to `lfx-v2-argocd`: add `lfx-crowdfunding-mentorship-sync` to `apps/prod/lfx-v2-applications.yaml` (values file already exists at `values/prod/lfx-crowdfunding-mentorship-sync.yaml`) | 1 h | ❌ Pending |
| F-M4 | Confirm prod pods are running (`kubectl get pods -n crowdfunding-backend` and `-n crowdfunding-frontend`) | 10 min | ❓ Self-verify |
| F-M5 | Write migration validation script `db/scripts/validate_migration.py` — scans each DynamoDB source table, counts items, queries Postgres row counts, prints comparison table, exits non-zero if counts are off. Reuses existing `boto3` + `psycopg2` setup. Expected counts in [`backend/docs/rewrite/05-migration-plan.md`](../backend/docs/rewrite/05-migration-plan.md). | 1 h | ❌ Pending |
| F-M6 | Add more e2e tests — priority: initiative creation flow, search/filter, logged-out redirects, error states | 2 h | ❌ Pending |

### Efren

| # | Task | Est. | Status |
|---|---|---|---|
| F-E1 | Create `/payment/complete` page — reads `redirect_status` param from URL, shows success if `succeeded`, error + retry link otherwise. Required for the 3DS redirect fallback path (`STRIPE_RETURN_URL` points here). | 1–2 h | ❌ Missing |

### Slack to DevOps (Robert / Alan) — send today

Send one Slack message now so they can plan their Monday. Ask for:

1. **LFF maintenance mode on Monday morning** — put `crowdfunding.lfx.linuxfoundation.org` into maintenance mode (display maintenance page, block donations) while we run the data migration. Window: ~45–60 min.
2. **URL forward on Monday** — once we confirm migration + smoke test pass, set a forward from `crowdfunding.lfx.linuxfoundation.org` to `https://crowdfunding.linuxfoundation.org`.

---

## Monday 2026-06-15 — Go-Live Sequence

**Suggested window: 8 AM Pacific. Budget ~2 hours.**

| # | Step | Who | Est. |
|---|---|---|---|
| GO1 | Deploy latest CF backend + frontend to prod (update image tags in `lfx-v2-argocd` prod values and sync ArgoCD) | Michal | 15 min |
| GO2 | Update Mandrill API key in 1Password prod to the real key (currently intentionally wrong) | Michal | 10 min |
| GO3 | Ask DevOps to put LFF into maintenance mode (`crowdfunding.lfx.linuxfoundation.org`) | Robert/Alan | 15 min |
| GO4 | Deploy Reimbursement Service to prod | Lewis | 20 min |
| GO5 | Deploy Ledger to prod | Lewis | 20 min |
| GO6 | Run data migration script (`migrate_dynamo_to_postgres.py`) against prod DynamoDB → prod Postgres | Lewis | 30 min |
| GO7 | Run validation script (`validate_migration.py`) — confirm counts match | Lewis + Michal | 10 min |
| GO8 | Manually trigger `ledger-stats-sync` CronJob, verify `amount_raised_in_cents` is populated for a sample of published initiatives | Michal | 15 min |
| GO9 | Ask DevOps to set URL forward from old CF to `https://crowdfunding.linuxfoundation.org` | Robert/Alan | 10 min |
| GO10 | Run manual smoke test (see below) | Efren + Michal | 45 min |
| GO11 | Watch logs for 1 hour: `kubectl logs -n crowdfunding-backend` + Datadog | Michal | 1 h |
| GO12 | **Rollback trigger:** if critical errors — ask DevOps to remove forward and restore old CF. Old DynamoDB untouched. | Robert/Alan | 5 min |

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

## Post-Launch Backlog (not blocking go-live)

| # | Item | Jira | Notes |
|---|---|---|---|
| PL1 | Intercom integration (stretch) | [LFXV2-2197](https://linuxfoundation.atlassian.net/browse/LFXV2-2197) | Not integrated yet. Needs `useIntercom` composable (Nuxt 4), Auth0 custom claim in `auth0-terraform`, Helm env vars. Use `/lfx-intercom` skill. ~1 day. |
| PL2 | AI-generated in-app docs (stretch) | [LFXV2-2198](https://linuxfoundation.atlassian.net/browse/LFXV2-2198) | LFX AI service SSE streaming. Needs agent ID from AI platform team before scoping. Do not start without that conversation. |
| PL3 | E2E test gaps | — | No tests for: actual payment submission, form validation, initiative creation flow, search/filter, logged-out redirects, subscription cancellation. Payment tests require a separate Stripe test-mode deployment. |

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

Bugs and gaps are tracked under [LFXV2-2145](https://linuxfoundation.atlassian.net/browse/LFXV2-2145).

- **OQ-20 (GitHub URL):** GitHub button in initiative header is permanently hidden. No `github_url` column in schema — data lives in `initiative_custom_websites`.

---

## Rollback

Remove the URL forward — old LFF Lambda is still running. Old DynamoDB is untouched (migration is read-only from DynamoDB). Safe at any point. Keep old Lambda running for minimum 2 weeks before decommission.
