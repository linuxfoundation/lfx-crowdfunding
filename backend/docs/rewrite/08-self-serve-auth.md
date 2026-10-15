<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Self Serve → Crowdfunding API Authentication

---

This document covers the **Self Serve (SS) side** of calling the Crowdfunding (CF) backend.
The canonical authentication model — scopes, route classes, token validation, the Go API decision
tree, and Auth0/infra configuration — lives in
[`../../../docs/authentication-architecture.md`](../../../docs/authentication-architecture.md). Read that first; this
document only adds the SS-specific integration details.

---

## 1. How SS authenticates to CF

SS forwards the **user's own LFX v2 access token** (`req.bearerToken`, the token from its primary
LFX One login) to CF through the platform API gateway, the same way every other SS module calls
LFX v2 services. There is no second token, no M2M token and no identity header.

```
User action in SS that needs CF data
  └─ SS BFF proxies to {CROWDFUNDING_API_BASE_URL}/crowdfunding/...   (lfx-api.<domain>)
       Authorization: Bearer {LFX v2 user access token}
  └─ Heimdall validates the token, runs the route's openfga_check, and forwards
       a Heimdall-minted JWT (aud = lfx-crowdfunding-backend, sub = LF username) to CF
  └─ CF accepts the Heimdall-issued token (dual-accept, LFXV2-3351) and keys
       Principal.Username on its subject
```

> **History.** Before CF moved behind Heimdall, SS ran a silent second `authorization_code` flow
> to mint a CF-audience token (`/crowdfunding/callback`, `CROWDFUNDING_API_AUDIENCE`,
> `CROWDFUNDING_REDIRECT_URI`). lfx-self-serve#3158 removed that flow; the leftover config was
> cleaned up in lfx-self-serve#3161.

---

## 2. SS-side implementation

Key components in `lfx-self-serve`:

| File | Purpose |
|---|---|
| `server/middleware/auth.middleware.ts` | `/api/crowdfunding/*` falls under the `/api` row (`auth: required`, `tokenRequired: true`) and gets `req.bearerToken` like every other route |
| `server/services/crowdfunding.service.ts` | `cfFetch` / `cfFetchNullable` forward `req.bearerToken` as `Authorization: Bearer` to CF |

**Environment variables (lfx-v2-argocd `values/*/lfx-self-serve.yaml`):**

| Env var | Purpose |
|---|---|
| `CROWDFUNDING_API_BASE_URL` | Platform API gateway base URL that fronts CF (`https://lfx-api.<domain>`) |

No dedicated CF client credentials, Auth0 client grant or extra Auth0 callback are needed.

---

## 3. Impersonation

Under Admin Mode, `req.bearerToken` is the **impersonated user's** token, so CF calls run as the
effective user with no CF-side special handling, consistent with the other SS modules.

Mutating crowdfunding routes in SS (payment method, subscription cancel, presigned URL, initiative
PATCH, announcements) are guarded with `blockDuringImpersonation` and return 403
`IMPERSONATION_READ_ONLY`. Reads are allowed.

---

## 4. Notes

- **Identity migration (OQ-23).** New CF treats the LF SSO username as the canonical user
  identifier; CF handlers key on `Principal.Username`, not the Auth0 `sub`. See
  [`09`](../../../docs/authentication-architecture.md) for how the username claim is sourced.
- **Heimdall.** SS reaches CF through the platform API gateway (Heimdall), which is what lets it
  forward its primary LFX v2 token. See the chart's `heimdall` values and
  [`12`](./12-fga-authorization-model.md) for the FGA checks Heimdall runs.

---

## Related Documents

- [`../../../docs/authentication-architecture.md`](../../../docs/authentication-architecture.md) — canonical CF authentication design (scopes, routes, validation, Auth0/infra)
