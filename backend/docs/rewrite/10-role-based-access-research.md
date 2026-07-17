<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Role-Based Access: Self Serve Model vs. Crowdfunding — Research

---

**Status:** Exploratory research only. No decision has been made to build this; this document
exists to inform a future discussion, not to spec an implementation.

This document compares the role-based access model used in **LFX Self Serve (SS)** against the
current authorization model in **LFX Crowdfunding (CF)**, and outlines what it would take to bring
an SS-style role model to CF.

---

## 1. Why this came up

CF currently has no concept of shared ownership or delegated access to an initiative — a single
`owner_id` decides who can manage it, plus a small hardcoded list of platform "approvers." SS has a
more mature multi-person, multi-role access model for orgs and projects. This doc surveys that SS
model as a reference point for what CF would need if it ever supports things like co-owners or
staff on an initiative.

---

## 2. SS role model (current state)

SS's model is **org/project-centric**, backed by the upstream **member-service**, not a
CF-style DB column, and not a direct FGA/ACS call at request time.

### 2.1 Two role types, two scopes

| Scope | Roles | Notes |
|---|---|---|
| Org | `writer` (full edit), `auditor` (read-only) | Direct assignment, stored in `b2b_org_settings` upstream |
| Org (cascading) | `auditor` only | Inherited top-down from parent org to child orgs. **Writer never cascades** — this is an explicit FGA-model design choice, not an oversight |
| Project | `manage` (writer-equivalent), `view` (auditor-equivalent) | Per-project staff list, binary role, stored in upstream project-service |

### 2.2 Backend: `OrgRoleGrantsService`

`lfx-self-serve/apps/lfx-one/src/server/services/org-role-grants.service.ts` is the single source
of truth for "what can this user do":

1. Queries member-service for the caller's role grants, tagged by `member:<username>`.
2. Classifies each grant as `writer` or `auditor` from the org's members list (writer wins if a
   user appears as both — matches the indexer's dedupe rule). Only `accepted` invites count.
3. Separately fetches cascading children for any org the user directly manages, to build the
   inherited-auditor set.
4. Returns four sets: `writers[]`, `auditors[]`, `cascadingWriters[]`, `cascadingAuditors[]`.
5. Caches the resolved result per-username for 5 minutes, to avoid recomputing the full
   settings/cascading fan-out on every debounced typeahead request.

### 2.3 Frontend gating

`lfx-self-serve/apps/lfx-one/src/app/shared/services/org-role-grants.service.ts` mirrors this
server-side shape into signals: `writerSet` / `auditorSet` (direct-only) and separate
`inheritedWriterSet` / `inheritedAuditorSet`. **Capability gates (`canWrite`) only ever check the
direct sets** — inherited/cascading roles are shown as badges in the UI (e.g. dropdowns, tooltips)
but never grant write access. This separation is called out explicitly in SS's spec notes (FR-011a)
as a deliberate design constraint, not an accident.

Project-level staff management is a separate, simpler service
(`app/shared/services/permissions.service.ts`) that fetches a project's `auditors[]`/`writers[]`
list and renders a staff table with `view`/`manage` roles.

### 2.4 What SS does *not* do

- No direct FGA or ACS API calls in the request path — authorization data lives in member-service,
  and SS treats it as an upstream read, not a live permission check against a policy engine.
- No role hierarchy beyond the two-level (org → child org) cascade.
- Auth middleware in SS is authentication-only (valid session / valid token) — role checks are a
  separate, later step performed by the services above, not baked into route middleware.

---

## 3. CF current model

CF's authorization model is intentionally minimal — see
[`08-self-serve-auth.md`](./08-self-serve-auth.md) for how identity itself flows from SS to CF.
This section is about what CF does with that identity once authenticated.

### 3.1 Principal has no role field

`backend/internal/domain/models/filters.go` (`Principal` struct) carries `UserID`, `Username`,
`Scope`, and profile claims (email, name) — no role or permission field. `Scope` is the OAuth2
scope from the token (e.g. `access:me`), not an app-level role.

### 3.2 Two access mechanisms, both binary

| Mechanism | Where | Behavior |
|---|---|---|
| Ownership | `internal/service/initiative_service.go` (`GetForUser`, `ResolveOwnedInitiativeID`) | `initiative.OwnerID == principal.UserID` — anything else returns `ErrInitiativeNotFound` (not `403`, to avoid confirming existence) |
| Approver allowlist | `internal/handler/initiative_handler.go` (`isApprover`, `allowedApprovers` field) | Static list of usernames from an env var, checked before approve/decline actions |

There is no concept of:
- Multiple people managing the same initiative (co-owners, staff, delegated editors)
- Any inherited or cascading access
- Any caching layer for permission resolution (each request re-checks ownership directly against
  the DB row already loaded)

### 3.3 Frontend

The Nuxt frontend has no role-based UI gating today — access decisions are entirely enforced
server-side by the Go API; the frontend simply reflects what the API returns or rejects.

---

## 4. Gap analysis

| Aspect | SS | CF |
|---|---|---|
| Role granularity | writer / auditor (org), manage / view (project) | owner / not-owner, plus approver allowlist |
| Multiple people per resource | Yes (org members, project staff) | No — single `owner_id` |
| Inheritance | Yes — auditor cascades parent→child org | No hierarchy exists |
| Data source | Upstream member-service / project-service | Local `owner_id` column |
| Caching | Per-username, 5-minute TTL | None — checked per request |
| Frontend awareness | Yes — signals gate UI capabilities | No — frontend has no auth logic |
| Policy engine (FGA/ACS) | Not called directly; upstream services encapsulate it | Not used at all |

**Bottom line:** SS's model solves a problem CF doesn't have yet — multiple people sharing
management of one resource, with different capability levels, across an org hierarchy. Porting it
to CF is not a small delta; it means introducing a concept (shared/delegated initiative access)
that doesn't exist in CF's domain model today, plus a resolution service and (if CF wants
parity) a caching layer.

---

## 5. Open questions (for a future decision, not resolved here)

1. Does CF actually need multi-person access per initiative, or does the single-owner model cover
   real usage? (i.e., is this a solution looking for a problem, or has co-ownership been requested?)
2. If needed, would CF model it per-initiative (like SS's project staff: `view`/`manage`) rather
   than adopting SS's full org-hierarchy/cascading model, which assumes an org structure CF's
   domain doesn't have?
3. Would CF source roles from an upstream service (mirroring SS's member-service pattern) or store
   them locally (a `initiative_staff` table), given CF already keeps ownership as a local column?
4. Does the `allowedApprovers` env-var allowlist get folded into any new role model, or stay a
   separate platform-admin concept?

---

## Related Documents

- [`08-self-serve-auth.md`](./08-self-serve-auth.md) — how SS authenticates to CF (identity, not authorization)
- [`02-decisions.md`](./02-decisions.md) — prior architecture decisions
- [`03-open-questions.md`](./03-open-questions.md) — existing open-questions log
