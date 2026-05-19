# Go-Live Checklist

Items that must be resolved before the new CF backend handles real traffic in production.
Each item references the relevant code location for the required follow-up work.

---

## Stripe / Payments

### 1. Implement LFF-era event guard in the Stripe webhook handler

**Location:** `backend/internal/handler/webhook_handler.go` — `dispatch()`

**Why:** CF and LFF share the same Stripe account. LFF Stripe objects carry `projectID` in
their metadata; CF objects carry `initiative_id`. Until LFF is fully retired and the Stripe
webhook endpoint is switched from LFF to CF, any events that arrive at CF for LFF objects
will produce not-found DB errors (currently acknowledged silently — see the not-found guard
added in this same file). Once the webhook is switched over, LFF events stop arriving and
the guard is no longer needed.

**However**, if the CF endpoint is registered in Stripe *before* LFF is retired (e.g. for
pre-production testing), LFF events will arrive at CF. The guard prevents those from
polluting error logs and metrics.

**What to do:**
- Before registering CF's webhook endpoint in Stripe, confirm LFF's endpoint is deregistered.
- If both endpoints must be active simultaneously (e.g. for parallel testing), add a metadata
  check at the top of `dispatch()`:

  ```go
  if event.Data != nil {
      // Peek at metadata without full unmarshal — check for CF's initiative_id key.
      // LFF events carry "projectID" instead; they have no "initiative_id".
      // ... extract metadata from event.Data.Raw and return 200 if initiative_id absent
  }
  ```

- Remove this TODO once the Stripe account webhook configuration is confirmed clean.

---

### 2. Verify Stripe webhook endpoint registration sequence

**Why:** The webhook secret (`STRIPE_WEBHOOK_SECRET`) must match the secret shown in the
Stripe Dashboard for the registered endpoint. If the secret from the old LFF endpoint is
used by mistake, all signature validations will fail and every webhook will be rejected.

**What to do:**
- Register CF's webhook endpoint in the Stripe Dashboard under the correct account.
- Copy the signing secret from that registration into the `STRIPE_WEBHOOK_SECRET` env var.
- Deregister LFF's old endpoint at the same time (or confirm it is already gone).
- Subscribe only to events CF handles: `payment_intent.succeeded`,
  `payment_intent.payment_failed`, `invoice.payment_succeeded`, `invoice.payment_failed`,
  `customer.subscription.deleted`.

---

### 3. Set `STRIPE_RETURN_URL` in all deployed environments

**Location:** `backend/cmd/initiatives-api/config.go` — `LoadConfig()`

**Why:** `STRIPE_RETURN_URL` is now required — the service will not start without it.
It must be the deployed frontend URL for the `/payment/complete` page. If this page does
not exist in the frontend, redirect-based 3DS (used as a fallback for older browsers)
will leave the payment hanging in `requires_action` permanently.

**What to do:**
- Set `STRIPE_RETURN_URL` in the Kubernetes ConfigMap / Helm values for staging and
  production. Example: `https://crowdfunding.lfx.linuxfoundation.org/payment/complete`.
- Confirm the frontend route `/payment/complete` exists and handles the Stripe redirect
  params (`payment_intent`, `payment_intent_client_secret`, `redirect_status`).

---

### 4. Integration-test the 3DS subscription flow with Stripe test card `4000 0025 0000 3155`

**Location:** `backend/internal/infrastructure/clients/stripe_client.go` — `CreateSubscription()`

**Why:** `ConfirmationSecret.ClientSecret` is the stripe-go v82 path to the `client_secret`
needed for 3DS on the first subscription invoice. This code path only fires for cards that
require 3DS authentication, which never happens in normal local testing.

**What to do:**
- Write an integration test (or run manually against Stripe test mode) using card number
  `4000 0025 0000 3155` (always requires 3DS authentication).
- Verify the `CreateSubscription` response includes a non-empty `client_secret`.
- Verify the frontend successfully calls `stripe.confirmPayment()` using that secret and
  the subscription transitions to `active` after the simulated challenge completes.
- Test card `4000 0000 0000 3220` (3DS required, challenge fails) to verify the
  `incomplete` → `canceled` path after the 23-hour Stripe expiry window.

---

## Data Migration

### 5. Confirm `ledger-stats-sync` CronJob is running before go-live

**Why:** The `amount_raised_in_cents` column on `crowdfunding.initiatives` starts as NULL.
The API returns 0 in that case, but list sorting and statistics will be incorrect until
the first CronJob run completes and populates the column.

**What to do:**
- Trigger a manual CronJob run immediately after the data migration and confirm the column
  is populated for all migrated initiatives before opening traffic to the new CF backend.

---

## Auth / Security

### 6. Remove or disable `DISABLED_MOCK_LOCAL_PRINCIPAL` in all non-local environments

**Location:** `backend/cmd/initiatives-api/config.go` — `LoadConfig()`

**Why:** This env var bypasses JWT validation entirely. It must never be set in staging
or production. Confirm it is absent from all Helm values files before go-live.
