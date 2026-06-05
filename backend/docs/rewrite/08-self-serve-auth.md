<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Self Serve → Crowdfunding API Authentication

---

This document covers the **Self Serve (SS) side** of calling the Crowdfunding (CF) backend.
The canonical authentication model — scopes, route classes, token validation, the Go API decision
tree, and Auth0/infra configuration — lives in
[`09-authentication-architecture.md`](09-authentication-architecture.md). Read that first; this
document only adds the SS-specific integration details.

---

## 1. How SS authenticates to CF

SS forwards the **logged-in user's own access token** to CF. There is no M2M token and no identity
header for SS — all SS→CF calls are me-style endpoints (`/v1/me/*`), and the user token carries the
acting user's identity via the LF SSO username claim (per Design Rule 3 in
[`09`](09-authentication-architecture.md#design-rules)).

```
User action in SS that needs CF data
  └─ SS BFF resolves the effective user's access token (see §3)
  └─ SS BFF proxies to CF /v1/me/*
       Authorization: Bearer {effective user's access token}
```

The CF audience (`access:me` scope) must be requested when SS mints the user's token, or CF will
reject the call. SS needs only the CF API base URL to make the call — no M2M client credentials.

---

## 2. Required SS-side changes

| Area | Change |
|---|---|
| `lfx-self-serve` | A `crowdfunding.service.ts` / proxy routes under `/api/crowdfunding/*` that forward the effective user's bearer token to CF `/v1/me/*`. |
| Auth0 token request | Include the CF API audience + `access:me` scope so the user token is accepted by CF. |
| `lfx-v2-argocd` | `values/*/lfx-self-serve.yaml`: `CROWDFUNDING_API_BASE_URL` (and CF audience for the token request). No M2M client/secret for CF. |

Infra specifics (Auth0 scopes/grants, CF backend env) are in
[`09`](09-authentication-architecture.md#auth0-terraform). SS needs no client-ID allowlist entry
and no identity header — the user token carries identity, and the `access:me` scope is the gate.

---

## 3. Impersonation

Admin impersonation works transparently with this model because SS swaps the **token**, not an
identity header. CF never needs to know impersonation is happening — it sees a normal `access:me`
user token.

The token SS forwards to CF must always represent the **effective user**:

- **Normal:** the logged-in user's own access token.
- **Impersonating:** a token minted for the **target user** (via a token exchange, authorized by
  the impersonator's `http://lfx.dev/claims/can_impersonate` claim) and used as the bearer while
  impersonation is active.

**Requirement:** the CF integration must forward this effective-user token — the same token used
for impersonation-aware upstream calls — and must **not** use a service/gateway token that carries
the admin's own identity. Because the forwarded token already represents the effective user, CF's
owner check and `Principal.Username` resolve correctly with no CF-side special handling.

> Write access under impersonation is a product-level decision, gated on the SS side.

---

## 4. Notes

- **Identity migration (OQ-23).** New CF treats the LF SSO username as the canonical user
  identifier; CF handlers key on `Principal.Username`, not the Auth0 `sub`. See
  [`09`](09-authentication-architecture.md) for how the username claim is sourced.
- **Heimdall.** CF sits outside the platform API gateway, so SS calls CF directly. The
  token-forwarding pattern here does not block a future move behind Heimdall. See the Known
  Deviations section in [`09`](09-authentication-architecture.md#known-deviations--future-direction).

---

## Related Documents

- [`09-authentication-architecture.md`](09-authentication-architecture.md) — canonical CF authentication design (scopes, routes, validation, Auth0/infra)
