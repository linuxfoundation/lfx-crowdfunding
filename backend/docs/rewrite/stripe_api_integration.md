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
  "client_secret": "seti_1ABC...secret_xyz"
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
  "client_secret": "pi_1ABC...secret_xyz",
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

**Purpose:** Start a recurring monthly or annual donation to a crowdfunding initiative.

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

`frequency` must be `"monthly"` or `"yearly"`.

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
  "client_secret": "pi_1ABC...secret_xyz",
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
| `invoice.payment_succeeded` | Sets `subscriptions.status = 'active'` for the related subscription |
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
