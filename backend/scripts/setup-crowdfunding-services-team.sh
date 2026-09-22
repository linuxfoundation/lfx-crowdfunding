#!/usr/bin/env bash
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
#
# setup-crowdfunding-services-team.sh
#
# Seeds (or updates) the `team:crowdfunding-services` membership tuples in
# OpenFGA. Each Auth0 M2M client in SERVICE_CLIENT_IDS gets a `member`
# relation on `team:crowdfunding-services`, which CF's Heimdall ruleset
# checks on the owner-info/published-list machine routes once
# `openfga.enabled` is true (see backend/docs/rewrite/12-fga-authorization-model.md,
# "M2M/client-credentials callers as team members").
#
# Prerequisites:
#   - kubectl port-forward to the target OpenFGA on localhost:8080, OR
#     run from inside the cluster (lfx-platform-nats-box has curl + jq)
#   - jq installed
#
# Required env vars:
#   OPENFGA_STORE_ID     — OpenFGA store ID for the target environment
#   SERVICE_CLIENT_IDS   — comma-separated Auth0 M2M client IDs (e.g. RS's)
#
# Optional env vars:
#   OPENFGA_URL          — default: http://localhost:8080
#
# IMPORTANT — user format:
#   Heimdall renders a client-credentials caller as `user:<client_id>@clients`.
#   SERVICE_CLIENT_IDS entries must be the raw Auth0 client ID, no prefix/suffix —
#   this script appends "@clients" itself.
#
# Usage:
#   kubectl -n lfx port-forward svc/lfx-platform-openfga 8080:8080
#
#   OPENFGA_STORE_ID=<store-id> \
#   SERVICE_CLIENT_IDS="<rs-client-id>" \
#   ./scripts/setup-crowdfunding-services-team.sh [--dry-run]
#
# To remove a member, use the matching revoke-crowdfunding-services-team.sh.

set -euo pipefail

BASE_URL="${OPENFGA_URL:-http://localhost:8080}"
STORE_ID="${OPENFGA_STORE_ID:?OPENFGA_STORE_ID must be set}"
SERVICE_CLIENT_IDS="${SERVICE_CLIENT_IDS:?SERVICE_CLIENT_IDS must be set (comma-separated Auth0 client IDs)}"
DRY_RUN=false

for arg in "$@"; do
	case "$arg" in
		--dry-run) DRY_RUN=true ;;
		*) echo "Unknown argument: $arg"; exit 1 ;;
	esac
done

if [[ "$DRY_RUN" == true ]]; then
	echo "=== DRY RUN MODE — no writes will be performed ==="
fi

TEAM_OBJECT="team:crowdfunding-services"

echo "Store:    $STORE_ID"
echo "Team:     $TEAM_OBJECT"
echo "Base URL: $BASE_URL"
echo ""

# fga_read paginates through /read for a given tuple_key JSON fragment,
# collecting all results into stdout as newline-separated user strings.
fga_read() {
	local filter_json="$1"
	local token=""
	while true; do
		local body
		if [[ -z "$token" ]]; then
			body=$(jq -n --argjson tk "$filter_json" '{"tuple_key":$tk,"page_size":100}')
		else
			body=$(jq -n --argjson tk "$filter_json" --arg ct "$token" \
				'{"tuple_key":$tk,"page_size":100,"continuation_token":$ct}')
		fi
		local resp
		if ! resp=$(curl -sf --show-error -X POST "${BASE_URL}/stores/${STORE_ID}/read" \
			-H 'Content-Type: application/json' \
			-d "$body" 2>&1); then
			echo "ERROR: curl failed: $resp" >&2
			exit 1
		fi
		if echo "$resp" | jq -e '.code' >/dev/null 2>&1; then
			echo "ERROR reading tuples: $(echo "$resp" | jq -r '.message')" >&2
			exit 1
		fi
		echo "$resp" | jq -r '.tuples[]?.key.user'
		token=$(echo "$resp" | jq -r '.continuation_token // ""')
		[[ -z "$token" ]] && break
	done
}

# ---------------------------------------------------------------------------
# Step 1: Read existing members (paginated)
# ---------------------------------------------------------------------------

echo "=== Step 1: Reading existing team members ==="

existing_users=$(fga_read "$(jq -n --arg obj "$TEAM_OBJECT" '{"object":$obj,"relation":"member"}')")

if [[ -n "$existing_users" ]]; then
	echo "Existing members:"
	while IFS= read -r line; do echo "  $line"; done <<< "$existing_users"
else
	echo "  (none)"
fi
echo ""

# ---------------------------------------------------------------------------
# Step 2: Determine which clients need to be written
# ---------------------------------------------------------------------------

echo "=== Step 2: Computing writes ==="

tuples_to_write="[]"
while IFS= read -r client_id; do
	fga_user="user:${client_id}@clients"

	if echo "$existing_users" | grep -qxF "$fga_user"; then
		echo "  SKIP (already member): $fga_user"
	else
		echo "  WRITE: $fga_user -> member -> $TEAM_OBJECT"
		tuples_to_write=$(echo "$tuples_to_write" | jq \
			--arg u "$fga_user" --arg r "member" --arg o "$TEAM_OBJECT" \
			'. + [{"user":$u,"relation":$r,"object":$o}]')
	fi
done < <(echo "$SERVICE_CLIENT_IDS" | tr ',' '\n' | tr -d '[:blank:]' | awk 'NF && !seen[$0]++')
echo ""

write_count=$(echo "$tuples_to_write" | jq 'length')
if [[ "$write_count" -eq 0 ]]; then
	echo "All clients are already members — nothing to do."
	exit 0
fi

# ---------------------------------------------------------------------------
# Step 3: Write new member tuples
# ---------------------------------------------------------------------------

echo "=== Step 3: Writing $write_count tuple(s) ==="

if [[ "$DRY_RUN" == true ]]; then
	echo "[DRY RUN] Would write:"
	echo "$tuples_to_write" | jq -r '.[] | "  \(.user) -> \(.relation) -> \(.object)"'
	exit 0
fi

payload=$(jq -n --argjson keys "$tuples_to_write" '{"writes":{"tuple_keys":$keys}}')
write_resp=""
if ! write_resp=$(curl -sf --show-error -X POST "${BASE_URL}/stores/${STORE_ID}/write" \
	-H 'Content-Type: application/json' \
	-d "$payload" 2>&1); then
	echo "ERROR: curl failed: $write_resp"
	exit 1
fi
if echo "$write_resp" | jq -e '.code' >/dev/null 2>&1; then
	echo "ERROR writing tuples: $(echo "$write_resp" | jq -r '.message')"
	exit 1
fi

echo "  Done — $write_count tuple(s) written."
echo ""
echo "NOTE: bust fga-sync-cache for this store after this write so cached"
echo "checks don't serve stale results (see lfx-v2-fga-sync)."
echo ""

# ---------------------------------------------------------------------------
# Step 4: Verify (paginated)
# ---------------------------------------------------------------------------

echo "=== Step 4: Verifying ==="

echo "Current members of $TEAM_OBJECT:"
fga_read "$(jq -n --arg obj "$TEAM_OBJECT" '{"object":$obj,"relation":"member"}')" | \
	while IFS= read -r line; do echo "  $line"; done
echo ""
echo "=== Done ==="
