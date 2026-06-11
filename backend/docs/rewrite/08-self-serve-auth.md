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
  └─ SS BFF resolves a CF-audience access token from session (see §2)
  └─ SS BFF proxies to CF /v1/me/*
       Authorization: Bearer {CF-audience user access token}
```

The token forwarded to CF must have the CF audience (`/api/`) and `access:me` scope, or CF will
reject the call. SS needs no M2M client credentials — the token is still user-issued.

### How SS obtains the CF-audience token

SS is a multi-audience BFF: its primary login audience is the LFX V2 cluster, which it needs for
committees, meetings, and other LFX v2 services. It cannot log in with the CF audience as its
primary audience without breaking those services.

SS runs a **silent second `authorization_code` flow** for the CF audience. This happens on the
first top-level navigation to a `/crowdfunding/*` page when no valid CF token is in the session:

```
1. SS BFF redirects browser to Auth0 /authorize
     client_id={LFX One client ID}
     audience=https://crowdfunding.{env}.lfx.dev/api/
     scope=openid profile access:me
     prompt=none          ← silent: no UI if Auth0 session exists
     redirect_uri=/crowdfunding/callback

2. Auth0 returns auth code to /crowdfunding/callback

3. SS BFF POSTs to Auth0 /oauth/token
     grant_type=authorization_code
     client_id + client_secret
     code + redirect_uri

4. Auth0 returns a CF-audience access token (access:me scope, username claim)

5. SS stores token in server session (5-minute expiry buffer);
   subsequent XHRs to /api/crowdfunding/* read it from session
```

The redirect is silent (`prompt=none`) when the user already has an Auth0 session — no second login
or consent screen is shown. If Auth0 returns `consent_required` or `interaction_required`, the
callback retries without `prompt=none` so the user sees the one-time CF audience consent screen.
Subsequent navigations are silent because consent is remembered.

**Why not a refresh-token exchange?** The refresh-token exchange approach
(`grant_type=refresh_token` + `audience=CF`) does not work with Auth0: Auth0 ignores the requested
audience on a refresh grant and returns the primary LFX V2 cluster token, which CF rejects with 401
(audience mismatch, confirmed by decoding the returned token). The second auth-code flow was
validated with the LFX platform team and is the confirmed approach. No Auth0 client grant is
required for this flow — `allow_all` on the CF resource server covers `authorization_code` flows.

---

## 2. SS-side implementation (lfx-self-serve PR #901)

The SS-side integration is implemented in `lfx-self-serve` (merged). Key components:

| File | Purpose |
|---|---|
| `server/services/crowdfunding-auth.service.ts` | Builds the Auth0 `/authorize` URL for the CF audience, exchanges the auth code for a token at `/oauth/token`, validates the token `sub`, stores in session |
| `server/controllers/crowdfunding.controller.ts` | Handles `/crowdfunding/callback`; retries without `prompt=none` on `consent_required`/`interaction_required`; returns `?error=login_required` without re-triggering the silent redirect |
| `server/middleware/auth.middleware.ts` | `extractCrowdfundingToken` reads the cached CF token from session onto `req.crowdfundingToken` on every authenticated request |
| `server/services/crowdfunding.service.ts` | All `/api/crowdfunding/*` proxy calls forward `req.crowdfundingToken` as `Authorization: Bearer` to CF `/v1/me/*` |

**Environment variables (lfx-v2-argocd `values/*/lfx-self-serve.yaml`):**

| Env var | Purpose |
|---|---|
| `CROWDFUNDING_API_BASE_URL` | CF API base URL |
| `CROWDFUNDING_API_AUDIENCE` | CF resource server identifier — used as `audience` in the auth-code flow |
| `CROWDFUNDING_REDIRECT_URI` | Auth0 callback URL (defaults to `{PCC_BASE_URL}/crowdfunding/callback`) |

SS reuses the existing `PCC_AUTH0_CLIENT_ID` / `PCC_AUTH0_CLIENT_SECRET` / `PCC_AUTH0_ISSUER_BASE_URL`
(the LFX One app client). No dedicated CF client credentials are needed. No auth0-terraform grant is
required — `authorization_code` flows work under Auth0's `allow_all` user policy on the CF resource
server without an explicit client grant.

SS needs no client-ID allowlist entry and no identity header — the user-issued access token carries
identity, and the `access:me` scope is the gate.

---

## 3. Impersonation

Admin impersonation works transparently with this model because SS swaps the **token**, not an
identity header. CF never needs to know impersonation is happening — it sees a normal `access:me`
user-issued access token.

The token SS forwards to CF must always represent the **effective user**:

- **Normal:** the logged-in user's own CF-audience access token (from the second auth-code flow).
- **Impersonating:** a CF-audience token minted for the **target user** and used as the bearer
  while impersonation is active.

**Requirement:** the CF integration must forward this effective-user token and must **not** use the
admin's own CF token. Because the forwarded token already represents the effective user, CF's owner
check and `Principal.Username` resolve correctly with no CF-side special handling.

> Write access under impersonation is a product-level decision, gated on the SS side.

> **Known gap (tracked separately).** The current second auth-code flow authenticates the
> **logged-in admin** — the resulting CF token is scoped to the admin's identity, not the
> impersonated user. CF calls during impersonation therefore use the admin's identity. The fix
> requires running the second auth-code flow (or a token exchange) for the impersonated user's
> identity — tracked as a separate work item.

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
