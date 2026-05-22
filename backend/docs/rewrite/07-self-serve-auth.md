<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Self Serve → Crowdfunding API Authentication

**Status:** Proposed — pending architect review
**Author:** Michal Lehotsky
**Related:** `auth0-terraform`, `lfx-self-serve`, `lfx-v2-argocd`
**Context:** Follows architecture discussion with Eric Searcy and David Deal. Eric confirmed M2M + audience pattern and explicit identity injection for impersonation.

---

## 1. Problem

LFX Self Serve (SS) needs to call Crowdfunding (CF) backend APIs on behalf of authenticated users, including when an admin is impersonating another user. CF validates JWTs on every protected route, so SS must obtain a token CF will accept and communicate the correct acting-user identity.

---

## 2. Approach: M2M + Explicit Identity Header

SS authenticates to CF using **M2M client credentials** — the same pattern SS already uses for CDP. SS obtains a CF-scoped Auth0 access token via `client_credentials`, passes it as the Bearer token, and sends the acting user's identity in an **`X-User-ID` header**. CF trusts this header only from verified M2M callers.

SS populates `X-User-ID` using the existing `getEffectiveSub()` helper, which returns the impersonated user's `sub` when impersonation is active and the logged-in user's `sub` otherwise.

```typescript
const token = await this.getM2MToken(CROWDFUNDING_API_AUDIENCE); // cached ~24hr

await fetch(`${CROWDFUNDING_API_BASE_URL}/v1/initiatives`, {
  headers: {
    'Authorization': `Bearer ${token}`,
    'X-User-ID': getEffectiveSub(req),
  },
});
```

**Why not forward the user token directly?** SS calls `user-service` via `apiGatewayToken`, but that token always carries the **admin's** identity — even during impersonation. Forwarding it to CF would silently operate on the admin's data instead of the target user's (see Appendix for the broader impact of this bug). M2M + explicit `X-User-ID` makes identity intentional.

---

## 3. Token Flow

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
  └─ SS resolves acting user identity via getEffectiveSub()
       normal:        X-User-ID = logged-in user's sub
       impersonating: X-User-ID = impersonated user's sub

  └─ SS BFF proxies request to CF
       Authorization: Bearer {M2M token}
       X-User-ID:     {acting user sub}

CF M2M middleware
  1. Validates Bearer token: signature, issuer, CF M2M audience, expiry
  2. Checks azp claim matches SS client ID (rejects unknown callers)
  3. Reads X-User-ID → Principal.UserID
  4. Handler proceeds with correct user identity
```

---

## 4. Impersonation

When impersonation is active, `getEffectiveSub()` returns the impersonated user's `sub`. CF receives a correctly identified call and has no need to know impersonation is occurring. The audit trail (who impersonated whom) is maintained in SS's session.

**Write access under impersonation** is a product-level decision. Any read-only gating is implemented on the product side. A future option is splitting the CF Auth0 scope into `read` and `write` — not in scope for initial release.

---

## 5. Required Changes

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
| `values/{dev,staging,prod}/lfx-crowdfunding-backend.yaml` | Add `M2M_JWT_AUDIENCE` and `M2M_JWKS_URL`; `JWT_AUDIENCE` unchanged |
| `values/{dev,staging,prod}/lfx-self-serve.yaml` | Add `CROWDFUNDING_API_BASE_URL` and `CROWDFUNDING_API_AUDIENCE` |

### `lfx-self-serve`

New `crowdfunding.service.ts` modelled on `cdp.service.ts`:
- M2M token via `client_credentials` using existing `PCC_AUTH0_CLIENT_ID/SECRET`
- Proxy routes under `/api/crowdfunding/*` with M2M Bearer + `X-User-ID`
- `getEffectiveSub()` for identity resolution (already exists)

No changes to auth middleware, session types, or existing token exchange logic.

### `lfx-crowdfunding` backend

- New `M2MMiddleware`: validates CF-scoped Bearer token against `M2M_JWT_AUDIENCE`, checks `azp`, reads `X-User-ID` → `Principal.UserID`
- Registered as an alternative to the existing user JWT middleware on protected routes
- Stripe webhook (`POST /v1/stripe/webhook`) is already outside the JWT middleware — no changes needed

---

## 6. What This Does Not Need

- New Auth0 resource server beyond the dedicated CF one
- Changes to Heimdall routing or `lfx-platform.yaml`
- New token exchange logic in SS auth middleware
- New session fields
- Changes to the existing user JWT middleware in CF
- Snowflake integration for CF data

---

## 7. Long-term: Heimdall Alignment

Every other LFX v2 service sits behind Heimdall. Adding Heimdall to CF would normalise both user and M2M tokens through the platform gateway, making `X-User-ID` unnecessary — impersonation identity would flow through the Heimdall-issued JWT directly.

This is the correct long-term architecture but is not in scope now. The M2M approach proposed here does not block it — migrating CF to Heimdall later is a contained ArgoCD + Helm chart change with no SS BFF changes required.

---

## 8. Open Items for Architect Sign-off

| Item | Status | Detail |
|---|---|---|
| M2M + audience mechanism | ✅ Confirmed by Eric | Same pattern as sanctions screening |
| `X-User-ID` header trust model | 🔲 Needs sign-off | Concrete header name and CF's trust policy (trusted only from verified M2M callers) |
| Impersonation write access | 🔲 Product decision | Read-only gating is product-side for launch; future option is `read`/`write` scope split |
| Impersonation bug in SS (Appendix) | 🔲 Needs Eric's input | Data integrity risk — should this be tracked as a separate SS ticket? |
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
