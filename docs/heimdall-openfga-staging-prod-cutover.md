<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Heimdall + OpenFGA Cutover Playbook — Staging then Prod

Staging and prod are still on the legacy Ingress with `openfga.enabled: false`.
Dev has been fully cut over since 2026-09-25. This is the replication
playbook for staging, then prod — same steps, do staging first, verify, then
repeat for prod.

Primary source of truth: [#271](https://github.com/linuxfoundation/lfx-crowdfunding/issues/271)
(sequencing plan, still open) and [#277](https://github.com/linuxfoundation/lfx-crowdfunding/issues/277)
(FGA tuple emission/backfill, closed 2026-09-25). Full design docs:
[`backend/docs/rewrite/12-fga-authorization-model.md`](../backend/docs/rewrite/12-fga-authorization-model.md)
and [`docs/authentication-architecture.md`](./authentication-architecture.md).
All values-file/PR edits below happen in `lfx-v2-argocd`, not this repo — this
doc lives here because the decisions and ordering are CF-specific, per the
open question left in #277's rollout comment.

## Precondition check (already true for both staging and prod as of 2026-09-28)

- [x] Chart/image for `lfx-crowdfunding-backend` and `-frontend` at `0.1.31` /
      `v0.1.31` in both `apps/staging|prod/lfx-v2-applications.yaml` and
      `values/staging|prod/*.yaml` (in `lfx-v2-argocd`) — this is the version
      that contains #295 (`fga.Publisher` + `cmd/fga-backfill` + the
      `fgaReconcileCronJob`/`openfga`/`heimdall` chart values). **If either
      environment gets bumped backward, or a fresh env is stood up on an older
      pin, this whole playbook is blocked until it's back on `>=0.1.31`.**
- [x] M2M FGA subject tuples (`user:<client_id>@clients → member →
      team:crowdfunding-services`) already written directly into OpenFGA for
      all 3 client IDs, confirmed live in dev **and** staging **and** prod
      (#281, 2026-09-23). Nothing to do here.
- [x] Reimbursement Service's Auth0 client grant for the `lfx_v2_api` audience
      merged (`auth0-terraform#386`, 2026-09-22). Nothing to do here.
- [x] `app.approversTeamName` (`crowdfunding_approvers`) and
      `app.servicesTeamName` (`crowdfunding-services`) are chart defaults in
      `backend/charts/lfx-crowdfunding-backend/values.yaml` — no per-env
      override needed unless an environment intentionally uses different team
      names (neither does today).

## Step-by-step (per environment, in `lfx-v2-argocd`)

Do these as **separate PRs in this order**, verifying each before the next.
Do not combine steps 1–2 with step 3 in one PR — enabling the CronJob without
having run+verified a backfill, or flipping `openfga.enabled` before the
backfill has run, fails every check closed and locks out every caller
(this is the whole point of #277's ordering).

### 1. Enable `fgaReconcileCronJob`

Edit `values/<env>/lfx-crowdfunding-backend.yaml`, add (pattern from dev,
`values/dev/lfx-crowdfunding-backend.yaml`):

```yaml
fgaReconcileCronJob:
  enabled: true
  image:
    repository: ghcr.io/linuxfoundation/lfx-crowdfunding-backend
    tag: v0.1.31   # pinned tag for staging/prod, not "development"
```

Commit, push, let ArgoCD auto-sync. Confirm the CronJob exists:

```bash
kubectl get cronjob -n crowdfunding-backend   # against the <env> cluster
```

### 2. Run and verify the backfill

Trigger manually rather than waiting for the `"0 7 * * *"` schedule:

```bash
kubectl create job --from=cronjob/lfx-crowdfunding-backend-fga-reconcile \
  fga-reconcile-manual-$(date +%s) -n crowdfunding-backend
```

Check the job log for `total_initiatives`, `published`, `failed: 0`.

**Then verify tuples actually landed in OpenFGA** (don't trust the job log
alone) — port-forward to `lfx-platform-openfga` in the `lfx` namespace
(`OPENFGA_STORE_ID` env var is on the `lfx-v2-fga-sync` Deployment) and hit
`/stores/{id}/read` with a real initiative UID from that env's public
`/crowdfunding/initiatives` endpoint. Expect `owner: user:<username>` and, for
published initiatives, `viewer: user:*`. Note: this OpenFGA version rejects a
type-only filter with an empty object id — you need a real object id.

Re-check the CronJob schedule's message-volume math (`~194k msgs/day` at the
dev-derived 15-minute interval, now defaulted to daily) against staging/prod's
actual initiative counts before assuming the same daily schedule is fine —
only re-tune if either env has a meaningfully different initiative count than
dev's ~74.

**Prod auth caveat:** both the manual job trigger and the direct OpenFGA
`/read` query need cluster access to the prod namespace. Confirm you (or
whoever runs this) actually has prod cluster access before assuming the dev
workflow repeats as-is. If not, either wait for the CronJob's own schedule to
fire (no manual trigger, just a longer wait before verifying) or get someone
with prod access to trigger/verify it.

### 3. Flip `openfga.enabled`

Only after step 2 is verified. Add to the same `values/<env>/lfx-crowdfunding-backend.yaml`:

```yaml
openfga:
  enabled: true
```

This turns on the real `openfga_check` authorizers (`me-initiative-write`,
the announcement rules, `process-approval`) — but has **no effect until
`heimdall.enabled` is also true** (these RuleSet rules only run for traffic
through the Heimdall gateway). It's safe to merge this ahead of step 4 if you
want to decouple the two changes; the flip is inert until the gateway cutover
below.

### 4. Heimdall gateway cutover (backend + frontend together)

This is the actual cutover — unlike steps 1–3, this is not gradual. Per
`heimdall.enabled`'s authenticator design (`oauth2_introspection` → `openfga_check`
→ `create_jwt` pipeline), a caller presenting the old Auth0 audience fails at
the gateway's own token introspection, not at FGA, the instant this flips —
there's no dual-accept window available at the gateway layer. `validate.yaml`
also hard-fails the chart render if `ingress.enabled` and `heimdall.enabled`
are both true, so there is never a state with both routes live.

**Backend** — `values/<env>/lfx-crowdfunding-backend.yaml` (pattern from
`values/dev/lfx-crowdfunding-backend.yaml`):

```yaml
heimdall:
  enabled: true
  add_middleware: true

ingress:
  enabled: false   # mutually exclusive with heimdall.enabled — validate.yaml enforces this
```

**Frontend** — `values/<env>/lfx-crowdfunding-frontend.yaml`, in the **same
commit/PR** (frontend has no dual-accept; flipping the audience alone 401s
every route until `NUXT_API_BASE_URL` also repoints):

```yaml
config:
  NUXT_API_BASE_URL: "https://lfx-api.<env>.v2.cluster.linuxfound.info"   # was the internal Service URL
  NUXT_PUBLIC_AUTH0_AUDIENCE: "https://lfx-api.<env>.v2.cluster.linuxfound.info/"   # was the crowdfunding-api.* audience
```

Both staging and prod already have their own `NUXT_API_BASE_URL` override
slot (added in argocd#1629) — this step is a one-line value change per file,
not a new key.

No 2-week dual-accept window — this was evaluated and dropped for CF
(#271, 2026-09-22 edit): RS has no session to protect (cron-driven, just
re-requests a token), and CF's session volume doesn't need staggering. Expect
one clean cutover and one burst of re-logins right after deploy, in staging
and in prod alike.

### 5. Post-cutover verification

From outside the cluster (same checks used for dev's argocd#1642):

- `GET /crowdfunding/statistics` and `/crowdfunding/initiatives` → 200
  through the new gateway host.
- `GET /crowdfunding/me/initiatives` with no/invalid token → 401.
- Frontend loads and a real login round-trips (session cookie, redirect).
- The old ingress host no longer responds (confirms `ingress.enabled: false`
  actually took effect, not just that the new path works).
- End-to-end FGA check: a gated route denies a non-owner/non-writer and
  allows the owner/approver, through the gateway (not just via `openfga.enabled`
  being set — confirm the RuleSet is actually being hit).
- Reimbursement Service (M2M) call against `owner-info`/`published-list`
  succeeds — this is the caller #281/#286 specifically unblocked; the
  `openfga.enabled: false` branch of that rule is `deny_all`, not `allow_all`,
  so this is the one path that was still broken even with tuples and grants
  in place until `openfga.enabled` is actually true.

## Rollback

Same as any GitOps rollback — revert the values PR(s) in `lfx-v2-argocd`.
`selfHeal: true` will restore the previous state on the next sync; do not
edit cluster resources by hand. Rolling back `heimdall.enabled`/`ingress.enabled`
alone is the fast path back to today's behavior; `openfga.enabled` and
`fgaReconcileCronJob.enabled` can stay on independently since they're inert
without the gateway.

## Open items not blocking this playbook

- [lfx-v2-argocd#1637](https://github.com/linuxfoundation/lfx-v2-argocd/pull/1637)
  (open, unmerged): staging `ALLOWED_APPROVERS` username fix — unrelated to
  this cutover but worth merging first so approval-flow testing in step 5
  uses correct approver accounts.
- #271 remains open on this repo as the tracking issue; comment there (or
  update this doc) after staging and after prod, following the pattern of the
  existing dev-completion comments.
