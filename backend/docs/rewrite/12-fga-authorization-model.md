<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Initiative Attribution — 12: FGA Authorization Model

Status: Proposal — for Architecture team review
Related: [11-initiative-attribution-and-access.md](./11-initiative-attribution-and-access.md)
(the product design this authorizes), tracked as
[lfx-crowdfunding#269](https://github.com/linuxfoundation/lfx-crowdfunding/issues/269) —
**stale as of this revision**: the issue still reflects a pre-2026-09-01 version of the model
(`approver`/`auditor` relations, per-initiative approver tuple, old open-questions A-D); this
doc's "The type," "Decided," and "Open questions" sections above supersede it. Needs a sync pass
before architecture review.

Doc 11 §3.4 already concluded that the idiomatic `crowdfunding_initiative` FGA type is the
gateway-milestone plan, but never wrote the model down. This doc proposes it — one type, not a
family: CF's whole authorization surface is "an initiative may hang off a project or an org, and
writers there can manage it," a much smaller shape than a multi-object service like Mentorship
(see [lfx-mentorship#119](https://github.com/linuxfoundation/lfx-mentorship/pull/119), whose
authorization-model doc this one follows in structure and convention).

## Principle

Postgres stays the system of record for initiatives, attribution, and status. OpenFGA holds only
what Heimdall needs to check at the edge — who may manage a given initiative. CF makes no
per-object authorization decisions in the backend and never queries FGA to gate a single request
at request time — that's Heimdall's job. Donations, sponsorship tiers, announcements, and statuses
stay Postgres-only, unchanged from doc 11.

**Carve-out for list population (see open question G resolution below):** the above principle
governs single-object *gating* (can this request proceed), which moves to the gateway entirely.
It does not forbid one bounded, batched `Check` call issued by CF itself to *populate a list view*
— that's a read-model concern, not an authorization decision, and the same shape doc 11 §5.1
already proposed for the hybrid model.

**Carve-out for the approver Phase-1 rollout (see "Rollout is two phases" below):** until CF sits
behind Heimdall, CF's backend does query FGA at request time to gate a single request (approver
check via `NATSResolver`). This is a temporary, explicitly-scoped exception to the Principle, not
a second pattern — it is retired at the gateway milestone when the check moves to a Heimdall
`openfga_check` rule.

## The type

Grounded in `lfx-v2-helm/charts/lfx-platform/files/model.fga`, two existing precedents:

- `meeting.organizer: [user] or writer from project or writer from committee` — the sanctioned
  shape for "direct grant OR inherited from one of two parents." This is the precedent doc 11
  §3.4 needed and never cited when it flagged the hybrid's OR-union as an antipattern — the
  concern was checking two entities from *outside* the gateway, not the union itself.
- `vote_response` — single owner plus parent, and the model's own guidance that a relation which
  is a mere alias of another (e.g. a `writer` defined as just `owner`) should not exist.

An earlier version of this proposal modeled CF's approvers as a **per-initiative**
`team#member` relation on `crowdfunding_initiative` itself (mirroring
`b2b_org.global_org_admin`), to retire the `ALLOWED_APPROVERS` env allowlist. PM decided
against that (2026-09-01, doc 11 open question 5): `ALLOWED_APPROVERS` stays as-is. That
decision is **reopened** as of 2026-09-09 — see "Approvers as a global team grant" below,
which proposes a different shape (one platform-wide team, no per-object tuple) that avoids
the objections the per-initiative shape raised. Until that proposal is accepted, approvers
hold no FGA tuple and are not part of `writer` or `viewer` below; approval/decline
authorization stays a CF-backend check against the env allowlist
(`backend/cmd/initiatives-api/config.go:110-112`,
`backend/internal/handler/initiative_handler.go:568-579`, `isApprover`), entirely outside
this type.

```
type crowdfunding_initiative
  relations
    # exclusive attribution; personal initiatives have neither
    define project: [project]
    define b2b_org: [b2b_org]
    # @fgadoc:alias Creator
    define owner: [user]
    # @fgadoc:jtbd Update, publish, hide & delete an initiative
    define writer: owner or writer from project or writer from b2b_org
    # @fgadoc:jtbd View & discover an initiative
    define viewer: [user:*] or writer
```

Notes:

- `writer` is doc 11 §2.2's flat manage capability verbatim: creator **or** attributed-entity
  writer, one capability, no view-only tier.
- **Private-view population, PM-confirmed (2026-09-01, doc 11 open question 5 discussion):** only
  the creator, the attributed entity's writers, and approvers may view a non-public initiative —
  no wider audience (e.g. project/org auditors). `writer` already covers creator + entity writer;
  `viewer: [user:*] or writer` extends that to public visibility once published, without adding
  any inherited-auditor population. Approvers view via the separate `isApprover` check (backend
  today, gateway-side under the proposal below), not through this relation.
- `viewer: [user:*]` is a **per-object** wildcard tuple, emitted only while `status == 'published'`
  (`backend/internal/domain/models/initiative.go:29-62`). `hidden` is the only path back down from
  `published` (`validateOwnerStatusTransition`, `backend/internal/service/initiative_service.go:818-833`
  — `declined` is reachable only from `submitted`/`pending`, never from `published`), so the
  wildcard must be withdrawn on the `published → hidden` transition, not just granted on the way up.
- `owner` becomes a tuple rather than a Postgres comparison — the one change that moves CF's last
  in-backend authorization decision to the edge.
- `b2b_org`-attributed writer access is deliberately non-cascading, PM-confirmed (2026-09-01, open
  question B below): only the org actually assigned to the initiative gets writer access — no
  parent- or child-org population is ever granted it.

## Approvers as a global team grant (proposed, reopens 2026-09-01 decision)

The earlier per-initiative proposal was `approver: [team#member]` on `crowdfunding_initiative`
itself — a tuple written to every initiative object at creation. PM rejected it for three
reasons: no known team to point at, no operational path to administer one, and
`delete_access` orphaning the per-initiative tuple, since fga-sync deliberately never deletes
a tuple whose subject is `team:*` (`lfx-v2-fga-sync/fga.go:274-283` — "these are managed by a
separate workflow and must not be clobbered by resource service sync operations"). All three
were sound objections to that shape.

The proposal below is a different shape: **one platform-wide team, checked as a global grant,
with no tuple ever written per initiative.** It resolves all three objections because the
condition each one depends on — a per-object tuple existing — never arises:

| Objection (2026-09-01) | Per-initiative shape | Global shape |
|---|---|---|
| No known team to point at | New team, no precedent | `team.member` (`[user]`) already carries other services' global capabilities as `@fgadoc:jtbd` lines — no new relation or type |
| No operational path to administer a team | None existed | Already shipped: `lfx-v2-member-service`'s `charts/lfx-v2-member-service/templates/ruleset.yaml:78-82` gates `POST /b2b_orgs` and `POST /admin/reindex` on `relation: member`, `object: "team:{{ globalOrgAdminTeamName }}"` — a constant object, no path param, proven pattern |
| `delete_access` orphans the per-object `team:` tuple | Fatal — every initiative deletion leaves one behind forever | Does not apply — no tuple is ever written against a `crowdfunding_initiative` object, so there is nothing for `delete_access` to orphan |

Team provisioning is self-service, not a request to a platform team: a team object exists
the moment a member tuple names it. `lfx-v2-member-service/scripts/setup-global-org-admin-team.sh`
is the reference implementation — read existing members, diff against a desired list, write
what's missing, verify. CF would adapt this into its own script (`backend/scripts/`)
against `team:crowdfunding_approvers`, with a matching revoke script (removal is not free:
fga-sync's `team:*` retention means there is no code path that deletes a membership tuple).

Two things that must accompany any writer of a `team:*` tuple directly against OpenFGA (as
opposed to going through a service that fga-sync itself manages): the LFID-username format
convention (Heimdall/CF principals are plain usernames, no `auth0|` prefix — getting this
wrong silently 403s every approver), and busting the `fga-sync-cache` KV's `inv` key after
the write, since fga-sync's cache invalidation only triggers on writes it makes itself
(`lfx-v2-fga-sync/docs/fga-sync-contract.md:276-295`) — a direct OpenFGA write does not bump
it, so a cached `false` for a newly-added approver can otherwise survive the grant.

**Rollout is two phases, because CF is not yet behind Heimdall** (no `ruleset.yaml` or
`httproute.yaml` in this repo's chart — CF serves through its own Traefik ingress today, per
the gateway-milestone framing in doc 11 §3.4). The tuples are the same in both phases; only
the caller changes:

1. **Now:** seed `team:crowdfunding_approvers`, and have CF's backend check it itself. CF
   already has the transport — `backend/internal/infrastructure/fga/resolver.go`'s
   `NATSResolver` does `object#relation@user` checks over `lfx.access_check.request`, and
   `FGA_NATS_URL` is already configured in every environment
   (`lfx-v2-argocd/values/global/lfx-crowdfunding-backend.yaml:86`). Add one method
   alongside the existing `CanManage` for `team:<name>#member@user:<username>`, and change
   `isApprover` (`initiative_handler.go:570`) to call it, with `ALLOWED_APPROVERS` kept as a
   fallback while `FGA_NATS_URL` is unset. This already retires the env var as the source of
   truth — an approver can be added or removed without a deploy.
2. **At the gateway milestone:** move the check to a Heimdall `openfga_check` rule on CF's
   own chart, exactly like member-service's `object: "team:{{ approversTeamName }}"` rule,
   and delete `isApprover` and the resolver method entirely. The read path
   (`GET /v1/initiatives/{id}`, approver-visible pre-publish) would use the
   `openfga_or_check` authorizer (`lfx-v2-helm/charts/lfx-platform/values.yaml:363-419`):
   `viewer` on the initiative OR `member` on the approvers team, one `BatchCheck`. That
   authorizer is defined in the platform chart but has no shipped `RuleSet` reference found
   in this repo's environment — verify it against OpenFGA in Docker before relying on it.

This section proposes the shape; it does not change `ALLOWED_APPROVERS` today. See "Decided"
below for the status of the reopened decision.

## Emission (summary, not a full contract)

Enough to judge the model, not a delivery design: `update_access` on initiative creation; on
approve/hide/republish (the `public` flag flipping with `status`); and on attribution change (a
re-parent — the one CF transition with no Mentorship analog, since a mentorship program's parent
is fixed at creation but an initiative's attribution can change). `delete_access` on delete, plus
a one-time backfill of existing initiatives. Delivery guarantees (outbox, reconciliation,
convergence after a missed publish — the gap doc 11 §3.4 names and leaves open) belong to a
separate CF-side emission issue once this model is accepted, not to this proposal.

## `tests.yaml` merge gate

The platform model ships with an OpenFGA test suite; the merge criterion is that scenarios pass,
not that the DSL parses. Minimum negative cases: a project-A writer is denied `writer` on a
project-B-attributed initiative; a `hidden` initiative grants no `viewer@user:*`. Minimum
inheritance case: a parent-project writer reaches an initiative attributed to the child project.
Validate locally against OpenFGA in Docker before proposing — a mis-scoped relation fails *open*.
(The global approver-team proposal above adds no scenarios here — it touches no relation on
`crowdfunding_initiative` and requires no platform model change.)

## Decided (PM, 2026-09-01)

Resolved during review — kept here for the record rather than left in the open-questions table:

- **Attribution-change authorization (was open question A).** The two-entity dual-check design
  this question raised is unnecessary: **the original creator always retains access to their
  initiative**, including the ability to move it to a different org/project they now belong to.
  Attribution changes are authorized by the standard `writer` check alone (owner, or the target
  entity's writer) — no separate check against the *current* attributed entity is required. This
  is a check against the target `project`/`b2b_org` object directly (`writer@user:X` on the
  entity being moved to), not the initiative's own `crowdfunding_initiative#writer` relation,
  which reflects only the current attribution.
- **`b2b_org` writer non-cascading (was open question B).** Confirmed as the intended design, not
  just an accepted platform limitation: **no parent- or child-org population ever gains writer
  access** — only the org actually assigned to the initiative does. No change needed; `writer from
  b2b_org` already has this shape.
- **`ALLOWED_APPROVERS` (was open question C).** Keep the current env var allowlist; approvers are
  not modeled in FGA. See "The type" above and doc 11 open question 5. **Reopened 2026-09-09:**
  new evidence (a shipped global-team-check precedent in `lfx-v2-member-service`, and
  confirmation that a global grant writes no per-object tuple, so the `delete_access` orphan
  objection doesn't apply) supports a different approver shape than the one this decision
  rejected. See "Approvers as a global team grant" above. The 2026-09-01 objections stand
  against the per-initiative shape they were made against; they were not re-litigated for the
  global shape until this reopening.
- **Private-view population (was open question E).** No wider audience than creator, entity
  writer, and approver — the `auditor` relation and its project/org inheritance are dropped from
  the type; see "The type" above.
- **`GET /v1/me/initiatives` list authorization (was open question G).** Resolved without an
  external service, using only Postgres + a bounded batched FGA `Check` (no `ListObjects`, no new
  dependency):
  1. Postgres: `SELECT id FROM initiatives WHERE owner_id = $1` — owned initiatives need no check.
  2. Postgres: `SELECT DISTINCT attributed_to_type, attributed_to_uid FROM initiatives WHERE
     attributed_to_type <> 'personal'` — the candidate set is bounded by *how many distinct
     orgs/projects have ever been attributed an initiative*, not by total initiatives or total
     platform entities.
  3. One batched `access_check.request` over NATS (transit C, fga-sync's own batching — not a
     direct OpenFGA call, per the Principle above) against that candidate set: "is `$1` a writer
     on `project:X` / `b2b_org:Y`" for each entity, one round trip. Bound and chunk the batch per
     doc 11 §5.1's bounding note (never silently truncate).
  4. Postgres: `SELECT id FROM initiatives WHERE (owner_id = $1 OR (attributed_to_type,
     attributed_to_uid) IN (<entities that came back true>))` — same parenthesized-OR shape as
     doc 11 §5.1's builder fix (the outer parens matter: `InitiativeRepository.List` appends
     further filters with a bare `AND`, so an unparenthesized `OR` would let owner-owned rows
     bypass them), so pagination/sorting/search stay ordinary SQL.

  This is the same batched-candidate-set shape doc 11 §5.1 already proposed for the hybrid model,
  and the same shape `lfx-self-serve`'s `AccessCheckService.checkAccess()` uses in production to
  annotate list rows — a bounded batch of named checks, not the open-ended `ListObjects`
  enumeration this doc rejects elsewhere. See the principle carve-out above.

## Open questions for the Architecture team

| # | Question | Proposed default | Directed to |
|---|---|---|---|
| D | **Ordering.** Model + `tests.yaml` in `lfx-v2-helm` first, CF-side tuple emission (with backfill) second, Heimdall RuleSets third — turning on enforcement before tuples exist fails every check closed, locking out every caller until backfill completes. | Model lands first, as its own PR; CF emission and backfill land next; RuleSet wiring (enforcement) lands last, once tuples are known to be present for every initiative. | Architecture team |
| H | **Slug-vs-UUID route contract.** `GET/PUT /v1/initiatives/{id}` accepts either a slug or a UUID in `{id}`; a Heimdall `openfga_check` RuleSet at the gateway milestone evaluates `object: "crowdfunding_initiative:{id}"` templated straight off the path param, before CF's handler can resolve a slug to the canonical UUID. A slug-addressed request would then check a `crowdfunding_initiative` object that never got any tuples (they're all emitted under the UUID), and fail closed for every valid slug-based request. | Either canonicalize slug-to-UUID upstream of the RuleSet (a Heimdall-side lookup or route-level redirect), or split the route so only the UUID-addressed path carries the `openfga_check` RuleSet and the slug-addressed path resolves first, then re-checks. | Architecture team |

(Open question F — `delete_access` orphaning a per-initiative `approver@team:…#member` tuple — no
longer applies: approvers are not modeled in FGA today, and the reopened global-team proposal in
"Approvers as a global team grant" above writes no per-initiative tuple either, so this objection
does not resurface under it. Open question G — the `GET /v1/me/initiatives` list mechanism — is
resolved above via a bounded batched `Check`, no longer open.)

## Prerequisite (named, not solved here)

[lfx-crowdfunding#263](https://github.com/linuxfoundation/lfx-crowdfunding/issues/263):
`attributed_to_uid` is typed `UUID` (migration `007_initiative_attribution.up.sql`), but a real
`b2b_org` uid may be an 18-character Salesforce Account SFID. Emitting a `b2b_org` reference
tuple is blocked on resolving that mismatch first.

This does **not** block the global approver-team proposal above: that proposal writes only
`user:<lfid>` → `member` → `team:crowdfunding_approvers` tuples, never a `b2b_org` reference,
so it can proceed independently of #263.
