<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Self Serve → Crowdfunding API Authentication

**Status:** Approved — implemented (DEV)
**Author:** Michal Lehotsky
**Related:** `auth0-terraform` #321, `lfx-v2-argocd` #882/#893, `lfx-secrets-management` #262
**Context:** Follows architecture discussions with Eric Searcy and David Deal (May 2026). Reviewed by Robert Detjens and Eric Searcy. Collapsed to a single Auth0 resource server per reviewer feedback — matches the platform gateway pattern (`lfx_v2_api`, `lfx_api_gateway`). Platform is migrating from Auth0 `sub` to LFID usernames — `X-Username` carries the username, not the sub.

---

## 1. Problem

LFX Self Serve (SS) needs to call Crowdfunding (CF) backend APIs on behalf of authenticated users, including when an admin is impersonating another user. CF validates JWTs on every protected route, so SS must obtain a token CF will accept and communicate the correct acting-user identity.

---

## 2. API Design: Me-Style Endpoints

CF's protected routes are **me-style endpoints** — they are scoped to the acting user and do not accept a user ID as a path or query parameter. The calling identity is the entire access control decision. This is intentional: SS (and the CF frontend) call `/v1/me/donations`, `/v1/me/subscriptions`, `/v1/me/payment-account`, etc. and always receive data for the user identified in the request.

Current me-style routes (server.go:167–174):

| Method | Route | Handler |
|---|---|---|
| `GET` | `/v1/me/donations` | `DonationHandler.ListForUser` |
| `GET` | `/v1/me/subscriptions` | `SubscriptionHandler.ListForUser` |
| `POST` | `/v1/me/setup-intent` | `PaymentHandler.CreateSetupIntent` |
| `POST` | `/v1/me/payment-method` | `PaymentHandler.AttachPaymentMethod` |
| `GET` | `/v1/me/payment-account` | `PaymentHandler.GetPaymentAccount` |
| `DELETE` | `/v1/me/payment-method` | `PaymentHandler.DeletePaymentMethod` |

This design is consistent with how Eric described it on the call: for a me-style endpoint, passing an explicit identity header makes sense — the header tells CF which user's slice of data to operate on.

---

## 3. Approach: M2M + Explicit Identity Header

SS authenticates to CF using **M2M client credentials** — the same pattern SS already uses for CDP. SS obtains an Auth0 access token via `client_credentials` for the CF API audience (`/api/`), passes it as the Bearer token, and sends the acting user's LFID username in an **`X-Username` header**. CF trusts this header only from verified M2M callers.

CF uses a **single resource server** (`lfx_crowdfunding_api`, `/api/`) for both user tokens (CF frontend) and M2M tokens (SS) — matching the platform gateway pattern. CF's middleware distinguishes callers by `gty`/`azp` claims and gates M2M callers via the `AUTHORIZED_CLIENTS` allowlist.

> **Why `X-Username` and not `X-User-ID`?**  
> The LFX v2 platform is migrating away from Auth0 `sub` identifiers (`auth0|...`) to LFID usernames across all services (see OQ-23). Eric explicitly recommended adopting usernames during this migration. `X-Username` makes the contract explicit — the value is a human-readable LFID username, not an opaque Auth0 identifier.

SS populates `X-Username` using the **LFID username** of the acting user — resolved via the existing `getEffectiveUsername()` helper (or equivalent), which returns the impersonated user's username when impersonation is active and the logged-in user's username otherwise.

```typescript
// CrowdfundingService mints its own client_credentials token using
// PCC_AUTH0_CLIENT_ID/SECRET (the lfx_one client), modelled on cdp.service.ts —
// not the shared generateM2MToken util, which is bound to a different client.
const token = await this.getM2MToken(CROWDFUNDING_API_AUDIENCE); // cached ~24hr

await fetch(`${CROWDFUNDING_API_BASE_URL}/v1/me/donations`, {
  headers: {
    'Authorization': `Bearer ${token}`,
    'X-Username': getEffectiveUsername(req), // LFID username, NOT Auth0 sub
  },
});
```

**Why not forward the user token directly?** SS calls `user-service` via `apiGatewayToken`, but that token always carries the **admin's** identity — even during impersonation. Forwarding it to CF would silently operate on the admin's data instead of the target user's (see Appendix for the broader impact of this bug). M2M + explicit `X-Username` makes identity intentional.

---

## 4. Token Flow

```
SS server start / first CF call
  └─ Auth0 token endpoint (client_credentials grant)
       client_id     = PCC_AUTH0_CLIENT_ID      (the lfx_one client)
       client_secret = PCC_AUTH0_CLIENT_SECRET
       audience      = https://crowdfunding.{env}.lfx.dev/api/   (dev/staging)
                     = https://crowdfunding.linuxfoundation.org/api/  (prod)
       → M2M access token (cached, ~24hr lifetime)

User navigates to a CF feature in SS
  └─ SS resolves acting user LFID username
       normal:        X-Username = logged-in user's LFID username
       impersonating: X-Username = impersonated user's LFID username

  └─ SS BFF proxies request to CF me-style endpoint
       Authorization: Bearer {M2M token}
       X-Username:    {LFID username of acting user}

CF JWT middleware (handles user + M2M tokens)
  1. Validates Bearer token: signature, issuer, JWT_AUDIENCE (/api/), expiry
  2. Detects M2M via gty=client_credentials (or sub ending @clients)
  3. For M2M: checks azp ∈ AUTHORIZED_CLIENTS (rejects unknown callers)
  4. Reads X-Username → Principal.Username (M2M only; user tokens use the
     username claim from https://sso.linuxfoundation.org/claims/username)
  5. Handler proceeds with correct user identity
```

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      LFX Self Serve (SS)                        │
│                                                                  │
│  User browser ──► Nuxt BFF server                               │
│                       │                                          │
│                       ├─ Auth0 client_credentials grant         │
│                       │   audience: https://crowdfunding.{env}.lfx.dev/api/ (dev/staging)  │
│                       │             https://crowdfunding.linuxfoundation.org/api/ (prod)  │
│                       │   → M2M access token (cached ~24hr)     │
│                       │                                          │
│                       ├─ Resolve acting user (impersonation?)   │
│                       │   getEffectiveUsername(req)             │
│                       │   → LFID username                       │
│                       │                                          │
│                       └─ Proxy to CF /v1/me/* endpoint          │
│                           Authorization: Bearer {M2M token}     │
│                           X-Username: {LFID username}           │
└─────────────────────────────────────────────────────────────────┘
                               │
                               │ HTTPS
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                  CF Go API (Kubernetes)                          │
│                                                                  │
│  JWT middleware (user + M2M)                                     │
│    1. Validate Bearer token (sig, issuer, /api/ audience, expiry)│
│    2. Detect M2M (gty); check azp ∈ AUTHORIZED_CLIENTS          │
│    3. Read X-Username → Principal.Username (M2M only)           │
│                                                                  │
│  Me-style handler                                                │
│    GET /v1/me/donations        → donations for this username     │
│    GET /v1/me/subscriptions    → subscriptions for this username │
│    GET /v1/me/payment-account  → Stripe customer for username   │
│    POST /v1/me/setup-intent    → Stripe SetupIntent             │
│    POST /v1/me/payment-method  → attach card                    │
│    DELETE /v1/me/payment-method → remove card                   │
└─────────────────────────────────────────────────────────────────┘
                               │
                 normal flow   │  impersonation flow
                               │
          ┌────────────────────┴──────────────────────┐
          │                                           │
          ▼                                           ▼
  X-Username = logged-in user's           X-Username = impersonated
  LFID username                           user's LFID username
  (SS getEffectiveUsername returns it)    (SS getEffectiveUsername returns it)
```

---

## 5. Impersonation

When impersonation is active, `getEffectiveUsername()` returns the impersonated user's LFID username. CF receives a correctly identified call and has no need to know impersonation is occurring. The audit trail (who impersonated whom) is maintained in SS's session.

**Write access under impersonation** is a product-level decision. Any read-only gating is implemented on the product side. A future option is splitting the CF Auth0 scope into `read` and `write` — not in scope for initial release.

---

## 6. Required Changes

### `auth0-terraform` — new file `grants_crowdfunding.tf`

A single client grant on the existing `lfx_crowdfunding_api` (`/api/`) resource server. No new resource server needed.

```hcl
resource "auth0_client_grant" "lfxone_crowdfunding" {
  client_id  = auth0_client.lfx_one.id
  audience   = auth0_resource_server.lfx_crowdfunding_api.identifier
  scopes     = ["access:api"]
  depends_on = [auth0_resource_server_scopes.lfx_crowdfunding_api]
}
```

### `lfx-v2-argocd`

| File | Change |
|---|---|
| `values/{dev,staging,prod}/lfx-crowdfunding-backend.yaml` | Add `AUTHORIZED_CLIENTS` (lfx_one client ID via ExternalSecret); `JWT_AUDIENCE`, `JWT_ISSUER`, `JWKS_URL` unchanged |
| `values/{dev,staging,prod}/lfx-self-serve.yaml` | Add `CROWDFUNDING_API_BASE_URL` and `CROWDFUNDING_API_AUDIENCE` (`/api/`) |

### `lfx-self-serve`

New `crowdfunding.service.ts` modelled on `cdp.service.ts`:
- M2M token via `client_credentials` using `PCC_AUTH0_CLIENT_ID/SECRET` (the `lfx_one` client), the same client SS uses for CDP — minted by the service directly (as `cdp.service.ts` does), not via the shared `generateM2MToken` util, which is bound to a different client
- Proxy routes under `/api/crowdfunding/*` with M2M Bearer + `X-Username`
- `getEffectiveUsername()` for identity resolution — resolves LFID username of the acting user (impersonated user's username when impersonation is active, logged-in user's username otherwise). The username is available via the `https://sso.linuxfoundation.org/claims/username` namespaced claim in the Auth0 JWT; if SS does not already have a helper that returns this for the effective user, one is needed.

No changes to auth middleware, session types, or existing token exchange logic.

### `lfx-crowdfunding` backend

- The existing `JWTAuthenticator.Middleware` handles both user and M2M tokens against the single `JWT_AUDIENCE` (`/api/`): validates the Bearer token, detects M2M via `gty`/`azp`, gates M2M callers against `AUTHORIZED_CLIENTS`, and reads `X-Username` → `Principal.Username` for trusted M2M callers only
- Stripe webhook (`POST /v1/stripe/webhook`) is outside the JWT middleware — no changes needed

> **Note for engineers — username is the canonical user identifier in new CF:**
>
> Per OQ-23, new CF uses **LFID username everywhere** for user identity — both the CF frontend (user JWT path) and SS → CF (M2M path). The change is **not** limited to me-routes or the M2M middleware: every handler that today reads `principal.UserID` for user identity must switch to `principal.Username`. This includes me-routes (`donation_handler.go`, `subscription_handler.go`, `payment_handler.go`) **and** non-me protected routes that perform ownership checks (`initiative_handler.go`, `subscription_handler.go:Cancel`). Note: `upload_handler.go` only checks that a principal is present — it performs no per-user ownership lookup via `UserID` and does not need to change. Service and repository layers update their parameter names and query columns to match.
>
> **LFF stays on Auth0 sub** — it's the retiring Lambda and is untouched. The username migration happens at the DynamoDB → Postgres boundary: existing Auth0 subs are bulk-resolved to LFID usernames via Auth0 Management API during the one-time migration script. Postgres stores both `users.user_id` (the sub, populated only for migrated rows) and `users.username` (LFID username, the join key). For users created after migration, `user_id` is NULL.
>
> Stripe customer metadata is updated in the same migration pass. The `users.stripe_customer_id` mapping is keyed by username going forward.
>
> The `Principal` struct already has both fields populated by the user JWT middleware (`UserID` = sub from the `sub` claim, `Username` = LFID username from `https://sso.linuxfoundation.org/claims/username`). On the M2M path, `Principal.UserID` is set to the M2M client's subject (the Auth0 client credential identifier, not a user identity) and must not be used as the acting user. Only `Principal.Username` (from `X-Username`) carries the acting user's identity on the M2M path — handlers must use `Username` exclusively.
>
> Full breakdown of schema, migration, and code changes is tracked in the Jira ticket created under LFXV2-1690.

---

## 7. What This Does Not Need

- New Auth0 resource server (reuses the existing `lfx_crowdfunding_api`)
- Changes to Heimdall routing or `lfx-platform.yaml`
- New token exchange logic in SS auth middleware
- New session fields
- Snowflake integration for CF data

---

## 8. Long-term: Heimdall Alignment

Every other LFX v2 service sits behind Heimdall. Adding Heimdall to CF would normalise both user and M2M tokens through the platform gateway, making `X-Username` unnecessary — impersonation identity would flow through the Heimdall-issued JWT directly.

This is the correct long-term architecture but is not in scope now. The M2M approach proposed here does not block it — migrating CF to Heimdall later is a contained ArgoCD + Helm chart change with no SS BFF changes required.

---

## 9. Open Items for Architect Sign-off

| Item | Status | Detail |
|---|---|---|
| M2M + explicit identity header | ✅ Confirmed by Eric | M2M client credentials + `X-Username` for me-style endpoints |
| Single resource server (`/api/`) | ✅ Resolved — reviewed by Robert Detjens, Eric Searcy | Matches platform gateway pattern; callers distinguished by `gty`/`azp` + `AUTHORIZED_CLIENTS` |
| Me-style endpoint design | ✅ Confirmed by Eric | Passing an explicit identity header makes sense for me-style endpoints |
| `X-Username` header name and trust model | ✅ Confirmed | Carries LFID username per Eric's recommendation; trusted only from verified M2M callers |
| sub → username migration impact on CF | 🔲 Needs implementation | DB and Stripe customer keys currently store Auth0 sub; must be updated — tracked as OQ-23 |
| Impersonation write access | 🔲 Product decision | Read-only gating is product-side for launch; future option is `read`/`write` scope split |
| Impersonation bug in SS (Appendix) | 🔲 Pending Jordan | Michal notified Jordan to confirm whether this is a real bug or by-design |
| Heimdall alignment | 🔲 Planned follow-up | No hard deadline; does not block launch |

---

## Appendix: Known Impersonation Bug in Self Serve

Analysing this auth approach exposed a broader impersonation bug in SS. `apiGatewayToken` is always derived from the admin's refresh token, so it always carries the **admin's** identity. Every feature that uses it during impersonation silently operates on the admin's data:

| Feature | Impact |
|---|---|
| Rewards tab | Shows admin's points and coupons, not the target user's |
| Visa letter requests | Submits under admin's Salesforce ID — wrong record created |
| Travel fund requests | Submits under admin's Salesforce ID — wrong record created |
| Membership / enrollment | Reads and modifies admin's membership, not the target's |

Visa and travel fund submissions are a **data integrity issue**. This is also why the `apiGatewayToken` forwarding pattern was not adopted for CF.

This is a separate SS bug, not CF scope. Proposed resolution:

**Option A (ship now):** Disable the affected features during impersonation with a clear "not available during impersonation" message.

**Option B (follow-up):** Fix the root cause — derive `apiGatewayToken` from the impersonation token when impersonation is active.

**Recommendation:** ship Option A now, track Option B separately.
