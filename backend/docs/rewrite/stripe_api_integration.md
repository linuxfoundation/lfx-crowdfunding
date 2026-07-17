# Stripe API Integration

This document describes every Stripe-backed HTTP endpoint in the LFX Crowdfunding
initiatives service, the 3DS authentication flow they participate in, and the
webhook events that complete async payment lifecycle transitions.

---

## Overview

The service uses **stripe-go v82** with the following high-level model:

- Every authenticated user gets a **Stripe Customer** (`cus_xxx`) created
  lazily on their first payment or when they explicitly save a card.
- **One-time donations** are fulfilled via a `PaymentIntent` with
  `confirm=true` and `request_three_d_secure=automatic`. If the card issuer
  requires a 3DS challenge the intent lands in `requires_action` and
  `client_secret` is returned to the frontend so Stripe.js can display the
  challenge modal. The final status is delivered asynchronously via webhook.
- **Recurring donations** use Stripe Subscriptions with
  `payment_behavior=default_incomplete`. The first invoice may also require a
  3DS challenge, in which case `client_secret` is returned and the subscription
  stays `incomplete` until the webhook confirms payment.
- A user can pre-save a card via a **SetupIntent** so that future charges
  succeed off-session without requiring a fresh 3DS challenge.

---

## Authentication

| Route group | Auth mechanism |
|-------------|----------------|
| `POST /v1/stripe/webhook` | **Stripe-Signature HMAC** (no JWT) |
| All other Stripe endpoints | **JWT Bearer token** (Auth0 / JWKS) |

---

## Endpoints

### 1. `POST /v1/me/setup-intent`

**Purpose:** Begin the card-saving flow. Returns a `client_secret` that the
frontend passes to `stripe.confirmSetupIntent()` (Stripe.js) to securely
collect card details and run 3DS authentication before the card is attached to
the account.

**Auth:** JWT required.

**Request body:** none.

**Response `201 Created`:**
```json
{
  "client_secret": "<setup-intent-client-secret>"
}
```

**Frontend flow:**
1. Call this endpoint to get `client_secret`.
2. Call `stripe.confirmSetupIntent(client_secret, { payment_method: { card: cardElement } })`.
3. On success Stripe returns a `pm_xxx` payment method ID.
4. POST that ID to `POST /v1/me/payment-method` (next endpoint).

**Backend behaviour:**
- Looks up the user's Stripe Customer ID from the `users` table.
- If no customer exists, creates one in Stripe and persists the `cus_xxx` in the DB.
- Creates a `SetupIntent` scoped to that customer with `usage=off_session` so
  the card can be charged in future without requiring the user to be present.

---

### 2. `POST /v1/me/payment-method`

**Purpose:** Attach a Stripe-confirmed payment method to the user's account and
save it as their default card for future charges.

**Auth:** JWT required.

**Request body:**
```json
{
  "payment_method_id": "pm_1ABC..."
}
```

**Response `200 OK`:**
```json
{
  "payment_method_id": "pm_1ABC...",
  "last_four": "4242",
  "brand": "visa",
  "expiry_month": 12,
  "expiry_year": 2027
}
```

**Backend behaviour:**
- Ensures the user has a Stripe Customer (creates one if needed).
- Calls `stripe.PaymentMethods.Attach` to link the `pm_xxx` to the customer.
- Persists the `pm_xxx` as `stripe_default_payment_method` on the `users` row.
- Returns card metadata from Stripe for immediate display in the UI.

---

### 3. `GET /v1/me/payment-account`

**Purpose:** Retrieve the user's currently saved card details for display in
account settings.

**Auth:** JWT required.

**Request body:** none.

**Response `200 OK`:**
```json
{
  "payment_method_id": "pm_1ABC...",
  "last_four": "4242",
  "brand": "visa",
  "expiry_month": 12,
  "expiry_year": 2027
}
```

**Backend behaviour:**
- Loads the user's `stripe_default_payment_method` from the DB.
- Fetches live card metadata from Stripe (`stripe.PaymentMethods.Get`).
- Returns `404` if no payment method is saved.

---

### 4. `DELETE /v1/me/payment-method`

**Purpose:** Remove the user's saved card from both Stripe and the local DB.

**Auth:** JWT required.

**Request body:** none.

**Response `204 No Content`.

**Backend behaviour:**
- Loads `stripe_default_payment_method` from the DB.
- Calls `stripe.PaymentMethods.Detach` to disassociate the card from the customer.
- Sets `stripe_default_payment_method = NULL` on the `users` row.

---

### 5. `POST /v1/initiatives/{id}/donations`

**Purpose:** Make a one-time donation to a crowdfunding initiative.

**Auth:** JWT required.

**Request body:**
```json
{
  "amount_in_cents": 5000,
  "stripe_payment_method_id": "pm_1ABC...",
  "category": "general",
  "po_number": "PO-123",
  "payment_method": "card",
  "organization_id": "org_xyz"
}
```

**Response `201 Created` — payment succeeded immediately:**
```json
{
  "id": "don_...",
  "user_id": "auth0|...",
  "initiative_id": "kubernetes",
  "current_amount_in_cents": 5000,
  "status": "succeeded",
  "stripe_payment_intent_id": "pi_1ABC...",
  "stripe_charge_id": "ch_1ABC...",
  "created_on": "2026-05-18T10:00:00Z",
  "updated_on": "2026-05-18T10:00:00Z"
}
```

**Response `201 Created` — 3DS challenge required:**
```json
{
  "id": "don_...",
  "status": "requires_action",
  "stripe_payment_intent_id": "pi_1ABC...",
  "client_secret": "<payment-intent-client-secret>",
  "created_on": "2026-05-18T10:00:00Z",
  "updated_on": "2026-05-18T10:00:00Z"
}
```

When `status == "requires_action"`, the frontend must call
`stripe.confirmCardPayment(client_secret)` to display the 3DS modal. The
donation record in the DB is updated to `succeeded` or `failed` asynchronously
by the `payment_intent.succeeded` / `payment_intent.payment_failed` webhooks.
`client_secret` is **never stored** — it is injected transiently before the
response is sent.

**Backend behaviour:**
1. Validates the initiative exists and has `accept_funding = true`.
2. Looks up or creates a Stripe Customer for the user.
3. Creates a `PaymentIntent` with:
   - `confirm=true`, `request_three_d_secure=automatic`
   - `return_url` set to `STRIPE_RETURN_URL` (redirect-based 3DS fallback)
   - `expand[]=latest_charge` so the `ch_xxx` charge ID is available immediately
4. Persists the donation row with `status` and `stripe_payment_intent_id`.
5. Sets `client_secret` on the response object when status is `requires_action`.

---

### 6. `POST /v1/initiatives/{id}/subscriptions`

**Purpose:** Start a recurring donation to a crowdfunding initiative.

**Auth:** JWT required.

**Request body:**
```json
{
  "amount_in_cents": 1000,
  "frequency": "monthly",
  "stripe_payment_method_id": "pm_1ABC...",
  "category": "general",
  "organization_id": "org_xyz"
}
```

`frequency` must be one of `"monthly"` (`"month"`), `"yearly"` (`"year"`, `"annual"`), `"weekly"` (`"week"`), or `"daily"` (`"day"`). Aliases are accepted; unsupported values return `400`.

**Response `201 Created` — subscription activated immediately:**
```json
{
  "id": "sub_local_...",
  "user_id": "auth0|...",
  "initiative_id": "kubernetes",
  "current_amount_in_cents": 1000,
  "frequency": "monthly",
  "status": "active",
  "stripe_subscription_id": "sub_1ABC...",
  "stripe_price_id": "price_1ABC...",
  "created_on": "2026-05-18T10:00:00Z",
  "updated_on": "2026-05-18T10:00:00Z"
}
```

**Response `201 Created` — first invoice requires 3DS:**
```json
{
  "id": "sub_local_...",
  "status": "incomplete",
  "stripe_subscription_id": "sub_1ABC...",
  "client_secret": "<payment-intent-client-secret>",
  "created_on": "2026-05-18T10:00:00Z",
  "updated_on": "2026-05-18T10:00:00Z"
}
```

When `status == "incomplete"`, the frontend must call
`stripe.confirmPayment(client_secret)` to complete 3DS on the first invoice.
The subscription is activated asynchronously by the `invoice.payment_succeeded`
webhook. `client_secret` is **never stored**.

**Backend behaviour:**
1. Validates the initiative accepts funding.
2. Looks up or creates a Stripe Customer for the user.
3. Calls `GetOrCreatePrice` — always creates a new Stripe Price with
   `ProductData` inline for the initiative's `stripe_product_id`, at the
   requested amount and interval. This supports variable-amount subscriptions
   (different donors can pay different amounts to the same initiative).
4. Creates a Subscription with `payment_behavior=default_incomplete` and
   `expand[]=latest_invoice` to access `ConfirmationSecret.ClientSecret`.
5. Persists the subscription row with `stripe_price_id` and initial status.
6. Sets `client_secret` on the response when status is `incomplete`.

---

### 7. `DELETE /v1/subscriptions/{id}`

**Purpose:** Cancel a recurring donation subscription.

**Auth:** JWT required. The caller must own the subscription.

**Request body:** none.

**Response `204 No Content`.**

**Backend behaviour:**
- Verifies `subscription.user_id == caller.user_id`, returns `403` otherwise.
- Calls `stripe.Subscriptions.Cancel` (immediate cancellation).
- Updates the local row to `status = "canceled"`.

---

## Webhook — `POST /v1/stripe/webhook`

**Auth:** No JWT. Every delivery is validated via `Stripe-Signature` HMAC
before any processing occurs (OWASP requirement). Requests missing the header
or with an invalid signature are rejected `401`.

The body is limited to 64 KiB. Configure the endpoint secret via
`STRIPE_WEBHOOK_SECRET`.

### Handled events

| Stripe event | Local effect |
|---|---|
| `payment_intent.succeeded` | Sets `donations.status = 'succeeded'`, stores `stripe_charge_id` from `latest_charge.id` |
| `payment_intent.payment_failed` | Sets `donations.status = 'failed'` |
| `invoice.payment_succeeded` | Sets `subscriptions.status = 'active'` for the related subscription and inserts an idempotent `donations` row keyed by `stripe_invoice_id` |
| `invoice.payment_failed` | Sets `subscriptions.status = 'past_due'` for the related subscription |
| `customer.subscription.deleted` | Sets `subscriptions.status = 'canceled'` (fired by Stripe after too many failed invoices or a Dashboard cancellation) |

All other event types are acknowledged with `200` and logged; they do not
affect the DB.

### Stripe Dashboard configuration

Register the webhook endpoint at:
```
https://<your-domain>/v1/stripe/webhook
```

Enable these event types:
- `payment_intent.succeeded`
- `payment_intent.payment_failed`
- `invoice.payment_succeeded`
- `invoice.payment_failed`
- `customer.subscription.deleted`

---

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `STRIPE_SECRET_KEY` | yes | — | Stripe secret key (`sk_live_...` or `sk_test_...`) |
| `STRIPE_WEBHOOK_SECRET` | yes | — | Webhook endpoint signing secret (`whsec_...`) |
| `STRIPE_RETURN_URL` | no | `http://localhost:3000/payment/complete` | Redirect URL Stripe uses after a redirect-based 3DS challenge |
| `STRIPE_TIMEOUT` | no | `30s` | HTTP timeout for outbound Stripe API calls |
| `STRIPE_WEBHOOK_ACK_UNIMPLEMENTED` | no | `false` | Reply `200` for handled-but-unimplemented events instead of `501` (pre-production only) |

---

## Donation and subscription status lifecycle

### Donation

```
created ──► requires_action ──(3DS confirmed)──► succeeded
       │                     └─(3DS failed / card declined)─► failed
       └──► succeeded   (no 3DS needed — synchronous)
```

### Subscription

```
created ──► incomplete ──(invoice.payment_succeeded)──► active
       │                └─(invoice.payment_failed)────► past_due ──► canceled
       └──► active   (no 3DS on first invoice — synchronous)
                │
                └──(customer.subscription.deleted)──► canceled
```

---

## Database columns added by migration `002_stripe_3ds`

| Table | Column | Purpose |
|---|---|---|
| `users` | `stripe_customer_id` | Stripe `cus_xxx` — created once per user |
| `users` | `stripe_default_payment_method` | Stripe `pm_xxx` of the user's saved card |
| `donations` | `stripe_payment_intent_id` | Stripe `pi_xxx` — used to match webhook events |
| `donations` | `status` | Payment state: `pending`, `requires_action`, `succeeded`, `failed` |
| `subscriptions` | `stripe_price_id` | Stripe `price_xxx` created per subscription |
| `subscriptions` | `status` | Subscription state: `incomplete`, `active`, `past_due`, `canceled` |

---

## Frontend Implementation Guide

This section is written for the frontend developer implementing the Stripe UI.
It covers every user-facing flow end-to-end with diagrams, Stripe.js call
signatures, and error handling.

### Prerequisites

Install the Stripe.js loader:

```bash
pnpm add @stripe/stripe-js
```

Initialise once at app startup (do **not** call `loadStripe` inside a
component render cycle):

```ts
// plugins/stripe.ts
import { loadStripe } from '@stripe/stripe-js'

export const stripe = await loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY)
```

---

### API Quick Reference

| Method | Endpoint | Auth | Purpose |
|--------|----------|------|---------|
| `POST` | `/v1/me/setup-intent` | JWT | Get a `client_secret` to save a card |
| `POST` | `/v1/me/payment-method` | JWT | Record the card after Stripe confirms it |
| `GET`  | `/v1/me/payment-account` | JWT | Fetch saved card details |
| `DELETE` | `/v1/me/payment-method` | JWT | Remove saved card |
| `POST` | `/v1/initiatives/{id}/donations` | JWT | Create a one-time donation |
| `POST` | `/v1/initiatives/{id}/subscriptions` | JWT | Create a recurring subscription |
| `DELETE` | `/v1/subscriptions/{id}` | JWT | Cancel a subscription |

---

### Flow 1 — Save a Card (SetupIntent)

Run this flow **before** any donation or subscription if the user has no card
on file. `GET /v1/me/payment-account` returns `404` when no card is saved.

```
Frontend                         Backend (API)                    Stripe
   │                                   │                             │
   │── POST /v1/me/setup-intent ───────▶                             │
   │                                   │── Customers.New ───────────▶│
   │                                   │◀──────────────── cus_xxx ───│
   │                                   │── SetupIntents.New ─────────▶│
   │                                   │◀──── seti_xxx + client_secret│
   │◀── { client_secret: "seti_..." } ─│                             │
   │                                   │                             │
   │  stripe.elements({ clientSecret })│                             │
   │  paymentElement.mount('#el')      │                             │
   │  [user fills card form]           │                             │
   │                                   │                             │
   │── stripe.confirmSetup() ──────────────────────────────────────▶│
   │                                   │        [Stripe validates card]
   │                         ┌─────────┴──────────────────────────┐  │
   │                         │   Does bank require 3DS challenge?  │  │
   │                         └─────────┬──────────────────────────┘  │
   │                              NO ◀─┘──▶ YES                      │
   │                              │              │                    │
   │◀── setupIntent.payment_method│              │ Stripe opens       │
   │    (pm_xxx, in-page result)  │              │ 3DS modal/redirect │
   │                              │              │                    │
   │── POST /v1/me/payment-method │       User completes challenge   │
   │   { payment_method_id: pm_xxx}       │                          │
   │◀── { last_four, brand, ... } │    Stripe redirects to           │
   │                              │    /payment/complete             │
   │                              │    ?setup_intent=seti_xxx        │
   │                              │    &redirect_status=succeeded    │
   │                              │              │                    │
   │                              │     [completion page runs]       │
   │                              │     stripe.retrieveSetupIntent() │
   │                              │     → setupIntent.payment_method │
   │                              │              │                    │
   │                              │── POST /v1/me/payment-method ───▶│
   │                              │◀── { last_four, brand, ... } ───│
```

#### Step 1 — Request a SetupIntent

```ts
const { client_secret } = await api.post('/v1/me/setup-intent')
// → { "client_secret": "seti_xxx_secret_yyy" }
```

#### Step 2 — Mount the Payment Element

```ts
const elements = stripe.elements({ clientSecret: client_secret })
const paymentElement = elements.create('payment')
paymentElement.mount('#payment-element')
```

#### Step 3 — Confirm on form submit

```ts
const { setupIntent, error } = await stripe.confirmSetup({
  elements,
  confirmParams: {
    return_url: `${window.location.origin}/payment/complete`,
  },
  redirect: 'if_required', // stay on page when no redirect is needed
})

if (error) {
  showError(error.message)
  return
}
// No redirect — pm_xxx available immediately
await api.post('/v1/me/payment-method', {
  payment_method_id: setupIntent.payment_method,
})
```

#### Two possible outcomes — same final call to the backend

```
redirect: 'if_required' result
┌──────────────────────────────────────────────────────────────┐
│ No redirect (most cards)                                     │
│   setupIntent.status === 'succeeded'                         │
│   setupIntent.payment_method === 'pm_xxx'  ← POST to API    │
├──────────────────────────────────────────────────────────────┤
│ Redirected to /payment/complete                              │
│   URL params:                                                │
│     setup_intent=seti_xxx                                    │
│     setup_intent_client_secret=seti_xxx_secret_yyy           │
│     redirect_status=succeeded                                │
│   Call stripe.retrieveSetupIntent(client_secret_from_url)    │
│   → setupIntent.payment_method  ← POST to API               │
└──────────────────────────────────────────────────────────────┘
```

---

### Flow 2 — One-Time Donation

#### Prerequisite check

```ts
let card
try {
  card = await api.get('/v1/me/payment-account')
} catch (e) {
  if (e.status === 404) {
    router.push('/account/payment/add') // → run Flow 1 first
    return
  }
  throw e
}
```

#### Full flow diagram

```
Frontend                         Backend (API)                  Stripe / Webhook
   │                                   │                              │
   │── POST /v1/initiatives/:id/donations ──────────────────────────▶│
   │   { amount_in_cents: 5000,        │   PaymentIntents.New         │
   │     stripe_payment_method_id:     │   (Confirm=true,             │
   │     pm_xxx }                      │    3DS=automatic)            │
   │                                   │◀─────────────────────────────│
   │                                   │                              │
   │          ┌────────────────────────┴────────────────────────────┐ │
   │          │ What did Stripe return?                              │ │
   │          └───────────┬──────────────────────┬──────────────────┘ │
   │                      │                      │                    │
   │               status=succeeded       status=requires_action      │
   │               (no 3DS needed)        client_secret present       │
   │                      │                      │                    │
   │◀── 201 { status:     │      201 { status: "requires_action",    │
   │    "succeeded" }     │           client_secret: "pi_xxx..." }   │
   │          │           │                      │                    │
   │    Show success ✓    │   stripe.confirmCardPayment(             │
   │   (webhook also      │     client_secret,                       │
   │    fires to confirm) │     { payment_method: pm_xxx }           │
   │                      │   )                  │                    │
   │                      │          ┌───────────┴──────────────────┐ │
   │                      │          │ 3DS challenge needed?        │ │
   │                      │          └──────────┬───────────────────┘ │
   │                      │               NO ◀──┘──▶ YES              │
   │                      │               │              │            │
   │                      │   paymentIntent.status       │            │
   │                      │   === 'succeeded'        Stripe modal     │
   │                      │         │            User authenticates   │
   │                      │   Show success ✓         │                │
   │                      │   (webhook still         ▼                │
   │                      │    fires separately) Stripe POSTs to      │
   │                      │                 /v1/stripe/webhook        │
   │                      │                 payment_intent.succeeded  │
   │                      │                      │                    │
   │                      │               backend updates DB          │
   │                      │               donation.status='succeeded' │
```

#### Submit the donation

```ts
const donation = await api.post(`/v1/initiatives/${initiativeID}/donations`, {
  amount_in_cents: 5000,
  stripe_payment_method_id: card.payment_method_id,
})

if (donation.status === 'succeeded') {
  showSuccess()
  return
}

if (donation.status === 'requires_action') {
  const { paymentIntent, error } = await stripe.confirmCardPayment(
    donation.client_secret,
    { payment_method: card.payment_method_id }
  )
  if (error) {
    showError(error.message)
    // The donation row stays 'pending' until the webhook fires 'failed'.
    // Do NOT re-POST to create another donation — the record already exists.
    return
  }
  showSuccess()
  // DB status is updated asynchronously by the payment_intent.succeeded webhook
}
```

#### Donation status state machine

```
                POST /donations
                      │
                       ▼
              ┌──────────────┐
              │   pending    │  ← initial DB state
              └──────┬───────┘
                     │
       ┌─────────────┴──────────────┐
       │                            │
       ▼                            ▼
payment_intent.succeeded   payment_intent.payment_failed
(webhook)                  (webhook)
       │                            │
       ▼                            ▼
┌────────────┐             ┌──────────────┐
│ succeeded  │             │   failed     │
└────────────┘             └──────────────┘
```

---

### Flow 3 — Recurring Subscription

#### Full flow diagram

```
Frontend                         Backend (API)                  Stripe / Webhook
   │                                   │                              │
   │── POST /v1/initiatives/:id/subscriptions ──────────────────────▶│
   │   { amount_in_cents: 1000,        │   Prices.New (fresh price)   │
   │     frequency: "monthly",         │   Subscriptions.New          │
   │     stripe_payment_method_id:     │   (default_incomplete,       │
   │     pm_xxx }                      │    expand latest_invoice)    │
   │                                   │◀─────────────────────────────│
   │                                   │                              │
   │          ┌────────────────────────┴────────────────────────────┐ │
   │          │ Did the first invoice need 3DS?                     │ │
   │          └───────────┬──────────────────────┬──────────────────┘ │
   │                      │                      │                    │
   │                status=active          status=incomplete          │
   │                (no 3DS needed         client_secret present      │
   │                 OR no invoice)               │                   │
   │                      │                      │                    │
   │◀── 201 { status:     │    201 { status: "incomplete",           │
   │    "active" }        │         client_secret: "pi_xxx..." }     │
   │          │           │                      │                    │
   │    Show success ✓    │   stripe.confirmPayment(                 │
   │                      │     clientSecret,                        │
   │                      │     confirmParams: {                     │
   │                      │       return_url: '/payment/complete'    │
   │                      │     },                                   │
   │                      │     redirect: 'if_required'              │
   │                      │   )                  │                    │
   │                      │          ┌───────────┴──────────────────┐ │
   │                      │          │ 3DS challenge needed?        │ │
   │                      │          └──────────┬───────────────────┘ │
   │                      │               NO ◀──┘──▶ YES              │
   │                      │               │              │            │
   │                      │   webhook fires         Redirect to       │
   │                      │   asynchronously        /payment/complete │
   │                      │   Show success ✓        (webhook handles  │
   │                      │                          status update)   │
   │                      │                              │            │
   │                      │             Stripe POSTs to /v1/stripe/webhook
   │                      │             invoice.payment_succeeded     │
   │                      │                   │                       │
   │                      │             backend: subscription → active│
```

#### Submit the subscription

```ts
const subscription = await api.post(`/v1/initiatives/${initiativeID}/subscriptions`, {
  amount_in_cents: 1000,
  frequency: 'monthly', // 'monthly' | 'yearly' | 'weekly' | 'daily'
  stripe_payment_method_id: card.payment_method_id,
})

if (subscription.status === 'active') {
  showSuccess('Subscription activated!')
  return
}

if (subscription.status === 'incomplete') {
  const { error } = await stripe.confirmPayment({
    clientSecret: subscription.client_secret,
    confirmParams: {
      return_url: `${window.location.origin}/payment/complete`,
    },
    redirect: 'if_required',
  })
  if (error) {
    showError(error.message)
    // Stripe will retry the invoice automatically (smart retry)
    return
  }
  showSuccess('Subscription activated!')
  // webhook (invoice.payment_succeeded) advances DB → 'active'
}
```

#### Subscription status state machine

```
           POST /subscriptions
                  │
                   ▼
         ┌─────────────────┐
         │   incomplete    │  ← initial DB state
         └────────┬────────┘
                  │
     invoice.payment_succeeded (webhook)
                  │
                  ▼
         ┌─────────────────┐
         │     active      │◀─── invoice.payment_succeeded (each renewal)
         └────────┬────────┘
                  │
     ┌────────────┴──────────────┐
     │                           │
invoice.payment_failed      DELETE /subscriptions/:id
(webhook)                   (frontend cancel)
     │                           │
     ▼                           ▼
┌──────────┐            ┌──────────────┐
│ past_due │            │   canceled   │◀── customer.subscription.deleted
└──────────┘            └──────────────┘    (Stripe-initiated webhook)
     │
     │  Stripe retries automatically (~4 attempts over several days)
     │
invoice.payment_succeeded ──▶ back to active
invoice.payment_failed (exhausted) ──▶ Stripe deletes ──▶ canceled
```

---

### Flow 4 — `/payment/complete` Return Page

This single page handles **all** Stripe redirect outcomes.

```
User lands on /payment/complete
         │
         ▼
Read URL params
         │
    ┌────┴──────────────────────────┐
    │        Which flow?            │
    └───────┬───────────────────────┘
            │                   │
    setup_intent=...     payment_intent=...
    (save card)          (donation/subscription)
            │                   │
            ▼                   ▼
  stripe.retrieveSetupIntent  stripe.retrievePaymentIntent
    (setup_intent_client_secret) (payment_intent_client_secret)
            │                   │
            ▼                   ▼
   status=succeeded?    redirect_status=succeeded?
            │                   │
     YES ───▼              YES ─▼
            │               Show success
  POST /v1/me/payment-method  (DB updated by webhook)
  { payment_method_id }
  → redirect to /account
            │
     NO ────▼
     Show "card verification
     failed" + retry button
```

```ts
// composables/usePaymentComplete.ts
const params = new URLSearchParams(window.location.search)

if (params.get('setup_intent')) {
  const { setupIntent } = await stripe.retrieveSetupIntent(
    params.get('setup_intent_client_secret')!
  )
  if (setupIntent?.status === 'succeeded' && setupIntent.payment_method) {
    await api.post('/v1/me/payment-method', {
      payment_method_id: setupIntent.payment_method,
    })
    router.replace('/account/payment?saved=true')
  } else {
    showError('Card verification failed. Please try again.')
  }

} else if (params.get('payment_intent')) {
  if (params.get('redirect_status') === 'succeeded') {
    // DB is updated by the webhook — just show confirmation
    router.replace('/account?payment=complete')
  } else {
    showError('Payment failed. Please try again.')
  }
}
```

---

### Flow 5 — View and Remove a Saved Card

```ts
// View
const card = await api.get('/v1/me/payment-account')
// { payment_method_id, last_four, brand, expiry_month, expiry_year }
// 404 → no card saved, show "Add a card" CTA

// Remove
await api.delete('/v1/me/payment-method')
// 204 No Content
// After deletion GET /v1/me/payment-account returns 404 again
```

---

### Flow 6 — Cancel a Subscription

```ts
await api.delete(`/v1/subscriptions/${subscriptionID}`)
// 204 No Content — cancelled immediately in Stripe and DB
```

---

### Error Handling Reference

| HTTP status | Condition | Frontend action |
|-------------|-----------|-----------------|
| `400` | Missing/invalid field (e.g. no `stripe_payment_method_id`) | Show field validation error |
| `401` | Missing or expired JWT | Redirect to login |
| `403` | Caller doesn't own the resource | Show "not authorised" |
| `404` on `/me/payment-account` | No card saved | Redirect to save-card flow |
| `404` on donation / subscription | Record not found | Show not found page |
| `409` | Conflict (duplicate resource) | Show "already exists" message |
| `503` | Stripe API unreachable | Show "payment service unavailable, please retry" |
| `500` | Unexpected server error | Show generic error; log to Sentry |

For Stripe.js errors, always show `error.message` directly — Stripe localises
it for the user's locale:

```ts
const { error } = await stripe.confirmCardPayment(...)
if (error) {
  // error.type: 'card_error' | 'validation_error' | 'api_connection_error' | ...
  // error.code: 'card_declined' | 'insufficient_funds' | 'expired_card' | ...
  showError(error.message) // Stripe-provided, user-friendly
}
```

---

### Which Stripe.js Call to Use

```
┌─────────────────────────────┬──────────────────────────────────────┐
│ Scenario                    │ Stripe.js method                     │
├─────────────────────────────┼──────────────────────────────────────┤
│ Saving a card               │ stripe.confirmSetup({ elements,      │
│ (SetupIntent flow)          │   redirect: 'if_required' })         │
├─────────────────────────────┼──────────────────────────────────────┤
│ One-time donation 3DS       │ stripe.confirmCardPayment(           │
│                             │   client_secret,                     │
│                             │   { payment_method: pm_xxx })        │
├─────────────────────────────┼──────────────────────────────────────┤
│ Subscription first invoice  │ stripe.confirmPayment({              │
│ 3DS                         │   clientSecret,                      │
│                             │   confirmParams: { return_url },     │
│                             │   redirect: 'if_required' })         │
├─────────────────────────────┼──────────────────────────────────────┤
│ Redirect return handler     │ stripe.retrieveSetupIntent() or      │
│ (/payment/complete page)    │ stripe.retrievePaymentIntent()       │
└─────────────────────────────┴──────────────────────────────────────┘
```

---

### Test Cards

Use these in Stripe test mode (`sk_test_...` / `pk_test_...`):

| Card number | Scenario |
|-------------|----------|
| `4242 4242 4242 4242` | Instant success — no 3DS required |
| `4000 0027 6000 3184` | 3DS required — always challenges |
| `4000 0025 0000 3155` | 3DS required — user must authenticate to succeed |
| `4000 0000 0000 9995` | Declined — `insufficient_funds` |
| `4000 0000 0000 0002` | Declined — `card_declined` |
| `4000 0000 0000 0069` | Declined — `expired_card` |

Use any future expiry date (e.g. `12/34`), any 3-digit CVC, and any postal code.

---

### Key Rules

1. **Always call the relevant `stripe.confirm*` method when `client_secret` is
   present** — the payment is not complete without it.
2. **Never store `client_secret`** — it is transient, injected at creation time
   only, and never written to the DB.
3. **The canonical final status comes from the webhook**, not the initial `POST`
   response. Show optimistic UI, but reconcile from a fresh `GET` if needed.
4. **`stripe_payment_method_id` is required** in both donation and subscription
   requests — save the card via the setup-intent flow first.
5. **`GET /v1/me/payment-account` returning 404 means no card** — always check
   this before rendering a payment form and redirect to the card-saving flow.
