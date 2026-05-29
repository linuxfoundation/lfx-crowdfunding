<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Self Serve → Crowdfunding API Authentication

**Status:** Proposed — pending architect review
**Author:** Michal Lehotsky
**Related:** `auth0-terraform`, `lfx-self-serve`, `lfx-v2-argocd`
**Context:** Follows architecture discussions with Eric Searcy and David Deal (May 2026). Eric confirmed M2M + audience pattern and explicit identity injection for impersonation. Eric also confirmed CF endpoints are **me-style** (user-scoped), and that the platform is migrating from Auth0 `sub` to LFID usernames — `X-Username` must carry the username, not the sub.

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

SS authenticates to CF using **M2M client credentials** — the same pattern SS already uses for CDP. SS obtains a CF-scoped Auth0 access token via `client_credentials`, passes it as the Bearer token, and sends the acting user's LFID username in an **`X-Username` header**. CF trusts this header only from verified M2M callers.

> **Why `X-Username` and not `X-User-ID`?**  
> The LFX v2 platform is migrating away from Auth0 `sub` identifiers (`auth0|...`) to LFID usernames across all services (see OQ-23). Eric explicitly recommended adopting usernames during this migration. `X-Username` makes the contract explicit — the value is a human-readable LFID username, not an opaque Auth0 identifier.

SS populates `X-Username` using the **LFID username** of the acting user — resolved via the existing `getEffectiveUsername()` helper (or equivalent), which returns the impersonated user's username when impersonation is active and the logged-in user's username otherwise.

```typescript
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
       client_id     = PCC_AUTH0_CLIENT_ID
       client_secret = PCC_AUTH0_CLIENT_SECRET
       audience      = https://crowdfunding.{env}.platform.linuxfoundation.org/
                        (new M2M-only audience — separate from the user-token
                         audience https://funding.{env}.platform.linuxfoundation.org/api/)
       → M2M access token (cached, ~24hr lifetime)

User navigates to a CF feature in SS
  └─ SS resolves acting user LFID username
       normal:        X-Username = logged-in user's LFID username
       impersonating: X-Username = impersonated user's LFID username

  └─ SS BFF proxies request to CF me-style endpoint
       Authorization: Bearer {M2M token}
       X-Username:    {LFID username of acting user}

CF M2M middleware
  1. Validates Bearer token: signature, issuer, CF M2M audience, expiry
  2. Checks azp claim matches SS client ID (rejects unknown callers)
  3. Reads X-Username → Principal.Username
  4. Handler proceeds with correct user identity
```

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      LFX Self Serve (SS)                        │
│                                                                  │
│  User browser ──► Nuxt BFF server                               │
│                       │                                          │
│                       ├─ Auth0 client_credentials grant         │
│                       │   audience: crowdfunding.{env}.platform │
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
│  M2M middleware                                                  │
│    1. Validate Bearer token (sig, issuer, M2M audience, expiry) │
│    2. Check azp == SS client ID                                  │
│    3. Read X-Username → Principal.Username                       │
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

Pure addition, ~40 lines. Follows `grants_sanctions_screening.tf` exactly. No existing resources touched.

```hcl
resource "auth0_resource_server" "lfx_crowdfunding" {
  identifier = {
    "dev"     = "https://crowdfunding.dev.platform.linuxfoundation.org/"
    "staging" = "https://crowdfunding.staging.platform.linuxfoundation.org/"
    "prod"    = "https://crowdfunding.platform.linuxfoundation.org/"
  }[terraform.workspace]
  signing_alg    = "RS256"
  token_lifetime = { "dev" = 86400, "staging" = 86400, "prod" = 86400 }[terraform.workspace]
  subject_type_authorization {
    user   { policy = "deny_all" }           # M2M-only audience
    client { policy = "require_client_grant" }
  }
}

resource "auth0_resource_server_scopes" "lfx_crowdfunding" {
  resource_server_identifier = auth0_resource_server.lfx_crowdfunding.identifier
  scopes { name = "access:api" description = "Access Crowdfunding API" }
}

resource "auth0_client_grant" "lfxone_crowdfunding" {
  client_id  = auth0_client.lfx_one.id
  audience   = auth0_resource_server.lfx_crowdfunding.identifier
  scopes     = ["access:api"]
  depends_on = [auth0_resource_server_scopes.lfx_crowdfunding]
}
```

> **Note:** `deny_all` applies only to this new M2M audience. The existing user-token flow uses a separate audience (`https://funding.{env}.platform.linuxfoundation.org/api/`) validated via `JWT_AUDIENCE`, `JWT_ISSUER`, and `JWKS_URL` (see `backend/cmd/initiatives-api/config.go`; per-environment values in `lfx-v2-argocd`). It is unaffected.

### `lfx-v2-argocd`

| File | Change |
|---|---|
| `values/{dev,staging,prod}/lfx-crowdfunding-backend.yaml` | Add `M2M_JWT_AUDIENCE` only; `JWT_AUDIENCE`, `JWT_ISSUER`, and `JWKS_URL` unchanged — M2M middleware reuses the same issuer and JWKS endpoint (same Auth0 tenant) |
| `values/{dev,staging,prod}/lfx-self-serve.yaml` | Add `CROWDFUNDING_API_BASE_URL` and `CROWDFUNDING_API_AUDIENCE` |

### `lfx-self-serve`

New `crowdfunding.service.ts` modelled on `cdp.service.ts`:
- M2M token via `client_credentials` using existing `PCC_AUTH0_CLIENT_ID/SECRET`
- Proxy routes under `/api/crowdfunding/*` with M2M Bearer + `X-Username`
- `getEffectiveUsername()` for identity resolution — resolves LFID username of the acting user (impersonated user's username when impersonation is active, logged-in user's username otherwise). The username is available via the `https://sso.linuxfoundation.org/claims/username` namespaced claim in the Auth0 JWT; if SS does not already have a helper that returns this for the effective user, one is needed.

No changes to auth middleware, session types, or existing token exchange logic.

### `lfx-crowdfunding` backend

- New `M2MMiddleware`: validates CF-scoped Bearer token against `M2M_JWT_AUDIENCE` (new env var), reusing existing `JWKS_URL` and `JWT_ISSUER`; checks `azp` claim matches SS client ID; reads `X-Username` header → `Principal.Username`
- Registered as an alternative to the existing user JWT middleware on protected routes
- Stripe webhook (`POST /v1/stripe/webhook`) is already outside the JWT middleware — no changes needed

> **Note for engineers — username is the canonical user identifier in new CF:**
>
> Per OQ-23, new CF uses **LFID username everywhere** for user identity — both the CF frontend (user JWT path) and SS → CF (M2M path). The change is **not** limited to me-routes or the M2M middleware: every handler that today reads `principal.UserID` for user identity must switch to `principal.Username`. This includes me-routes (`donation_handler.go`, `subscription_handler.go`, `payment_handler.go`) **and** non-me protected routes that perform ownership checks (`initiative_handler.go`, `upload_handler.go`, `subscription_handler.go:Cancel`). Service and repository layers update their parameter names and query columns to match.
>
> **LFF stays on Auth0 sub** — it's the retiring Lambda and is untouched. The username migration happens at the DynamoDB → Postgres boundary: existing Auth0 subs are bulk-resolved to LFID usernames via Auth0 Management API during the one-time migration script. Postgres stores both `users.user_id` (the sub, populated only for migrated rows) and `users.username` (LFID username, the join key). For users created after migration, `user_id` is NULL.
>
> Stripe customer metadata is updated in the same migration pass. The `users.stripe_customer_id` mapping is keyed by username going forward.
>
> The `Principal` struct already has both fields populated by the user JWT middleware (`UserID` = sub from the `sub` claim, `Username` = LFID username from `https://sso.linuxfoundation.org/claims/username`). The M2M middleware will set only `Principal.Username` (from `X-Username`); `Principal.UserID` will be empty on the M2M path — handlers must not depend on it.
>
> Full breakdown of schema, migration, and code changes is tracked in the Jira ticket created under LFXV2-1690.

---

## 7. What This Does Not Need

- New Auth0 resource server beyond the dedicated CF one
- Changes to Heimdall routing or `lfx-platform.yaml`
- New token exchange logic in SS auth middleware
- New session fields
- Changes to the existing user JWT middleware in CF
- Snowflake integration for CF data

---

## 8. Long-term: Heimdall Alignment

Every other LFX v2 service sits behind Heimdall. Adding Heimdall to CF would normalise both user and M2M tokens through the platform gateway, making `X-User-ID` unnecessary — impersonation identity would flow through the Heimdall-issued JWT directly.

This is the correct long-term architecture but is not in scope now. The M2M approach proposed here does not block it — migrating CF to Heimdall later is a contained ArgoCD + Helm chart change with no SS BFF changes required.

---

## 9. Open Items for Architect Sign-off

| Item | Status | Detail |
|---|---|---|
| M2M + audience mechanism | ✅ Confirmed by Eric | Same pattern as sanctions screening |
| Me-style endpoint design | ✅ Confirmed by Eric | Eric confirmed: passing an explicit identity header makes sense for me-style endpoints |
| `X-Username` header name and trust model | ✅ Confirmed | Renamed from `X-User-ID` to carry LFID username per Eric's recommendation; trusted only from verified M2M callers |
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
