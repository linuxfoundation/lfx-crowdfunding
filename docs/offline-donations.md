<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Recording offline organization donations, returns and payouts

Internal guide for approvers (#464, #489). Some organizations donate to an initiative through LF
finance, by invoice or wire, instead of through Stripe on the site. Recording the donation in
Crowdfunding puts it in the Ledger, so it counts toward the initiative's balance and donor list, and on
the organization's donations in the Self Serve Organization lens. The same page records money taken
back out: unused funds returned to the donor, and payouts from the initiative's fund. This replaces
posting a hand-built JSON payload to the Ledger from Postman.

This page is deliberately not under `docs/user/`: those pages are public, and only approvers can use
this tool.

## Who can use it

Approvers only: usernames in the backend's `ALLOWED_APPROVERS` (set per environment in
`lfx-v2-argocd` `values/<env>/lfx-crowdfunding-backend.yaml`). Where the Heimdall rule checks OpenFGA
(`openfga.enabled: true`), the user must also be a member of `team:crowdfunding_approvers`. Everyone
else gets a 403 and doesn't see the button; opening `/approvals/offline-donations` directly shows the
"Page not found" page.

## Recording a donation

1. Sign in and open the initiative's page. It must be published.
2. Click **Record offline donation** in the header, next to **Donate**. This opens
   `/approvals/offline-donations?initiative=<slug>` with the initiative filled in. You can also open
   `/approvals/offline-donations` directly and search for the initiative.
3. Under **Type**, keep **Donation**, then fill in:
   - **Organization**: search for it, or choose **Can't find it? Enter it manually** and give its name
     and website. The website's domain is how the donation is matched to the organization's Salesforce
     record (b2b_org), which is what puts it on the organization's Self Serve page.
   - **Amount (USD)**: dollars, e.g. `1300` or `1300.50`. It is converted to cents for you.
   - **Date received**: the day LF finance received the money. It becomes the Ledger transaction date.
   - **Category**: defaults to `general fund` (`mentorship` for mentorship programs), as finance
     records them. Pick one of the initiative's funding goals instead to count the donation toward
     that goal's progress.
   - **Source reference**: the reference finance uses for this payment, e.g. an invoice number or
     `LFX Mentorship CNCF Fall 2026 Row 52`. It must be unique, see below.
4. Click **Review**, check the summary, then **Record donation**.

The recorded donations for the initiative are listed below the form, newest received first, with who
recorded each one. **Load more** shows older ones, 50 at a time.
The initiative's totals and donor list pick it up within the hour (the hourly Ledger sync).

## Recording a return or a payout

Use these to take money out of the initiative's balance. Pick the **Type** on the same form:

- **Return to donor**: unused funds sent back to the organization that donated them. For example, at
  the end of a mentorship program a mentee failed an evaluation, so their unpaid stipend goes back to
  the donor and the program ends at a zero balance.
- **Payout**: money paid to a person out of the initiative's fund, e.g. a reward payout.

Fill in:

- **Returned to organization** (return, required) or **Funds from organization** (payout,
  optional): the donor whose money it is, picked the same way as for a donation. For a payout, leave
  it empty or click **Remove** if there's no specific donor.
- **Paid to** (payout only, required): the payee's name.
- **Amount (USD)**: the amount taken out, typed as a positive number. The Ledger entry gets the minus
  sign; you don't type it.
- **Date returned** / **Date paid**: becomes the Ledger transaction date.
- **Category**: same options as for a donation. A funding goal's name counts the amount as spent from
  that goal.
- **Reason**: why the money is taken out, e.g. `Mentee withdrew, CNCF Fall 2026`. Up to 500
  characters.
- **Source reference**: unique, shared with donations (see below).

Click **Review**. The summary shows the amount with a minus sign and what the entry will do; then click
**Record return** or **Record payout**. Recorded returns and payouts are listed under **Recorded
returns and payouts**, below the donations.

What changes once it's recorded:

- The initiative's available balance drops by the amount within the hour. For a return, the total
  received drops too. The donor's own donation still shows what they sent, and the Organization lens
  is unchanged (it lists donations, not returns).
- The public **Financials** tab lists it under **Expense breakdown**: a return as "Returned to
  *organization*", a payout under the payee's name, each with the **reason**. The reason and payee are
  public, so write them accordingly.

## The source reference must be unique

The Ledger keeps one transaction per source reference (`sourceTxnID`) and silently drops a second one.
Crowdfunding checks it first, across donations, returns and payouts, so:

- Case doesn't matter: `Invoice 12` and `invoice 12` are the same reference. The entry keeps the
  case it was first recorded with. Spaces around it are trimmed.
- Submitting the same entry again is safe: the page says **Already recorded** in an amber notice (not
  the green success screen), with when and by whom,
  and no second Ledger row is created. For a donation "the same" means the same reference, initiative,
  organization name and domain, amount, category and date; for a return or payout also the type,
  payee and reason. Two submits at once (two tabs, a double click) are handled one after the other.
- The same reference with any different detail is rejected as already recorded, including a
  reference already used by an entry of another type.
- A reference that was already posted to the Ledger by hand (e.g. from Postman) is rejected too, and
  nothing is stored.
- If a submit was saved but didn't reach the Ledger, and someone then posts the same reference to the
  Ledger by hand, resubmitting is rejected unless that Ledger row matches (same initiative, amount and
  organization domain; for a return or payout also the type). The entry stays *Not in the Ledger yet*
  until someone checks the Ledger.

## If something goes wrong

- **"The Ledger isn't reachable right now"**: the entry is saved but not in the Ledger yet; it shows
  as *Not in the Ledger yet* in the list. Submit the same details again later to finish it. This
  can't create a duplicate.
- **"This source reference is already recorded in the Ledger"**: the reference is taken by a
  different entry. Check it against the lists below the form and the Ledger.
- **Wrong details after recording**: there is no edit or reverse in the UI. To undo a donation, record
  a **Payout** of the same amount with a reason that says it's a correction (or post the offsetting
  entry to the Ledger by hand), then record the correct entry under a new reference. The CF row for
  the wrong entry has to be fixed by hand.

## What gets stored

The Ledger transaction matches what approvers posted by hand:

| Field | Donation | Return | Payout |
|---|---|---|---|
| `projectID` | the initiative's ID | same | same |
| `userID` / `accountEmail` | `auth0\|communitybridge` / `admin@communitybridge.org` (the shared LF account for offline funds) | same | same |
| `organizationName`, `organizationDomain`, `organizationSFID` | from the organization picked; the SFID is matched from the domain | same | same, when an organization is picked |
| `submitterName` | (empty) | (empty) | the payee (**Paid to**) |
| `txnComment` | (empty) | the reason | the reason |
| `sourceType` / `sourceAccountID` | `expensify` / `manual_from_lf` | same | same |
| `sourceTxnID` | the source reference | same | same |
| `txnType` / `fee` | `credit` / `0` | `credit` / `0` | `debit` / `0` |
| `txnCategory` | the category | same | same |
| `amount` | cents | negative cents | negative cents |
| `txnDate` | the date, 00:00 UTC | same | same |

A return is a negative `credit`, not a `debit`, because that's how finance has recorded them: it's the
donor's unused money going back, not money spent. The public Financials tab reads expenses with the
Ledger's `txnType=outgoing` filter (debits plus negative credits), so returns posted by hand before
this page existed show up there too. Only rows posted for the shared LF account
(`auth0|communitybridge` + `manual_from_lf`) show their `txnComment`; synced Expensify reimbursements
keep theirs private.

Crowdfunding also stores each donation in its `donations` table (`payment_method = 'offline'`, with
`source_txn_id`, `received_on` and `recorded_by`), owned by the `lf:offline-donations` account, whose
name, "LF Finance (offline donation)", is what the Organization lens shows as the donor. Returns and
payouts go in the `ledger_adjustments` table (`kind`, amount as entered, organization, `paid_to`,
`reason`, `adjusted_on`, `source_txn_id`, `status`, `recorded_by`). Because `sourceType` is not
`stripe`, none of these appear in the site-wide recent donations feed, the same as the manual entries
before.

API: `POST` and `GET /crowdfunding/offline-donations` and `/crowdfunding/ledger-adjustments` (see
[architecture.md](architecture.md)).
