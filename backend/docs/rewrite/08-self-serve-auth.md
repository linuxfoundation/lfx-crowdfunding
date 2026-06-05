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
[`09`](09-authentication-architecture.md#auth0-terraform). SS needs **no** `AUTHORIZED_CLIENTS`
entry and **no** `X-Username` wiring — both were part of the superseded design.

---

## 3. Impersonation

SS supports admin impersonation, and it works transparently with this model because SS swaps the
**token**, not an identity header. CF never needs to know impersonation is happening — it sees a
normal `access:me` user token.

How SS handles it (in `lfx-self-serve`):

- An admin may impersonate only if their own token carries the `http://lfx.dev/claims/can_impersonate` claim.
- On start, SS performs a **token exchange** to mint a genuine access token **for the target user**, stored in the session as the impersonation token (with its own expiry).
- While impersonation is active, the SS BFF uses the **target user's** access token as the bearer for upstream calls (including CF); otherwise it uses the logged-in user's own token.

So the bearer token SS forwards to CF always represents the **effective user**. CF's owner check
and `Principal.Username` resolve to that effective user with no special handling.

> Write access under impersonation is a product-level decision, gated on the SS side.

---

## 4. Notes

- **Identity migration (OQ-23).** New CF treats the LF SSO username as the canonical user
  identifier; CF handlers key on `Principal.Username`, not the Auth0 `sub`. See
  [`09`](09-authentication-architecture.md) for how the username claim is sourced.
- **Heimdall.** CF is not behind the platform API gateway today; SS calls CF directly. If CF later
  moves behind Heimdall, the token-forwarding pattern here does not block it. See the Known
  Deviations section in [`09`](09-authentication-architecture.md#known-deviations--future-direction).

---

## Related Documents

- [`09-authentication-architecture.md`](09-authentication-architecture.md) — canonical CF authentication design (scopes, routes, validation, Auth0/infra)
