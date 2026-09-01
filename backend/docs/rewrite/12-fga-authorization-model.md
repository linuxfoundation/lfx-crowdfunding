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

Grounded in `lfx-v2-helm/charts/lfx-platform/files/model.fga`, three existing precedents:

- `meeting.organizer: [user] or writer from project or writer from committee` — the sanctioned
  shape for "direct grant OR inherited from one of two parents." This is the precedent doc 11
  §3.4 needed and never cited when it flagged the hybrid's OR-union as an antipattern — the
  concern was checking two entities from *outside* the gateway, not the union itself.
- `b2b_org.global_org_admin: [team#member]` — *"written to every b2b_org at creation time … providing
  writer access … without requiring a hierarchical root object."* The answer for CF's global
  approvers, which today are a flat env allowlist (`ALLOWED_APPROVERS`).
- `vote_response` — single owner plus parent, and the model's own guidance that a relation which
  is a mere alias of another (e.g. a `writer` defined as just `owner`) should not exist.

```
type crowdfunding_initiative
  relations
    # exclusive attribution; personal initiatives have neither
    define project: [project]
    define b2b_org: [b2b_org]
    # LF staff approvers, stamped on every initiative at creation
    # (mirrors b2b_org.global_org_admin — no root object invented)
    # @fgadoc:jtbd Approve or decline a submitted initiative
    define approver: [team#member]
    # @fgadoc:alias Creator
    define owner: [user]
    # @fgadoc:jtbd Update, publish, hide & delete an initiative
    define writer: owner or writer from project or writer from b2b_org
    # @fgadoc:jtbd View a non-public initiative & its transactions
    define auditor: writer or approver or auditor from project or auditor from b2b_org
    # @fgadoc:jtbd View & discover an initiative
    define viewer: [user:*] or auditor
```

Notes:

- `writer` is doc 11 §2.2's flat manage capability verbatim: creator **or** attributed-entity
  writer, one capability, no view-only tier.
- `approver` sits outside `writer` deliberately — approvers cannot edit content, and a creator
  cannot approve their own initiative. This retires the `ALLOWED_APPROVERS` allowlist
  (`backend/cmd/initiatives-api/config.go:110-112`,
  `backend/internal/handler/initiative_handler.go:570-579`) and resolves doc 11 open question 5.
- `viewer: [user:*]` is a **per-object** tuple, emitted only while `status == 'published'`
  (`backend/internal/domain/models/initiative.go:29-62`). `hidden` and `declined` are paths back
  down from `published`, so the wildcard must be withdrawn on those transitions, not just granted
  on the way up.
- `owner` becomes a tuple rather than a Postgres comparison — the one change that moves CF's last
  in-backend authorization decision to the edge.

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
not that the DSL parses. Minimum negative cases: an `approver` is denied `writer`; an `owner` is
denied `approver` on their own initiative; a project-A writer is denied `writer` on a
project-B-attributed initiative; a `hidden` initiative grants no `viewer@user:*`. Minimum
inheritance case: a parent-project writer reaches an initiative attributed to the child project.
Validate locally against OpenFGA in Docker before proposing — a mis-scoped relation fails *open*.

## Open questions for the Architecture team

| # | Question | Proposed default |
|---|---|---|
| A | **How is an attribution *change* authorized at the edge?** The decision spans the current and target entity (doc 11 §2.2), but the target UID is a request-body value today, and Heimdall templates its `openfga_check` object from URL captures (verified in `lfx-v2-campaign-service/docs/knowledge/kubernetes/ruleset.md`). | Put the target in the path (`PUT /initiatives/{uid}/attribution/{type}/{entity_uid}`) and attach two `openfga_check` entries. This is the direct answer to the 2026-08-25 guidance that the decision *"should be done by the API GW, not the backend service."* |
| B | **Is `writer from b2b_org` acceptable given it doesn't cascade?** `b2b_org.writer` is scoped to the directly-assigned org by design (parent→child is auditor-only in the platform model); `project.writer` inherits from parent. Org- and project-attributed initiatives therefore inherit differently. | Accept it — it's the platform's own scoping decision, and CF shouldn't invent a broader org-write population. Flagged so it isn't discovered later as a surprise. |
| C | **Who owns and administers the `approver` team?** A `team#member` grant needs a platform team to exist and be administered. | Reuse an existing LF-staff team if one fits; otherwise a CF-specific team administered the way `global_org_admin` is. |
| D | **Ordering.** Model + `tests.yaml` in `lfx-v2-helm` first, Heimdall RuleSets second, CF-side tuple emission third — a RuleSet referencing a relation that doesn't exist yet fails closed. | Model lands first, as its own PR; RuleSet wiring and CF emission are tracked separately once the model is accepted. |

## Prerequisite (named, not solved here)

[lfx-crowdfunding#263](https://github.com/linuxfoundation/lfx-crowdfunding/issues/263):
`attributed_to_uid` is typed `UUID` (migration `007_initiative_attribution.up.sql`), but a real
`b2b_org` uid may be an 18-character Salesforce Account SFID. Emitting a `b2b_org` reference
tuple is blocked on resolving that mismatch first.
