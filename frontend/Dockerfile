# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT

# ── Build stage ──────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder

# Enable corepack so pnpm is available without a separate install step
RUN corepack enable

WORKDIR /build

# Cache pnpm store separately from source — rebuild only when lockfile changes
COPY pnpm-lock.yaml package.json ./
RUN pnpm fetch

COPY . .
RUN pnpm install --offline && pnpm build

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM node:22-alpine AS runtime

RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 65532

WORKDIR /app

# Nuxt 4 output: .output/server (Node server) + .output/public (static assets)
COPY --from=builder --chown=appuser:appgroup /build/.output ./

USER appuser

EXPOSE 3000

ENV NODE_ENV=production
ENV NITRO_HOST=0.0.0.0
ENV NITRO_PORT=3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:3000/api/health || exit 1

ENTRYPOINT ["node", "server/index.mjs"]
