<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Org Donations — 13: Requirements and Organization List Source

---

**Status:** Investigation / draft requirements, October 2026. Not yet signed off by PM or
architecture. Tracked in
[lfx-crowdfunding#366](https://github.com/linuxfoundation/lfx-crowdfunding/issues/366) (epic
[#365](https://github.com/linuxfoundation/lfx-crowdfunding/issues/365); background in
[#261](https://github.com/linuxfoundation/lfx-crowdfunding/issues/261)). Implementation follows
in [#367](https://github.com/linuxfoundation/lfx-crowdfunding/issues/367).

**TL;DR.** Today "donate as an organization" picks from a private, per-user list of free-text
orgs that nothing else in LFX knows about. PM's preferred source for the org list is the user's
Self Serve **Work history & Affiliations**. That data lives in **CDP (crowd.dev)** and is keyed by
**CDP organization UUIDs**, not Salesforce Account IDs / `b2b_org` UIDs. The alternatives are
`b2b_org` (member service; covers only Salesforce Accounts with a membership) and raw Salesforce
Accounts. Each source trades off coverage against being canonical.
**Leaning: option A (CDP work history)**, pending architecture review (§4.1). §5 lists the
decisions needed.

---

## 1. Current state in Crowdfunding

### 1.1 How it works

**Schema.** `organizations` was created in `backend/db/migrations/001_initial.up.sql:53-61` and
no migration since has changed it:

```sql
CREATE TABLE IF NOT EXISTS organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id UUID NOT NULL REFERENCES users(id),
  name TEXT NOT NULL, avatar_url TEXT, status VARCHAR(50),
  created_on TIMESTAMPTZ DEFAULT NOW(), updated_on TIMESTAMPTZ DEFAULT NOW());
```

- Apart from the `id` primary key, the table has no unique constraint (nothing on name or any
  business key) and no external or canonical ID. Its only other index is on `owner_id`.
- `donations.organization_id` and `subscriptions.organization_id` reference it with
  `ON DELETE SET NULL` (`001_initial.up.sql:363`, `:385`).
- 606 orgs were migrated from v1 (`05-migration-plan.md:367`). The migration nulled out donation
  `orgId` values that pointed at unknown orgs (`backend/db/scripts/migrate_dynamo_to_postgres.py:1271-1305`).

**API.** All routes are user-scoped: `GET/POST /me/organizations` and
`PATCH/DELETE /me/organizations/{id}` (`backend/cmd/initiatives-api/server.go:294-297`).
- The only validation is a non-empty name (`backend/internal/service/organization_service.go:55`).
- Create is a plain `INSERT` with no upsert (`backend/internal/infrastructure/db/organization_repository.go:86-89`).
- Update and delete are scoped to `owner_id`.
- v1's `exists/{name}` and approval endpoints were not carried over.

**Donation flow.**
- *Picking an org.* Step 2 of the donate drawer, "I'm donating as"
  (`frontend/app/components/modules/donate/components/donate-step-contact.vue`), lists
  "Individual" plus the **user's own** orgs. "Add organization" opens
  `donate-add-org-modal.vue`, which takes a free-text name and requires a logo upload. It has
  no search and no duplicate warning.
- *Ownership check.* The backend accepts `organization_id` only if the caller owns the row
  (`ownedOrg()`, `backend/internal/service/donation_service.go:358-382`). This is marked as an
  interim guard until #261.
- *Where the org ID goes.* Stripe metadata gets `org_id` (the CF UUID) and `org_name`
  (`stripe_client.go:284-318`). The webhook forwards `org_id` to Ledger as `OrganizationID`
  (`webhook_handler.go:214`), so **Ledger stores the CF-local UUID**.
- *Display.* The initiative page enriches donations live from `organizations`: transactions
  (`initiative_service.go:1434-1483`), sponsors (`cmd/ledger-stats-sync/syncer.go:238-269`) and
  top-org stats (`statistics_service.go:113-121`). If the row is missing, the donor shows as
  "Anonymous", with an empty name, or as "Unknown Organization".
- *Featured companies.* "Investing companies" is a hardcoded list of 16 org UUIDs
  (`backend/internal/config/featured_orgs.json`). These point at **user-owned** rows.

### 1.2 Why it doesn't work

1. **Duplicates by design.** Two employees of the same company create two unrelated rows, and
   one user can create the same org twice. Sponsor lists, Ledger totals and top-org stats show
   the company more than once. The CSV summary merges rows only on an exact name match.
2. **No canonical identity.** Nothing links a row to Salesforce, `b2b_org` or CDP. Org donations
   never show up in the Self Serve Organization lens. Initiative attribution already uses
   `b2b_org` SFIDs (migrations 007/008), so CF now has two incompatible "organization" concepts.
3. **Private ownership.** Only the creator can donate as "their" org. Coworkers can't reuse it.
4. **Edits rewrite history.** A rename changes the name shown on every past donation. A delete
   turns past org donations into individual ones (`SET NULL`), while Ledger still holds the
   orphaned UUID.
5. **Self-asserted, unvalidated data.** Anyone can donate as "Google" with any logo. The only
   check is a non-empty name. The logo URL isn't validated on the server, and the v1 approval
   workflow is gone (the `status` column is never written).
6. **Featured companies are fragile.** The curated list depends on whoever owns each row.
7. **No user docs.** `docs/user/donations/make-donation/index.md` skips the "I'm donating as"
   step entirely.

---

## 2. PM direction

PM wants the donor's org list to come from the orgs the user has added under **Self Serve →
Profile → Work history & Affiliations**. This can change depending on architecture constraints
and decisions. §3 records how that data is stored, and §4 compares it with the alternatives.

---

## 3. Self Serve "Work history & Affiliations": investigation

Repo: `linuxfoundation/lfx-self-serve` (paths relative to `apps/lfx-one/src/` unless noted).

### 3.1 Where work history is stored

**Work history lives in CDP (crowd.dev)**, as member → organization rows (`memberOrganizations`).
It isn't stored in Salesforce, member service, NATS KV or Auth0 metadata.

- **UI.** The tab is `app/modules/profile/attribution/profile-attribution.component.*`
  (route `attributions`). It has two sections:
  - "Work history" (`profile-work-experience.component.ts`)
  - "Project Involvement" (CDP project affiliations)
- **Entry fields.** Organization, role/job title, start month/year, and end month/year or
  "currently work here" (`work-experience-form-dialog.component.ts:39-49`).
- **Write path.**
  1. `POST /api/profile/work-experiences` (`server/routes/profile.route.ts:165`) runs behind the
     user's OIDC session.
  2. `profile.controller.ts:1674-1708` adds `verified: true, verifiedBy: <lfid>`.
  3. `server/services/cdp.service.ts` calls CDP. It first resolves the user to a CDP member with
     `POST {CDP}/v1/members/resolve`, creating the member if missing. Then it calls
     `POST {CDP}/v1/members/{memberId}/work-experiences`.
  4. Self Serve authenticates with an Auth0 client-credentials token for the CDP audience.
     The env vars are `CDP_API_URL`, `CDP_AUDIENCE` and `PCC_AUTH0_*`.
- **Org identifier stored.** A **CDP organization UUID**. The shared type says so explicitly:
  *"CDP internal organization ID (not a Salesforce SFID — never forward to v1 Project Service)"*
  (`packages/shared/src/interfaces/organization.interface.ts:34`).
- **Read API.** Self Serve's `GET /api/profile/work-experiences` only works for the logged-in
  Self Serve session, so other services can't call it. The CDP calls behind it are:
  1. `POST /v1/members/resolve {lfids:[username]}` (scope `read:members`)
  2. `GET /v1/members/{memberId}/work-experiences` (scope `read:work-experiences`)

  Both need an Auth0 M2M token with the CDP audience. **CF can't call these directly**: the
  architecture team has ruled out giving external consumers CDP `read:members` access (§5.2
  item 3).

### 3.2 Where the autocomplete list comes from

The org field (`app/shared/components/organization-search/`) calls
`GET /api/organizations/search?query=` (minimum 2 characters, 300 ms debounce). The BFF
(`server/services/organization.service.ts:120-164`) merges three lookups that run in parallel:

| Lookup | Backend | Behavior |
|---|---|---|
| `GET {LFX_V2_SERVICE}/query/orgs/suggest` | lfx-v2-query-service → **Clearbit** autocomplete | Fuzzy, broad coverage of companies worldwide. Returns name/domain/logo, **no ID** |
| `GET {CDP}/v1/organizations?name=` | **CDP** | **Exact** case-insensitive name match, only orgs with a verified primary domain |
| `GET {CDP}/v1/organizations?domain=` | **CDP** | Only when the query looks like a domain |

CDP hits are listed first and have a 3 s timeout. Orgs the user picked or created during the
session are also cached in the browser (`sessionOrgs`, not persisted). **None of these lookups
touch Salesforce or `b2b_org`.**

### 3.3 Creating an organization during affiliation entry

If nothing matches, the dropdown footer offers **"I want to create '<term>'"**
(`organization-search.component.html:45-54`).

- **Fields.** Name plus **Organization Website** (https, required in the work-history dialog).
  There is no logo field; a logo is included only when the user picked a Clearbit suggestion.
- **Request.** `POST /api/organizations/resolve {name, domain, logo?}`
  (`organization.controller.ts:73-128`), handled by `cdp.service.ts:890-913`. Picking a Clearbit
  suggestion goes through the same resolve call.
- **Find-or-create in CDP.** The BFF strips the URL to its hostname. If
  `GET /v1/organizations?domain=` finds an org, it reuses it. Otherwise it calls
  `POST {CDP}/v1/organizations {name, domain, source:'lfxOne'}`.
- **CDP dedup** (crowd.dev `findOrCreateOrganization`) checks a verified identity, then a
  verified domain, then **display name**. A name match attaches the new domain to the existing
  org, so a typed name can attach a domain to an unrelated org with the same name (inferred from
  code).
- **No review or approval, and no Salesforce Account.** The CDP org exists immediately, and its
  UUID is returned and stored on the work-history entry.
- **Searchability afterwards.** The Clearbit typeahead never returns it. Only the exact-name or
  domain CDP lookups find it.

### 3.4 What this means for CF

- Work-history orgs are **CDP org UUIDs**, so they don't join to `b2b_org`/Salesforce or to
  Ledger without a mapping.
- That mapping exists today **only in Snowflake**. Self Serve's org lens reads `ACCOUNT_ID ↔
  CDEV_ORG_ID` from `ANALYTICS.PLATINUM_LFX_ONE.ORG_LENS_ACCOUNT_CONTEXT` and
  `ACCOUNT_TO_CDEV_ORG_MAPPING` (`server/services/organization.service.ts:1043-1070`). Coverage
  of arbitrary orgs, and of lookups in the CDP → SFID direction, hasn't been verified.
- Orgs created by users in Self Serve are **self-asserted**, just like CF's current orgs: there's
  no review and dedup is weak. The difference is that they're **shared across users** (one CDP
  org per domain) instead of private per user.
- Using work history limits CF to orgs the donor has already added in Self Serve. **Decided
  (designer):** CF won't add orgs inline. If the donor's org isn't listed, CF directs them to
  Self Serve's Work history & Affiliations tab to add it.

---

## 4. Org list sources compared

| | **A. Self Serve work history (CDP)** | **B. `b2b_org` (member service)** | **C. Salesforce Accounts** |
|---|---|---|---|
| ID | CDP org UUID | 18-char SFID | SFID |
| Coverage | Any org a user has added; many | **Only Accounts with a membership Asset** (`account_repo.go:25-46`) | All Accounts |
| Canonical / joins to SS org lens | Via Snowflake mapping only | **Yes** | Yes |
| User ↔ org link | **Yes** (that's what work history is) | Only roles: writer/auditor/key contact | No |
| Search API | CDP exact name/domain lookups; no fuzzy search | `GET /query/resources?type=b2b_org&name=`, non-public: results filtered to orgs where the caller has `auditor`; CF has no credential today | Member service removed its SOQL search; no LFX API exposes it |
| Create new | Self-serve find-or-create, no review | **No.** `POST /b2b_orgs` only registers an *existing* SF Account, restricted to org-admin clients | Out of scope from checkout |
| CF access today | None (needs a v2 API behind Heimdall; direct CDP access ruled out, §5.2) | None for listing | None |

**Notes on each source:**

- **A** best matches PM's intent ("orgs I work for"). It covers non-member companies, and it
  verifies affiliation in the same loose way Self Serve does. Costs:
  - a new v2 API, behind Heimdall, that reads the caller's work history from CDP. CF can't get
    its own CDP M2M client (§5.2 item 3), so this is a cross-team dependency;
  - CDP IDs in CF and Ledger that need mapping to SFIDs before org donations can show in the
    org lens;
  - self-asserted orgs.
- **B** is the canonical record and already used for initiative attribution
  (`11-initiative-attribution-and-access.md` §4). But it covers only member companies, so most
  donor companies aren't in it. Listing needs a credential with `auditor` on all `b2b_org`s,
  which is the same blocker that deferred the M1 org picker (doc 11, open questions 4 and 7).
- **C** has the widest canonical coverage, but no supported LFX API searches it, and checkout
  must never create Accounts.

Clearbit (`/query/orgs/suggest`, an external third-party company-data API) isn't listed as a
source. It returns no stable ID, so it can't be the record of truth. With option A, CF still
gets Clearbit data indirectly: Self Serve's autocomplete (§3.2) saves picked Clearbit
suggestions into CDP.

### 4.1 Leaning: option A (CDP work history)

Pending architecture review. Option A matches PM's direction and the designer's "send them to
Self Serve" flow, covers non-member companies, and gives CF a real user ↔ org link that B and C
lack.

**Draft shape** (for discussion):

- **Picker.** At checkout, CF lists the donor's work-history orgs through a v2 API behind
  Heimdall, scoped to the caller (§5.2 item 3). If their org isn't listed, CF links them to Self
  Serve's Work history & Affiliations tab (§3.4). CF has no free-text fallback, which replaces
  #261's "fall back to unlinked" idea.
- **Storage.** A CF org row keyed by the CDP org UUID (unique), shared across donors and not
  editable from CF. The name and logo are refreshed from CDP.
- **Authorization.** Shared rows break today's `ownedOrg()` check, which rejects any donor who
  doesn't own the row (`donation_service.go:266-270`, `subscription_service.go:189-193`).
  Replace it with a server-side check that the selected CDP org is in the donor's CDP work
  history. Sharing the row or dropping the check alone would let anyone donate as any org.
  **This check alone isn't enough.** Work history is self-attested: anyone can add any org in
  Self Serve with no review (§3.3), and `verifiedBy: <lfid>` is the user verifying themselves.
  A donor can add an org to their work history and then donate as it, which reopens the hole
  #349 closed. What counts as proof of affiliation is a PM decision (§5.1 item 6).
- **Ledger.** Ledger receives the CF org row ID at posting time. That ID is known at checkout,
  so posting doesn't depend on any CDP → SFID mapping.
- **Org lens.** CDP → SFID resolution (Snowflake mapping or a back-office job) is a read-time
  lookup so org donations can show in the Self Serve Organization lens. Nothing at checkout or
  Ledger posting waits on it.

**Open questions for option A:**

- Whether donors may pick past employers or only current ones (§5.1 item 1).
- Which v2 service owns the work-history API, and its cost (§5.2 item 3).
- What counts as proof of affiliation, since work history is self-attested (§5.1 item 6).
- How CDP → SFID mapping coverage looks for typical donor orgs.
- How to handle CDP's weak dedup (§3.3), since it can merge unrelated same-named orgs.
- **Self Serve Organization lens visibility** (§5.1 item 5). The lens only exists for `b2b_org`
  orgs, so CDP-only orgs can never appear there.

**Org lens constraint (confirmed in lfx-self-serve).** The Organization lens is built entirely on
`b2b_org`:

- The org selector lists the caller's `b2b_org` role grants and searches the catalogue with
  `GET /query/resources?type=b2b_org` (`server/services/org-navigation.service.ts:153`, `:221`).
- Lens routes only accept an 18-char Salesforce Account ID (`server/helpers/org-uid.helper.ts`,
  `assertOrgUid`).
- Access is checked as `auditor`/`writer` on `b2b_org` (`server/helpers/org-lens-read-access.helper.ts:142`).

The lens does bridge to CDP for some data. It reads `CDEV_ORG_ID` for an account from the
Snowflake table `ORG_LENS_ACCOUNT_CONTEXT` (`server/services/organization.service.ts:1043-1070`).
So with option A:

- Donations to a CDP org that maps to a `b2b_org` *could* appear in that org's lens, by looking
  them up with the mapped CDP ID.
- Donations to a CDP org with no `b2b_org` never appear in the lens, because there is no lens
  page for that org. `b2b_org` records are created only by the org admins (§4, option B).

If PM requires every org donation to show in the lens, option A doesn't meet that and the source
has to be `b2b_org` (option B), with its member-only coverage.

---

## 5. Decisions needed

### 5.1 Product (PM)

1. **Who can donate as an org.** Any org in the donor's work history, or only current
   employment? (Free-text entry is ruled out by the designer decision in item 2.)
2. ~~**Inline add.**~~ **Decided (designer):** no inline add in CF. Donors whose org isn't
   listed are directed to Self Serve to add it to their work history.
3. **Display.** Can a donor edit an org's name and logo? Today they can, which rewrites history.
   Where should org donations appear (initiative page, SS Organization lens, emails)?
4. **Existing data.** Migrate or merge the existing per-user orgs (606 from v1 plus those
   created since), including the 16 featured companies?
5. **Self Serve Organization lens.** Must org donations show in the SS Organization lens? The
   lens only covers `b2b_org` orgs (§4.1). If yes for *every* donation, option A can't work and
   the source must be `b2b_org`. If showing only for orgs that have a `b2b_org` is acceptable,
   option A still works.
6. **Proof of affiliation.** Work history is self-attested (§4.1 Authorization), so a
   work-history check alone lets anyone donate as any org. What counts as enough proof: an email
   domain matching the org, a `b2b_org` role, an Auth0-linked or otherwise verified signal, or
   accepting self-attestation as a known risk?

### 5.2 Architecture

1. **Source of truth for the list:** CDP work history (A), `b2b_org` (B), or a hybrid.
   Leaning A (§4.1); needs architecture confirmation.
2. **Which ID CF and Ledger store** for an org donation: CDP UUID, SFID, or a CF UUID with
   external refs. Also how CDP → SFID resolution happens (Snowflake mapping, CDP API, back-office
   job).
3. **Credentials and data path.** A CDP M2M client for CF is **ruled out**. The architecture
   team doesn't give external consumers unscoped CDP `read:members` access. Another system's data
   goes through a v2 service behind Heimdall (Eric Searcy, 2026-09-29, "Maintainer GitHub ↔ LFID
   Resolution", `lfx-architecture-scratch/2026-09-Maintainer-GitHub-Resolution`). For option A:
   - **A v2 API scoped to the caller** (for example `GET /me/work-experiences`). Heimdall
     authorizes the request, and the service reads CDP service-to-service. CDP keeps ownership of
     the data; v2 owns the contract and the access control. Still open: which service owns it (a
     new service, `lfx-v2-auth-service`, or member service) and who builds it.
   - **Snowflake** isn't a good fit. It can be up to 12 hours behind, so a donor who adds their
     org in Self Serve and comes back wouldn't see it.
   - Option B already goes through v2 (query-service), so this narrows the cost gap between A
     and B. B still needs a `b2b_org`-wide read credential.
4. **Org-row ownership.** Shared or platform-managed linked rows vs today's per-user editable
   rows. This is the open fork carried over from #261.

### 5.3 Sign-off

- [ ] Requirements agreed (§5.1)
- [ ] Org list source decided (§5.2)
- [ ] PM sign-off
- [ ] Architecture sign-off

---

## 6. References

- `backend/docs/rewrite/11-initiative-attribution-and-access.md`: §4 "Organization donations"
  (the earlier hybrid-picker design) and open questions 4 and 7 (org picker source and
  credential).
- lfx-self-serve:
  - `apps/lfx-one/src/server/services/cdp.service.ts`
  - `apps/lfx-one/src/server/services/organization.service.ts`
  - `apps/lfx-one/src/app/shared/components/organization-search/`
- lfx-v2-member-service:
  - `internal/infrastructure/salesforce/account_repo.go` (membership gate)
  - `internal/service/messaging.go` (`b2b_org` indexing, non-public)
  - `cmd/member-api/design/membership.go` (no list/search endpoint)
- lfx-v2-query-service:
  - `design/query-svc.go` (`/query/resources`, `/query/orgs/suggest`)
  - `internal/infrastructure/clearbit/`
- crowd.dev:
  - `backend/src/api/public/v1/members/work-experiences/`
  - `backend/src/api/public/v1/organizations/createOrganization.ts`
