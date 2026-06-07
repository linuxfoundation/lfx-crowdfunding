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

SS obtains a **user-issued access token scoped to the CF audience** and forwards it to CF. There is
no M2M token and no identity header — all SS→CF calls are me-style endpoints (`/v1/me/*`), and the
access token carries the acting user's identity via the LF SSO username — a custom claim (per
Design Rule 3 in [`09`](09-authentication-architecture.md#design-rules)).

```
User action in SS that needs CF data
  └─ SS BFF resolves (or exchanges for) a CF-audience access token (see §2)
  └─ SS BFF proxies to CF /v1/me/*
       Authorization: Bearer {CF-audience user access token}
```

The token forwarded to CF must have the CF audience (`/api/`) and `access:me` scope, or CF will
reject the call. SS needs no M2M client credentials — the token is still user-issued.

### How SS obtains the CF-audience token

SS is a multi-audience BFF: its primary login audience is the LFX V2 cluster, which it needs for
committees, meetings, and other LFX v2 services. It cannot log in with the CF audience as its
primary audience without breaking those services.

Instead, SS performs a **silent cross-audience refresh token exchange** on each authenticated
request:

```
POST /oauth/token
  grant_type=refresh_token
  refresh_token={user's refresh token from OIDC session}
  client_id={LFX One client ID}
  client_secret={LFX One client secret}
  audience=https://crowdfunding.{env}.lfx.dev/api/
  scope=access:me
```

Auth0 returns a CF-scoped access token carrying the user's identity. SS caches it in the server
session (with a 5-minute expiry buffer) and forwards it to CF. This is the same mechanism SS uses
to obtain legacy API Gateway tokens — no user interaction, and no `client_credentials` (M2M) grant.
The request is authenticated with SS's client secret (SS is a confidential client), but the
resulting token represents the **user**, not the SS client.

This exchange requires LFX One to have a client grant registered for the CF audience in
`auth0-terraform` (see [`09`](09-authentication-architecture.md#client-grants) for the HCL).
Auth0's `allow_all` user policy on the CF resource server enables interactive logins without a
grant, but cross-audience refresh token exchanges require explicit grant registration regardless of
the `allow_all` policy.

---

## 2. Required SS-side changes

| Area | Change |
|---|---|
| `lfx-self-serve` | `crowdfunding.service.ts` / proxy routes under `/api/crowdfunding/*` that perform the CF token exchange and forward the resulting CF-audience token to CF `/v1/me/*`. |
| `auth0-terraform` | Register LFX One client grant for CF audience (`grants_crowdfunding.tf`) — required for cross-audience refresh token exchange. See [`09`](09-authentication-architecture.md#client-grants). |
| `lfx-v2-argocd` | `values/*/lfx-self-serve.yaml`: `CROWDFUNDING_API_BASE_URL` + `CROWDFUNDING_API_AUDIENCE`. No M2M client/secret for CF. |

SS needs no client-ID allowlist entry and no identity header — the user-issued access token carries
identity, and the `access:me` scope is the gate.

---

## 3. Impersonation

Admin impersonation works transparently with this model because SS swaps the **token**, not an
identity header. CF never needs to know impersonation is happening — it sees a normal `access:me`
user-issued access token.

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

> **Known gap (tracked separately).** The current cross-audience refresh token exchange uses the
> logged-in admin's refresh token, so the CF-audience token is scoped to the **admin** rather than
> the impersonated user during an impersonation session. This means CF calls during impersonation
> use the admin's identity rather than the target user's. The fix requires the exchange to use the
> impersonated user's token as the subject — tracked as a separate work item.

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
