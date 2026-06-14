# LFX Crowdfunding — E2E Manual Test Plan

**Environment:** https://crowdfunding.dev.lfx.dev  
**Self Serve:** https://app.dev.lfx.dev/crowdfunding  
**Test initiative:** https://crowdfunding.dev.lfx.dev/initiatives/test-html-text  

> **For automated payment tests**, see `frontend/e2e/tests/donate.spec.ts` — run with:
> ```
> E2E_BASE_URL=https://crowdfunding.dev.lfx.dev pnpm test:e2e --grep "@dev"
> ```
> This file covers exploratory flows, initiative creation, and corporate donations not yet in the Playwright suite.

---

## How to use this file with an AI

Point your AI assistant at this file and say:

> "Follow the test plan in docs/testing/e2e-manual-test-plan.md. Use Playwright MCP to drive the browser. Sign in with my DEV credentials before starting. Report pass/fail and findings for each section."

The AI will need:
- Your DEV login credentials (provide at runtime — do not store here)
- Playwright MCP enabled in your Claude Code session
- Access to the DB tunnel if running DB verification steps

---

## Prerequisites

1. Sign in at https://crowdfunding.dev.lfx.dev using your DEV LFID credentials.
2. The test initiative `test-html-text` must exist and be **published** with `accept_funding = true`.
3. For DB verification: a tunnel to the DEV RDS instance must be up (see team runbook for tunnel setup).

### Stripe test cards

| Scenario | Card number | Expiry | CVC |
|---|---|---|---|
| Valid (succeeds) | `4242 4242 4242 4242` | Any future | Any |
| Declined | `4000 0000 0000 0002` | Any future | Any |
| Expired | `4000 0000 0000 0069` | Any future | Any |
| Wrong ZIP (accepted) | `4000 0000 0000 0010` | Any future | Any |
| 3DS — requires auth | `4000 0000 0000 3220` | Any future | Any |

---

## Section 1 — Individual Donation Flows

Navigate to https://crowdfunding.dev.lfx.dev/initiatives/test-html-text and click **Donate**.

### 1.1 Valid card (one-time)

1. Select **$25**, click Continue.
2. Select **Individual**. Fill Full name and Email (use a personal email to verify receipt).
3. Click Continue to Payment.
4. Click **Use a different card**. Enter the valid card (`4242...`).
5. Click Donate.
6. **Assert:** "Thank you for your donation! Your $25 donation to Test HTML Text has been processed."

### 1.2 Expired card

1. Donate $10 as Individual.
2. At payment step, enter expired card (`...0069`).
3. **Assert:** Inline error "Your card is expired. Try a different card." — donation does NOT succeed.

### 1.3 Declined card

1. Donate $10 as Individual.
2. Enter declined card (`...0002`).
3. **Assert:** Error "Your card has been declined." — donation does NOT succeed.

### 1.4 3DS — approve

1. Donate $5 as Individual.
2. Enter 3DS card (`...3220`).
3. When Stripe 3DS test page appears, click **COMPLETE**.
4. **Assert:** Thank-you screen. Donation processed.

### 1.5 3DS — fail

1. Donate $5 as Individual.
2. Enter 3DS card (`...3220`).
3. When Stripe 3DS test page appears, click **FAIL**.
4. **Assert:** Error "We are unable to authenticate your payment method. Please choose a different payment method and try again."

---

## Section 2 — Corporate Donation Flows

Same initiative. Click Donate.

### 2.1 Valid card — company

1. Select **$25**, Continue.
2. Select **Company**. Fill: Company name, Contact name, Email.
3. Continue to Payment → Use a different card → enter valid card (`4242...`).
4. **Assert:** Thank-you screen. $25 processed.

### 2.2 Declined card — company

1. Donate $25 as Company (same contact details).
2. Enter declined card (`...0002`).
3. **Assert:** "Your card has been declined." shown. No charge.

---

## Section 3 — Recurring (Monthly) Donation

### 3.1 Subscribe with valid card

1. Click Donate → select **Monthly** frequency.
2. Select **$10/mo**, Continue → fill Individual contact → Continue to Payment.
3. Enter valid card or use saved card.
4. **Assert:** Thank-you screen. Check https://app.dev.lfx.dev/crowdfunding/donations — recurring subscription appears under "Recurring Donations" with status Active, $10/mo.

### 3.2 Cancel subscription

1. Go to https://app.dev.lfx.dev/crowdfunding/donations.
2. Find the active subscription, click the **…** menu → Cancel.
3. **Assert:** Subscription status changes to Canceled.

---

## Section 4 — Initiative Creation (Fundraise flow)

Navigate to https://crowdfunding.dev.lfx.dev and click **Start Fundraise**.

> After each submission, verify in Self Serve at https://app.dev.lfx.dev/crowdfunding/initiatives (Pending tab).

### 4.1 Project type

1. Select **Project** → Continue.
2. Choose **Git URL** → enter any public GitHub repo URL → Continue.
3. Fill: Name, Elevator Pitch, select at least one Topic.
4. Upload a logo (min 600×600px).
5. Set Annual Funding Goal. Enable at least one Fund Distribution category and set it to 100%.
6. Continue → check both compliance boxes → Submit.
7. **Assert:** "Initiative submitted with success!" shown.

### 4.2 OSTIF Security Audit type

1. Select **OSTIF Security Audit** → Continue.
2. Fill: Name, Elevator Pitch, Topic, Repository URL.
3. Upload logo. Fill Primary Contact (first name, last name, email).
4. Continue → check compliance → Submit.
5. **Assert:** Success screen.
6. **Extra:** Check Primary Contact fields mapped correctly (first name ≠ last name field).

### 4.3 General Fund type

1. Select **General Fund** → Continue.
2. Fill: Name, Elevator Pitch, Topic, Funding Goal, enable one fund distribution category at 100%.
3. Upload logo → Continue → compliance → Submit.
4. **Assert:** Success screen.

### 4.4 Event / Meetup type

1. Select **Event / Meetup** → Continue.
2. Fill: Event name, Summary, Topic, Registration URL (required), Start date, End date.
3. Upload logo. Set Sponsorship Goal, enable one Budget Distribution category at 100%.
4. Continue → compliance → Submit.
5. **Assert:** Success screen.

---

## Section 5 — Self Serve Validation

After completing Sections 1–4:

### 5.1 My Donations

1. Go to https://app.dev.lfx.dev/crowdfunding/donations.
2. **Assert:** Donation History shows all successful donations (correct amounts, initiative names, dates).
3. **Assert:** Failed/declined donations do NOT appear in history.
4. **Assert:** Invoice links render for each donation.

### 5.2 My Initiatives

1. Go to https://app.dev.lfx.dev/crowdfunding/initiatives → Pending tab.
2. **Assert:** All submitted initiatives appear with status "Submitted / Under review by our team".

---

## Section 6 — DB Verification (optional, requires DB tunnel)

Connect to the DEV database (`crowdfunding` schema) and verify:

### 6.1 Donations table

```sql
SELECT d.id, d.status, d.current_amount_in_cents, d.payment_method,
       d.organization_id, d.stripe_payment_intent_id, d.stripe_charge_id,
       i.name AS initiative
FROM crowdfunding.donations d
JOIN crowdfunding.users u ON d.user_id = u.id
LEFT JOIN crowdfunding.initiatives i ON d.initiative_id = i.id
WHERE u.username = '<your-lfid-username>'
ORDER BY d.created_on DESC;
```

**Assert:**
- All successful donations have `status = 'succeeded'`
- `stripe_payment_intent_id` (`pi_...`) is populated on every row
- `stripe_charge_id` (`ch_...`) is populated on succeeded rows
- Declined/failed donations: either absent (rejected client-side) or `status = 'failed'`
- Corporate donations: check whether `organization_id` is set (known gap — currently NULL)

### 6.2 Initiatives table

```sql
SELECT i.name, i.slug, i.initiative_type, i.status, i.accept_funding,
       i.stripe_product_id, i.industry, i.event_start_date, i.event_end_date
FROM crowdfunding.initiatives i
JOIN crowdfunding.users u ON i.owner_id = u.id
WHERE u.username = '<your-lfid-username>'
ORDER BY i.created_on DESC;
```

**Assert:**
- All 4 types present: `project`, `security_audit`, `general_fund`, `event`
- All have `status = 'submitted'`, `accept_funding = true`
- All have a non-empty `stripe_product_id` (`prod_...`)
- Event has `event_start_date` and `event_end_date` populated

### 6.3 Initiative goals

```sql
SELECT i.name AS initiative, g.name AS goal, g.amount_in_cents, g.allocation
FROM crowdfunding.initiative_goals g
JOIN crowdfunding.initiatives i ON g.initiative_id = i.id
JOIN crowdfunding.users u ON i.owner_id = u.id
WHERE u.username = '<your-lfid-username>'
ORDER BY i.created_on, g.sort_order;
```

**Assert:** Goals exist for Project (Annual + Development), General Fund (Annual), Event (Sponsorship Goal).

### 6.4 OSTIF contacts

```sql
SELECT i.name, c.contact_type, c.first_name, c.last_name, c.email
FROM crowdfunding.initiative_contacts c
JOIN crowdfunding.initiatives i ON c.initiative_id = i.id
JOIN crowdfunding.users u ON i.owner_id = u.id
WHERE u.username = '<your-lfid-username>';
```

**Assert:** Primary contact has correct first_name, last_name, email in the right columns.

### 6.5 Ledger stats

```sql
SELECT i.name, ls.total_raised_cents, ls.supporters, ls.updated_on
FROM crowdfunding.initiative_ledger_stats ls
JOIN crowdfunding.initiatives i ON ls.initiative_id = i.id
WHERE i.slug = 'test-html-text';
```

**Assert:** `total_raised_cents` matches sum of all succeeded donations. `updated_on` is within the last hour (cron runs hourly).

---

## Known gaps / open issues

| # | Area | Issue | Severity |
|---|---|---|---|
| 1 | Corporate donation | `organization_id` is NULL — company name not persisted to `organizations` table | Medium |
| 2 | Donations | `payment_method` column is NULL on all rows — not set by service | Low |
| 3 | OSTIF detail | No `initiative_ostif_detail` row created when optional fields left blank | Low (expected) |
| 4 | Balance widget | Does not refresh after donation without page reload (by design — cron-driven) | Info |
| 5 | Self Serve | My Initiatives page was initially empty — needed fresh page load to show Pending count | Low |
