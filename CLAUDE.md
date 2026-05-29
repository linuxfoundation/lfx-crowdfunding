# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Kubernetes-native rewrite of a legacy Lambda-based crowdfunding platform. It is a monorepo containing:

- `backend/` — Go HTTP API (Chi router, PostgreSQL, Stripe)
- `frontend/` — Nuxt 4 BFF (Vue 3, TypeScript, Tailwind, PrimeVue)

Both are independently deployed to the LFX v2 shared Kubernetes cluster via ArgoCD GitOps (`linuxfoundation/lfx-v2-argocd`).

## Backend Commands (Go)

All commands run from `backend/`:

```bash
make deps          # Download Go module dependencies
make build         # Compile binary → bin/initiatives-api
make run           # Build and run locally (requires .env)
make test          # Run unit tests with race detector
make fmt           # Format Go code (gofmt + goimports)
make lint          # Run golangci-lint
make license-check # Verify SPDX license headers on .go files
make db-seed       # Load dev seed data (localhost only)
make docker-build  # Build Docker image
make deploy-kind   # Deploy to local Kind cluster with Helm
```

Entry points:
- `cmd/initiatives-api/` — HTTP API server (port 8080, `GET /livez` for health)
- `cmd/ledger-stats-sync/` — CronJob that syncs financial data hourly from ledger service

## Frontend Commands (Node/pnpm)

All commands run from `frontend/`:

```bash
pnpm install       # Install dependencies (requires pnpm 9+, Node 22+)
pnpm dev           # Dev server
pnpm build         # Production build
pnpm lint          # ESLint check
pnpm lint:fix      # ESLint with auto-fix
pnpm format        # Prettier format
pnpm tsc-check     # TypeScript type check
pnpm test          # Vitest unit tests
```

## Architecture

### Backend (`backend/internal/`)

Layered architecture: handler → service → domain/infrastructure

- `domain/` — Domain models, repository interfaces, domain errors
- `service/` — Business logic (initiative, payment, donation, statistics, subscription)
- `handler/` — HTTP handlers wired to Chi router
- `infrastructure/` — DB (pgx/v5), Stripe, Auth0 JWT middleware, Ledger client, OpenTelemetry

Database: PostgreSQL schema `crowdfunding` on shared LFX v2 RDS. Migrations in `db/migrations/` via golang-migrate. Local dev uses Docker Compose (`docker-compose.yml` at repo root starts Postgres on port 5432).

### Frontend (`frontend/app/`)

Nuxt 4 BFF — server-side auth with HTTP-only session cookies (OAuth2 PKCE, Auth0). All API calls proxy through the Nuxt server to the Go backend.

- `pages/` — Nuxt auto-routes
- `components/` — Reusable Vue 3 components
- `composables/` — Logic reuse (Vue Query for async, Pinia for state)
- `server/` — Nuxt server routes (BFF layer, auth)
- `types/` — TypeScript interfaces

## Environment Setup

**Backend** — create `backend/.env`:
- `DATABASE_URL` — e.g. `postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding`
- `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`
- `LEDGER_BASE_URL`, `LEDGER_API_KEY`
- `JWKS_URL` — Auth0 JWKS (or use `DISABLED_MOCK_LOCAL_PRINCIPAL=true` to skip JWT validation locally)

**Frontend** — create `frontend/.env` from `frontend/.env.example`:
- `NUXT_PUBLIC_AUTH0_CLIENT_ID`, `NUXT_AUTH0_CLIENT_SECRET`
- `NUXT_JWT_SECRET` — random string for session signing
- Auth0 domain defaults to `linuxfoundation-dev.auth0.com` when `NUXT_APP_ENV=development`

## Conventions

- **DCO:** All commits require `--signoff` (`git commit --signoff`)
- **License headers:** All `.go` files must start with `// SPDX-License-Identifier: MIT`
- **Frontend UIKit:** When building any UI element, **always load the `uikit` skill first** to check whether an existing component covers the need before writing any HTML or creating a new component
- **TypeScript:** Strict mode enforced; no `any` without justification
- **CI:** MegaLinter runs on every PR; license header check enforced on every commit
- **Frontend types:** Never define `type` or `interface` inline in `.vue` files, server API routes, or middleware. Place all types in a dedicated `*.types.ts` file:
  - App-level types (components, composables): `frontend/app/types/<domain>.types.ts` — import via `~/types/<domain>.types`
  - Server-side types (wire shapes, middleware interfaces): `frontend/server/types/<domain>.types.ts` — import via relative path
