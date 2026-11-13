<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Recording offline organization donations

Internal guide for approvers (#464). Some organizations donate to an initiative through LF finance, by
invoice or wire, instead of through Stripe on the site. Recording the donation in Crowdfunding puts it
in the Ledger, so it counts toward the initiative's balance and donor list, and on the organization's
donations in the Self Serve Organization lens. This replaces posting a hand-built JSON payload to the
Ledger from Postman.

This page is deliberately not under `docs/user/`: those pages are public, and only approvers can use
this tool.

## Who can use it

Approvers only: usernames in the backend's `ALLOWED_APPROVERS` (set per environment in
`lfx-v2-argocd` `values/<env>/lfx-crowdfunding-backend.yaml`). Where the Heimdall rule checks OpenFGA
(`openfga.enabled: true`), the user must also be a member of `team:crowdfunding_approvers`. Everyone
else gets a 403 and doesn't see the button.

## Recording a donation

1. Sign in and open the initiative's page. It must be published.
2. Click **Record offline donation** in the header, next to **Donate**. This opens
   `/approvals/offline-donations?initiative=<slug>` with the initiative filled in. You can also open
   `/approvals/offline-donations` directly and search for the initiative.
3. Fill in:
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

The recorded donations for the initiative are listed below the form, with who recorded each one.
The initiative's totals and donor list pick it up within the hour (the hourly Ledger sync).

## The source reference must be unique

The Ledger keeps one transaction per source reference (`sourceTxnID`) and silently drops a second one.
Crowdfunding checks it first, so:

- Submitting the same donation again (same reference, initiative, organization, amount and date) is
  safe: it returns the entry already recorded and doesn't create a second Ledger row.
- The same reference with different details is rejected as already recorded.
- A reference that was already posted to the Ledger by hand (e.g. from Postman) is rejected too, and
  nothing is stored.

## If something goes wrong

- **"The Ledger isn't reachable right now"**: the entry is saved but not in the Ledger yet; it shows
  as *Not in the Ledger yet* in the list. Submit the same details again later to finish it. This
  can't create a duplicate.
- **"Already recorded"**: check the reference against the list below the form and the Ledger.
- **Wrong details after recording**: there is no edit or reverse in the UI. As with other manual
  corrections, post an offsetting `debit` with a negative amount to the Ledger (by hand until #489
  adds adjustments to this page), then record the correct donation under a new reference. The CF row
  for the wrong entry has to be fixed by hand.

## What gets stored

The Ledger transaction matches what approvers posted by hand:

| Field | Value |
|---|---|
| `projectID` | the initiative's ID |
| `userID` / `accountEmail` | `auth0\|communitybridge` / `admin@communitybridge.org` (the shared LF account for offline funds) |
| `organizationName`, `organizationDomain`, `organizationSFID` | from the organization picked; the SFID is matched from the domain |
| `sourceType` / `sourceAccountID` | `expensify` / `manual_from_lf` |
| `sourceTxnID` | the source reference |
| `txnType` / `fee` | `credit` / `0` |
| `txnCategory` | the category |
| `amount` | cents |
| `txnDate` | the date received, 00:00 UTC |

Crowdfunding also stores the donation in its `donations` table (`payment_method = 'offline'`, with
`source_txn_id`, `received_on` and `recorded_by`), owned by the `lf:offline-donations` account, whose
name, "LF Finance (offline donation)", is what the Organization lens shows as the donor. Because
`sourceType` is not `stripe`, offline donations don't appear in the site-wide recent donations feed,
the same as the manual entries before.

API: `POST` and `GET /crowdfunding/offline-donations` (see [architecture.md](architecture.md)).
