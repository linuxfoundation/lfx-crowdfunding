<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Role-Based Access: Self Serve Model vs. Crowdfunding — Research

---

**Status:** Exploratory research with an agreed direction (§5) — not yet a spec or implementation
plan. Summary of the direction: every initiative belongs to a parent LF project; a single flat
project-level editor role, checked against the platform's OpenFGA store via `lfx-v2-fga-sync`,
grants management of all the project's initiatives; the creator always retains edit access to
their own initiative. Remaining open items are in §6.

This document compares the role-based access model used in **LFX Self Serve (SS)** against the
current authorization model in **LFX Crowdfunding (CF)**, and outlines what it would take to bring
an SS-style role model to CF.

---

## 1. Why this came up

CF currently has no concept of shared ownership or delegated access to an initiative — a single
`owner_id` decides who can manage it, plus a small hardcoded list of platform "approvers." SS has a
more mature multi-person, multi-role access model for orgs and projects. CF has confirmed it needs
multiple users to be owner/editor per initiative, so this doc surveys the SS model as a reference
point for how CF's version of that could work.

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

**Bottom line:** CF has confirmed it needs multiple owner/editors per initiative — a concept that
doesn't exist in its domain model today. SS's full model (tiered roles, org hierarchy, cascading
inheritance) is more than CF currently needs. The direction: each initiative belongs to a parent
**LF project**, and a single flat editor role at the **project level** — checked against the
platform's OpenFGA store via `lfx-v2-fga-sync`, not member-service — grants management of all
initiatives under that project, with the creator retaining edit access to their own initiative.
The main structural prerequisite is the project relationship itself: CF initiatives have no
parent-project column today, so this direction requires a schema change and backfill before any
role resolution can be keyed on it (see §5).

---

## 5. Direction so far

These points were resolved in discussion (July 2026). They are direction, not a spec — each still
needs validation with the platform team before implementation.

### Access model

1. **Multi-person access is a confirmed requirement.** CF needs multiple users to be owner/editor
   per initiative — not speculative.
2. **Every initiative belongs to a parent LF project.** New relationship: CF initiatives today
   have no parent-project column — only `cii_project_id` (CII badge lookup) and
   `jobspring_project_id` (mentorship sync), neither of which is a parent link. Requires a schema
   change plus a **data backfill mapping every existing initiative to an LF project**, including
   General Fund and Event types.
3. **One flat editor role, granted at the project level.** Anyone with the editor/writer role on
   the LF project can manage **every initiative under that project**. No per-initiative grants, no
   view-only tier (can be added later if a real need surfaces). This deliberately skips SS's org
   hierarchy and cascading inheritance, which assume an org structure CF's domain doesn't have.
4. **Access rule: project editor OR creator.** `owner_id` becomes the record of who created the
   initiative and no longer carries general access semantics — with one narrow exception: the
   creator always retains edit access to their own initiative. Without this exception, open
   creation (below) would let a non-editor create an initiative they can never edit again.
   Consequence worth stating: other project editors' access comes and goes with their project
   role; only the creator's access is permanent.
5. **Creation stays open to everyone.** Any authenticated user can create an initiative; the
   "LF Project" field becomes mandatory, with a soft warning ("make sure you are part of that
   project"). Nothing prevents attaching an initiative to a project the creator has no role in —
   the existing approver flow is the backstop against abuse.

### Authorization source: platform OpenFGA, not member-service

An earlier draft of this doc pointed at member-service (by analogy with SS). That analogy was
wrong: per the platform FGA inventory (`lfx-v2-fga-sync/docs/fga-protected-types.md`),
**member-service owns `b2b_org`** — org roles, which is what SS's `OrgRoleGrantsService` reads —
plus only a narrow `project_membership.key_contact` relation. The **`project` authorization object
is owned by `lfx-v2-project-service`**, and platform services check it through
**`lfx-v2-fga-sync`**: a NATS request/reply ("is user U a writer on project P?") backed by
OpenFGA, with a cache-first JetStream layer built in.

**Recommendation:** CF should ask the question through that platform FGA path rather than reading
role lists from any service's data API:

- It is the canonical enforcement source — it captures grants CF can't see in raw data reads
  (committee-derived access, inheritance).
- CF already runs in the LFX v2 shared cluster, so in-cluster NATS is reachable in principle.
- It is the integration every other v2 service uses, so it's the pattern the platform team
  supports.

Fallback if NATS access turns out to be blocked for CF: an HTTP read of project-service's
writer/auditor settings — workable, but weaker (a data read, not an authorization check). Either
way, CF should hide the choice behind a small `ProjectRoleResolver` interface in the domain layer
so the upstream can be swapped without touching business logic.

### Caching

**Recommendation: none on day one.** fga-sync is already cache-first (JetStream KV), so CF does
not need its own SS-style per-username TTL cache initially. Add an in-process cache only if
measured latency demands it. One rule regardless: **fail closed** — if the upstream check is
unavailable, deny management access rather than allow it.

## 6. Still open

- **NATS/FGA access for CF.** Confirm with the platform team that CF (outside Heimdall) can use
  the fga-sync access-check path, and what onboarding it requires (see
  `lfx-v2-fga-sync/docs/fga-catalog.md` for service-owner onboarding).
- **Backfill ownership.** Someone has to produce the initiative → LF project mapping for all
  existing initiatives, including General Fund and Event types. This is a data exercise, not just
  a migration.
- **`allowedApprovers`.** Does the env-var allowlist get folded into the new model (e.g. a
  platform-level FGA relation), or stay a separate platform-admin concept?
- **Frontend gating.** The Nuxt frontend currently has no role awareness; it will need to know
  "can this user manage this initiative" to show/hide management UI (server-enforced either way).

---

## Related Documents

- [`08-self-serve-auth.md`](./08-self-serve-auth.md) — how SS authenticates to CF (identity, not authorization)
- [`02-decisions.md`](./02-decisions.md) — prior architecture decisions
- [`03-open-questions.md`](./03-open-questions.md) — existing open-questions log
