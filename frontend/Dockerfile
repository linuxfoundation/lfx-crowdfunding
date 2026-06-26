# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT

# ── Build stage ──────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder

# Enable corepack so pnpm is available without a separate install step
RUN corepack enable

WORKDIR /build

# Cache pnpm store separately from source — rebuild only when manifest changes.
# pnpm-workspace.yaml contains overrides/allowBuilds that affect resolution so
# it must be in this layer alongside the lockfile.
COPY frontend/pnpm-lock.yaml frontend/package.json frontend/pnpm-workspace.yaml ./
RUN pnpm fetch

# Copy frontend source and docs content so the prebuild script (build-docs.mjs)
# and the runtime server can resolve docs/user/ via ../docs/user relative paths.
COPY frontend/ .
COPY docs/user/ /docs/user/

RUN pnpm install --frozen-lockfile --prefer-offline && pnpm build

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM node:22-alpine AS runtime

RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 65532

WORKDIR /app

# Nuxt 4 output: .output/server (Node server) + .output/public (static assets)
COPY --from=builder --chown=appuser:appgroup /build/.output/ ./
# Docs markdown files for runtime API — getDocsDir() finds them via ../docs/user
# (process.cwd() = /app, so ../docs/user = /docs/user)
COPY --from=builder --chown=appuser:appgroup /docs/user/ /docs/user/

USER appuser

EXPOSE 3000

ENV NODE_ENV=production
ENV NITRO_HOST=0.0.0.0
ENV NITRO_PORT=3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:${NITRO_PORT:-3000}/api/health || exit 1

ENTRYPOINT ["node", "server/index.mjs"]
