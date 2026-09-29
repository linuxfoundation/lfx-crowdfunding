<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Heimdall + OpenFGA Cutover Playbook — Staging then Prod

Staging and prod are still on the legacy Ingress with `openfga.enabled: false`.
Dev has been cut over since 2026-09-25, but **dev's own step 3 was never run**
(`openfga.enabled` shipped in argocd#1660 with `team:crowdfunding_approvers`
still unseeded) — dev's approve/decline calls are almost certainly 403ing
today. Do not treat dev as a working reference; re-run step 3 in dev first,
and use "dev has approvals, RS, Self Serve, and Ledger all passing
post-cutover verification (step 6)" as the actual gate before starting
staging, not just "dev is cut over." This is the replication playbook for
staging, then prod — same steps, do staging first, verify, then repeat for
prod.

## Per-environment hosts and audiences

| Env | Legacy host (Ingress) | Gateway host (Heimdall) | Auth0 `lfx_v2_api` audience |
| --- | --- | --- | --- |
| dev | `crowdfunding-api.dev.lfx.dev` | `lfx-api.dev.v2.cluster.linuxfound.info` | `https://lfx-api.dev.v2.cluster.linuxfound.info/` |
| staging | `crowdfunding-api.staging.lfx.dev` | `lfx-api.staging.v2.cluster.linuxfound.info` | `https://lfx-api.staging.v2.cluster.linuxfound.info/` |
| prod | `crowdfunding-api.linuxfoundation.org` | `lfx-api.v2.cluster.lfx.dev` (not `.prod.` — matches `auth0-terraform#resource_servers.tf`'s `lfx_v2_api` identifier and the existing `lfx-v2-argocd/values/prod/lfx-platform.yaml` gateway host) | `https://lfx-api.v2.cluster.lfx.dev/` |

Every `<env>.../` placeholder below resolves against this table — prod is the
one environment where the pattern breaks (no `.prod.` segment).

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

## Step-by-step (per environment, in `lfx-v2-argocd` unless noted)

Do these as **separate PRs in this order**, verifying each before the next.
Do not combine steps 1–2 with step 4 in one PR — enabling the CronJob without
having run+verified a backfill, or flipping `openfga.enabled` before the
backfill has run, fails every check closed and locks out every caller
(this is the whole point of #277's ordering). Step 3 is a separate, easy-to-miss
blocker on the `process-approval` route specifically — see below.

**Steps 1–2 can run well ahead of steps 4–5, in prod especially.** The
reconcile job only publishes to fga-sync — it has no dependency on
`openfga.enabled` or `heimdall.enabled` and doesn't touch the gateway or auth
path. If your prod cutover window is scheduled for daytime hours when most of
devops is asleep, merge step 1 for prod early and let the daily
`0 7 * * *` UTC schedule fire and get verified on its own timeline, so step 2
is already done (or just needs re-verifying) by the time you start the rest
of the playbook — instead of needing an on-demand trigger during the live
window. (`kubectl create job --from=cronjob/...` also isn't guaranteed to
work ad hoc — it needs `jobs.batch` create RBAC in that namespace, which
`sso-power-user` doesn't have as of 2026-09-29.)

### 1. Enable `fgaReconcileCronJob`

Edit `values/<env>/lfx-crowdfunding-backend.yaml`, add (pattern from dev,
`values/dev/lfx-crowdfunding-backend.yaml`):

```yaml
fgaReconcileCronJob:
  enabled: true
```

Don't add an explicit `image:` block — `cronjob-fga-reconcile.yaml`'s
template already falls back to the top-level `.Values.image.tag` (same tag
the backend Deployment runs), so a pinned tag here would just fall behind on
the next chart bump.

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
alone — `published`/`failed` only track whether the fire-and-forget NATS
publish succeeded, not whether fga-sync actually consumed and wrote the
tuple; a per-row consumer failure downstream can still leave an initiative
without tuples). Compare the complete Postgres-derived expected tuple set —
every initiative's `owner`, `viewer: user:*` for published ones, and for
attributed initiatives its `project` or `b2b_org` reference tuple — against
what's actually in OpenFGA before moving to step 4. Spot-checking a single
initiative's owner/viewer tuples is not sufficient: it would miss both a
stalled consumer on another initiative and an omitted attribution reference.
Port-forward to `lfx-platform-openfga` in the `lfx` namespace
(`OPENFGA_STORE_ID` env var is on the `lfx-v2-fga-sync` Deployment) to query
`/stores/{id}/read`; note this OpenFGA version rejects a type-only filter
with an empty object id — you need real object ids.

Re-check the CronJob's message volume before assuming the same schedule is
fine: one reconcile run emits one NATS update per initiative, so daily
messages = `initiative_count × runs_per_day` (1 run/day on the default daily
schedule, 96 on the dev-era 15-minute interval). Dev's ~74 initiatives is
~74 msgs/day on today's daily schedule, or ~7.1k/day at a 15-minute interval
— the `~194k msgs/day` figure some earlier notes cite is that same 15-minute
interval applied to prod's ~2,023 initiatives, not a default threshold.
Compute this for staging/prod's actual initiative counts before assuming no
re-tuning is needed.

**Prod auth caveat:** both the manual job trigger and the direct OpenFGA
`/read` query need cluster access to the prod namespace. Confirm you (or
whoever runs this) actually has prod cluster access before assuming the dev
workflow repeats as-is. If not, either wait for the CronJob's own schedule to
fire (no manual trigger, just a longer wait before verifying) or get someone
with prod access to trigger/verify it.

### 3. Seed `team:crowdfunding_approvers` — blocker, not optional

**This step is not covered by #295/#277 and has never been done in any
environment.** Without it, flipping `openfga.enabled` denies every approve/
decline call, in every env, with no fallback — this is a real regression, not
a theoretical one.

Why: the `process-approval` rule in `ruleset.yaml` does not check anything
CF's backend emits. It checks `relation: member` on a separate, global
`team:crowdfunding_approvers` object (per `backend/docs/rewrite/12-fga-authorization-model.md`,
"Approvers as a global team grant"). Doc 12's Phase 1 called for `isApprover`
(`backend/internal/handler/initiative_handler.go:598`) to check that team via
FGA, falling back to the `ALLOWED_APPROVERS` env var — but that Phase 1 was
never implemented. `isApprover` today still only checks `ALLOWED_APPROVERS`
directly, and no setup script for `team:crowdfunding_approvers` was ever
written (the only team-seeding PR that shipped, #285, was for the unrelated
`team:crowdfunding-services` M2M grant). So `team:crowdfunding_approvers`
almost certainly has **zero member tuples** in every environment today.

Once `openfga.enabled` is true, `process-approval` stops being `allow_all`
and starts requiring FGA team membership — checked at the gateway, before the
request ever reaches `isApprover`'s in-app fallback. An empty team means
every approver is denied, `ALLOWED_APPROVERS` included.

**Before step 4 in any environment:**

1. Write (or adapt) a setup script analogous to
   `lfx-v2-member-service/scripts/setup-global-org-admin-team.sh` — read
   existing `team:crowdfunding_approvers` members, diff against that env's
   current `ALLOWED_APPROVERS` list, write what's missing. Use the plain
   LFID username (no `auth0|` prefix) as the tuple subject — getting this
   wrong silently 403s every approver.
2. Run it against that environment's OpenFGA store (same pattern as verifying
   the backfill in step 2 — a direct `/write` this time, not a `/read`).
3. No fga-sync cache to bust for this one: `process-approval`'s
   `openfga_check` (`lfx-v2-helm/charts/lfx-platform/values.yaml`) calls
   OpenFGA's `/check` endpoint directly from Heimdall, not through fga-sync,
   so a direct OpenFGA write is visible immediately.
4. Verify with an OpenFGA `/read` for `team:crowdfunding_approvers` — confirm
   every username currently in that env's `ALLOWED_APPROVERS` (fix
   [lfx-v2-argocd#1637](https://github.com/linuxfoundation/lfx-v2-argocd/pull/1637)
   first for staging, so you're seeding against the corrected list, not the
   typo'd one) shows up as a `member` tuple.
5. Smoke-test one real approve/decline call against that env, as a known
   approver, before moving to step 4.

### 4. Flip `openfga.enabled`

Only after steps 2 and 3 are both verified. Add to the same
`values/<env>/lfx-crowdfunding-backend.yaml`:

```yaml
openfga:
  enabled: true
```

This turns on the real `openfga_check` authorizers (`me-initiative-write`,
the announcement rules, `process-approval`) — but has **no effect until
`heimdall.enabled` is also true** (these RuleSet rules only run for traffic
through the Heimdall gateway). It's safe to merge this ahead of step 5 if you
want to decouple the two changes; the flip is inert until the gateway cutover
below.

### 5. Heimdall gateway cutover (backend, frontend, RS, Ledger, Self Serve, Stripe — all synchronized)

This is the actual cutover — unlike steps 1–3, this is not gradual. Per
`heimdall.enabled`'s authenticator design (`oidc` → `openfga_check` →
`create_jwt` pipeline, `ruleset.yaml`), a caller presenting the old Auth0
audience fails at the gateway's own JWT validation, not at FGA, the instant
this flips. `validate.yaml` (#282) also hard-fails the chart render if
`ingress.enabled` and `heimdall.enabled` are both true, so there is never a
state with both routes live at once — this is a deliberate choice, not a
gateway limitation: the backend itself (v0.1.31) already accepts both Auth0
and Heimdall JWTs on both `/v1` and `/crowdfunding`, so a temporary dual-route
window (Ingress + HTTPRoute both live, each caller repointing on its own
schedule) is technically possible and would make rollback simpler. We're
keeping the synchronized, all-at-once cutover here to preserve #282's
gateway-only guarantee (every request provably passes through Heimdall's
RuleSet once cut over, with no window where a caller could still be reaching
the backend directly) — if that trade-off should be revisited, raise it with
the reviewers on #282 rather than deciding it in this playbook.

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
  NUXT_API_BASE_URL: "<gateway host, from the table above>"   # was the internal Service URL
  NUXT_PUBLIC_AUTH0_AUDIENCE: "<gateway host, from the table above>/"   # was the crowdfunding-api.* audience
```

Both staging and prod already have their own `NUXT_API_BASE_URL` override
slot (added in argocd#1629) — this step is a one-line value change per file,
not a new key. Remember prod's gateway host is `lfx-api.v2.cluster.lfx.dev`,
not `lfx-api.prod.v2.cluster.linuxfound.info`.

No 2-week dual-accept window — this was evaluated and dropped for CF
(#271, 2026-09-22 edit): RS has no session to protect (cron-driven, just
re-requests a token), and CF's session volume doesn't need staggering. Expect
one clean cutover and one burst of re-logins right after deploy, in staging
and in prod alike.

**Four more callers must repoint in the same window** (#271 rows 7b/7f,
tracked in [#286](https://github.com/linuxfoundation/lfx-crowdfunding/issues/286)
and [#292](https://github.com/linuxfoundation/lfx-crowdfunding/issues/292)).
None of these have a safe window on either side of the flip — the old host
stops routing the instant `ingress.enabled` goes false for that env, so each
must be pre-staged (PR written + approved ahead of time) and merged/applied
in the same deploy window as the backend/frontend values above, not before
and not after:

- **Reimbursement Service** (`reimbursement-service` repo) — one PR per
  environment (not one combined PR: RS's CI deploys dev on every push to
  `master`, and staging+prod together on the same tag, so a shared PR would
  force environments to cut over together and block unrelated RS releases).
  Repoints `serverless.yml`'s `cfAudience`/`cfAPIURL` (and prod's
  `cfAPIPathPrefix`, which is `/v1` for prod today vs. `/crowdfunding` for
  dev/staging) from the legacy host to that env's `lfx_v2_api` audience and
  Heimdall gateway host — same pattern as the merged dev PR,
  `reimbursement-service#268`, and the open staging PR,
  `reimbursement-service#269`. Note `m2mAudience` is a separate, unrelated
  value shared with Expensify/user-service token requests against the legacy
  api-gw — leave it pointed at the pre-Heimdall host; only `cfAudience`/
  `cfAPIURL`/`cfAPIPathPrefix` move. Validate past RS's token cache TTL (~5
  min default) before calling the env's cutover clean — warm Lambda
  containers can keep presenting the old audience for up to one token
  lifetime after deploy.
- **Ledger Service** (`ledger-service` repo) — its Expensify sync runs every
  10 minutes (`cron(0/10 * * * ? *)`) and calls
  `GET {CF_API_URL}/v1/initiatives/{slug}` on the legacy host
  (`cfclient/cfclient.go:155`, `serverless.yml`'s `cfAPIURL`). This call is
  unauthenticated/public (the initiative detail route), so it needs no new
  Auth0 client grant — just repoint `CF_API_URL` to that env's gateway host
  and change the hardcoded `/v1` prefix in `cfclient.go` to `/crowdfunding`
  (or make it a per-stage config value, matching RS's `cfAPIPathPrefix`
  pattern, if prod ever needs to keep `/v1`). Requires a tag deploy in the
  same window — this call is almost certainly already failing in dev, since
  dev cut over without a corresponding Ledger Service change.
- **Self Serve** (`lfx-v2-argocd` `values/<env>/lfx-self-serve.yaml`) —
  repoint `CROWDFUNDING_API_BASE_URL` and `CROWDFUNDING_API_AUDIENCE` from
  the legacy host (still `https://crowdfunding-api.<env>...` in both staging
  and prod today) to that env's gateway host, same as dev's values. The code
  side of this (calling `/crowdfunding/...` instead of `/v1/...`) already
  shipped org-wide in `lfx-self-serve#2846` and needs no further change —
  only this values-file repoint is synchronized with the flip.
- **Stripe webhook endpoint** — no Terraform/IaC manages this; it's a manual
  edit in the Stripe dashboard (or `POST /v1/webhook_endpoints/{id}` via the
  Stripe API) per environment. **Edit the existing endpoint's URL in place —
  do not delete and recreate it** — changing it to
  `https://lfx-api.<env>.../crowdfunding/stripe/webhook` in place preserves
  `STRIPE_WEBHOOK_SECRET`; recreating the endpoint rotates it and breaks
  webhook verification until the new secret is also updated in that env's
  `STRIPE_WEBHOOK_SECRET` config.

### 6. Post-cutover verification

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
  succeeds — this is the caller #281/#286 specifically unblocked. This rule's
  `openfga_check` runs unconditionally once traffic reaches Heimdall (it is
  not gated on `openfga.enabled` — `ruleset.yaml:98-123`); it was blocked
  purely by this step's gateway cutover being the first time these routes
  reach Heimdall at all, with the M2M subject tuples from the precondition
  check already satisfying it.
- A known approver successfully approves/declines a test initiative through
  the gateway — confirms step 3's `team:crowdfunding_approvers` seeding
  actually took, not just that the tuple write succeeded.
- Self Serve's CF module still loads and its silent second Auth0 login still
  mints a usable token (see the note on `CrowdfundingAuthService` in #292 —
  it keeps working post-cutover because `lfx_v2_api` silently drops the
  unknown `access:me` scope it requests and the gateway's `create_jwt`
  finalizer adds the real scope back).
- Stripe dashboard shows a 2xx on the next webhook delivery after the
  endpoint edit, and a real (or test) event still passes HMAC validation.
- Ledger Service's next Expensify sync (within 10 minutes) succeeds against
  the gateway host — check its logs for the `GET .../crowdfunding/initiatives/{slug}`
  call returning 200, not a 404/401 against the old host or old path.

## Rollback

Rolling back steps 1–4 (`fgaReconcileCronJob.enabled`, `openfga.enabled`) is
just reverting that values PR in `lfx-v2-argocd` — both are inert without the
gateway, so nothing else needs to move in the same window. Note
`fgaReconcileCronJob` itself is not inert while it's enabled: it publishes a
tuple update for every initiative on its own schedule (see the message-volume
note in step 2) regardless of `openfga.enabled`, so rolling it back does stop
that traffic if that's desired — it's not required for a *safe* rollback,
since a live reconcile job with `openfga.enabled: false` has no
authorization-path effect either way.

Rolling back step 5 (the gateway cutover) is not a backend-only revert —
every synchronized consumer from step 5 must move back together, in the same
window, the same way they moved forward:

- Backend: `heimdall.enabled: false`, `ingress.enabled: true` (values PR).
- Frontend: revert `NUXT_API_BASE_URL`/`NUXT_PUBLIC_AUTH0_AUDIENCE` to the
  legacy host (values PR).
- Self Serve: revert `CROWDFUNDING_API_BASE_URL`/`CROWDFUNDING_API_AUDIENCE`
  (values PR).
- Reimbursement Service: revert `cfAudience`/`cfAPIURL`/`cfAPIPathPrefix` —
  requires a new tag deploy, not just a values change.
- Ledger Service: revert `CF_API_URL` and the `/crowdfunding` prefix change —
  also requires a tag deploy.
- Stripe: edit the webhook endpoint URL back to the legacy host in the Stripe
  dashboard (in place, same secret-preservation caveat as the forward move).

For the values-file changes, `selfHeal: true` restores the previous state on
the next ArgoCD sync once the PR is reverted; do not edit cluster resources by
hand. For RS and Ledger, the revert isn't complete until their tag deploys
land — the legacy Ingress host works again immediately, but RS/Ledger keep
failing against it until redeployed, since the reverted `ingress.enabled: true`
alone doesn't undo their config/code changes.

## Open items not blocking this playbook

- [lfx-v2-argocd#1637](https://github.com/linuxfoundation/lfx-v2-argocd/pull/1637)
  (open, unmerged): staging `ALLOWED_APPROVERS` username fix — merge this
  **before** step 3's approvers-team seeding for staging, so you seed the
  corrected username list, not the typo'd one.
- [#286](https://github.com/linuxfoundation/lfx-crowdfunding/issues/286) and
  [#292](https://github.com/linuxfoundation/lfx-crowdfunding/issues/292) are
  the tracking issues for the RS/Self-Serve/Stripe repoints in step 5.
  Staging already has pre-staged PRs open: `lfx-v2-argocd#1668`–`#1671`
  (backend steps 1/4/5) and `reimbursement-service#269` (RS). Prod has none
  yet — pre-stage prod's backend, frontend, Self Serve, RS, and Ledger Service
  PRs before prod's cutover window. No Ledger Service PR (staging or prod)
  exists yet either; it isn't tracked by #286/#292 today and should be, since
  it's a fourth synchronized consumer of step 5.
- #271 remains open on this repo as the tracking issue; comment there (or
  update this doc) after staging and after prod, following the pattern of the
  existing dev-completion comments.
