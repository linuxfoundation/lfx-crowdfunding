<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Initiative Attribution & Role-Based Access

---

**Status:** Design proposal, July 2026 — reviewed at the July architecture sync, settled in the
follow-up exchange, then partially superseded 2026-08-25. Three decisions: **(1)** transit — CF
reaches the FGA checks over direct NATS (option C), approved by Eric and Jordan (§3.1); **(2)**
approach — **the hybrid per-entity write gate is deferred, not shipped as an interim step; the
idiomatic `crowdfunding_initiative` type is the plan for the gateway milestone, with no mechanism
before it** (§3.4) — Eric's July initial-step exception is read as superseded by the 2026-08-25
guidance (CF's interpretation, architecture confirmation pending — §6 open question 8); **(3)**
attribution's access-granting half deferred entirely — **permission-aware attachment for both
`b2b_org` and `project` (the M1 org picker's FGA-brokered candidate list and the writer-access grant
for either entity type alike) waits for the gateway milestone** (architecture team, 2026-08-25 —
§3.4 "org attribution" addendum, §3.3, §3.5, §6 open questions 7 and 8). Not yet a spec or
implementation plan. Related story:
[LFXV2-2537](https://linuxfoundation.atlassian.net/browse/LFXV2-2537) *"Initiatives on behalf of
projects and/or organizations"*; epic
[LFXV2-2759](https://linuxfoundation.atlassian.net/browse/LFXV2-2759).

**TL;DR.** Today a Crowdfunding (CF) initiative is manageable by exactly one user (`owner_id`).
The proposal: each initiative carries an **attribution** — *personal* (default), *organization*
(`b2b_org` UID), or *project* (LF project UID) — and one flat access rule follows it:

> **A user may manage an initiative if they are its creator, OR a `writer` on the attributed
> entity** (checked against the platform's OpenFGA store via `fga-sync`).

The same attribution field drives the details-page source label and the Self Serve (SS) lens
"Initiatives" pages. Existing initiatives default to *personal* — **no data backfill**.

---

## 1. Problem

Three confirmed needs, one root cause — CF has no relationship between initiatives and the
platform's real orgs and projects:

1. **Multi-person management.** Only the single `owner_id` can manage an initiative. Teams,
   foundations, and companies running fundraisers need shared access.
2. **Attribution (LFXV2-2537).** An initiative can't be marked as run *on behalf of* a company or
   project; visitors can't tell official project fundraisers from personal ones, and SS lenses
   have no "Initiatives" page for their org/project.
3. **Organization donations.** CF's `organizations` table
   ([001_initial.up.sql:53-61](../../db/migrations/001_initial.up.sql)) is free-form per-user
   (name + avatar + creator), with no uniqueness constraint and no external identifier — two users
   donating for the same company create two unrelated rows that can never be reconciled with the
   orgs SS manages.

### Current access model (for contrast)

| Mechanism | Behavior |
|---|---|
| Ownership (reads) | Owner-scoped reads compare `initiative.owner_id` to the caller's resolved user row; a non-owner gets 404 (`ErrInitiativeNotFound`), deliberately concealing existence |
| Ownership (mutations) | `Update`/`Delete` return `ErrForbidden` → **403** for a non-owner (`initiative_service.go:570,1070`; `respond.go:73`) — existence is not concealed on the write path |
| Approvers | Hardcoded username allowlist from an env var (`allowedApprovers`) gates approve/decline |

Identity note: `owner_id` is the **CF user's database UUID**, not the JWT principal directly. The
service resolves `Principal.Username` (LF SSO username) to a user row and compares `owner_id` to
that row's `ID` (`initiative_service.go:252,570,1070`). `Principal` itself carries identity and
OAuth2 scope only — no role fields. The frontend has no role awareness; all enforcement is
server-side.

---

## 2. Proposed model

### 2.1 Attribution

Each initiative carries exactly one attribution, chosen at creation in a new fundraise-form step
(per LFXV2-2537 — picked from the user's real affiliations, never free text):

| Attribution | Entity reference | Managed by |
|---|---|---|
| `personal` (default) | none | creator only — today's behavior |
| `organization` | `b2b_org` UID (canonical platform org, owned by member-service, backed by Salesforce Accounts) | creator + org writers |
| `project` | LF project UID (owned by project-service) | creator + project writers |

One field, three consumers: **access control**, the **details-page source label**, and the **SS
lens listing pages**. Existing initiatives default to `personal` and behave exactly as today.

> **`b2b_org` UID format is documented two conflicting ways — verify before the M1 migration.** Per
> `lfx-v2-member-service/docs/indexer-contract.md`, a `b2b_org` uid is an 18-char Salesforce Account
> SFID (e.g. `0012M00002qnukOQAQ`); but `lfx-v2-helm/charts/lfx-platform/files/model.fga` describes
> it as "an invertible UUID v8 encoded from the Salesforce Account SFID." Only the `project` UID is
> unambiguously a v2 UUID. The M1 schema and `models.Attribution.Validate()` currently assume UUID
> for both attribution kinds; whether that's a bug depends on which of the two descriptions the live
> index actually uses — check a real `b2b_org` document before relying on either — tracked as a
> pre-migration verification, not settled fact.

**The picker lists affiliated entities only — no free text.** The org/project pickers show only
entities the user is already affiliated with (per LFXV2-2537's functional requirements: no free
text, and a disabled option with an explanation when the user has none). "Any existing org" is
deliberately rejected: attribution is a public claim of representation — the org's name and logo on
a fundraising page — and under the access rule it would also hand the org's writers edit access to
(and lens visibility of) an initiative nobody at the org sanctioned. Escape hatches **deep-link
out of the form** rather than inlining platform workflows: "can't find your org" links to the
platform's org-creation flow; "org exists but I'm not listed" links to where affiliations are
managed. Users with multiple affiliations simply see them all in the single-select.

**Eligibility gate: affiliation, not writer (decided).** A user may attribute to any org/project
they are *affiliated* with — they need not be a `writer` on it (PM decision, 2026-07). This is the
weaker of the two gates: someone affiliated with, but not a writer on, an org can publish a page
carrying that org's name and logo without a writer signing off first. The org's writers *can*
correct or remove it — but only once the gateway milestone ships the access rule that grants them
management (§3.4, §3.5, §5 — deferred from M2, which no longer exists as a separate milestone). Two
consequences follow directly (see §5): the public attribution label cannot ship in a standalone
M1, and server-side validation checks an *affiliation* source, not an FGA `writer` relation.

**New eligibility rule (architecture team, 2026-08): private/formation projects are not
attributable at all.** Even a project *writer* cannot attribute an initiative to a project that
isn't public — attribution is a public claim of representation, and a formation-stage project has
no public identity to claim on behalf of yet. This is enforced the same way the rest of §2.1's
eligibility is: server-side, on the entity lookup at submit time, not by the picker alone. It also
happens to make the project picker's anonymous query-service lookup (§3.3) correct by
construction — a private project would never need to appear there.

**Known gap: the org picker is narrower than the affiliation gate (§3.3).** The org candidate
source is the query service's `filter_grants=direct` filter for `object_type=b2b_org` — it returns
any *direct* tuple on the org (writer, auditor, or owner alike), so it is closer to the affiliation
gate than a writer-only filter would be, but it still cannot see affiliation itself, and it still
excludes **inherited-only** grants (e.g. an org member who only inherits access through a parent
org). That narrower population won't see their org in the picker at all, even though the
affiliation gate above would allow the attribution. Their path is the escape hatch, not the select.
This is a real narrowing of who can use the picker, not just a best-effort miss — see §3.3 for why
it's accepted anyway.

> **Architecture ruling on affiliation data (2026-08, Eric).** Involvement/affiliation data (CDP
> contribution history, persona-service detections) is **self-attested and must never be an
> authorization signal** — anyone in the public can contribute to a project and appear "involved."
> This is platform doctrine (epic [LFXV2-1654](https://linuxfoundation.atlassian.net/browse/LFXV2-1654),
> [permission-persona-navigation-model doc](https://github.com/linuxfoundation/lfx-self-serve/blob/docs/permission-persona-preread/docs/architecture/frontend/permission-persona-navigation-model-preread.md)):
> persona/involvement shapes what a user *sees*; only manage/write permission gates what a user
> *does*. The affiliation gate above is compatible with this ruling **only because attribution
> derives no access for the claimant**: the claim grants access *to* the entity's real FGA writers
> (deferred to the gateway milestone, §3.4/§3.5), never *from* the claim to the claimer, and the
> public label stays suppressed until those writers can police it (§5). What this changes: server-side "affiliation validation" cannot be an
> authoritative check (no authoritative affiliation source exists — involvement is best-effort and
> self-attested), so attribution is a **self-attested claim**; the server validates that the entity
> exists and is the right type, and the false-claim risk is mitigated by label suppression + writer
> policing, not by an affiliation lookup. Confirmation of this framing with the architect is
> tracked in open question 4.

**Eligibility is enforced server-side, not by the picker.** Constraining the dropdown is UX only —
a caller can POST a create/update request with any entity UID directly. The API must therefore
re-validate the submitted attribution *before persisting it*: the entity UID must exist and match
the claimed type (project-service / member-service lookup). Per the ruling above, this validation
is an existence/shape check, not an authorization check — affiliation data cannot authoritatively
prove or disprove the claim. (Picker suggestion sources: open question 4.)

### 2.2 Access decision

**This is the semantic spec for "who can manage an initiative," not a near-term build.** Per §3.4/
§3.5, the write gate ships at the gateway milestone, not before — and by then the mechanism is
likely §3.4's idiomatic `crowdfunding_initiative` type rather than the hybrid per-entity flowchart
below. The flowchart still documents the intended access decision correctly (creator OR entity
writer, fail-closed, one flat capability); only *how* it's evaluated at runtime may change.

```mermaid
flowchart TD
    A[Manage request on initiative] --> B{Caller is creator?<br/>owner_id == caller user row}
    B -- yes --> ALLOW([Allow])
    B -- no --> C{Attribution}
    C -- personal --> DENY([Deny — 404])
    C -- "organization (b2b_org UID)" --> D[fga-sync check:<br/>b2b_org:UID#writer@user:PRINCIPAL]
    C -- "project (LF project UID)" --> E[fga-sync check:<br/>project:UID#writer@user:PRINCIPAL]
    D -- writer --> ALLOW
    E -- writer --> ALLOW
    D -- "definitively not writer" --> DENY
    E -- "definitively not writer" --> DENY
    D -- "resolver error" --> ERR([Deny — 503])
    E -- "resolver error" --> ERR
```

`user:PRINCIPAL` is the caller's **LFID username** — member-service and project-service key their
`writer` tuples by LFID username (per their FGA contracts), and CF already carries it as
`Principal.Username`. So the check CF sends is `entity:UID#writer@user:{username}`.

Design rules:

- **One flat capability.** No view-only tier, no per-initiative grants. Either can be added later
  if a real need surfaces.
- **At most one FGA check per request.** Attribution is exclusive, so the resolver checks only
  the attributed entity's branch; personal initiatives make no FGA call at all.
- **The creator always retains access** (PM-confirmed, 2026-08 — including after the creator
  leaves the attributed org/project; the entity's writers can edit or unpublish the initiative but
  cannot revoke the creator). `owner_id` (the creator's CF user row) stays as
  "created by" and guarantees the creator can always edit — without this, a user could create an
  initiative attributed to an entity they're not a writer on and be locked out immediately.
  Everyone else's access comes and goes with their writer role on the attributed entity. This
  deliberately keeps one access decision (creator) in CF code while entity decisions defer to
  FGA — the trade is discussed in §3.4.
- **Fail closed, but distinguish the reason.** A *definitive* non-writer result denies as 404
  (consistent with today's read concealment). A *resolver error* (NATS/OpenFGA unavailable) also
  denies, but returns **503** — never a false 404 — so the outage is visible and clients can
  retry.
- **CF stores no roles.** No membership tables, no role columns — CF stores one entity reference
  and asks the platform the membership question at request time.
- **Changing attribution is not the flat manage capability.** Editing an initiative's *content*
  is one flat capability; changing its *attribution* is a separate, more restricted action.
  Validating only the target entity (§2.1) is insufficient: under the flat rule, any current
  writer could reattribute the initiative to another entity they write — or to `personal` —
  silently revoking the original entity's writers and moving (or erasing) the public claim of
  representation. An attribution change must therefore be authorized on **both** the current and
  the target entity; transferring a non-personal initiative to `personal` is **creator-only**.
  Tracking *which* writer made a given change is a separate concern, out of scope here (open
  question 6).

---

## 3. Architecture

```mermaid
flowchart LR
    SS[Self Serve<br/>lens pages] -->|HTTP /v1/me/*| API
    FE[CF frontend<br/>Nuxt BFF] -->|HTTP| API

    subgraph CF[Crowdfunding Go API]
        API[Handlers / Services] --> DB[(Postgres<br/>initiatives.attributed_to)]
        API --> RES[EntityRoleResolver]
    end

    RES <-->|NATS request/reply<br/>lfx.access_check.*| FGASYNC[fga-sync]

    subgraph PLATFORM[LFX v2 platform]
        FGASYNC --> KV[(JetStream KV<br/>cache)]
        FGASYNC --> FGA[(OpenFGA store)]
        PS[project-service] -.->|project tuples| FGASYNC
        MS[member-service] -.->|b2b_org tuples| FGASYNC
    end
```

### 3.1 Why fga-sync (and not the alternatives)

| Alternative | Why not |
|---|---|
| Copy SS's model (read role lists from member-service) | SS's `OrgRoleGrantsService` reads **org** roles (`b2b_org`) — the wrong axis for project attribution. It also re-implements resolution (cascading, dedup, 5-min cache) that FGA already computes. |
| Call OpenFGA directly | Not the platform pattern — v2 services go through fga-sync, which provides one shared cache with invalidation (per `lfx-v2-fga-sync/docs/fga-sync-contract.md`). |
| Local role tables in CF | CF would own org/project membership it can't keep correct; the platform already maintains it. |
| Literal org ownership (transfer `owner_id` to an org) | Forces CF to answer "who is in the org" — inventing membership. Attribution + FGA-derived access stores one UID instead. |

fga-sync is the canonical enforcement source (it captures committee-derived and inherited grants
that raw data reads miss), CF already runs in the LFX v2 shared cluster, and it's the integration
every other v2 service uses. Both `project` and `b2b_org` live in the same OpenFGA store, so **one
integration covers both attribution kinds**.

**"Reachable in principle" is now reachable in fact (2026-08-17).** `FGA_NATS_URL` was set to
`nats://lfx-platform-nats.lfx.svc.cluster.local:4222` in
`lfx-v2-argocd/values/global/lfx-crowdfunding-backend.yaml:86` (commit `e882c635`, "set
FGA_NATS_URL for direct NATS access-check transit") — the same value every other v2 service is
handed (e.g. `values/dev/lfx-v2-meeting-service.yaml:42-43`), applied globally rather than
per-environment. DevOps separately confirmed the same day that network-policy connectivity from
CF's pods to `lfx-platform-nats` is open. What's left is CF-side code, not a platform
prerequisite: the `NATSResolver` shipped in M1 (`backend/internal/infrastructure/fga/resolver.go`)
implements only `AccessCheckSubject`/`CanManage` (`lfx.access_check.request`).

**That remaining CF-side work is now deferred, not scheduled for M1.** With the write gate itself
pushed to the gateway milestone (§3.4, §3.5, §6 open question 8), CF builds no
`EntityRoleResolver`/NATS access-check integration in M1 — the platform-side prerequisites above
(`FGA_NATS_URL`, network policy) stay live and cost CF nothing to leave in place, but there is no
near-term consumer for them.

**Correction (2026-08, architecture team): being outside Heimdall is a provider-side constraint,
not a consumer-side one.** An earlier version of this doc read "CF sits fully outside Heimdall" as
"CF cannot reach v2 HTTP APIs" and used that to justify routing the org-picker's name/logo lookup
through Snowflake instead (§3.3). That conflates two different things. CF's own decisions around
access control **as a provider** — no gateway routes to attach `openfga_check` rules to — are what
§3.4 is about, and that section's reasoning is unaffected by this correction. But **as a consumer**,
CF is no different from any other client, human or machine: the v2 query service is a public API
behind the gateway, and any caller — including CF's backend — can call it directly over HTTPS. The
mechanism is not optional but also not a blocker: the query service validates a *Heimdall-minted*
PS256 JWT against Heimdall's own JWKS (`JWKS_URL: http://lfx-platform-heimdall:4457/.well-known/jwks`
per its Helm chart), so a caller must go through the gateway to get a token the service accepts —
which is exactly the normal path for an external consumer, not something CF is locked out of. §3.3
below now uses this path for both halves of the picker; Snowflake is no longer part of the design.

CF still holds only one outbound platform credential: a private-key-JWT M2M client for the **v1**
api-gw (`reimbursement_client.go`), audience `https://api-gw.*.platform.linuxfoundation.org/` — not
an LFX v2 audience, and no RFC 8693 token exchange exists anywhere in the repo. That gap now matters
only for the **org** half of the picker (§3.3, open question 7): public project lookups need no
credential at all, but `b2b_org` is indexed non-public and still needs one.

The fga-sync NATS subjects relevant to CF:

| Subject | Use |
|---|---|
| `lfx.access_check.request` | Batch yes/no checks; each tuple is `{object_type}:{object_id}#{relation}@{user_type}:{user_id}` (e.g. `project:UID#writer@user:alice`) — the edit-access gate, and the batch-verify half of §5.1 |
| `lfx.access_check.read_tuples` | Returns **all direct** OpenFGA tuples for a (user, object_type) — request `{"user": "user:<lfid>", "object_type": "b2b_org"}`, response `{"results": ["b2b_org:<uid>#writer@user:<lfid>", …]}`. Direct-only, so rejected as the §5.1 `ListForUser` candidate source (same reasoning applies to the query service's `filter_grants=direct`, §3.3). **Not adopted by CF** — the org picker's equivalent need is served by the query service's `filter_grants=direct` instead (§3.3), which returns names/logos in the same call; a `read_tuples` client is not on CF's build list |

One contract detail that isn't obvious from the flow: `access_check.request` replies are
**unordered** (cached results may return first, per `fga-sync-contract.md`) — callers must
correlate each result by its echoed request token, never by array position.

**Transit — how CF reaches the check (decided: C).** The architecture sync framed three options:
**(A)** token exchange — swap the CF-audience user token for an LFX v2 token and call the public
`/access-check` HTTP endpoint (the MCP-server precedent; the wrapper is not a drop-in otherwise,
since it expects a Heimdall-minted JWT and derives the principal from it); **(B)** a new privileged
`/user-access-check` HTTP endpoint that accepts arbitrary users, authorized for specific machine
clients via an LFX v2 M2M token; **(C)** direct on-network NATS access to the fga-sync subjects
above. **Eric and Jordan approved C.** The rationale: when CF eventually moves behind the Heimdall
gateway (the §3.4 target state), it adopts NATS anyway for the indexing/tuple-write path — so using
NATS now means its access checks never have to migrate off HTTP later. NATS access control is
**not a prerequisite** (raised at the sync, settled in Eric's follow-up): a platform-owned
operational risk that grows with NATS sprawl, not something CF rolls out in isolation. The
integration hides behind a small `EntityRoleResolver` interface (entity type + UID + principal →
can manage?) so the transport can be swapped without touching business logic if that ever proves
necessary.

**Caching:** none in CF on day one — fga-sync is already cache-first. Add an in-process cache only
if measured latency demands it.

### 3.2 Flow: edit access check

```mermaid
sequenceDiagram
    participant U as User (SS or CF frontend)
    participant API as CF Go API
    participant DB as Postgres
    participant FS as fga-sync (NATS)

    U->>API: PATCH /v1/me/initiatives/{id}
    API->>DB: load initiative (owner_id, attribution)
    alt caller is creator
        API-->>U: 200 OK
    else attributed to org or project
        API->>FS: lfx.access_check.request<br/>entity:UID#writer@user:PRINCIPAL
        Note over FS: cache-first (JetStream KV),<br/>OpenFGA on miss
        FS-->>API: true / false
        alt writer (true)
            API-->>U: 200 OK
        else definitively not writer (false)
            API-->>U: 404 (existence concealed)
        else resolver error (NATS/OpenFGA down)
            API-->>U: 503 (fail closed, retryable)
        end
    end
```

### 3.3 Flow: attribution options in the fundraise form

```mermaid
sequenceDiagram
    participant FE as Fundraise form
    participant API as CF Go API
    participant AF as persona-service (suggestions)
    participant FS as fga-sync (NATS)
    participant QS as v2 query service (HTTP, via gateway)

    FE->>API: GET attribution options
    API->>AF: lfx.personas-api.get (caller's involvement)
    AF-->>API: candidate project UIDs (best-effort)
    API->>QS: GET /query/resources?type=project (anonymous)
    QS-->>API: project names/logos (public only)
    API->>QS: GET /query/resources?type=b2b_org&filter_grants=direct
    QS-->>API: caller's directly-granted orgs, with name/logo
    API-->>FE: suggested projects + organizations
    Note over FE: user picks Personal (default),<br/>an org, or a project — no free text
```

Because the picker is a **suggestion surface, not the gate** (§2.1 architecture ruling — persona
shapes what you see, permission gates what you do), its candidate source may be best-effort:

- **Projects: persona-service** for candidates (`lfx.personas-api.get`, NATS request/reply —
  reachable from outside Heimdall, same transit posture as C), **v2 query service** for
  names/logos and the type-ahead fallback. It aggregates the caller's *involvement*
  (writer/auditor grants, board/committee membership, maintainer detection, mailing lists, meeting
  attendance) and is explicitly **not** an authorization source — acceptable here precisely because
  no access derives from the pick. A type-ahead project search is the fallback for anything the
  best-effort list misses.

  **Names/logos, and the type-ahead search itself, now resolve through
  `GET /query/resources?v=1&type=project`** (open question 4, project half — resolved,
  architecture team, 2026-08). This corrects an earlier premise in this doc (§3.1): CF being
  outside Heimdall constrains it as a *provider*, not as a *consumer* — any client can call the
  public v2 query service through the gateway. For `project`, it's better than "reachable": the
  endpoint runs an anonymous authenticator for public resources, filtering to `public: true` docs
  and returning `name`/`slug`/`logo_url` with `Cache-Control: public, max-age=300` — **no
  credential at all**, and the new eligibility rule above (private/formation projects aren't
  attributable) means the picker never needs to reach past what's public anyway.
- **Organizations — superseded again (architecture team, 2026-08-25, §3.5): deferred to the
  gateway milestone, not built for M1.** The design below was resolved as of open question 4's org
  half, superseding the earlier Snowflake answer — kept here for the reasoning, not as the current
  plan. `GET /query/resources?v=1&type=b2b_org&filter_grants=direct`, one call for both uids
  and names/logos. `filter_grants=direct` narrows the query-service search to resources where
  the authenticated caller has a direct OpenFGA tuple on that object type — any relation, so
  writers, auditors, and owners all match — and the same response documents already carry `name`
  and `logo_url` (per `lfx-v2-member-service/docs/indexer-contract.md`; the same fields Self Serve
  already reads for orgs, `lfx-self-serve/apps/lfx-one/src/server/services/org-role-grants.service.ts:308-325`).
  This is a single HTTP round trip in place of the two-step NATS-plus-Snowflake design the earlier
  version of this doc proposed, and it removes the `lfx.access_check.read_tuples` client entirely
  — nothing in M1 now depends on it (update open question 3 accordingly).

  Unlike `project`, `b2b_org` is indexed non-public
  (`lfx-v2-member-service/docs/indexer-contract.md`: `public: false`), and `/query/orgs` /
  `/query/orgs/suggest` have no anonymous authenticator either — so this call needs a credential.
  ~~Which one is open question 7: try a user-scoped v2 token from the CF backend first (the
  architecture team's preference); fall back to CF's own M2M client, extended with a second
  audience, if that proves impractical.~~ **Superseded (§3.5, architecture team, 2026-08-25):**
  needing a credential at all is the problem, not which credential — see open question 7's
  resolution. Neither the user-scoped-token nor the M2M path is being built.

  **Known gap, restated (§2.1) — now moot for M1.** `filter_grants=direct` is *direct*-tuples-only
  — the same property that disqualifies it as the §5.1 `ListForUser` candidate source (§5.1
  explains why) — so an inherited-only org grant still won't surface here. This gap no longer
  matters for M1 scope now that the org picker itself is deferred (§3.5); recorded for whoever
  picks this back up post-migration.

**This closes the picker's project name/logo question — no Heimdall route or NATS subject needed
for it.** The prior version of this doc routed the org half through Snowflake, believing
query-service was unreachable from outside Heimdall; that premise was wrong (§3.1). The design
above was the intended replacement for both halves, but the org half is now deferred to the gateway
milestone (§3.5) rather than shipped — see §6 open question 4's org bullet and open question 7. The
edit-access check (§3.2) was never affected by any of this — it stays a per-entity `access_check`
over the NATS transit confirmed above.

### 3.4 Considered alternative: an idiomatic `crowdfunding_initiative` FGA type (target state)

> **Read vs. write — the key difference from the hybrid.** Everything above (§2.2, §3.1) is
> **read-only**: CF only *asks* FGA "is this user a writer on this project/org?" and stores its own
> data (creator, attribution) in Postgres. The idiomatic alternative below is the only part that
> would have CF **write** to FGA — because FGA would then own "who can manage this initiative," so
> CF must keep those tuples in sync. That write-sync burden is one reason it's deferred, not the
> plan.

At the architecture sync, Eric flagged the hybrid's OR-union of per-entity checks (project OR
b2b_org OR local creator) as **something of an FGA antipattern** — while also noting that the
fan-out fits a service *outside* the system, a new type is the *inside-the-gateway* pattern, and
the hybrid is fine **as an initial step until CF is behind the gateway**.

**Superseded (CF's reading, 2026-08-25 guidance — §3.5, §6 open question 8): that initial-step
exception no longer stands.** The later guidance's own opening line — permission inheritance from
LF projects was withheld "until the milestone where it had been converted to an idiomatic v2
service" — reasserts the constraint July's exception had waived, and does so for `project`, not
only `b2b_org`. Under that reading there is no interim mechanism: the hybrid does not ship as a
first step, and the idiomatic `crowdfunding_initiative` type below is the plan for the gateway
milestone directly, with the write gate absent until then (§2.2, §5). This is CF's interpretation of
guidance that named org attribution as its concrete case, not a statement architecture has confirmed
against the project gate specifically — flagged for confirmation at gateway-milestone scoping.

The idiomatic alternative he proposed: add a `crowdfunding_initiative` type to the platform
model — `define writer: [user] or writer from project or writer from b2b_org`
(`project_membership` in `model.yaml` is an existing precedent for the shape) — have CF emit
`update_access`/`delete_access` tuples on create/attribution-change/delete (including the creator
as a direct `writer` tuple, so *all* access decisions move to FGA), backfill existing initiatives,
and reduce every runtime check to one `crowdfunding_initiative:{id}#writer` query. A side benefit:
the platform's auto-generated access documentation would then describe the type's roles and
permission inheritance — visibility an in-backend union never gets.

**Non-LF initiatives are the simple case, not a driver.** There is no requirement (nor a timeline
for one) to model a non-LF *project entity* with multiple managers. If non-LF support ever lands,
the scope is the trivial one Eric identified: an initiative with no `project`/`b2b_org` attribution,
managed by its owner only — which today's `personal` attribution already covers, and which the
idiomatic type would cover via the creator's direct `writer` tuple. The more complex path (a CF-
defined non-LF project with its own multi-user membership, e.g. a future `crowdfunding_group` type)
is explicitly out of scope until such a requirement is real.

**This is the right target state, but not the right first step while CF sits outside the API
gateway:**

- **The platform's own contract assumes gateway enforcement.** Per `fga-sync-contract.md`, a new
  model object type ships with Heimdall `openfga_check` rules in the same PR cycle. CF has no
  gateway routes to attach rules to, so `crowdfunding_initiative` would be the first type
  maintained and checked entirely from outside the platform. The only existing external-consumer
  precedent (the MCP server) does read-only brokered checks — exactly this proposal's hybrid.
- **It copies CF-local facts into FGA to read them back.** Creator and attribution live in CF's
  Postgres; the genuinely *external* facts are only "who is a writer on this project/org," which
  the hybrid reads directly. Duplicating local facts adds a sync failure mode (a missed publish
  locks the creator out of their own initiative) whose natural mitigation — falling back to the
  local `owner_id` check — reintroduces the hybrid anyway.
- **Model changes are the platform's most expensive kind.** Relation renames are breaking
  (per the model-evolution policy in `fga-sync-contract.md`); freezing CF's access model into
  `lfx-v2-helm` before the non-LF-initiative design exists is premature.

**Deferral is cheap by design.** The `EntityRoleResolver` seam is the migration path — though with
the hybrid itself now deferred (above), CF builds this seam *at* the gateway migration rather than
swapping into it from an already-shipped interim implementation: when CF moves behind the gateway,
the resolver's implementation is a single `crowdfunding_initiative:{id}#writer@user:{id}` check
from the start, CF adds tuple emission plus a one-time backfill, and handlers/services are written
against the seam once. One migration-time design item to note now: tuple
emission (`update_access`) is asynchronous with no reply, so the migration must define a
read-after-write/convergence strategy — a create or attribution change is otherwise briefly
inconsistent with FGA-backed decisions. The model addition and its Heimdall ruleset then
land together, as the platform contract expects.

### 3.5 Org attribution deferred to the same milestone (architecture team, 2026-08-25)

**New guidance, broader than §3.4's access-check antipattern alone.** Relayed from the
architecture team:

> The "hybrid" approach for crowdfunding was predicated on initiatives being fully decoupled from
> LFX. We didn't bring in the idea of linking to LF projects in a way that could support permission
> inheritance until the milestone where it had been converted to an idiomatic v2 service. Adding
> org permissions falls into the same category — it puts authorization decisions (who can set what
> values on an attribute) into the backend instead of the edge, and it implies some degree of
> republishing entities, which should otherwise be avoided. I'd recommend that attaching b2b orgs
> in a permission-aware way wait until the milestone where this is behind the API GW — the
> authorization decision for setting the property to a given b2b org should be done by the API GW,
> not the backend service.

This extends §3.4's reasoning (deferred until CF is behind the gateway) from *just the M2
access-check hybrid* to *the org attribution feature as a whole*, including the parts §3.3
previously treated as low-risk because they were read-only and not the gate:

- **The M1 org picker's credentialed candidate list is in scope, not exempt.** §3.3 justified the
  `filter_grants=direct` query-service call and its user-scoped-token-vs-M2M question (open
  question 7) on the grounds that the picker is "a suggestion surface, not the gate" — no access
  derives from the pick. This guidance reads that lookup itself as already backend-brokered,
  FGA-informed decision-making about a `b2b_org` attribute — the "who can set what values on an
  attribute" the quote calls out — and says it belongs at the edge regardless of whether the pick
  itself grants access.
- **Practical result: the credentialed org-picker work is dropped, not merely re-sequenced.**
  [LFXV2-3322](https://linuxfoundation.atlassian.net/browse/LFXV2-3322) (the Auth0 custom token
  exchange for a user-scoped v2 token) is **Discarded**, and its implementation,
  [auth0-terraform PR #366](https://github.com/linuxfoundation/auth0-terraform/pull/366), is
  **closed without merging** — see §6 open question 7 for why it wouldn't have been reusable
  post-migration anyway (it bridges an audience mismatch the migration itself removes). CF's
  M2M-fallback path (a second audience on `reimbursement_client.go`) is dropped for the same reason
  — it's the same category of backend-brokered decision, just via a service identity instead of a
  user-scoped one.
- **Project attribution's M1 scope is unaffected.** The project picker (§3.3) uses persona-service
  candidates plus an *anonymous* query-service call — no credential, no FGA-brokered lookup, no
  backend authorization decision about who may see what. Nothing in this guidance's own reasoning
  reaches it.
- **Resolved (CF's interpretation, pending architecture confirmation — open question 8): yes, it
  also revisits project's write gate, not only org's.** The quote's first sentence — permission
  inheritance from LF projects was withheld "until the milestone where it had been converted to an
  idiomatic v2 service" — is read here as reasserting that constraint against §3.4's own
  initial-step exception (which Eric approved for project *and* org alike at the July sync). Under
  this reading, July's approval is superseded: the hybrid write gate does not ship as an interim
  step for either entity type, and §3.4's idiomatic type is the plan for the gateway milestone
  directly (§3.4, §2.2, §5). This is CF's reading, not a statement architecture has made against the
  project case specifically — the guidance's own example was org attribution — so it stays flagged
  for confirmation at gateway-milestone scoping rather than treated as settled.

**M1 scope consequence (§5): the org attribution option has no candidate source left for M1.**
Without a curated, FGA-aware list, offering `organization` attribution would mean either free text
(rejected outright, §2.1 — "no free text" is an LFXV2-2537 functional requirement) or an
uncurated/anonymous org list (defeats the affiliation-gate purpose of the picker entirely, and
`b2b_org` isn't indexed public like `project` — §3.3). **M1 ships `personal` and `project`
attribution only; `organization` attribution is deferred to the gateway milestone**, not scoped
down to a lesser M1 mechanism. §5's milestones table and §6's open question 7 reflect this.

---

## 4. Organization donations (separable)

Independent of attribution/access, CF's free-form `organizations` table needs linking to canonical
platform orgs. Donations and subscriptions FK to these rows, so the table can't be dropped. The
fix:

1. Add a nullable `b2b_org` UID column to `organizations`, plus a **partial unique index on the
   non-null UID** so two donors can't create two rows for the same canonical org (multiple null
   `unlinked` rows are still allowed). Existing duplicates must be merged *before* the index is
   installed.
2. Make the picker path **reuse/upsert** the canonical linked row rather than inserting a new one —
   the unique index alone doesn't prevent concurrent duplicate inserts without upsert-on-conflict.
   **Reuse conflicts with the table's current user-ownership model** (`001_initial.up.sql:55`;
   `organization_repository.go` create/update/delete): today the first creator owns the row and
   can rename or delete it, and a delete nulls every linked donation/subscription FK. A shared
   canonical row can't stay user-owned — the first donor would control an org record others depend
   on. So before introducing reuse, linked (`b2b_org`-backed) rows must become
   **platform-managed/immutable** (name/logo sourced from member-service, not donor-editable, not
   user-deletable), or canonical identity must be separated from the per-user organization record.
   Unlinked rows keep today's per-user ownership.
3. Replace free-text org creation in the donation flow with a **hybrid picker**: typeahead against
   canonical platform orgs first; if the donor's org isn't found, free text still works and
   creates a **local, unlinked CF row** (flagged `unlinked`) — never a platform/Salesforce org
   from a checkout. A donation is never blocked on data plumbing.
4. Reconcile asynchronously: a back-office task matches `unlinked` rows against Salesforce
   Accounts and links or escalates them. This is the same mechanism as the dedup of existing rows,
   so the escape hatch adds no new machinery.

No *affiliation* check for donating — attribution and donation carry different risk and get
different gates: attributing an initiative **claims authority** to raise money in the org's name
(strict, affiliated-only, §2.1), while donating **gives** money in its name (lenient, hybrid
picker). Accepted consequence: unlinked-org donations don't appear in the SS Organization lens
until reconciled — correct behavior, not a bug.

---

## 5. Milestones

M1 and M3 ship independently (M1 with the public label suppressed — see the coupling note below).
The former M2 scope no longer exists as a separate milestone: per §3.4/§3.5 (CF's reading of the
2026-08-25 guidance, §6 open question 8), the write gate for either attribution type waits for the
gateway milestone, so "multi-person management" folds in there instead of following M1 directly. M3
can move ahead of M1 and the gateway milestone alike.

| # | Scope | Delivers |
|---|---|---|
| M1 | **Attribution foundation, `personal` + `project` only (`organization` deferred, §3.5)** — schema (`attributed_to` type + entity UID, plus nullable benefit-project field per resolved OQ1; schema keeps the `organization` enum value so no migration is needed later), form step with affiliation pickers (projects: persona-service candidates + anonymous query-service names/logos + type-ahead, §3.3; ~~orgs: query-service `filter_grants=direct`, one call for uids and names/logos, §3.3 — needs a credential, open question 7~~ **org picker dropped from M1 scope — no FGA-brokered candidate source is being built, §3.5**), **server-side entity validation** (existence/type, not authorization — §2.1; includes the new private/formation-project exclusion), details-page source label **suppressed until the gateway milestone** (§5 coupling). No access changes. | Most of LFXV2-2537 (org half and access deferred) |
| Gateway milestone | **Everything the write gate touches, for both attribution types, folded into one delivery (§3.4, §3.5, §6 open question 8)** — the idiomatic `crowdfunding_initiative` FGA type (§3.4: tuple emission, backfill, Heimdall `openfga_check` rules), `organization` attribution and its credentialed picker (§3.5, dropped from M1), writers manage attributed initiatives, frontend "can manage" signal, SS lens "Initiatives" pages (authorization-aware — entity writers also see unpublished initiatives), and the access-aware `ListForUser` query plan (§5.1). These were already converging on the same milestone from two directions (org attribution via §3.5, the write gate via §3.4's target state) — this folds the last piece (project's write gate) in alongside them. | Multi-person management, `organization` attribution |
| M3 | **Org donations cleanup** — `b2b_org` link + partial unique index + upsert, canonical-org picker, dedup | Reconciled org donors |

**The gateway milestone must migrate every owner-gated path, not just editing.** Today `owner_id`
gates more than Update/Delete: `ListForUser` hard-filters `owner_id` (the CF management list),
private/unpublished initiative reads, owner-scoped transaction reads, and announcement
create/update/delete (in `AnnouncementService`). If only the edit path routes through the resolver,
entity writers get inconsistent partial access — e.g. they could edit an initiative but not see it
in their list or manage its announcements. The "one flat capability" rule requires all of these to
move together.

**`ListForUser` needs an access-aware query plan (gateway-milestone prerequisite) — see §5.1.** The
single-entity boolean resolver works for per-initiative gates (edit, read, delete) but not for
*discovery*. Today `ListForUser` filters `WHERE i.owner_id = $1`, counts, then paginates
([initiative_repository.go:230,261,308](../../internal/infrastructure/db/initiative_repository.go)).
Extending that to "initiatives I can manage" is not a point check applied per-row — the plan is
below.

### 5.1 The `ListForUser` query plan

The list must return every initiative the caller owns **or** is a writer on the attributed entity
of, correctly paginated. Two naive approaches both fail:

- **Check each row after pagination** — the point resolver applied to a fetched page produces
  short pages (some rows drop out) and wrong totals.
- **Check every initiative before pagination** — an FGA call per initiative doesn't scale.

The correct shape inverts it: **resolve the caller's writable entity set first, then push it into
the existing SQL as a second `WHERE` branch.**

1. **Resolve the writable set — from CF's own data (resolved 2026-08).** The set of
   `project`/`b2b_org` UIDs the caller can write, *including inherited writers* (e.g. a
   parent-project writer). Two platform-side sources were evaluated and are dead:
   - **FGA enumeration (`list-objects`) is a platform anti-pattern.** A draft fga-sync subject
     existed ([lfx-v2-fga-sync#57](https://github.com/linuxfoundation/lfx-v2-fga-sync/pull/57)) but
     is being closed: OpenFGA's ListObjects is "explicitly NOT guaranteed to be affordable," and the
     platform (including the Query Service) is deliberately built to *avoid* that call, using batch
     permission checks instead (Eric,
     [LFXV2-2753 comment](https://linuxfoundation.atlassian.net/browse/LFXV2-2753?focusedCommentId=116226);
     Jordan, 2026-08).
   - **Direct-grant reads are not a valid superset.** `read_tuples` (and `filter_grants=direct`
     Query Service reads) return **direct** tuples only — an inherited-only writer has *zero* direct
     tuples yet can evaluate as writer on 1,000+ projects (verified in prod, LFXV2-2753). An omitted
     candidate can never be recovered by batch-verify.

   The resolution is to invert the candidate source: **CF's own `initiatives` table bounds the
   candidates.** The only entities that can matter for this list are the distinct
   `(attributed_to_type, attributed_to_id)` pairs that actually appear on non-personal CF
   initiatives — a set bounded by CF's own data, not by the platform's entity universe. So:

   ```
   SELECT DISTINCT attributed_to_type, attributed_to_id
     FROM initiatives WHERE attributed_to_type <> 'personal'
   ```

   then confirm `writer` on each pair with **one batched `access_check.request`** (transit C — the
   subject already accepts multiple checks per message, and `Check` evaluates inheritance, so
   inherited writers are included for free). This is the same batch-check-as-filter shape the Query
   Service itself uses, needs **no new platform capability**, and is correct by construction: the
   candidate set is a superset of every entity that could put an initiative in the caller's list.

   **This shape may itself be superseded if §3.4's idiomatic `crowdfunding_initiative` type ships
   directly (§3.4, §6 open question 8).** A single `crowdfunding_initiative:{id}#writer` relation
   turns "which entities can the caller write" into a question FGA answers about initiatives
   themselves, which changes the discovery problem this candidate-set-then-batch-check design
   solves. Kept here as the design for the gateway milestone unless/until the idiomatic type
   changes the shape of the problem.

   **Bounding note:** the batch size grows with the number of *distinct attributed entities across
   CF*, not per-user. At CF's scale that is small; the per-user result is cacheable (short TTL,
   step below), and if the distinct-entity count ever grows past a comfortable batch size, chunk
   the batch and monitor — never silently truncate.
2. **Extend the SQL, don't post-filter.** The writable set becomes a second branch on the existing
   query, matched as **(attribution type, entity UID) pairs** — not bare UIDs — so a writable
   `project` UID can never authorize a `b2b_org`-attributed initiative (or vice versa):
   `WHERE i.owner_id = $1 OR (i.attributed_to_type, i.attributed_to_id) IN ((…,…), …)`. Count and
   `LIMIT/OFFSET` pagination then work unchanged — totals and page sizes stay correct because the
   filter is applied *in* the query, not after it.
3. **Bound the set.** If a caller writes an unusually large number of entities, keep the filter
   database-side: pass the set as an array parameter (`= ANY($n::uuid[])` per attribution type)
   or a joined temporary relation instead of an inline `IN` list, so sort, count, and pagination
   stay in the query at any set size. If a hard cap is ever imposed, log when it is hit (no
   silent truncation).

Net: the writable-set resolution is one call per list request (cacheable per-user for a short TTL,
mirroring SS's 5-min role cache), not one call per initiative. The SQL change is additive — the
existing sort, search, and pagination are untouched.

**M1/gateway-milestone coupling under affiliation eligibility (now in force).** Eligibility is
*affiliation*, not *writer* (§2.1). So a standalone M1 would publish an entity's public label for a
creator who may not be a writer, while the entity's writers can't correct or remove it until the
gateway milestone grants them management. M1 therefore cannot ship the public attribution label
alone. **Resolution: ship M1's data model and pickers with the public project label suppressed
until the gateway milestone** (reflected in the M1 row above) — the attribution is captured and
validated, just not shown publicly until the people who can police it have the tools to. Coupling
M1 into one release with that milestone was the rejected alternative; suppression keeps M1
independently shippable.

**Consequence, stated plainly rather than glossed: this is now a longer wait than originally
scoped.** The gateway milestone folds in the entire write gate for both attribution types (§5
table above), not just the org half — so M1's captured-but-suppressed attribution stays invisible
on the details page for the full duration of that milestone, whatever it turns out to be, not a
near-term M2 that follows shortly after M1. Accepted knowingly per §3.5's M1-scope consequence
(`organization` attribution deferred outright) and §6 open question 8's resolution above — not
revisited here as a separate open question, but flagged so it isn't mistaken for an oversight.

Scope-reduction lever: ship the Project lens page before the Organization lens page (the
maintainer story is the strongest).

---

## 6. Open questions

1. ~~**PM: benefit vs. attribution axis.**~~ **Resolved (PM, 2026-08): yes** — add a separate,
   nullable benefit-project field to the M1 schema, independent of attribution. The PM's framing
   ("it should always be clear which project a donation is attributed to") suggests the benefit
   relation may eventually be *required*; the schema is nullable either way, and requiredness is
   API/form validation to settle before M1's form ships — not a migration.
2. ~~**Eligibility vs. access populations.**~~ **Resolved (PM, 2026-07): affiliation.** A user may
   attribute to any org/project they are *affiliated* with; they need not be a `writer`. Details
   and consequences: §2.1 and the M1/gateway-milestone coupling note in §5.
3. ~~**Platform onboarding for NATS (transit C).**~~ **Resolved on the platform side (2026-08-17).**
   `FGA_NATS_URL` is now set globally (`lfx-v2-argocd/values/global/lfx-crowdfunding-backend.yaml:86`,
   commit `e882c635`), and DevOps confirmed the same day that network-policy connectivity from CF's
   pods to `lfx-platform-nats` is open, per `lfx-v2-fga-sync/docs/fga-catalog.md`. (NATS access
   control is explicitly *not* a prerequisite — a platform-owned operational risk, per Eric's
   follow-up; §3.1.) **Remaining work is CF-side, not platform-side, and is now smaller than
   originally scoped — but also deferred, not scheduled for M1:** `resolver.go` only implements
   `lfx.access_check.request` (`AccessCheckSubject`/`CanManage`), which is all it would need to; the
   org picker no longer depends on a `lfx.access_check.read_tuples` client (superseded, §3.3: the
   org picker moved to the v2 query service's `filter_grants=direct`, then dropped from M1 entirely,
   §3.5). With the write gate itself deferred to the gateway milestone (§3.4, §3.5, open question
   8), there is no near-term consumer for this resolver work either.
4. **Candidate enumeration — largely resolved (2026-08); org picker source superseded again, gate
   framing remains open.** The shared "enumerate a user's entities" dependency dissolved once the
   platform ruled FGA enumeration out (ListObjects is an anti-pattern; fga-sync PR #57 being closed
   — see §5.1):
   - ~~**Gateway-milestone writable-set**~~ **Resolved:** CF-local candidates (distinct attributed
     entities from CF's own `initiatives` table) + one batched `access_check` — no platform work
     needed (§5.1). (Originally scoped to "M2"; that milestone folded into the gateway migration,
     open question 8.)
   - ~~**M1 project picker**~~ **Resolved (architecture team, 2026-08 — superseding the earlier
     answer).** persona-service supplies candidates (unchanged); names/logos and type-ahead now
     come from the v2 query service's `GET /query/resources?type=project`, called **anonymously** —
     project docs are indexed `public` and the endpoint runs an anonymous authenticator, so no
     credential or M2M grant is needed at all (§3.3). This corrects the earlier premise that CF
     couldn't reach v2 HTTP APIs from outside Heimdall (§3.1) — that constraint is provider-side,
     not consumer-side.
   - ~~**M1 org picker source**~~ **Deferred (architecture team, 2026-08-25 — supersedes the
     2026-08 re-resolution below, §3.5).** The org picker itself — not just its credential — is out
     of M1 scope. `organization` attribution ships once CF is behind the gateway, alongside §3.4's
     idiomatic type. The 2026-08 answer is kept below for the reasoning it recorded, not as the
     current plan.
     <details><summary>2026-08 answer (superseded 2026-08-25, kept for history)</summary>

     `GET /query/resources?v=1&type=b2b_org&filter_grants=direct` — one
     HTTP call returns the caller's directly-granted org uids *and* their names/logos together
     (§3.3), replacing the two-part `read_tuples` + Snowflake design entirely: no nightly CronJob,
     no new CF-local Postgres table, no Snowflake grant request. `b2b_org` is indexed non-public,
     so this call still needs a credential — tracked as open question 7, not blocking this
     resolution. Known narrowing, revised: `filter_grants=direct` matches any direct relation
     (writer, auditor, owner), which is *closer* to the affiliation gate than the earlier
     writer-only filter — the remaining gap is inherited-only grants (§2.1).
     </details>
     <details><summary>Superseded answer (kept for history)</summary>

     `lfx.access_check.read_tuples` (`object_type=b2b_org`, filtered to `#writer`) for UIDs,
     Snowflake (`ANALYTICS.PLATINUM_LFX_ONE.ORG_LENS_ACCOUNT_CONTEXT`, synced nightly into
     Postgres) for names/logos — reasoned at the time that this needed no Heimdall route or
     platform ask, since CF already held Snowflake credentials for `mentorship-sync`. That
     reasoning rested on the premise (since corrected, §3.1) that CF could not reach v2 HTTP APIs
     from outside Heimdall.
     </details>
   - ~~**New, still open: project name/logo half**~~ **Resolved** by the same v2 query-service
     answer above — it was the missing half, not a separate dependency.
   - ~~**Gate framing.**~~ **Resolved (PM, 2026-08).** Attribution stays a self-attested claim — no
     access derived from it, entity existence validated server-side, public label suppressed until
     gateway-milestone writer policing (§2.1 ruling note). Both sub-parts raised by architecture review: (i)
     excluding private/formation projects (§2.1) is confirmed sufficient — no further framing
     needed beyond the public/private line; (ii) the affiliation-vs-writer gate stands as-is, with
     `filter_grants=direct` accepted as the practical org-picker source even though it's narrower
     than affiliation and broader than writer-only. Both are application-level policy, not a schema
     or migration decision — either can be revisited later without a data migration if the
     architecture team's formal sign-off (still pending, informational) lands differently.
   - **Org-picker credential question — see open question 7.** Originally raised here; promoted to
     a top-level item below once it was resolved, since it no longer depends on the candidate-
     enumeration framing this item groups.
5. **`allowedApprovers`.** Fold the env-var allowlist into the new model, or keep it as a separate
   platform-admin concept?
6. **Edit attribution once multiple writers exist.** Neither `initiatives` nor
   `initiative_announcements` tracks *which* writer made a given change today — `initiatives` has
   no `updated_by`, and `initiative_announcements.created_by` is stamped once at creation and never
   revisited by later `PUT`s, so an announcement edited by a second writer still displays the
   original author. That's harmless while an initiative has exactly one manager; once the gateway
   milestone lets multiple org/project writers manage the same initiative, "who changed this"
   becomes answerable only from the last `updated_on` timestamp, not a which-writer record. Decide
   before that milestone ships whether that's acceptable or whether initiatives and announcements
   need a minimal `updated_by` column (not a full audit/edit-history log unless a real need
   surfaces). Confirm with PM.
7. ~~**The credential for the org picker's `filter_grants=direct` call.**~~
   **Moot (architecture team, 2026-08-25, §3.5): there is no credential to choose, because the
   org picker isn't being built for M1.** The original framing (below, kept for history) picked
   between a user-scoped v2 token and an M2M fallback; both are backend-brokered authorization
   decisions about a `b2b_org` attribute, which is exactly what the 2026-08-25 guidance says
   belongs at the gateway instead. Consequently:
   - [LFXV2-3322](https://linuxfoundation.atlassian.net/browse/LFXV2-3322) (Auth0 token-exchange
     plumbing for the user-scoped path) is **Discarded**, and its implementation,
     [auth0-terraform PR #366](https://github.com/linuxfoundation/auth0-terraform/pull/366), is
     **closed without merging**. It wasn't just deferred-and-shelved: the gateway migration
     removes the audience mismatch it bridges (CF's Auth0-audience tokens vs. Heimdall-audience
     ones), so there's nothing to resume post-migration either — see the PR's closing comment for
     the full reasoning.
   - The M2M fallback (a second audience on `reimbursement_client.go`) was never built and is
     dropped for the same reason, not picked as the alternative.
   - LFXV2-3323 (the CF-side consumer of the exchange) and its branch
     (`feat/lfxv2-3323-org-token-exchange-client`) are moot for the same reason but untouched —
     no PR exists for it and no action has been taken on it.

   <details><summary>Original framing (superseded 2026-08-25, kept for history)</summary>

   The architecture team's preferred order: attempt a user-scoped v2 token from the CF backend
   first, since CF's frontend already holds a per-user session (LFXV2-2537's premise that "any
   frontend can call the new LFX API with user-scoped tokens" per the architecture reply);
   fall back to CF's existing M2M client (extended with a second, v2 audience) if that proves
   impractical during implementation. Not free either way: CF's BFF currently mints and refreshes
   an Auth0 token scoped to *CF's own* audience only (`frontend/server/routes/auth/callback.ts:111`,
   `frontend/server/api/auth/refresh.post.ts:51`, scope `openid profile email offline_access
   access:me` at `frontend/server/api/auth/login.get.ts:56`) — reaching the v2 API with a
   user-scoped token needs a second Auth0 authorization or an RFC 8693 exchange, neither of which
   exists in the repo today. The M2M fallback is comparatively cheap: a second audience on the
   private-key-JWT client already built for the v1 api-gw (`reimbursement_client.go`), plus an
   FGA tuple granting `auditor` (or `writer`, per what the picker actually needs) to the M2M
   client's platform identity, `user:<client_id>@clients` — not a scope, since platform
   permissions are plain FGA tuples on that principal, not something `PERMISSIONS.md` or an
   `access:api`-style scope currently encodes.
   </details>
8. ~~**New (§3.5): does the 2026-08-25 guidance also revisit project's write gate?**~~ **Resolved
   (CF's interpretation, pending architecture confirmation — §3.4, §3.5).** The architecture reply
   names "adding org permissions" as the concrete case, and §3.5 applies that reasoning to *all* of
   org attribution (M1 picker included, not just the access check). CF reads the same reasoning as
   reopening the July sync's separate, already-approved decision to ship the hybrid per-entity
   OR-union check (§2.2, §3.1) as project's *own* write-gate mechanism, even though that approval
   predated this guidance and wasn't itself about attaching a new attribute — just about which FGA
   check pattern to use for an entity CF already writes. **Practical effect:** the hybrid does not
   ship as an interim mechanism for project either; the write gate for both attribution types waits
   for the gateway milestone, where §3.4's idiomatic `crowdfunding_initiative` type is the plan
   directly (§5 milestones table — the former M2 folds into that milestone). This is CF's reading of
   guidance whose stated example was org attribution, not a ruling architecture has made against the
   project case by name — confirm with architecture before gateway-milestone scoping rather than
   treating this as settled.

---

## Appendix: how Self Serve does it (reference)

SS's role model was the starting reference but is **not** what CF adopts. In short: SS reads
**org** roles (writer/auditor, with auditor-only parent→child cascading) from member-service
(`OrgRoleGrantsService`, per-username 5-min cache), and per-project staff (`view`/`manage`) from
project-service; capability gates only ever use *direct* roles, inherited ones are display-only.
It is a data-read model, not a live policy check. CF instead needs project- *and* org-scoped
access with one mechanism — which is what the FGA check path provides without re-implementing
SS's resolution logic.
