<!-- Copyright The Linux Foundation and each contributor to LFX. -->
<!-- SPDX-License-Identifier: MIT -->

# Auth0 Cross-Application Integration Guide

## LfxSelfServe → Crowdfunding API Handshake

**Audience:** backend engineers, frontend engineers, DevOps/Auth0 admins  
**Scope:** how a user authenticated in LfxSelfServe can make authenticated requests
to the Crowdfunding `/me` endpoints (donations, subscriptions, initiatives) without
a second login prompt, and how the Crowdfunding backend correctly identifies that
user.

---

## Table of Contents

1. [Situation and Core Problem](#1-situation-and-core-problem)
2. [Auth0 Concepts Primer](#2-auth0-concepts-primer)
3. [The Recommended Architecture: Access Tokens with a Shared API Audience](#3-the-recommended-architecture-access-tokens-with-a-shared-api-audience)
4. [Full Token Flow: Step by Step](#4-full-token-flow-step-by-step)
5. [Auth0 Dashboard Configuration](#5-auth0-dashboard-configuration)
6. [Custom Claims: LF SSO Username in the Access Token](#6-custom-claims-lf-sso-username-in-the-access-token)
7. [Crowdfunding Backend Configuration](#7-crowdfunding-backend-configuration)
8. [CORS: Browser Security Boundary](#8-cors-browser-security-boundary)
9. [LfxSelfServe Frontend Implementation](#9-lfxselfserve-frontend-implementation)
10. [What the Crowdfunding Backend Does with the Token](#10-what-the-crowdfunding-backend-does-with-the-token)
11. [Local Development Without Auth0](#11-local-development-without-auth0)
12. [Security Considerations](#12-security-considerations)
13. [Troubleshooting](#13-troubleshooting)
14. [Quick Reference: Configuration Matrix](#14-quick-reference-configuration-matrix)

---

## 1. Situation and Core Problem

### The setup

```
┌─────────────────────── Single Auth0 Tenant ───────────────────────────────┐
│                                                                             │
│   ┌────────────────────┐         ┌──────────────────────────────────────┐  │
│   │  LfxSelfServe App  │         │  Crowdfunding App (this service)     │  │
│   │  (Auth0 "SPA"      │         │  (Auth0 "Regular Web App" or "SPA")  │  │
│   │   Application)     │         │  client_id: <CF_CLIENT_ID>           │  │
│   │  client_id: <SS>   │         └──────────────────────────────────────┘  │
│   └────────────────────┘                                                    │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │  Crowdfunding API  (Auth0 "API" / Resource Server)                  │  │
│   │  identifier: https://crowdfunding.api.lfx.dev                       │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│   JWKS endpoint (shared by all apps in tenant):                            │
│   https://<tenant>.auth0.com/.well-known/jwks.json                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

### The problem in plain English

When a user logs into **LfxSelfServe**, Auth0 issues two tokens:

| Token | Purpose | `aud` claim |
|---|---|---|
| **ID token** | Proves identity to LfxSelfServe | `<SS_CLIENT_ID>` (the LfxSelfServe app) |
| **Access token** _(default)_ | Calls Auth0's `/userinfo` endpoint | `https://<tenant>.auth0.com/userinfo` |

Neither of those tokens can be sent to the Crowdfunding backend as-is:

- **The ID token** has `aud = <SS_CLIENT_ID>`. The Crowdfunding backend would
  reject it because the audience doesn't match its configured `JWT_AUDIENCE`.
- **The default access token** is opaque and scoped only to `/userinfo`.

The Crowdfunding backend validates tokens with:
```
JWT_AUDIENCE = https://crowdfunding.api.lfx.dev
JWT_ISSUER   = https://<tenant>.auth0.com/
```

So you need a token whose `aud` contains `https://crowdfunding.api.lfx.dev` — an
**access token for the Crowdfunding API**.

---

## 2. Auth0 Concepts Primer

### Applications vs APIs

Auth0 has two different object types you register:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Auth0 "Application" (represents a client)                                  │
│  Has a client_id. Used by frontends and backends to authenticate users      │
│  or machines. Generates ID tokens and access tokens.                        │
│                                                                             │
│  Examples:  LfxSelfServe SPA,  Crowdfunding SPA,  Crowdfunding Backend     │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  Auth0 "API" (represents a resource server — the thing being called)       │
│  Has an identifier (a URI). Never an OAuth client. Its identifier is used  │
│  as the "audience" in token requests and appears in the `aud` claim of     │
│  access tokens.                                                             │
│                                                                             │
│  Example:  Crowdfunding API  →  https://crowdfunding.api.lfx.dev           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### ID Token vs Access Token

| | ID Token | Access Token |
|---|---|---|
| **Purpose** | Who the user is (authentication) | What the user can do (authorization) |
| **Audience** | The Application's `client_id` | The API identifier |
| **Should be sent to APIs?** | **No** — only for the app | **Yes** — this is its purpose |
| **Contains user identity?** | Yes, always | Yes, when a user is involved |
| **Contains custom claims?** | Only if Auth0 Action adds them | Only if Auth0 Action adds them |
| **Format** | Always a signed JWT | JWT (when a custom API audience is specified) |

> **Rule:** Never send an ID token to a backend API as a bearer credential.
> ID tokens are for the client that requested them — not for downstream services.
> Always request and send an **access token** with the target API's audience.

### How Access Tokens carry user identity

When a user authenticates in LfxSelfServe and the `audience` parameter includes
`https://crowdfunding.api.lfx.dev`, Auth0 issues a signed **JWT access token**
that contains:

```json
{
  "sub": "auth0|abc123",
  "iss": "https://<tenant>.auth0.com/",
  "aud": ["https://crowdfunding.api.lfx.dev"],
  "exp": 1748736000,
  "iat": 1748649600,
  "azp": "<SS_CLIENT_ID>",
  "https://sso.linuxfoundation.org/claims/username": "jdoe",
  "email": "jdoe@example.com",
  "email_verified": true,
  "given_name": "John",
  "family_name": "Doe",
  "picture": "https://cdn.example.com/jdoe.png"
}
```

Key points:
- `sub` is the **global** Auth0 user ID — the same value regardless of which app
  issued the token, because all apps share one tenant.
- `azp` (authorized party) is the `client_id` of the application that requested
  the token. This lets Crowdfunding know the token came from LfxSelfServe.
- Custom claims like `https://sso.linuxfoundation.org/claims/username` are added
  by an Auth0 **Action** and will appear in access tokens if the Action is
  configured to add them to both ID tokens and access tokens.

---

## 3. The Recommended Architecture: Access Tokens with a Shared API Audience

```
                    ┌─────────────────────────────────────────┐
                    │              Auth0 Tenant                │
                    │                                         │
                    │  ┌──────────────────────────────────┐  │
                    │  │  "Crowdfunding API" Resource      │  │
                    │  │  Server                           │  │
                    │  │  id: https://crowdfunding.api.    │  │
                    │  │      lfx.dev                      │  │
                    │  └──────────────────────────────────┘  │
                    │                                         │
                    │  ┌───────────────┐ ┌─────────────────┐ │
                    │  │ LfxSelfServe  │ │ Crowdfunding    │ │
                    │  │ Application   │ │ Application     │ │
                    │  │ (SPA)         │ │ (SPA)           │ │
                    │  └───────────────┘ └─────────────────┘ │
                    └────────────────┬────────────────────────┘
                                     │
              ┌──────────────────────┼──────────────────────────┐
              │                      │                          │
              ▼                      ▼                          ▼
   ┌─────────────────┐   ┌────────────────────┐   ┌──────────────────────┐
   │  User's Browser │   │  LfxSelfServe UI   │   │  Crowdfunding API    │
   │                 │   │  (Nuxt / Angular)  │   │  (Go / Chi)          │
   │  Stores:        │   │                    │   │                      │
   │  - access_token │──▶│  Sends access_token│──▶│  Validates JWT:      │
   │    for CF API   │   │  to CF /me endpoints│  │  - sig via JWKS      │
   │  - refresh_token│   │                    │   │  - aud == CF API id  │
   └─────────────────┘   └────────────────────┘   │  - iss == tenant     │
                                                   │  Extracts Principal: │
                                                   │  - UserID = sub      │
                                                   │  - Username = claim  │
                                                   └──────────────────────┘
```

### Why this is the correct approach

1. **No second login** — The access token is obtained during the LfxSelfServe
   login flow by including the Crowdfunding API `audience` in the auth request.
   Auth0 issues it silently alongside the ID token.

2. **Standards-compliant** — Using access tokens for APIs is exactly the OAuth
   2.0 / OIDC intended pattern. Auth0 explicitly recommends against sending ID
   tokens to APIs.

3. **User identity is preserved** — The `sub` claim in the access token is the
   same `auth0|...` ID that Crowdfunding stores in its `users.user_id` column,
   so filtering by `principal.UserID` works correctly.

4. **Single JWKS** — Both apps are in the same tenant, so the Crowdfunding
   backend uses the same JWKS URL as any other Auth0-protected service.

5. **No coupling between apps** — LfxSelfServe does not need to know anything
   about the Crowdfunding application's client_id. It only needs to know the
   API identifier (the audience string).

---

## 4. Full Token Flow: Step by Step

### 4.1 Login Flow (happens once per session)

```
Browser                   LfxSelfServe UI              Auth0 Tenant
   │                            │                            │
   │  User clicks "Log in"      │                            │
   │───────────────────────────▶│                            │
   │                            │                            │
   │                            │  /authorize request        │
   │                            │  ?client_id=<SS>           │
   │                            │  &response_type=code       │
   │                            │  &scope=openid profile     │
   │                            │    email                   │
   │                            │  &audience=                │  ← KEY PARAM
   │                            │    https://crowdfunding    │
   │                            │    .api.lfx.dev            │
   │                            │  &redirect_uri=...         │
   │                            │───────────────────────────▶│
   │                            │                            │
   │◀───────────────────────────────────────────────────────▶│
   │         (Auth0 hosted login UI — LF SSO)                │
   │                            │                            │
   │                            │  authorization_code        │
   │                            │◀───────────────────────────│
   │                            │                            │
   │                            │  POST /oauth/token         │
   │                            │  grant=authorization_code  │
   │                            │  code=...                  │
   │                            │───────────────────────────▶│
   │                            │                            │
   │                            │  {                         │
   │                            │    access_token: <JWT>,    │  ← for CF API
   │                            │    id_token: <JWT>,        │  ← for SS app
   │                            │    refresh_token: <opaque>,│
   │                            │    expires_in: 86400       │
   │                            │  }                         │
   │                            │◀───────────────────────────│
   │                            │                            │
   │  Store tokens in memory    │                            │
   │◀───────────────────────────│                            │
```

**The critical line:** the `audience` parameter in the `/authorize` URL.
When `audience` is set to the Crowdfunding API identifier, Auth0 issues an
**access token in JWT format** scoped to that API, instead of the default
opaque token.

### 4.2 API Call Flow (every request to /me endpoints)

```
Browser              LfxSelfServe UI          Crowdfunding API          Database
   │                       │                        │                      │
   │  Navigate to           │                        │                      │
   │  "My Donations"        │                        │                      │
   │──────────────────────▶│                        │                      │
   │                        │                        │                      │
   │                        │  GET /v1/me/donations  │                      │
   │                        │  Authorization:        │                      │
   │                        │    Bearer <access_token│                      │
   │                        │───────────────────────▶│                      │
   │                        │                        │                      │
   │                        │                        │  1. Extract token    │
   │                        │                        │     from header      │
   │                        │                        │                      │
   │                        │                        │  2. Fetch JWKS from  │
   │                        │                        │     Auth0 (cached)   │
   │                        │                        │                      │
   │                        │                        │  3. Verify:          │
   │                        │                        │     - RS256 sig      │
   │                        │                        │     - aud == CF API  │
   │                        │                        │     - iss == tenant  │
   │                        │                        │     - not expired    │
   │                        │                        │                      │
   │                        │                        │  4. Build Principal  │
   │                        │                        │     UserID = sub     │
   │                        │                        │                      │
   │                        │                        │  SELECT * FROM       │
   │                        │                        │  donations WHERE     │
   │                        │                        │  user_id = sub ──────▶
   │                        │                        │                      │
   │                        │                        │◀─ [{...}, {...}]    │
   │                        │  200 { data: [...] }   │                      │
   │                        │◀───────────────────────│                      │
   │◀──────────────────────│                        │                      │
```

### 4.3 Token Refresh Flow

Access tokens expire (default 24 hours; configurable in Auth0 dashboard). The
LfxSelfServe frontend should use the refresh token to obtain a new access token
silently before the current one expires.

```
LfxSelfServe UI                Auth0 Tenant
      │                              │
      │  (access_token expiring)     │
      │                              │
      │  POST /oauth/token           │
      │  grant=refresh_token         │
      │  refresh_token=<opaque>      │
      │  audience=https://           │
      │    crowdfunding.api.lfx.dev  │
      │─────────────────────────────▶│
      │                              │
      │  {                           │
      │    access_token: <new JWT>,  │
      │    expires_in: 86400         │
      │  }                           │
      │◀─────────────────────────────│
      │                              │
      │  Replace stored access_token │
```

---

## 5. Auth0 Dashboard Configuration

### 5.1 Create the Crowdfunding API (Resource Server)

In Auth0 Dashboard → **Applications → APIs → Create API**:

| Field | Value |
|---|---|
| Name | `Crowdfunding API` |
| Identifier | `https://crowdfunding.api.lfx.dev` |
| Signing Algorithm | `RS256` |
| Token Expiration | `86400` seconds (24 hours) |
| Allow Offline Access | `true` (enables refresh tokens) |
| Enable RBAC | optional — only if you need permission scopes |

> The identifier is just a URI used as the audience string. It does not need
> to be a reachable URL. Choose something stable and environment-specific
> (e.g. `https://crowdfunding.api.staging.lfx.dev` for staging).

### 5.2 Authorize LfxSelfServe to use the Crowdfunding API

In Auth0 Dashboard → **Applications → APIs → Crowdfunding API → Machine to Machine Applications**:

Add the **LfxSelfServe SPA Application** to the authorized applications list.
This tells Auth0 it is allowed to request access tokens for this API.

> Note: "Machine to Machine Applications" is Auth0's label for this setting
> even for SPAs. You are not configuring M2M authentication here — you are
> simply authorizing the LfxSelfServe app to request access tokens for the
> Crowdfunding API audience.

### 5.3 Verify the JWKS endpoint

Both apps in the same tenant share a single JWKS endpoint:

```
https://<your-tenant>.auth0.com/.well-known/jwks.json
```

This is the value you set in `JWKS_URL` for the Crowdfunding backend. No
additional configuration is needed — Auth0 manages key rotation automatically.

### 5.4 Auth0 tenant settings summary

```
┌────────────────────────────────────────────────────────────────────────────┐
│ Auth0 Tenant: <your-tenant>.auth0.com                                      │
│                                                                            │
│ Applications:                                                              │
│   ├── LfxSelfServe           (SPA)   client_id: <SS_CLIENT_ID>            │
│   └── Crowdfunding Frontend  (SPA)   client_id: <CF_CLIENT_ID>            │
│                                                                            │
│ APIs:                                                                      │
│   └── Crowdfunding API               id: https://crowdfunding.api.lfx.dev │
│         Authorized apps:                                                   │
│           └── LfxSelfServe ✓                                              │
│           └── Crowdfunding Frontend ✓                                     │
│                                                                            │
│ JWKS: https://<tenant>.auth0.com/.well-known/jwks.json                    │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Custom Claims: LF SSO Username in the Access Token

The Crowdfunding backend reads the following custom claim from every token:

```
https://sso.linuxfoundation.org/claims/username
```

This claim is **not** a standard OIDC claim — it is injected by an Auth0 Action
(formerly "Rules"). The Action must be configured to add this claim to **both**
ID tokens **and** access tokens, because different consumers read different token
types.

### 6.1 Auth0 Action template

In Auth0 Dashboard → **Actions → Flows → Login → Custom Action**:

```javascript
// Auth0 Action: "Add LF SSO custom claims"
// Trigger: Login / Post Login
// Runs for EVERY token issuance in the tenant

exports.onExecutePostLogin = async (event, api) => {
  const namespace = 'https://sso.linuxfoundation.org/claims/';

  // event.user contains the Auth0 user profile
  const username = event.user.username
    || event.user.nickname
    || event.user.email?.split('@')[0]
    || '';

  // Add to ID token (for apps)
  api.idToken.setCustomClaim(`${namespace}username`, username);

  // Add to Access token (for APIs like Crowdfunding)
  api.accessToken.setCustomClaim(`${namespace}username`, username);

  // Optionally propagate other profile fields into access token
  // (Crowdfunding reads email, email_verified, given_name, family_name, picture
  //  from the token — these are standard OIDC claims and are included by default
  //  when 'profile' and 'email' scopes are requested)
};
```

> **Important:** If this Action only calls `api.idToken.setCustomClaim(...)` and
> not `api.accessToken.setCustomClaim(...)`, the username claim will be absent
> from access tokens and `principal.Username` will be empty in the Crowdfunding
> backend. Check this first during troubleshooting.

### 6.2 Standard claims in access tokens

Auth0 includes standard profile claims in the access token when the following
scopes are requested during login:

| Scope | Claims included |
|---|---|
| `openid` | `sub`, `iss`, `aud`, `exp`, `iat` |
| `email` | `email`, `email_verified` |
| `profile` | `name`, `given_name`, `family_name`, `picture`, `nickname` |

LfxSelfServe must request `scope: "openid profile email"` alongside the
`audience` parameter to ensure all these claims are present in the access token.

---

## 7. Crowdfunding Backend Configuration

### 7.1 Environment variables

```bash
# .env (production values via Helm/K8s secrets)

JWKS_URL=https://<tenant>.auth0.com/.well-known/jwks.json
JWT_AUDIENCE=https://crowdfunding.api.lfx.dev
JWT_ISSUER=https://<tenant>.auth0.com/
```

The `JWT_AUDIENCE` must exactly match the API identifier registered in Auth0
(step 5.1). The `JWT_ISSUER` must be the Auth0 tenant URL with a trailing slash.

### 7.2 What the middleware validates

The Go JWT middleware (`internal/infrastructure/auth/jwt.go`) performs the
following checks on every protected request:

```
Incoming Bearer token
        │
        ▼
┌───────────────────────────────────────────────────────────┐
│  1. Token present?                                        │
│     Authorization: Bearer <token>   → proceed            │
│     Missing / malformed header      → 401                │
├───────────────────────────────────────────────────────────┤
│  2. Algorithm allowed?                                    │
│     RS256, RS384, RS512, ES256, ES384, ES512   → proceed  │
│     HS256, HS384, HS512 (symmetric)            → 401      │
│     (prevents algorithm confusion attacks)               │
├───────────────────────────────────────────────────────────┤
│  3. Signature valid?                                      │
│     Verified with JWKS public key   → proceed            │
│     Invalid signature               → 401                │
├───────────────────────────────────────────────────────────┤
│  4. Token not expired?                                    │
│     exp > now (±5s clock skew)      → proceed            │
│     Expired                         → 401                │
├───────────────────────────────────────────────────────────┤
│  5. Audience matches?                                     │
│     aud contains JWT_AUDIENCE       → proceed            │
│     Audience mismatch               → 401                │
├───────────────────────────────────────────────────────────┤
│  6. Issuer matches?                                       │
│     iss == JWT_ISSUER               → proceed            │
│     Issuer mismatch                 → 401                │
├───────────────────────────────────────────────────────────┤
│  7. Subject present?                                      │
│     sub non-empty                   → proceed            │
│     Empty sub                       → 401                │
└───────────────────────────────────────────────────────────┘
        │ All checks pass
        ▼
   Build Principal{
     UserID:        claims.Subject,   // "auth0|abc123"
     Username:      claims.Username,  // "jdoe"
     Email:         claims.Email,
     EmailVerified: claims.EmailVerified,
     GivenName:     claims.GivenName,
     FamilyName:    claims.FamilyName,
     Picture:       claims.Picture,
   }
        │
        ▼
   Store in request context → handler reads via PrincipalFromContext()
```

### 7.3 How handlers use the Principal

```go
// GET /v1/me/donations
func (h *DonationHandler) ListForUser(w http.ResponseWriter, r *http.Request) {
    principal := auth.PrincipalFromContext(r.Context())
    // principal.UserID == "auth0|abc123" (the Auth0 sub from the token)

    donations, meta, err := h.svc.ListByUser(r.Context(), principal.UserID, ...)
    // SQL: SELECT * FROM donations WHERE user_id = $1
    //      $1 = "auth0|abc123"
```

The `sub` claim in Auth0 is **stable and global** within a tenant. The same user
logging in via LfxSelfServe and via the Crowdfunding frontend will have identical
`sub` values. This is what allows Crowdfunding to correctly identify the user
regardless of which app they authenticated through.

---

## 8. CORS: Browser Security Boundary

When the LfxSelfServe frontend (served from `https://selfserve.lfx.linuxfoundation.org`)
makes a fetch/axios call to the Crowdfunding API
(`https://crowdfunding-api.lfx.linuxfoundation.org`), the browser enforces
**cross-origin resource sharing (CORS)**. The Crowdfunding API must respond with
the correct headers.

### 8.1 What happens without CORS headers

```
Browser
  │
  │  OPTIONS /v1/me/donations           ← preflight request
  │  Origin: https://selfserve.lfx...
  │──────────────────────────────────▶ Crowdfunding API
  │
  │  (no Access-Control-Allow-Origin)  ← API doesn't set CORS headers
  │◀──────────────────────────────────
  │
  │  ✗ CORS error — request blocked
  │    even though API returned 200
```

### 8.2 Required CORS headers from Crowdfunding API

```
Access-Control-Allow-Origin:  https://selfserve.lfx.linuxfoundation.org
Access-Control-Allow-Methods: GET, POST, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-Id
Access-Control-Max-Age:       86400
```

For local development, add `http://localhost:4200` (Angular) or
`http://localhost:3000` (Nuxt) to the allowed origins.

> **Security note:** Never set `Access-Control-Allow-Origin: *` on authenticated
> API endpoints. The wildcard restriction applies when `Access-Control-Allow-Credentials: true`
> is set (required for cookie-based flows) — but even for token-based flows with
> an `Authorization` header, using a wildcard origin is poor practice: it grants any
> domain the ability to read responses from your API. Always use an explicit allowlist.

### 8.3 Implementing CORS in the Go router

```go
// server.go — add before route registration
import "github.com/go-chi/cors"

r.Use(cors.Handler(cors.Options{
    AllowedOrigins: cfg.CORS.AllowedOrigins, // from env: CORS_ALLOWED_ORIGINS
    AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
    AllowedHeaders: []string{"Authorization", "Content-Type", "X-Request-Id"},
    MaxAge:         86400,
}))
```

```bash
# .env
CORS_ALLOWED_ORIGINS=https://selfserve.lfx.linuxfoundation.org,https://crowdfunding.lfx.linuxfoundation.org
```

---

## 9. LfxSelfServe Frontend Implementation

### 9.1 Auth0 SDK configuration (Auth0 SPA SDK / auth0-angular)

The key change from a standard single-app setup is the `audience` parameter:

```typescript
// auth0-angular or @auth0/auth0-spa-js configuration in LfxSelfServe

AuthModule.forRoot({
  domain: '<your-tenant>.auth0.com',
  clientId: '<SS_CLIENT_ID>',
  authorizationParams: {
    redirect_uri: window.location.origin,
    scope: 'openid profile email',       // ← required for profile claims
    audience: 'https://crowdfunding.api.lfx.dev',  // ← KEY: CF API audience
  },
  useRefreshTokens: true,                // ← enables silent token refresh
  cacheLocation: 'memory',              // ← prefer memory over localStorage
})
```

### 9.2 Making authenticated calls to Crowdfunding

```typescript
// donations.service.ts in LfxSelfServe

import { AuthService } from '@auth0/auth0-angular';

@Injectable({ providedIn: 'root' })
export class DonationsService {
  constructor(
    private http: HttpClient,
    private auth: AuthService,
  ) {}

  getMyDonations(): Observable<DonationListResponse> {
    return this.auth.getAccessTokenSilently({
      authorizationParams: {
        audience: 'https://crowdfunding.api.lfx.dev',
      },
    }).pipe(
      switchMap(accessToken =>
        this.http.get<DonationListResponse>(
          'https://crowdfunding-api.lfx.linuxfoundation.org/v1/me/donations',
          {
            headers: { Authorization: `Bearer ${accessToken}` },
          }
        )
      )
    );
  }
}
```

> `getAccessTokenSilently()` returns the cached access token if it is still
> valid, or uses the refresh token to obtain a new one. It never shows the
> login screen unless the refresh token has also expired.

### 9.3 Using an HTTP interceptor (recommended pattern)

For cleaner code, attach the token automatically to all Crowdfunding API calls
using an HTTP interceptor:

```typescript
// crowdfunding-auth.interceptor.ts in LfxSelfServe

@Injectable()
export class CrowdfundingAuthInterceptor implements HttpInterceptor {
  private readonly CF_API_BASE = 'https://crowdfunding-api.lfx.linuxfoundation.org';
  private readonly CF_AUDIENCE = 'https://crowdfunding.api.lfx.dev';

  constructor(private auth: AuthService) {}

  intercept(req: HttpRequest<unknown>, next: HttpHandler): Observable<HttpEvent<unknown>> {
    // Only intercept calls to the Crowdfunding API
    if (!req.url.startsWith(this.CF_API_BASE)) {
      return next.handle(req);
    }

    return this.auth.getAccessTokenSilently({
      authorizationParams: { audience: this.CF_AUDIENCE },
    }).pipe(
      switchMap(token => {
        const authedReq = req.clone({
          setHeaders: { Authorization: `Bearer ${token}` },
        });
        return next.handle(authedReq);
      }),
      catchError(err => {
        // Token acquisition failed — likely session expired
        if (err.error === 'login_required') {
          this.auth.loginWithRedirect();
        }
        return throwError(() => err);
      })
    );
  }
}
```

```typescript
// app.module.ts — register the interceptor
providers: [
  { provide: HTTP_INTERCEPTORS, useClass: CrowdfundingAuthInterceptor, multi: true },
]
```

---

## 10. What the Crowdfunding Backend Does with the Token

This section shows the full chain from token → principal → database query,
mapping the Auth0 claim to the database column.

```
Auth0 Access Token (JWT)                 Crowdfunding Database
─────────────────────────────            ─────────────────────────────────────
{                                        crowdfunding.donations
  "sub": "auth0|abc123",         ──────▶ WHERE user_id = 'auth0|abc123'
  "email": "jdoe@example.com",
  "given_name": "John",                  crowdfunding.subscriptions
  "family_name": "Doe",          ──────▶ WHERE user_id = 'auth0|abc123'
  "email_verified": true,
  "picture": "https://...",              crowdfunding.users
  "https://sso.linuxfoundation         (upserted on first donation/subscription
    .org/claims/username": "jdoe" ────▶  username = 'jdoe')
}
         │
         ▼
   models.Principal{
     UserID:        "auth0|abc123",
     Username:      "jdoe",
     Email:         "jdoe@example.com",
     EmailVerified: true,
     GivenName:     "John",
     FamilyName:    "Doe",
     Picture:       "https://...",
   }
```

The `sub` claim (`auth0|<id>`) is the **join key** between Auth0 and every
Crowdfunding database table. It is:

- **Stable** — never changes for a given user, even if they change email/username
- **Unique** — guaranteed unique across the entire Auth0 tenant
- **Portable** — the same value regardless of which application the user logs in through

---

## 11. Local Development Without Auth0

For local development, the Crowdfunding backend supports a **bypass mode** that
skips JWT validation entirely and uses a static mock principal:

```bash
# .env.local — never use in staging or production
DISABLED_MOCK_LOCAL_PRINCIPAL=local-dev-user
# Leave JWKS_URL, JWT_AUDIENCE, JWT_ISSUER empty or unset
```

When `DISABLED_MOCK_LOCAL_PRINCIPAL` is set, every request is treated as
authenticated with:

```go
Principal{
  UserID:        "local-dev-user",
  Username:      "local-dev-user",
  EmailVerified: true,
}
```

The backend logs four prominent warning lines at startup to prevent this being
accidentally deployed.

For LfxSelfServe local development, you have two options:

**Option A: Both services in bypass mode** (simplest for frontend dev)
- Set `DISABLED_MOCK_LOCAL_PRINCIPAL=local-dev-user` on the Crowdfunding backend
- LfxSelfServe frontend makes calls without a token (or with a dummy one)
- Crowdfunding ignores the token and uses the mock principal

**Option B: Real Auth0 tokens against a dev tenant**
- Use a dedicated Auth0 dev tenant (separate from production)
- Configure a `Dev Crowdfunding API` in the dev tenant
- LfxSelfServe dev build points to the dev tenant
- Crowdfunding local instance uses dev tenant's `JWKS_URL` and `JWT_AUDIENCE`

---

## 12. Security Considerations

### 12.1 Token storage on the frontend

| Storage | Recommendation | Risk |
|---|---|---|
| **Memory** (recommended) | Store `access_token` in JS memory only | Token lost on page refresh — mitigated by silent refresh via `useRefreshTokens: true` |
| `localStorage` | Avoid | XSS can read localStorage; token stolen → full account access |
| `sessionStorage` | Acceptable compromise | XSS can still read it, but scoped to tab |
| `httpOnly` cookie | Ideal for SSR | Requires BFF (Backend for Frontend) pattern |

Use the Auth0 SDK with `cacheLocation: 'memory'` and `useRefreshTokens: true`.
The SDK handles silent refresh automatically.

### 12.2 Audience validation prevents token misuse

Because the Crowdfunding backend validates `aud == "https://crowdfunding.api.lfx.dev"`,
a token issued for a different audience (e.g. the LfxSelfServe application's
`client_id`) cannot be replayed against the Crowdfunding API:

```
Token with aud="<SS_CLIENT_ID>"
        │
        ▼
JWT Middleware checks: aud contains "https://crowdfunding.api.lfx.dev"?
        │
        ▼
        ✗  → 401 Unauthorized
```

### 12.3 Algorithm restriction

The backend only accepts asymmetric signing algorithms (`RS256`, `RS384`,
`RS512`, `ES256`, `ES384`, `ES512`). Symmetric algorithms (`HS256`, etc.) are
rejected. This prevents the "algorithm confusion" attack where an attacker crafts
a token signed with `HS256` using the JWKS public key bytes as the HMAC secret.

### 12.4 Clock skew tolerance

The backend allows ±5 seconds of clock drift (`ClockSkew: 5 * time.Second`).
Keep server clocks synchronized with NTP. A larger skew would extend the
acceptance window for just-expired tokens.

### 12.5 JWKS caching and key rotation

Auth0 rotates signing keys periodically. The `keyfunc/v3` library (`github.com/MicahParks/keyfunc/v3`)
automatically fetches fresh keys when a token arrives signed by a key ID (`kid`)
not currently in the cache. The JWKS goroutine runs for the lifetime of the
cancellable root context created with `context.WithCancel(...)` in `main.go`.

No manual key rotation steps are required.

### 12.6 The `sub` claim and user identity

The `sub` (subject) claim is the canonical user identifier. It takes the form
`auth0|<id>` for users authenticated through Auth0's own database, or
`google-oauth2|<id>` for social logins, etc. It is:

- Set by Auth0 — not user-controlled
- Permanent for the lifetime of the user account
- Consistent across all applications in the same tenant

**Never** use `email` as the primary user key in the database. Emails change.
The `sub` does not.

---

## 13. Troubleshooting

### 401 from Crowdfunding when called from LfxSelfServe

**Check 1: Is the access token's audience correct?**

Decode the token at [jwt.io](https://jwt.io) and inspect the `aud` claim:

```json
{
  "aud": ["https://crowdfunding.api.lfx.dev"]
}
```

If `aud` is instead the LfxSelfServe `client_id` or `https://<tenant>.auth0.com/userinfo`,
the `audience` parameter was not set (or not set correctly) in the LfxSelfServe
auth configuration. Fix: add `audience: 'https://crowdfunding.api.lfx.dev'` to
the auth config.

**Check 2: Is the token expired?**

Inspect `exp`. A newly-minted token should be valid. If the frontend is caching
stale tokens, ensure `getAccessTokenSilently` is used (not a stored string).

**Check 3: Does the issuer match?**

The `iss` claim must exactly match `JWT_ISSUER` including the trailing slash:

```
iss: "https://your-tenant.auth0.com/"   ← note trailing slash
JWT_ISSUER=https://your-tenant.auth0.com/
```

**Check 4: Is LfxSelfServe authorized to use the Crowdfunding API?**

In Auth0 Dashboard → APIs → Crowdfunding API → Machine to Machine Applications:
confirm LfxSelfServe is in the authorized applications list.

---

### `principal.Username` is empty

The `https://sso.linuxfoundation.org/claims/username` custom claim is missing
from the access token. Check:

1. The Auth0 Action calls `api.accessToken.setCustomClaim(...)` (not just `api.idToken`)
2. The Action is deployed and active in the Login flow
3. The user's Auth0 profile has a `username` or `nickname` field set

---

### CORS preflight failing

Symptoms: browser console shows `Access to fetch at '...' from origin '...' has
been blocked by CORS policy`.

1. Confirm the Crowdfunding API includes `cors.Handler` middleware
2. Confirm `CORS_ALLOWED_ORIGINS` includes the LfxSelfServe origin exactly
   (no trailing slash, correct protocol: `https://`)
3. Check that `OPTIONS` is in `AllowedMethods`

---

### Token valid but user's donations not found

The `sub` in the token does not match `user_id` in the database. This can happen
if:

- The database was seeded with a different `user_id` format (e.g. old DynamoDB
  records used a different identifier scheme)
- The user record was created under a different Auth0 tenant (dev vs production)

Check: `SELECT user_id FROM users LIMIT 10;` — the values should look like
`auth0|abc123` or `google-oauth2|abc123`. If they don't, a data migration is
needed.

---

## 14. Quick Reference: Configuration Matrix

| Setting | LfxSelfServe (frontend) | Crowdfunding API (backend) |
|---|---|---|
| Auth0 domain | `<tenant>.auth0.com` | (not directly used) |
| Client ID | `<SS_CLIENT_ID>` | (not used — API validates audience) |
| Audience in auth request | `https://crowdfunding.api.lfx.dev` | (set in `JWT_AUDIENCE`) |
| Scopes | `openid profile email` | (not set — validates received token) |
| JWKS URL | (handled by SDK) | `JWKS_URL=https://<tenant>.auth0.com/.well-known/jwks.json` |
| Token type sent to CF API | Access token (JWT) | (validated — not an ID token) |
| Token storage | Memory (`cacheLocation: 'memory'`) | (stateless — no session) |
| Refresh | `useRefreshTokens: true` | (not applicable) |
| Local dev bypass | No token required | `DISABLED_MOCK_LOCAL_PRINCIPAL=local-dev-user` |

---

### Sequence diagram — complete login to API call

```
User     LfxSS Browser    LfxSS SPA SDK    Auth0 Tenant    Crowdfunding API    CF DB
 │            │                │                 │                  │              │
 │ Login      │                │                 │                  │              │
 │───────────▶│                │                 │                  │              │
 │            │ loginWithRedirect(               │                  │              │
 │            │   audience: CF_API_ID)           │                  │              │
 │            │────────────────────────────────▶│                  │              │
 │            │                │                 │                  │              │
 │            │◀─ hosted login UI (LF SSO) ─────▶│                 │              │
 │ enters creds            │                │                  │              │
 │────────────────────────────────────────▶│                  │              │
 │            │                │           auth code          │              │
 │            │                │◀────────────────│             │              │
 │            │                │ POST /oauth/token│             │              │
 │            │                │────────────────▶│             │              │
 │            │                │ access_token    │             │              │
 │            │                │ (aud=CF_API_ID) │             │              │
 │            │                │◀────────────────│             │              │
 │            │                │ id_token        │             │              │
 │            │                │ (aud=SS_CLIENT) │             │              │
 │            │                │◀────────────────│             │              │
 │            │                │                 │                  │              │
 │ "My Donations"             │                 │                  │              │
 │───────────▶│                │                 │                  │              │
 │            │ getAccessTokenSilently()         │                  │              │
 │            │────────────────▶                 │                  │              │
 │            │◀── cached access_token ──────────│                  │              │
 │            │                                  │                  │              │
 │            │        GET /v1/me/donations       │                  │              │
 │            │        Authorization: Bearer ...  │                  │              │
 │            │──────────────────────────────────────────────────▶│              │
 │            │                                  │  validate token  │              │
 │            │                                  │  (JWKS cached)   │              │
 │            │                                  │  extract sub     │              │
 │            │                                  │                  │ SELECT WHERE │
 │            │                                  │                  │ user_id=sub──▶
 │            │                                  │                  │◀─ rows ──────│
 │            │        200 { data: [...] }        │                  │              │
 │            │◀──────────────────────────────────────────────────│              │
 │◀───────────│                                  │                  │              │
```
