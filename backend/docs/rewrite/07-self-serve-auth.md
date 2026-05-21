<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Self Serve → Crowdfunding API Authentication

**Status:** Proposed — pending architect review
**Author:** Michal Lehotsky
**Related:** `auth0-terraform`, `lfx-self-serve`, `lfx-v2-argocd`
**Context:** Follows architecture discussion with Eric Searcy and David Deal. Eric confirmed M2M + audience pattern (same as sanctions screening service) and explicit identity injection for impersonation.

---

## 1. Problem

LFX Self Serve (SS) needs to call Crowdfunding (CF) backend APIs on behalf of authenticated users, including when an admin is impersonating another user. CF validates JWTs on every protected route, so SS must obtain a token CF will accept and communicate the correct user identity — including the impersonated user's identity when applicable.

All CF data in SS is served in real time via direct API calls. There is no Snowflake dependency for crowdfunding — donations and subscription changes must be visible immediately.

---

## 2. Approach: M2M + Explicit User Identity Header

SS authenticates to CF using **M2M client credentials** — the same pattern as the sanctions screening service. SS obtains a CF-scoped Auth0 access token using its own `client_id`/`client_secret`, and passes it as the Bearer token on every CF API call.

The acting user's identity is passed separately as an **`X-User-ID` header**. CF trusts this header only from authenticated M2M callers. SS populates it using the existing `getEffectiveSub()` helper, which returns the impersonated user's `sub` when impersonation is active and the logged-in user's `sub` otherwise.

This is what Eric described in the architecture discussion: *"the UI checks if it's currently impersonating someone else, and injects that identity into the machine-authorised API call."*

### Why not forward the user token directly

SS already calls `user-service` via `apiGatewayToken`. Despite the name, this is not an M2M token — it is a user token derived from the admin's OIDC refresh token with a different audience. Its `sub` is always the **admin's** user ID, even during impersonation.

This means calling CF with `apiGatewayToken` during impersonation would silently use the admin's identity instead of the target user's — the same bug that currently affects the Rewards tab (see Section 8). M2M + explicit `X-User-ID` avoids this: identity is always intentional, not inferred from whichever token happened to be available.

---

## 3. How SS Does M2M Today

SS already uses this exact pattern for CDP. The `lfx_one` Auth0 client has `client_credentials` grant enabled, and SS uses `PCC_AUTH0_CLIENT_ID` / `PCC_AUTH0_CLIENT_SECRET` to obtain audience-scoped tokens, cached with a buffer before expiry.

The CF integration follows `cdp.service.ts`:

```typescript
// M2M token obtained via client_credentials, cached ~24hr
const token = await this.getM2MToken(CROWDFUNDING_API_AUDIENCE);

// Every CF API call includes the M2M token + acting user identity
await fetch(`${CROWDFUNDING_API_BASE_URL}/v1/initiatives`, {
  headers: {
    'Authorization': `Bearer ${token}`,
    'X-User-ID': getEffectiveSub(req),  // impersonated sub when active, otherwise logged-in sub
  },
});
```

---

## 4. Token Flow

```
SS server start / first CF call
  └─ Auth0 token endpoint (client_credentials grant)
       client_id     = PCC_AUTH0_CLIENT_ID
       client_secret = PCC_AUTH0_CLIENT_SECRET
       audience      = https://crowdfunding.{env}.platform.linuxfoundation.org/
       → M2M access token (cached, ~24hr lifetime)

User (or admin impersonating user) navigates to a CF feature in SS
  └─ SS resolves acting user identity
       normal:        X-User-ID = logged-in user's sub
       impersonating: X-User-ID = impersonated user's sub  ← via getEffectiveSub()

  └─ SS BFF proxies request to CF
       Authorization: Bearer {M2M token}
       X-User-ID:     {acting user sub}

CF M2M middleware
  1. Validates Bearer token: signature, issuer, CF audience, expiry
  2. Checks azp claim matches SS client ID (rejects unknown callers)
  3. Reads X-User-ID → Principal.UserID
  4. Handler proceeds with correct user identity
```

---

## 5. Impersonation

When impersonation is active, `getEffectiveSub()` returns the **impersonated user's** `sub`. CF receives a correctly identified M2M call and has no need to know impersonation is occurring. The audit trail (who impersonated whom) is maintained in SS's session (`impersonator` and `impersonationUser` fields).

**Write access under impersonation** is a product-level decision. Eric noted any read-only gating would be implemented on the product side. A future option is splitting the CF Auth0 scope into `read` and `write`, enforcing this at the API level — not in scope for initial release.

---

## 6. Callers: Reimbursement Service

The reimbursement service (AWS Lambda, outside the LFX v2 cluster) calls CF via M2M to update travel fund ticket statuses. The request carries only `ticketID` and `ticketStatus` — no user identity is needed. CF validates the M2M token and checks the `azp` is an allowed caller; no `X-User-ID` header required.

The endpoint (`POST /v1/service/travel-fund/ticket-status`) does not yet exist in the new CF backend and is not on the current critical path.

---

## 7. Callers: Stripe Webhook

Stripe calls CF directly via `POST /v1/stripe/webhook`, already outside the JWT middleware, validated with HMAC (`STRIPE_WEBHOOK_SECRET`). No changes needed.

---

## 8. Required Changes

### `auth0-terraform` — new file `grants_crowdfunding.tf` (~40 lines, pure addition)

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
    user   { policy = "deny_all" }           # user tokens cannot authenticate directly
    client { policy = "require_client_grant" }
  }
}

resource "auth0_resource_server_scopes" "lfx_crowdfunding" {
  resource_server_identifier = auth0_resource_server.lfx_crowdfunding.identifier
  scopes { name = "access:api" description = "Access Crowdfunding API" }
}

# Grant SS (lfx_one) permission to request CF-scoped tokens
resource "auth0_client_grant" "lfxone_crowdfunding" {
  client_id  = auth0_client.lfx_one.id
  audience   = auth0_resource_server.lfx_crowdfunding.identifier
  scopes     = ["access:api"]
  depends_on = [auth0_resource_server_scopes.lfx_crowdfunding]
}
```

No existing resources touched. Follows `grants_sanctions_screening.tf` exactly.

### `lfx-v2-argocd`

| File | Change |
|---|---|
| `values/{dev,staging,prod}/lfx-crowdfunding-backend.yaml` | Update `JWT_AUDIENCE` to the new CF audience per environment |
| `values/{dev,staging,prod}/lfx-self-serve.yaml` | Add `CROWDFUNDING_API_BASE_URL` and `CROWDFUNDING_API_AUDIENCE` |

### `lfx-self-serve`

New `crowdfunding.service.ts` following `cdp.service.ts`:
- M2M token via `client_credentials` using existing `PCC_AUTH0_CLIENT_ID/SECRET`
- Proxy routes under `/api/crowdfunding/*` with M2M Bearer + `X-User-ID` header
- `getEffectiveSub()` for identity resolution (already exists)

No changes to auth middleware, session types, or existing token exchange logic.

### `lfx-crowdfunding` backend

- New `M2MMiddleware`: validates CF-scoped Bearer token, checks `azp`, reads `X-User-ID` → `Principal.UserID`
- Register as alternative to existing user JWT middleware on protected routes
- `JWT_AUDIENCE` updated via ArgoCD (no code change)

---

## 10. Long-term: Heimdall Alignment

Every other LFX v2 native service sits behind Heimdall. Adding Heimdall to CF would normalise both user and M2M tokens through the platform gateway, making the `X-User-ID` header unnecessary — user identity would flow through the Heimdall-issued JWT directly, and impersonation would work transparently.

This is the correct long-term architecture but is not in scope for the current timeline. The M2M approach proposed here does not block it — migrating CF to Heimdall later is a contained ArgoCD + CF Helm chart change with no SS BFF changes required.

---

## 11. Open Items for Architect Sign-off

| Item | Status | Detail |
|---|---|---|
| M2M + audience mechanism | ✅ Confirmed by Eric | Same pattern as sanctions screening |
| `X-User-ID` header for identity injection | 🔲 Needs sign-off | Eric confirmed the pattern; this document proposes the concrete header name and CF's trust model (trusted only from verified M2M callers) |
| Impersonation write access | 🔲 Product decision | Read-only gating is product-side for launch; future option is `read`/`write` scope split on the CF resource server |
| Impersonation bug in SS (Section 8) | 🔲 Needs Eric's input | Affects Rewards, visa/travel applications, membership — data integrity risk. Recommend disabling affected features during impersonation (Option A) immediately. Should this be tracked as a separate SS ticket? |
| Reimbursement Service M2M | 🔲 Not on critical path | Addressed when travel fund endpoint is implemented |
| Heimdall alignment | 🔲 Planned follow-up | No hard deadline; does not block launch |

---

## 12. Appendix: Known Impersonation Bug in Self Serve (separate ticket)

Analysing this auth approach exposed a broader impersonation bug in SS. Because `apiGatewayToken` is always derived from the admin's refresh token, it always carries the **admin's** identity. Every feature that uses it during impersonation silently operates on the admin's data instead of the target user's:

| Feature | Impact |
|---|---|
| Rewards tab | Shows admin's points and coupons, not the target user's |
| Visa letter requests | Submits under admin's Salesforce ID — wrong person's record created |
| Travel fund requests | Submits under admin's Salesforce ID — wrong person's record created |
| Membership / enrollment | Reads and modifies admin's membership, not target's |

Visa and travel fund submissions are a **data integrity issue** — records are created under the wrong person's identity. This is also why the `apiGatewayToken` forwarding pattern was not adopted for CF.

This is a separate SS bug, not CF scope. Proposed resolution:

**Option A (ship now):** Disable the affected features during impersonation with a clear "not available during impersonation" message. Stops bad data being written immediately.

**Option B (follow-up):** Fix the root cause — derive `apiGatewayToken` from the impersonation token when impersonation is active rather than the admin's refresh token. Requires investigating whether Auth0 supports this token exchange.

**Recommendation:** ship Option A now, track Option B separately.
