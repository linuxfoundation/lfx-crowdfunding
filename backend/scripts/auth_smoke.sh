#!/usr/bin/env bash
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
#
# Smoke tests for the Initiatives API.
# Usage: BASE_URL=http://localhost:8080 TOKEN=<jwt> bash scripts/auth_smoke.sh

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
TOKEN="${TOKEN:-}"

pass=0
fail=0

check() {
  local desc="$1"
  local expected="$2"
  local actual="$3"
  if [ "$actual" = "$expected" ]; then
    echo "[PASS] $desc"
    ((pass++)) || true
  else
    echo "[FAIL] $desc — expected HTTP $expected, got $actual"
    ((fail++)) || true
  fi
}

echo "=== Smoke tests against $BASE_URL ==="

# Health checks
check "GET /livez → 200" 200 \
  "$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/livez")"

check "GET /healthz → 200" 200 \
  "$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz")"

check "GET /readyz → 200" 200 \
  "$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/readyz")"

# Unauthenticated request to protected endpoint — expect 401
check "GET /v1/initiatives (no token) → 401" 401 \
  "$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/v1/initiatives")"

# Authenticated request — expect 200
if [ -n "$TOKEN" ]; then
  check "GET /v1/initiatives (with token) → 200" 200 \
    "$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "$BASE_URL/v1/initiatives")"
else
  echo "[SKIP] GET /v1/initiatives (with token) — TOKEN not set"
fi

# Stripe webhook without signature — expect 401
check "POST /v1/stripe/webhook (no sig) → 401" 401 \
  "$(curl -s -o /dev/null -w "%{http_code}" \
      -X POST -H "Content-Type: application/json" \
      -d '{}' "$BASE_URL/v1/stripe/webhook")"

echo ""
echo "Results: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
