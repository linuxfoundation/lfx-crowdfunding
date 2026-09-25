# FGA Contract — Crowdfunding (Initiatives) Service

This document is the authoritative reference for all messages the crowdfunding
(initiatives) service sends to the fga-sync service, which writes and deletes
[OpenFGA](https://openfga.dev/) relationship tuples to enforce access control.

The full OpenFGA type definitions (relations, schema) are defined in the
[platform model](https://github.com/linuxfoundation/lfx-v2-helm/blob/main/charts/lfx-platform/templates/openfga/model.yaml).

**Update this document in the same PR as any change to FGA message construction.**

---

## Object Types

- [Crowdfunding Initiative](#crowdfunding-initiative)

---

## Message Format

All messages use the generic FGA message format on the following NATS subjects:

| Subject | Used for |
|---|---|
| `lfx.fga-sync.update_access` | Full-sync of an initiative's current access state |
| `lfx.fga-sync.delete_access` | Withdraw every publisher-managed tuple for an initiative |

Publication is fire-and-forget core NATS — no request/reply, no wait on
fga-sync processing or OpenFGA convergence (`internal/infrastructure/fga/publisher.go`).

### Delivery Semantics

`UpdateAccess` calls are best-effort: a publish failure is logged and does not
fail the caller's request. This is safe because every initiative's current
state is also republished by the `fga-reconcile` CronJob
(`cmd/fga-backfill`), which self-heals a missed or out-of-order update on its
next pass.

`DeleteAccess` blocks the caller (bounded by a 5s timeout) and flushes the
NATS connection before `Delete` returns, so a pod exit immediately after
delete can't silently strand the tuple withdrawal — a missed delete has no
DB row left to reconstruct it from, unlike a missed update.

**Known accepted risk:** a hard-deleted initiative row is gone from
`ListForFGABackfill`, so if a delete's publish is lost after the flush
succeeds (e.g. fga-sync itself fails to process it and exhausts retries), no
automated path can recover it — worst case is an orphaned tuple on a
UID nothing resolves to, which grants nothing. Recovery is a manual replay
per the [fga-sync contract](https://github.com/linuxfoundation/lfx-v2-fga-sync/blob/main/docs/fga-sync-contract.md).
Decided acceptable in PR #295 review (see thread on `initiative_service.go`'s
`syncFGADelete`).

---

## Crowdfunding Initiative

**Source struct:** `internal/domain/models` — `Initiative`, via
`internal/infrastructure/fga.InitiativeAccess`

**Synced on:** create, update (approve, hide/publish, attribution change),
delete of an initiative; and on every `fga-reconcile` CronJob pass (full
republish of all initiatives, self-healing any missed or stale update).

### update_access

Published to `lfx.fga-sync.update_access`.

#### Message Envelope

| Field | Value |
|---|---|
| `object_type` | `crowdfunding_initiative` |
| `operation` | `update_access` |

#### Data Fields

| Field | Value |
|---|---|
| `uid` | `Initiative.ID` |
| `public` | `true` iff `Initiative.Status == published` — drives the `viewer:[user:*]` wildcard |

#### Relations

| Relation | Value | Condition |
|---|---|---|
| `owner` | Owner's LFX username | Always |

This is a full sync — any relation not present in the message is removed by
fga-sync.

#### References

| Reference | Value | Condition |
|---|---|---|
| `project` | `Initiative.Attribution.EntityUID` | Only when `Attribution.Type == project` and `EntityUID` is non-empty |

> `b2b_org` attribution is deliberately not emitted as a reference yet —
> blocked on lfx-crowdfunding#263 (`attributed_to_uid` is typed UUID, but a
> real b2b_org UID may be an 18-character Salesforce SFID). Project
> attribution and owner are unaffected and ship now.

### delete_access

Published to `lfx.fga-sync.delete_access` on initiative delete, carrying
only `uid`. Removes every FGA tuple for `crowdfunding_initiative:{uid}`.

---

## Triggers

| Operation | Subject | Notes |
|---|---|---|
| Create initiative | `lfx.fga-sync.update_access` | Owner relation set; `public` false until approved+published |
| Approve / publish / hide initiative | `lfx.fga-sync.update_access` | `public` toggles the `viewer:[user:*]` wildcard |
| Update attribution | `lfx.fga-sync.update_access` | `project` reference set/cleared |
| Delete initiative | `lfx.fga-sync.delete_access` | Blocks on a 5s-bounded flush before `Delete` returns |
| `fga-reconcile` CronJob (daily) | `lfx.fga-sync.update_access` | Republishes every initiative's current DB state; self-heals stale/missed live-path publishes |
