# Backend Requirements: Sponsorship Tiers

## 1. Background & Scope

The frontend landed a full donation-options / sponsorship-tiers UX ahead of backend
support:

- `feat/sponsorship-tiers-step` (merged to `main` as PR [#211](https://github.com/linuxfoundation/lfx-v2-crowdfunding/pull/211)) added a
  "donation options" step to the fundraise wizard (choose fixed tiers vs. open
  donations, configure tier goals/benefits) and re-enabled sponsorship-tier display
  on the donate drawer.
- `lfx-self-serve` [PR #1058](https://github.com/linuxfoundation/lfx-self-serve/pull/1058) added the matching
  "Sponsorship Tiers" tab to the initiative settings drawer, gated behind the
  `crowdfunding-sponsor-tiers` LaunchDarkly flag (LFXV2-2535).

Both were originally UI-only, pass-through implementations, sending payloads the
backend had nowhere to put:

- Self-serve PR #1058 is documented as "backend passes data through unchanged... no
  real tier data served yet."
- This repo's `frontend/server/api/fundraise/index.post.ts` originally sent a
  placeholder nested `donation_options: {mode, tiers}` shape with the comment *"No
  backend field exists yet for this — sent as-is so it's ready once one lands."*
  **This has since been updated** (see §3/§6) to send the flat `donation_mode` +
  `sponsorship_tiers[]` contract this document specifies, so the write side is
  ready ahead of the backend.
- `frontend/server/services/initiatives.services.ts` still serves a hard-coded
  `MOCK_SPONSORSHIP_TIERS` array (hidden in production) — this stays in place until
  the backend GET response actually returns usable tier data; see §6.

This document specifies what the Go backend needs to implement so the frontend can
stop mocking sponsorship-tier data on read.

**Non-goal:** sponsorship tiers here are descriptive configuration + benefits
display for donors, not a payment enforcement mechanism. Whether a donation actually
matches a tier amount is a UI nicety, not a backend-enforced constraint, in this pass.

## 2. Gap Analysis

The backend already has partial support from the legacy entity migration, but the
shape doesn't match what the new UIs produce.

| Layer | Exists today | Required |
|---|---|---|
| `initiatives` row | no donation-mode concept | `donation_mode` enum `tiers` \| `open`, default `open` |
| `initiative_sponsorship_tiers` table | `name, description, color, icon, minimum (BIGINT cents), sort_order` | add `enabled BOOLEAN`, add `benefits TEXT[]` |
| `SponsorshipTierInput` (write) | `Name, Description, Color, Icon, Minimum, SortOrder` | add `Enabled bool`, `Benefits []string`; add `DonationMode` on the initiative input |
| `SponsorshipTier` (read) | `ID, Name, Description, Color, Icon, Minimum` | add `Enabled`, `Benefits`; expose `donation_mode` on `Initiative` |
| Validation | none | reject invalid `donation_mode`; reject invalid tier `name` |
| Read wiring | tiers returned without benefits/mode | drop frontend `MOCK_SPONSORSHIP_TIERS`; serve real tiers + mode |

Relevant existing code (`backend/internal/domain/models/`):
- `initiative.go`: `Initiative`, `InitiativeCreateInput`, `InitiativeUpdateInput`
- `initiative_relations.go`: `SponsorshipTierInput` (write), `SponsorshipTier` (read)
- `backend/internal/infrastructure/db/initiative_repository.go`: insert (`~389`),
  delete (`~424`), write loops (`~570`, `~775`), read (`~1173` `listSponsorshipTiers`)

## 3. Canonical Wire Contract

The backend adopts the **flat** shape (`donation_mode` + `sponsorship_tiers[]`) — it
aligns with the existing `sponsorship_tiers` field name and table. Both frontends
already send this shape:

- Self-serve PR #1058 sent it from the start.
- This repo's Nuxt BFF (`frontend/server/api/fundraise/index.post.ts`) was updated
  to match — see §6 for what changed.

### Create / Update request (`POST /v1/me/initiatives`, initiative update)

```jsonc
{
  "initiative_type": "general_fund",
  "name": "...",
  "donation_mode": "tiers",              // "tiers" | "open", default "open"
  "sponsorship_tiers": [
    {
      "name": "gold",                    // constrained enum — see §5
      "enabled": true,
      "goal_amount_cents": 2500000,      // JSON alias for the `minimum` column
      "benefits": ["Logo on homepage", "Direct access to audit team", "Custom briefing"],
      "sort_order": 1
    }
  ]
}
```

Field notes:
- `goal_amount_cents` is accepted as a JSON field name on `SponsorshipTierInput` but
  persists to the existing `minimum` column — no column rename needed.
- `donation_mode: "open"` — tiers are ignored on write (existing rows are cleared,
  see below).
- Update semantics follow the existing convention: a non-nil `sponsorship_tiers`
  (including empty `[]`) replaces all rows for the initiative — same
  delete-then-insert pattern already used for other child tables
  (`initiative_repository.go` update path, `~771`–`781`).

### Read response (`GET` initiative detail/list)

```jsonc
{
  "donation_mode": "tiers",
  "sponsorship_tiers": [
    {
      "id": "uuid",
      "name": "gold",
      "enabled": true,
      "minimum": 2500000,
      "benefits": ["Logo on homepage", "..."]
    }
  ]
}
```
`description`, `color`, `icon` remain as existing optional fields; unused by the new
UI but not removed (other consumers may still rely on them).

## 4. Schema / Migration

New migration pair, e.g. `db/migrations/00X_sponsorship_tiers.up.sql` /
`.down.sql`:

```sql
-- up
ALTER TABLE initiatives
  ADD COLUMN donation_mode TEXT NOT NULL DEFAULT 'open'
    CHECK (donation_mode IN ('tiers', 'open'));

ALTER TABLE initiative_sponsorship_tiers
  ADD COLUMN enabled  BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN benefits TEXT[]  NOT NULL DEFAULT '{}';
```

```sql
-- down
ALTER TABLE initiative_sponsorship_tiers
  DROP COLUMN benefits,
  DROP COLUMN enabled;

ALTER TABLE initiatives
  DROP COLUMN donation_mode;
```

The existing `set_updated_on` trigger on `initiative_sponsorship_tiers`
(`001_initial.up.sql:454`) already covers the table — no trigger changes needed.

## 5. Validation Rules

- `donation_mode` ∈ `{"tiers", "open"}`. Follow the existing
  `InitiativeStatus.IsValid()` pattern in `initiative.go` — add a
  `ValidDonationModes` map + an `IsValid()`-style helper. Invalid value ⇒ domain
  validation error, surfaced by the handler as HTTP 400 (matches how invalid
  `status` is already rejected).
- Tier `name` ∈ `{"platinum", "gold", "silver", "bronze"}` — source of truth is
  `frontend/app/components/modules/fundraise/config/donation-options.config.ts`
  (`SPONSORSHIP_TIER_NAMES`), and matches the validation self-serve PR #1058
  already added client-side (`ServiceValidationError` on invalid tier name).
  Backend must enforce the same set server-side rather than trust the client.
- `benefits[]`: trim/drop blank entries server-side. The Nuxt BFF already filters
  empty strings before sending, but the backend must not depend on that.
- `goal_amount_cents` (`minimum`) `>= 0` — already enforced by the existing
  `CHECK (minimum >= 0)` constraint.

## 6. Read-Side Wiring

- Add `DonationMode` to the `Initiative` struct (`initiative.go`) and to the
  list/detail SQL selects.
- Extend `listSponsorshipTiers` (`initiative_repository.go:1173`) to select
  `enabled, benefits` and populate the extended `SponsorshipTier` read struct.

**Frontend write side — already done, ahead of the backend:**
`frontend/server/api/fundraise/index.post.ts` (`buildSponsorshipTiers()`) now sends
the flat `donation_mode` + `sponsorship_tiers` contract from §3 instead of the old
placeholder `donation_options: {mode, tiers}` shape. The dollar-string tier goal is
converted to cents client-side in `useFundraiseSubmit.ts`
(`buildDonationOptionsPayload()`) and sent as `goalCents`, which the server route
maps to `goal_amount_cents`. This payload is currently dropped by the backend since
`sponsorship_tiers`/`donation_mode` aren't yet accepted on `POST
/v1/me/initiatives` — it will start taking effect once §3–§5 land.

**Frontend read side — still pending, blocked on this backend work:**
`frontend/server/services/initiatives.services.ts` still serves
`MOCK_SPONSORSHIP_TIERS` (hidden in production) because the backend GET response
has no real tier data to map. Once §6's read wiring ships, that file should drop the
mock and map the real `sponsorship_tiers` + `donation_mode` from
`BackendInitiative`, renaming backend `minimum` → frontend
`SponsorshipTier.amountCents`.

## 7. Open Decisions

- **`donation_mode: "open"` on update** — this spec treats it as clearing
  (delete-then-no-insert) any existing tier rows, consistent with the "non-nil
  replaces" convention. Confirm this is the desired product behavior (vs. hiding
  rows via `enabled=false` and preserving history).
- **Tier name enum vs. free text** — this spec fixes tier names to the four-tier
  enum used by both frontends today. If a future requirement needs project-defined
  tier names, `name` would need to become free text with `sort_order` as the only
  ordering signal (no schema blocker either way — `name` is already `TEXT`).
