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

**Where we are (October 2026).** Architecture recommends option D and against option A (§4).
The PM has answered the two open product questions: anyone can publicly credit an org, as
stated by the donor (§5.1 item 1), and donors can type an org by hand (§5.1 item 2). PM also
wants a donations list page on the Self Serve Organization lens (§5.1 item 3). The PM
didn't ask to take work history back to architecture. Formal sign-off (§5.3) is still pending.
Option D's `b2b_org` half depends on platform work designed in LFXV2-2400 (Q1/Q2), which was
discarded in September 2026 as not requested by product. The PM's lens page is that product
request (§5.2 item 3).

**TL;DR.** Today "donate as an organization" picks from a private, per-user list of free-text
orgs that nothing else in LFX knows about. PM's preferred source for the org list is the user's
Self Serve **Work history & Affiliations**. That data lives in **CDP (crowd.dev)** and is keyed by
**CDP organization UUIDs**, not Salesforce Account IDs / `b2b_org` UIDs. The alternatives are
`b2b_org` (member service; covers only Salesforce Accounts with a membership) and raw Salesforce
Accounts. Each source trades off coverage against being canonical.
**Architecture recommends against CDP.** Their recommended shape is **option D**, the
committee-member pattern: each org donation stores a name and domain, plus a `b2b_org` reference
when the org is a member company, with no CDP IDs (§4.1). Option A (CDP work history) isn't
formally rejected, but direct CDP access is ruled out, which makes it costly (§4.2). PM has
decided that **anyone can publicly credit an org, as stated**, and that donors can type an org
by hand (§5.1 items 1 and 2).

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

Architecture feedback (Eric Searcy, Luis Mori Guerra, October 2026) is that how donation orgs
connect to the rest of the LFX entity graph needs product and technical alignment. The only
foreign key that matters in LFX is `b2b_org` (it drives the Organization lens), so CF shouldn't
store CDP org IDs. Eric framed this as an initial impression, not a ruling, and PM input is
still needed. Work history as the source is not recommended (§4.2).

---

## 3. Self Serve "Work history & Affiliations": investigation

Repo: `linuxfoundation/lfx-self-serve` (paths relative to `apps/lfx-one/src/` unless noted).

> Research for option A, which architecture doesn't recommend (§4.2).

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
  architecture team has ruled out giving external consumers CDP `read:members` access (§4.2).

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
- Using work history limits CF to orgs the donor has already added in Self Serve. The designer
  decided CF wouldn't add orgs inline and would send donors to Self Serve's Work history &
  Affiliations tab instead. That decision assumed option A and has been replaced: donors can type
an org by hand (§5.1 item 2).

---

## 4. Org list sources compared

| | **A. Self Serve work history (CDP)** | **B. `b2b_org` (member service)** | **C. Salesforce Accounts** | **D. Committee pattern (name + domain, optional `b2b_org`)** |
|---|---|---|---|---|
| ID | CDP org UUID | 18-char SFID | SFID | `b2b_org` SFID when matched, else none |
| Coverage | Any org a user has added; many | **Only Accounts with a membership Asset** (`account_repo.go:25-46`) | All Accounts | Any org (Clearbit or typed) |
| Canonical / joins to SS org lens | Via Snowflake mapping only | **Yes** | Yes | Only when the `b2b_org` reference is set |
| User ↔ org link | **Yes** (that's what work history is) | Only roles: writer/auditor/key contact | No | No |
| Search API | CDP exact name/domain lookups; no fuzzy search | `GET /query/resources?type=b2b_org&name=`, non-public: results filtered to orgs where the caller has `auditor`; CF has no credential today | Member service removed its SOQL search; no LFX API exposes it | query-service org search, `b2b_org` + Clearbit with `b2b_org` first (designed in LFXV2-2400 Q1/Q2; ticket discarded, not scheduled) |
| Create new | Self-serve find-or-create, no review | **No.** `POST /b2b_orgs` only registers an *existing* SF Account, restricted to org-admin clients | Out of scope from checkout | Nothing created; the record is just a name and domain |
| CF access today | None (needs a v2 API behind Heimdall; direct CDP access ruled out, §4.2) | None for listing | None | Clearbit suggest only; `b2b_org` search not yet open to donors |

**Notes on each source:**

- **A** best matches PM's intent ("orgs I work for"). It covers non-member companies, and it
  verifies affiliation in the same loose way Self Serve does. Costs:
  - a new v2 API, behind Heimdall, that reads the caller's work history from CDP. CF can't get
    its own CDP M2M client (§4.2), so this is a cross-team dependency;
  - CDP IDs in CF and Ledger that need mapping to SFIDs before org donations can show in the
    org lens;
  - self-asserted orgs.
- **B** is the canonical record and already used for initiative attribution
  (`11-initiative-attribution-and-access.md` §4). But it covers only member companies, so most
  donor companies aren't in it. Listing needs a credential with `auditor` on all `b2b_org`s,
  which is the same blocker that deferred the M1 org picker (doc 11, open questions 4 and 7).
- **C** has the widest canonical coverage, but no supported LFX API searches it, and checkout
  must never create Accounts.
- **D** is how committee members store orgs today, and what doc 11 §4 already sketched as the
  "hybrid picker". It needs no new CDP or v2 integration, and the lens works through the
  `b2b_org` reference. Costs: no user ↔ org link (§5.1 item 1), and the `b2b_org` half of the
  search depends on platform work (§5.2 item 3).

Clearbit (`/query/orgs/suggest`, an external third-party company-data API) isn't listed as a
source of record. It returns no stable ID, so it can't be the record of truth. In option D it
supplies names and domains for orgs that aren't member companies.

### 4.1 Recommended: option D (committee-member pattern)

Proposed by Eric Searcy in architecture review, and matches the committee service. Pending PM
and architecture sign-off.

**Draft shape** (for discussion):

- **Data.** An org donation carries an `organization` attribute: required **name** and
  **domain**, plus an optional **`b2b_org` SFID**. This mirrors committee members
  (`organization: {id, name, website}`, lfx-v2-committee-service
  `cmd/committee-api/design/type.go`, `OrganizationInfoAttributes`). There are no foreign keys
  to CDP or anything else. Committee members stopped storing CDP org IDs (LFXV2-2647).
- **Picker.** The same org lookup committees use: the query-service org search, which as
  designed in LFXV2-2400 (Q2) would combine `b2b_org` and Clearbit, with `b2b_org` results
  first and duplicates removed by domain. That search isn't built and the ticket was discarded
  (§5.2 item 3). If the donor picks a `b2b_org` result, the reference is filled in.
  Otherwise the donation stores only the name and domain. CF calls query-service directly. It
  doesn't copy Self Serve's committee picker, which still creates orgs in CDP.
- **Free text.** If search doesn't find the org, the donor types a name and domain by hand.
  PM confirmed this (§5.1 item 2); it replaces the earlier designer decision to send donors to
  Self Serve.
- **Domain.** The server validates and normalizes every domain, from the picker or typed by
  hand, before storing it: parse a URL down to its hostname, lowercase it, strip a trailing dot
  and a leading `www.`, and reject anything that isn't a hostname. `https://Acme.com/path`,
  `acme.com` and `acme.com.` all become `acme.com`. Dedup, `b2b_org` matching and adoption of
  older records (below) all compare this canonical form.
- **Recurring donations.** Option D covers monthly subscriptions too, since checkout sends the
  same org choice down the subscription path. `SubscriptionService.Create` drops `ownedOrg()`
  and stores the same name, domain and optional `b2b_org` SFID on the subscription and in the
  Stripe subscription metadata. Every renewal's Ledger transaction carries that snapshot. It
  isn't re-checked at renewal: with no affiliation check (§5.1 item 1) there's nothing to
  re-check.
- **Org lens.** PM wants a donations list page on the Organization lens (§5.1 item 3). Donations
  with a `b2b_org` reference show on that org's page. Name and domain only donations can't,
  since the lens only exists for `b2b_org` orgs (see the constraint in §4.3). So the page is
  only as complete as the `b2b_org` matching: until donors can find member companies in search
  (§5.2 item 3), every new org donation is name and domain only and the page stays empty.
  CF serves the list (§5.2 item 5); Self Serve builds the page.
- **Adopting older records (later).** Services can listen for org created and updated events
  and attach a `b2b_org` reference to records whose domain matches. That would also cover the
  existing per-user orgs and the featured companies. CF has no such subscriber today, so this is
  out of scope for the first version. The lens page makes this more valuable, since it is the
  only way donations made before matching existed can reach the page (§5.2 item 4).
- **Authorization.** Nothing links the donor to the org, so anyone can donate "as Google". PM
  accepted this: the donor's stated org is shown as is (§5.1 item 1). It's the same hole #349
  closed for today's per-user orgs, now accepted on purpose.
- **Ledger.** Today Ledger gets the CF org row ID. With option D it would get the `b2b_org`
  SFID when there is one, plus the name and domain. Unmatched orgs (Clearbit or typed) have no
  SFID, and today a blank `organizationID` marks the donation as individual
  (`internal/infrastructure/clients/ledger.go:254`), with org totals keyed by that ID. So Ledger
  needs an explicit org discriminator, or a stable fallback key such as the normalized domain,
  and the historical CF org UUIDs must keep resolving so existing totals don't split. Needs
  agreement with ledger-service (§5.2 item 2).

### 4.2 Not recommended: option A (CDP work history)

Option A lists the donor's Self Serve work-history orgs and stores CDP org UUIDs. Architecture
recommends against it. It isn't formally rejected, and it remains possible if PM still wants it
and architecture agrees to the cost:

- **Data path.** A CDP M2M client for CF is ruled out. External consumers don't get unscoped
  CDP `read:members` access (Eric Searcy, 2026-09-29, "Maintainer GitHub ↔ LFID Resolution",
  `lfx-architecture-scratch/2026-09-Maintainer-GitHub-Resolution`; applied to CF by Michal
  Lehotsky in review). Option A would need a new v2 API behind Heimdall, which makes it much
  more expensive.
- **Entity graph.** Eric's view is that CDP org IDs aren't foreign keys that matter in LFX;
  only `b2b_org` is (LFXV2-2647 removed them from committee members).
- **Authorization.** Work history is self-attested (§3.3), so it doesn't prove affiliation
  either.

### 4.3 Org lens constraint (all options)

Confirmed in lfx-self-serve: the Organization lens is built entirely on `b2b_org`.

- The org selector lists the caller's `b2b_org` role grants and searches the catalogue with
  `GET /query/resources?type=b2b_org` (`server/services/org-navigation.service.ts:153`, `:221`).
- Lens routes only accept an 18-char Salesforce Account ID (`server/helpers/org-uid.helper.ts`,
  `assertOrgUid`).
- Access is checked as `auditor`/`writer` on `b2b_org` (`server/helpers/org-lens-read-access.helper.ts:142`).

The lens does bridge to CDP for some data. It reads `CDEV_ORG_ID` for an account from the
Snowflake table `ORG_LENS_ACCOUNT_CONTEXT` (`server/services/organization.service.ts:1043-1070`).
Orgs with no `b2b_org` have no lens page, and `b2b_org` records are created only by the org
admins (§4, option B). This applies to every option.

---

## 5. Decisions needed

### 5.1 Product (PM)

1. **Who can publicly credit an org. Decided (PM, October 2026): anyone, as stated by the
   donor.** The donor's chosen org is shown as is, with no proof of affiliation. PM's reasoning:
   people are unlikely to donate on behalf of an org they aren't part of, and doc 11 §4 took the
   same view (donating gives money in the org's name, which is lower risk than raising money in
   its name). Options considered and not chosen:
   - **Anyone, shown differently.** Matched `b2b_org` orgs show as the org; everything else
     shows as stated by the donor, or isn't shown publicly.
   - **Only with proof.** For example, a verified email domain that matches the org's domain,
     or a role on the `b2b_org`. This blocks most donors at non-member companies.
2. **Free text. Decided (PM, October 2026): yes.** When search doesn't find their org, donors
   type the name and domain by hand. PM checked other donation platforms and a free-text org
   field is standard. This replaces the earlier designer decision to send donors to Self Serve
   work history (option A).
3. **Display.** *Where org donations appear.* **Decided (PM, October 2026): the SS Organization
   lens gets a donations list page.** Still open: emails, and
   whether a donor can edit an org's name and logo after donating (today they can, which
   rewrites history).
4. **Existing data.** Convert the existing per-user orgs (606 from v1 plus those created since),
   including the 16 featured companies, to name and domain records? Many have no domain today.
5. **Lens page coverage.** Only donations with a `b2b_org` reference can show on the lens page
   (§4.1). Is it acceptable to ship the page while it misses name and domain only donations, or
   must `b2b_org` search (§5.2 item 3) or domain matching (§5.2 item 4) ship with it?
6. **Lens page content.** What the page shows per donation: initiative, amount, date, and
   which donor details (name, email). Anonymous donations need a rule too.

### 5.2 Architecture

1. **Source of truth:** option D (§4.1), proposed in architecture review. Needs formal
   architecture sign-off.
2. **What CF and Ledger store** for an org donation: name, domain and optional `b2b_org` SFID,
   either on the donation or on shared CF rows that donors can't edit. Ledger needs to agree to
   the change from the CF org row ID, including how it marks an unmatched org donation (no
   SFID) as an org donation and how historical CF org UUIDs keep their totals (§4.1, Ledger).
3. **Search dependency.** Today `/query/orgs/suggest` is Clearbit only, and `b2b_org` search
   only returns orgs where the caller is an `auditor`. Donors can only find member companies
   once query-service ships the combined search and `b2b_org` names and domains are visible to
   all logged-in users. Until then, CF gets Clearbit results with no `b2b_org` reference. Both
   pieces were designed in LFXV2-2400 (Q1: public `b2b_org` field subset; Q2: combined search),
   which architecture agreed to, but the ticket was discarded on 2026-09-03 as not requested by
   product. The PM's lens page (§5.1 item 3) is the product request that would revive them.
   The alternative is matching by domain when the donation is saved, using member-service
   `ResolveByDomain` (LFXV2-2650, Backlog). Eric argued against domain lookup in LFXV2-2400
   (Q3B, Q4).
4. **Adopting older records.** Whether and when CF subscribes to org created and updated events
   to attach `b2b_org` references by domain (§4.1). Not needed for the first version, unless
   PM wants the lens page to cover older donations (§5.1 item 5).
5. **Lens donations API.** CF exposes `GET /crowdfunding/organizations/{uid}/donations`,
   following the per-entity initiatives list that already ships for the lens
   (`GET /crowdfunding/organizations/{uid}/initiatives`, #333; doc 11 §5.1): a filtered query on
   the donation's `b2b_org` SFID, guarded in Heimdall with `openfga_check` on `b2b_org:{uid}`.
   Open: `auditor` (matches the lens read gate, §4.3) or `writer` (what the initiatives list
   uses). The response is per caller, so like the initiatives list it sets
   `Cache-Control: private, no-store` and `Vary: Authorization`
   (`backend/internal/handler/initiative_handler.go:154-155`). Self Serve builds the page
   (cross-team).

### 5.3 Sign-off

- [ ] Requirements agreed (§5.1)
- [ ] Option D confirmed (§5.2)
- [ ] PM sign-off
- [ ] Architecture sign-off

---

## 6. References

- `backend/docs/rewrite/11-initiative-attribution-and-access.md`: §4 "Organization donations"
  (the earlier hybrid-picker design) and open questions 4 and 7 (org picker source and
  credential). §5.1 (superseded note) describes the per-entity lens lists from #333 that the
  lens donations API follows.
- Architecture feedback on this doc (October 2026):
  - Eric Searcy: treat donation orgs like committee-member orgs (§4.1).
  - Luis Mori Guerra: Self Serve's committee picker still writes to CDP; `b2b_org` search isn't
    open to donors yet; name and domain alone reopen #349.
- lfx-v2-committee-service:
  - `cmd/committee-api/design/type.go` (`OrganizationInfoAttributes`)
  - `gen/http/openapi3.yaml` (add committee member payload, `organization`)
- Jira:
  - LFXV2-2400: committee-member org ID divergence. Its Q1–Q3 are the public `b2b_org` subset,
    combined `b2b_org` + Clearbit search and sync-from-`b2b_org` design. Discarded 2026-09-03.
  - LFXV2-2647: stop storing CDP org IDs on committee members.
  - LFXV2-2650: member-service `ResolveByDomain` (Backlog).
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
