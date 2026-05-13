# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT

# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Cache dependencies separately from source code
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" \
    -o /build/initiatives-api ./cmd/initiatives-api

# ── Final stage ───────────────────────────────────────────────────────────────
FROM alpine:3.20

# Security: run as non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup -u 1000

WORKDIR /app
COPY --from=builder /build/initiatives-api ./initiatives-api

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/initiatives-api"]
