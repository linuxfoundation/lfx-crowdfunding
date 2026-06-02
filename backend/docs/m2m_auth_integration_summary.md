# M2M Auth Integration Summary

## Overview

The JWT middleware (`internal/infrastructure/auth/jwt.go`) supports two caller
types — interactive users and machine-to-machine (M2M) services — with a
single, hardened validation pipeline. A runtime-configurable allowlist
(`AUTHORIZED_CLIENTS`) governs which clients may call the API and whether
header-based identity impersonation is permitted.

---

## Token validation pipeline

Every request passes through the following checks in order:

1. **Bearer token extraction** — `Authorization: Bearer <token>` is required.
2. **Signature + standard claims** — validated against the Auth0 JWKS endpoint
   (cached via Auth0 `jwks.NewCachingProvider` + custom JWKS URI). Checks:
   signature, algorithm (RS256), audience, issuer, expiry, and clock skew
   leeway.
3. **Client allowlist** (`isAuthorizedClient`) — when `AUTHORIZED_CLIENTS` is
   non-empty, the token's client ID (`azp` → `client_id` → `@clients` subject
   suffix for M2M) must appear in the list. Applies equally to user tokens
   (SPA `azp`) and M2M tokens.
4. **Identity resolution** — principal `UserID` and `Username` are extracted
   from the validated claims. For trusted M2M callers, they may be supplied via
   headers (see below).
5. **Empty subject guard** — requests with no resolvable `UserID` are rejected.

---

## M2M identity impersonation

M2M services act on behalf of users. When a trusted M2M client (one whose
`azp` is in `AUTHORIZED_CLIENTS`) sends a token with no `username` claim, it
may supply the acting user's identity via two companion headers:

| Header | Content |
|--------|---------|
| `X-Username` | The acting user's username |
| `X-User-ID` | The acting user's real Auth0 subject (e.g. `auth0|abc123`) |

Both headers must be present together. If `X-Username` is supplied without
`X-User-ID` the request is rejected (401).

**Security boundary:** the ingress layer **must** strip both `X-Username` and
`X-User-ID` from all inbound requests before forwarding. A startup warning is
logged whenever the feature is active. User tokens can never use the headers
regardless of their content.

---

## Configuration

| Environment variable | Default | Description |
|----------------------|---------|-------------|
| `JWKS_URL` | — | Auth0 JWKS endpoint for key rotation |
| `JWT_AUDIENCE` | — | API identifier registered in Auth0 |
| `JWT_ISSUER` | — | Auth0 domain (with trailing `/`) |
| `AUTHORIZED_CLIENTS` | `""` (disabled) | Whitespace-separated list of allowed client IDs |
| `ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS` | `false` | Must be `true` to allow local JWT bypass |
| `DISABLED_MOCK_LOCAL_PRINCIPAL` | `""` | Local dev bypass principal; requires `ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS=true` and is mutually exclusive with `JWKS_URL` |

When `AUTHORIZED_CLIENTS` is empty:
- All valid tokens are accepted regardless of client ID.
- Header impersonation is disabled entirely.

---

## Key design decisions

| Decision | Rationale |
|----------|-----------|
| RS256-only validation | Rejects symmetric algorithm confusion and unexpected signing methods |
| Explicit mock bypass gate (`ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS`) | Fail-closed startup behavior for shared environments |
| Client set as `map[string]struct{}` | O(1) lookup, parsed once at construction — not on every request |
| `X-User-ID` required alongside `X-Username` | Eliminates the need to fabricate an Auth0 subject from a username string |
| `@clients` fallback guarded by `isM2MToken` | Prevents nonsensical client ID extraction from user token subjects |
| `AUTHORIZED_CLIENTS` applies to all token types | Closes the bypass where non-M2M tokens could skip the client ID check |
| `slog.Default()` nil-guard on logger | Prevents panic if logger is not injected |

---

## Files changed

| File | Change |
|------|--------|
| `internal/infrastructure/auth/jwt.go` | Core implementation |
| `internal/infrastructure/auth/jwt_test.go` | Full test coverage |
| `cmd/initiatives-api/server.go` | Wiring, startup warnings |
| `cmd/initiatives-api/config.go` | `AuthorizedClients` + local bypass allow-gate config |
| `.env.example` | Environment variable documentation |
