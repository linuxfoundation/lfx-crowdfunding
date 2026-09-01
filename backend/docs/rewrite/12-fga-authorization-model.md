<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Initiative Attribution — 12: FGA Authorization Model

Status: Proposal — for Architecture team review
Related: [11-initiative-attribution-and-access.md](./11-initiative-attribution-and-access.md)
(the product design this authorizes), tracked as
[lfx-crowdfunding#269](https://github.com/linuxfoundation/lfx-crowdfunding/issues/269)

Doc 11 §3.4 already concluded that the idiomatic `crowdfunding_initiative` FGA type is the
gateway-milestone plan, but never wrote the model down. This doc proposes it — one type, not a
family: CF's whole authorization surface is "an initiative may hang off a project or an org, and
writers there can manage it," a much smaller shape than a multi-object service like Mentorship
(see [lfx-mentorship#119](https://github.com/linuxfoundation/lfx-mentorship/pull/119), whose
authorization-model doc this one follows in structure and convention).

## Principle

Postgres stays the system of record for initiatives, attribution, and status. OpenFGA holds only
what Heimdall needs to check at the edge — who may manage a given initiative. CF makes no
authorization decisions in the backend and never queries FGA at request time. Donations,
sponsorship tiers, announcements, and statuses stay Postgres-only, unchanged from doc 11.

## The type

Grounded in `lfx-v2-helm/charts/lfx-platform/files/model.fga`, two existing precedents:

- `meeting.organizer: [user] or writer from project or writer from committee` — the sanctioned
  shape for "direct grant OR inherited from one of two parents." This is the precedent doc 11
  §3.4 needed and never cited when it flagged the hybrid's OR-union as an antipattern — the
  concern was checking two entities from *outside* the gateway, not the union itself.
- `vote_response` — single owner plus parent, and the model's own guidance that a relation which
  is a mere alias of another (e.g. a `writer` defined as just `owner`) should not exist.

An earlier version of this proposal also modeled CF's approvers as a `team#member` relation
(mirroring `b2b_org.global_org_admin`), to retire the `ALLOWED_APPROVERS` env allowlist. **PM
decided against that (2026-09-01, doc 11 open question 5): `ALLOWED_APPROVERS` stays as-is.**
Approval/decline authorization stays a CF-backend check against that allowlist
(`backend/cmd/initiatives-api/config.go:110-112`,
`backend/internal/handler/initiative_handler.go:568-579`, `isApprover`), entirely outside this
type — approvers hold no FGA tuple and are not part of `writer` or `viewer` below.

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
  any inherited-auditor population. Approvers view via the separate `isApprover` backend check
  above, not through this relation.
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

## Decided (PM, 2026-09-01)

Resolved during review — kept here for the record rather than left in the open-questions table:

- **Attribution-change authorization (was open question A).** The two-entity dual-check design
  this question raised is unnecessary: **the original creator always retains access to their
  initiative**, including the ability to move it to a different org/project they now belong to.
  Attribution changes are authorized by the standard `writer` check alone (owner, or the target
  entity's writer) — no separate check against the *current* attributed entity is required.
- **`b2b_org` writer non-cascading (was open question B).** Confirmed as the intended design, not
  just an accepted platform limitation: **no parent- or child-org population ever gains writer
  access** — only the org actually assigned to the initiative does. No change needed; `writer from
  b2b_org` already has this shape.
- **`ALLOWED_APPROVERS` (was open question C).** Keep the current env var allowlist; approvers are
  not modeled in FGA. See "The type" above and doc 11 open question 5.
- **Private-view population (was open question E).** No wider audience than creator, entity
  writer, and approver — the `auditor` relation and its project/org inheritance are dropped from
  the type; see "The type" above.

## Open questions for the Architecture team

| # | Question | Proposed default | Directed to |
|---|---|---|---|
| D | **Ordering.** Model + `tests.yaml` in `lfx-v2-helm` first, Heimdall RuleSets second, CF-side tuple emission third — a RuleSet referencing a relation that doesn't exist yet fails closed. | Model lands first, as its own PR; RuleSet wiring and CF emission are tracked separately once the model is accepted. | Architecture team |
| G | **`GET /v1/me/initiatives` (the caller's manageable-initiatives list) has no authorization mechanism under this model.** That route has no initiative ID for Heimdall to check; doc 11 §5.1 requires CF-side batched FGA checks against a CF-bounded candidate set, but this doc's principle says CF never queries FGA at request time, and the per-initiative `crowdfunding_initiative` type (unlike the rejected `ListObjects` enumeration) doesn't itself support "list all initiatives I can manage." | No default proposed — needs an edge/query/index strategy that returns the caller's manageable initiatives with correct pagination before this doc's "all backend authorization removed" claim can stand for the list endpoint. | Architecture team |

(Open question F — `delete_access` orphaning a per-initiative `approver@team:…#member` tuple — no
longer applies now that approvers are not modeled in FGA at all, per the "Decided" section above.)

## Prerequisite (named, not solved here)

[lfx-crowdfunding#263](https://github.com/linuxfoundation/lfx-crowdfunding/issues/263):
`attributed_to_uid` is typed `UUID` (migration `007_initiative_attribution.up.sql`), but a real
`b2b_org` uid may be an 18-character Salesforce Account SFID. Emitting a `b2b_org` reference
tuple is blocked on resolving that mismatch first.
