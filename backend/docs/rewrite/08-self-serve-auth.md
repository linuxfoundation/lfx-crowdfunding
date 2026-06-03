<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Self Serve → Crowdfunding API Authentication

---

## 1. Problem

LFX Self Serve (SS) needs to call Crowdfunding (CF) backend APIs on behalf of authenticated users, including when an admin is impersonating another user. CF validates JWTs on every protected route, so SS must obtain a token CF will accept and communicate the correct acting-user identity.

---

## 2. Approach: M2M + Explicit Identity Header

SS authenticates to CF using **M2M client credentials** (same pattern as CDP). SS obtains an Auth0 access token via `client_credentials` for the CF API audience (`/api/`), passes it as the Bearer token, and sends the acting user's LFID username in an **`X-Username` header**.

CF uses a **single resource server** (`lfx_crowdfunding_api`, `/api/`) for both user tokens (CF frontend) and M2M tokens (SS) — matching the platform gateway pattern (`lfx_v2_api`, `lfx_api_gateway`). The JWT middleware distinguishes caller types by `gty`/`azp` claims and gates M2M callers via the `AUTHORIZED_CLIENTS` allowlist.

**Why not forward the user token directly?** SS's `apiGatewayToken` always carries the **admin's** identity, even during impersonation. Forwarding it to CF would silently operate on the admin's data instead of the target user's (see Appendix). M2M + explicit `X-Username` makes identity intentional.

**Why `X-Username`?** The platform is migrating from Auth0 `sub` identifiers to LFID usernames (OQ-23). `X-Username` carries a human-readable LFID username, not an opaque Auth0 sub.

---

## 3. Token Flow

```
SS server start / first CF call
  └─ Auth0 token endpoint (client_credentials grant)
       client_id     = PCC_AUTH0_CLIENT_ID   (the lfx_one client)
       client_secret = PCC_AUTH0_CLIENT_SECRET
       audience      = https://crowdfunding.{env}.lfx.dev/api/   (dev/staging)
                     = https://crowdfunding.linuxfoundation.org/api/  (prod)
       → M2M access token (cached, ~24hr lifetime)

User navigates to a CF feature in SS
  └─ SS resolves acting user LFID username via getEffectiveUsername()
       normal:        logged-in user's LFID username
       impersonating: impersonated user's LFID username

  └─ SS BFF proxies to CF /v1/me/* endpoint
       Authorization: Bearer {M2M token}
       X-Username:    {LFID username of acting user}
```

### Middleware flow

The `/v1/me/*` routes use `JWTAuthenticator.Middleware` which handles both user and M2M tokens against `JWT_AUDIENCE` (`/api/`).

```mermaid
flowchart TD
    A[Request hits /v1/me/*] --> B{bypass active?\nlocal dev only}
    B -- yes --> C[inject mock Principal\ncall next]
    B -- no --> D{validate Bearer token\nsig · issuer · JWT_AUDIENCE · expiry}
    D -- invalid --> E[401]
    D -- valid --> F{isM2MToken?\ngty=client_credentials\nor sub @clients}
    F -- no\nuser token --> G[Principal.Username\nfrom username JWT claim]
    F -- yes\nM2M token --> H{azp ∈\nAUTHORIZED_CLIENTS?}
    H -- no --> I[401]
    H -- yes --> J[Principal.Username\nfrom X-Username header]
    G --> K[call next]
    J --> K
```

**Key behaviour:**
- User token (CF frontend, `/api/` audience) → `Principal.Username` from `https://sso.linuxfoundation.org/claims/username` JWT claim
- M2M token (SS, `/api/` audience, `gty=client_credentials`) → `azp` checked against `AUTHORIZED_CLIENTS` → `Principal.Username` from `X-Username` header
- M2M token with unknown `azp` → **401**

---

## 4. Impersonation

`getEffectiveUsername()` returns the impersonated user's LFID username when impersonation is active. CF sees a normal authenticated call — it has no need to know impersonation is occurring. The audit trail is maintained in SS's session.

Write access under impersonation is a product-level decision; read-only gating is product-side for launch.

---

## 5. Required Changes

### `auth0-terraform` — `grants_crowdfunding.tf`

Single client grant on the existing `lfx_crowdfunding_api` (`/api/`) resource server. No new resource server needed — `lfx_crowdfunding_api` already accepts both users (`allow_all`) and M2M clients (`require_client_grant`).

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
| `values/{dev,staging,prod}/lfx-crowdfunding-backend.yaml` | `JWT_AUDIENCE`, `JWT_ISSUER`, `JWKS_URL` unchanged; `AUTHORIZED_CLIENTS` added (lfx_one client ID via ExternalSecret) |
| `values/{dev,staging,prod}/lfx-self-serve.yaml` | Add `CROWDFUNDING_API_BASE_URL` and `CROWDFUNDING_API_AUDIENCE` (`/api/`) |

### `lfx-self-serve`

New `crowdfunding.service.ts` modelled on `cdp.service.ts`:
- M2M token via `client_credentials` using `PCC_AUTH0_CLIENT_ID/SECRET` (the `lfx_one` client), minted directly by the service
- Proxy routes under `/api/crowdfunding/*` forwarding M2M Bearer + `X-Username`
- `getEffectiveUsername()` resolves the acting user's LFID username from the `https://sso.linuxfoundation.org/claims/username` claim

### `lfx-crowdfunding` backend

`JWTAuthenticator.Middleware` on `/v1/me/*` routes handles both caller types against the single `JWT_AUDIENCE`. It detects M2M via `gty`/`azp`, gates M2M callers against `AUTHORIZED_CLIENTS`, and reads `X-Username` → `Principal.Username` for trusted M2M callers. See flow diagram in §3.

> **Username is the canonical user identifier in new CF (OQ-23).** Both paths (user JWT and M2M) resolve to `Principal.Username` (LFID username). Handlers must use `Username`, not `UserID` — `UserID` on the M2M path is the Auth0 client credential subject, not a user identity. Full details in LFXV2-1690.

---

## 6. What This Does Not Need

- New Auth0 resource server (reuses `lfx_crowdfunding_api`)
- Changes to Heimdall routing or `lfx-platform.yaml`
- New token exchange logic in SS
- New session fields

---

## 7. Long-term: Heimdall Alignment

Every other LFX v2 service sits behind Heimdall. Adding Heimdall to CF would normalise both user and M2M tokens through the platform gateway, making `X-Username` unnecessary — impersonation identity would flow through the Heimdall-issued JWT directly. Not in scope now; the M2M approach does not block it.

---

## 8. Open Items

| Item | Status | Detail |
|---|---|---|
| M2M + single RS + `X-Username` | ✅ Confirmed | Eric Searcy (me-style + identity header); Robert Detjens (single RS) |
| sub → username migration | 🔲 Needs implementation | DB and Stripe customer keys store Auth0 sub; update via migration — tracked as OQ-23 |
| Impersonation write access | 🔲 Product decision | Read-only gating is product-side for launch |
| Impersonation bug in SS (Appendix) | 🔲 Pending Jordan | Confirm whether by-design or a real bug |
| Heimdall alignment | 🔲 Planned follow-up | Does not block launch |

---

## Appendix: Known Impersonation Bug in Self Serve

`apiGatewayToken` is always derived from the admin's refresh token and always carries the **admin's** identity. Every SS feature that uses it during impersonation silently operates on the admin's data:

| Feature | Impact |
|---|---|
| Rewards tab | Shows admin's points and coupons, not the target user's |
| Visa letter requests | Submits under admin's Salesforce ID — wrong record created |
| Travel fund requests | Submits under admin's Salesforce ID — wrong record created |
| Membership / enrollment | Reads and modifies admin's membership, not the target's |

Visa and travel fund submissions are a **data integrity issue**. This is why the `apiGatewayToken` forwarding pattern was not adopted for CF.

This is a separate SS bug, not CF scope.

**Option A (ship now):** Disable affected features during impersonation with a clear message.
**Option B (follow-up):** Fix root cause — derive `apiGatewayToken` from the impersonation token.

**Recommendation:** ship Option A now, track Option B separately.
