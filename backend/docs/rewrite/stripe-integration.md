# Stripe Integration Reference

This document covers everything needed to implement and operate Stripe payments in the new
Crowdfunding (CF) backend. It is the primary reference for the developer implementing this work.

---

## Context: LFF → CF migration

LFF (the retiring system) and CF share the **same Stripe account** per environment. This is
intentional and gives CF a migration advantage: existing Customers, Plans, Products, and
active subscriptions are all immediately accessible in CF without any data migration in Stripe.

When CF goes to production, LFF is retired at the same time. The two services never coexist
in production — there is no period where both are live simultaneously charging the same account.

---

## Stripe account structure

| Environment | Stripe mode | API key prefix |
|---|---|---|
| Dev | Test | `sk_test_...` / `pk_test_...` |
| Staging | Test | `sk_test_...` / `pk_test_...` |
| Prod | Live | `sk_live_...` / `pk_live_...` |

The publishable key (`pk_...`) is shared between LFF and CF — it's safe to reuse since it's
account-scoped and carries no secrets.

---

## API keys

LFF currently uses a single secret key shared across all its services. Before CF goes to
production, create a **restricted key** for CF scoped to only what it needs:

| Resource | Permission needed |
|---|---|
| PaymentIntents | Write |
| Subscriptions | Write |
| Customers | Read + Write |
| Products / Prices | Read |
| Webhook endpoints | None (managed in dashboard) |

This means a LFF key rotation or revocation does not affect CF and vice versa.

**Environment variables:**
- `STRIPE_SECRET_KEY` — the restricted secret key (`sk_...`)
- `STRIPE_PUBLISHABLE_KEY` — the publishable key (same as LFF, set in frontend runtime config)

---

## Webhook endpoints

**This is the most important configuration step.** Each running service needs its own webhook
endpoint registered in the Stripe dashboard. Webhook secrets (`whsec_...`) are per-endpoint —
they cannot be shared.

### Endpoints to register

> **Note:** The URLs below are inferred from the ArgoCD/Helm config. Confirm the actual
> hostnames with the infrastructure team before registering in the Stripe dashboard.

| Environment | URL | Events to subscribe |
|---|---|---|
| Dev | `https://api.initiatives.dev.lfx.linuxfoundation.org/v1/stripe/webhook` | See below |
| Staging | `https://api.initiatives.staging.lfx.linuxfoundation.org/v1/stripe/webhook` | See below |
| Prod | `https://api.initiatives.lfx.linuxfoundation.org/v1/stripe/webhook` | See below |

**Events CF must subscribe to:**

| Event | Why |
|---|---|
| `payment_intent.succeeded` | Confirm one-time donation completed, update `donations.status` → `active` |
| `payment_intent.payment_failed` | Mark one-time donation `failed` so user can retry |
| `invoice.paid` | Confirm subscription first payment and renewals cleared, update `subscriptions.status` → `active` |
| `invoice.payment_failed` | Subscription first payment or renewal failed — mark subscription `past_due` or `failed` so user is notified. **Different event from `payment_intent.payment_failed`**, which only fires for standalone PaymentIntents (donations). |
| `customer.subscription.deleted` | Subscription cancelled by Stripe (payment failure, dashboard cancel, fraud) — update `subscriptions.status` → `canceled` |
| `customer.subscription.updated` | Optional: capture status changes (paused, past_due) |

**Do not subscribe to `charge.succeeded`** — that is the legacy Charges API used by LFF and
ledger-service. CF uses PaymentIntents exclusively.

After registering each endpoint, copy the signing secret into the environment variable:
`STRIPE_WEBHOOK_SECRET`

### Local development

Use Stripe CLI instead of a registered endpoint:

```bash
stripe listen --forward-to localhost:8080/v1/stripe/webhook
```

The CLI prints a local signing secret (`whsec_...`) — set that as `STRIPE_WEBHOOK_SECRET`
in your `.env`. No dashboard registration needed for local dev.

---

## Existing Stripe objects (inherited from LFF)

When CF migrates an initiative from DynamoDB, it brings along:
- `stripe_plan_id` — a Stripe Plan ID (legacy Plans API, format `plan_...`)
- `stripe_product_id` — a Stripe Product ID (format `prod_...`)

These IDs still work in CF because it's the same Stripe account. **Do not delete or recreate
them.** Existing active subscriptions reference these Plan IDs and continue billing correctly.

### Plans vs Prices

LFF used the **legacy Plans API** (`stripe.Plan`). Stripe has since replaced it with the
Prices API (`stripe.Price`), but Plans still work and are not being removed.

**Rules for CF:**
- **Existing initiatives** with a `stripe_plan_id` → use that Plan ID when creating new
  subscriptions for those initiatives. Don't force a migration.
- **New initiatives** created in CF → create a `stripe.Price` (not a Plan). Prices are the
  current API.
- **Existing subscribers** on legacy Plans → leave them alone. They continue billing on their
  existing Plan until they cancel and re-subscribe.

This means CF's subscription creation code must handle both Plan IDs and Price IDs. The
`stripe.SubscriptionItemsParams` accepts both in the `Price` field — Stripe normalises them.

---

## Stripe Customers

LFF stores the Stripe Customer ID on the user record in DynamoDB (`lff-{stage}-users`).
CF's `users` table does not currently have a `stripe_customer_id` column.

**Required:** Add `stripe_customer_id TEXT` to the `users` table. When a user makes their
first payment in CF, look up or create a Stripe Customer and store the ID.

**Do not create duplicate customers.** Before creating a new Stripe Customer, check whether
the user already has one (either migrated from LFF or created in a previous CF session).
If the user exists in LFF's DynamoDB with a Stripe Customer ID, that ID is valid in CF.

**How to obtain existing Customer IDs from LFF:** The data migration (see
`docs/rewrite/data-design_and_migration.md`) should pre-populate `users.stripe_customer_id`
from LFF's `lff-{stage}-users` DynamoDB table as part of the migration step. If it does not,
CF would need direct read access to that DynamoDB table at runtime — which is undesirable.
**Confirm with Lewis/data migration owner that `stripe_customer_id` is populated during
migration before implementing the look-up-or-create path.**

---

## Payment flow: one-time donations

CF uses **PaymentIntents** (not the legacy Charges API that LFF uses).

### Flow

```
1. User fills card details (Stripe Elements in frontend)
2. Frontend calls stripe.createPaymentMethod({ type: 'card', card: cardEl })
   → returns paymentMethodId

3. Frontend POSTs { initiativeId, amountCents, paymentMethodId, ... }
   to Nuxt server route /api/donate

4. Nuxt server route forwards to CF backend POST /v1/initiatives/{id}/donations
   with the paymentMethodId

5. CF backend:
   a. Validates initiative accepts funding
   b. Looks up or creates Stripe Customer for the user
   c. Creates PaymentIntent (Confirm: false):
      stripe.PaymentIntents.New({
        amount:         amountCents,
        currency:       "usd",
        customer:       stripeCustomerID,
        metadata: {
          initiative_id: initiativeId,
          user_id:       userId,
        },
      })
   d. Writes donation record to DB: status = "pending", stripe_payment_intent_id = pi.ID
   e. Returns { clientSecret: pi.ClientSecret } to frontend

6. Frontend calls stripe.confirmCardPayment(clientSecret, { payment_method: paymentMethodId })
   → Stripe.js handles 3DS challenge natively if required
   → Returns { paymentIntent: { status: "succeeded" } } on success

7. Frontend shows success screen optimistically

8. Stripe fires payment_intent.succeeded webhook to CF
   → CF handler looks up donation by stripe_payment_intent_id
   → Updates donation.status = "active"
   → Deduplicates using processed_stripe_events table (see below)
```

### Why not Confirm: true on the server?

Confirming server-side with `Confirm: true` fails silently for cards requiring 3DS
authentication — Stripe returns `requires_action` instead of `succeeded`, and the payment
stalls with no error to the user and no recovery path. Client-side confirmation via
`stripe.confirmCardPayment` handles 3DS natively.

---

## Payment flow: recurring subscriptions

### Stripe objects required per initiative

Before a user can subscribe to an initiative, that initiative needs a Stripe Price (or
legacy Plan). This is created when the initiative is approved for funding.

CF supports **two subscription frequencies: monthly and annual.** Each initiative needs
**two Prices** — one per interval. Create both when the initiative is approved:

```go
// Monthly Price
stripe.Prices.New({
  currency:    "usd",
  unit_amount: 1,           // quantity-per-subscription carries the actual amount
  recurring: { interval: "month" },
  product:     stripeProductID,
  metadata: { initiative_id: initiativeId, frequency: "monthly" },
})

// Annual Price
stripe.Prices.New({
  currency:    "usd",
  unit_amount: 1,
  recurring: { interval: "year" },
  product:     stripeProductID,
  metadata: { initiative_id: initiativeId, frequency: "annual" },
})
```

Store both Price IDs on the initiative: `stripe_monthly_price_id` and
`stripe_annual_price_id`. When creating a subscription, select the Price matching the
user's chosen frequency.

For **existing LFF initiatives** that carry a single `stripe_plan_id` (monthly-only Plans),
use that Plan ID for monthly subscriptions. If CF needs to offer annual subscriptions on
migrated initiatives, create an annual Price at migration time.

### Subscription creation flow

```
1. User selects amount + frequency, enters card details
2. Frontend creates paymentMethodId (same as donation flow)
3. CF backend:
   a. Creates/fetches Stripe Customer
   b. Attaches payment method to customer:
      stripe.PaymentMethods.Attach(paymentMethodId, { customer: customerId })
   c. Sets as default payment method on customer
   d. Creates Subscription:
      stripe.Subscriptions.New({
        customer:               customerId,
        items: [{ price: priceId, quantity: amountCents }],
        payment_behavior:       "default_incomplete",
        payment_settings: { save_default_payment_method: "on_subscription" },
        expand: ["latest_invoice.payment_intent"],
        metadata: { initiative_id: initiativeId, user_id: userId },
      })
   e. Writes subscription record: status = "pending",
      stripe_subscription_id = sub.ID
   f. Returns { clientSecret: sub.LatestInvoice.PaymentIntent.ClientSecret }

4. Frontend calls stripe.confirmCardPayment(clientSecret)
   → Handles 3DS if needed

5. Webhooks update state:
   invoice.paid              → subscriptions.status = "active"
   customer.subscription.deleted → subscriptions.status = "canceled"
```

Note `payment_behavior: "default_incomplete"` — this creates the subscription in
`incomplete` status until the first payment clears. Never grant access or show "active"
until `invoice.paid` fires.

### Subscription cancellation (user-initiated)

```
CF backend:
  stripe.Subscriptions.Cancel(stripeSubscriptionID)
  → Update subscriptions.status = "canceled" immediately (don't wait for webhook)
```

For user-initiated cancellations, update CF's DB synchronously. The
`customer.subscription.deleted` webhook will also fire — the deduplication table prevents
double-processing.

---

## Webhook handler implementation

### Idempotency — required

Stripe delivers webhooks **at least once**. The same event can arrive multiple times (network
retries, Stripe internal retries). Every handler must be idempotent.

**Implementation:** add a `processed_stripe_events` table:

```sql
CREATE TABLE IF NOT EXISTS processed_stripe_events (
  event_id    VARCHAR(255) PRIMARY KEY,   -- stripe event.ID, e.g. "evt_..."
  event_type  VARCHAR(100) NOT NULL,
  processed_at TIMESTAMPTZ DEFAULT NOW()
);
```

In each webhook handler, wrap the business logic and the event ID insert in a single
transaction:

```go
func (s *WebhookService) handlePaymentIntentSucceeded(ctx context.Context, event stripe.Event) error {
  return s.db.WithTx(ctx, func(tx pgx.Tx) error {
    // 1. Try to record the event — unique constraint rejects duplicates
    _, err := tx.Exec(ctx,
      `INSERT INTO processed_stripe_events (event_id, event_type) VALUES ($1, $2)
       ON CONFLICT (event_id) DO NOTHING`,
      event.ID, event.Type,
    )
    if err != nil {
      return err
    }
    // Check if the INSERT was a no-op (duplicate)
    // ... use pgconn.CommandTag.RowsAffected() == 0 to detect

    // 2. Business logic within the same transaction
    var pi stripe.PaymentIntent
    if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
      return err
    }
    initiativeID := pi.Metadata["initiative_id"]
    _, err := tx.Exec(ctx,
      `UPDATE donations SET status = 'active' WHERE stripe_payment_intent_id = $1`,
      pi.ID,
    )
    return err
  })
}
```

### Guarding against LFF-era events

CF shares the same Stripe account as LFF. When CF's webhook endpoint is registered, Stripe
will deliver **all** matching events to it — including events for Stripe objects created by
LFF (payments, subscriptions, invoices on LFF-era customers and Plans).

LFF objects carry `projectID` in metadata. CF objects carry `initiative_id`. Use this to
distinguish them:

```go
func initiativeIDFromMetadata(metadata map[string]string) (string, bool) {
  id, ok := metadata["initiative_id"]
  return id, ok && id != ""
}
```

At the top of every handler, check for `initiative_id`:

```go
initiativeID, ok := initiativeIDFromMetadata(pi.Metadata)
if !ok {
  // LFF-era event — not ours, skip silently
  return nil
}
```

Return `nil` (not an error) so CF responds 200 to Stripe. Returning an error causes Stripe
to retry, flooding CF with events it will never be able to process.

### STRIPE_WEBHOOK_ACK_UNIMPLEMENTED flag

The codebase has a `STRIPE_WEBHOOK_ACK_UNIMPLEMENTED` env var (config key
`StripeWebhookAckUnimplemented`). When `true`, the webhook handler returns 200 for known
event types that don't have a handler yet, instead of returning 400 and triggering Stripe
retries.

**This is scaffolding only.** It must be set to `false` (or removed) once real handlers
are in place. It must never be `true` in a production environment. Check:

- `.env` / `.env.example` — set to `false`
- `backend/charts/lfx-v2-initiatives-service/values.yaml` — ensure absent or `"false"`
- `backend/charts/lfx-v2-initiatives-service/secret.yaml` — same

If this flag is left `true` after handlers are implemented, CF will silently acknowledge
events it failed to process, losing webhook deliveries with no error signal.

### Response timing

Stripe requires a 2xx response within **5 seconds** or it marks the delivery as failed and
retries. The webhook handler must:
1. Verify signature
2. Write event to a processing queue or handle synchronously if fast
3. Return 200 immediately

For the initial implementation, synchronous handling is fine — DB updates are fast. Add async
queuing later if handler latency becomes an issue.

### Stripe retries

If CF returns a non-2xx response, Stripe retries with exponential backoff over **3 days**
(attempts at 5s, 5min, 30min, 2h, 5h, 10h, ...). This means:
- Transient DB errors will self-heal on retry
- The idempotency table prevents double-processing on retry

---

## Schema changes required

The `donations` table currently has a `stripe_charge_id` column — a misnomer inherited from
the legacy Charges API. CF uses PaymentIntents, not Charges. **Rename this column** in the
migration so the schema is correct from the start:

```sql
-- Rename misleading column (Charges API → PaymentIntents API)
ALTER TABLE donations RENAME COLUMN stripe_charge_id TO stripe_payment_intent_id;
```

Add remaining columns and tables:

```sql
-- Track Stripe Customer ID per user
ALTER TABLE users ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(255);

-- Webhook idempotency
CREATE TABLE IF NOT EXISTS processed_stripe_events (
  event_id     VARCHAR(255) PRIMARY KEY,
  event_type   VARCHAR(100) NOT NULL,
  processed_at TIMESTAMPTZ  DEFAULT NOW()
);

-- Index for fast lookups (stripe_payment_intent_id index replaces the old stripe_charge_id index)
CREATE INDEX IF NOT EXISTS idx_donations_stripe_pi
  ON donations(stripe_payment_intent_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_sub
  ON subscriptions(stripe_subscription_id);
```

Update the `Donation` domain model (`internal/domain/models/donation.go`) to rename the
`StripeChargeID` field to `StripePaymentIntentID` and update all references in the
repository and service layers.

---

## Metadata conventions

All Stripe objects created by CF must carry these metadata fields so webhooks can route
events back to the correct CF records:

| Object | Required metadata |
|---|---|
| PaymentIntent | `initiative_id`, `user_id` |
| Subscription | `initiative_id`, `user_id` |
| Price (new) | `initiative_id` |

LFF used `projectID` as the metadata key. CF uses `initiative_id`. Since both services
share the same Stripe account but are active at different times (LFF retires when CF
launches), there is no conflict — Stripe objects created by LFF carry `projectID`,
objects created by CF carry `initiative_id`. The CF webhook handler only processes events
for objects it created (identifiable by `initiative_id` in metadata).

---

## Ledger-service relationship

Ledger-service has its own independently registered Stripe webhook endpoint listening for
`charge.succeeded`. It will continue receiving events triggered by CF's PaymentIntents —
when a PaymentIntent succeeds, Stripe also creates a Charge, which fires `charge.succeeded`
to ledger-service. **This is intentional.** Ledger-service remains the financial audit trail
regardless of whether the originating payment was a Charge (LFF) or a PaymentIntent (CF).

CF does not call ledger-service directly for payment recording. The flow is:
```
CF creates PaymentIntent → Stripe → charge.succeeded → ledger-service (audit trail)
                                  → payment_intent.succeeded → CF (status update)
```

Ledger-service also handles `customer.subscription.deleted` and writes subscription state
into the DynamoDB tables (`lff-{stage}-subscriptions`, `lff-{stage}-entity-subscriptions`)
that it shares with LFF. When CF fires a `customer.subscription.deleted` event for a CF
subscription, ledger-service will attempt to find and update the matching DynamoDB record —
it will find nothing (CF subscriptions live in Postgres, not DynamoDB) and no-op silently.
This is harmless, but Lewis should understand why this will happen.

---

## 3DS / SCA — pending PM decision

Whether CF supports 3D Secure for the initial release is a product decision currently pending.
The implementation above assumes 3DS support (client-side confirmation via
`stripe.confirmCardPayment`). If the PM decides to defer 3DS, the frontend step changes to
server-side confirm, but this creates tech debt that will need to be addressed before
significant EU traffic.

See open question in [03-open-questions.md](03-open-questions.md) once confirmed.

---

## Checklist for Lewis

**Stripe dashboard setup**
- [ ] Stripe restricted API key created for CF (separate from LFF)
- [ ] Confirm actual service hostnames with infra team before registering webhook endpoints
- [ ] Webhook endpoint registered in Stripe dashboard for dev environment
- [ ] Webhook endpoint registered in Stripe dashboard for staging environment
- [ ] `STRIPE_WEBHOOK_SECRET` env var populated for dev and staging

**Schema migrations**
- [ ] `donations.stripe_charge_id` renamed to `stripe_payment_intent_id`
- [ ] `Donation` domain model field renamed to `StripePaymentIntentID`; all repo/service references updated
- [ ] `stripe_customer_id` column added to `users` table
- [ ] `processed_stripe_events` table created
- [ ] `stripe_monthly_price_id` and `stripe_annual_price_id` columns added to `initiatives` table

**Webhook handler implementation**
- [ ] LFF-era event guard in every handler (skip if `initiative_id` metadata absent)
- [ ] `payment_intent.succeeded` handler implemented
- [ ] `payment_intent.payment_failed` handler implemented (one-time donations)
- [ ] `invoice.paid` handler implemented
- [ ] `invoice.payment_failed` handler implemented (subscriptions)
- [ ] `customer.subscription.deleted` handler implemented
- [ ] `STRIPE_WEBHOOK_ACK_UNIMPLEMENTED` set to `false` (or removed) in `.env`, `.env.example`, and Helm chart values once real handlers are in place

**Subscription Price creation**
- [ ] Monthly and annual Prices created per initiative (two Prices per initiative)
- [ ] Price IDs stored on initiative record

**Testing**
- [ ] Local dev tested end-to-end with Stripe CLI
- [ ] One-time donation tested with basic success card: `4242 4242 4242 4242`
- [ ] Decline case tested with: `4000 0000 0000 0002` (card declined)
- [ ] Subscription failure tested with: `4000 0000 0000 0341` (attaches OK, fails on charge)
- [ ] 3DS tested with Stripe test card `4000 0025 0000 3155` (pending PM decision — see 3DS section)
- [ ] Idempotency verified: replaying the same webhook event twice does not double-update DB

**Pre-production**
- [ ] Prod webhook endpoint registered before LFF retirement
- [ ] `STRIPE_WEBHOOK_ACK_UNIMPLEMENTED` confirmed absent/false in all deployed environments
