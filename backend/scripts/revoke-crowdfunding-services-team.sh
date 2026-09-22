#!/usr/bin/env bash
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
#
# revoke-crowdfunding-services-team.sh
#
# Removes `member` tuples from `team:crowdfunding-services` in OpenFGA.
# Matching revoke script for setup-crowdfunding-services-team.sh — see that
# script's header and backend/docs/rewrite/12-fga-authorization-model.md for
# the tuple shape this manages.
#
# Prerequisites:
#   - kubectl port-forward to the target OpenFGA on localhost:8080, OR
#     run from inside the cluster (lfx-platform-nats-box has curl + jq)
#   - jq installed
#
# Required env vars:
#   OPENFGA_STORE_ID     — OpenFGA store ID for the target environment
#   SERVICE_CLIENT_IDS   — comma-separated Auth0 M2M client IDs to remove
#
# Optional env vars:
#   OPENFGA_URL          — default: http://localhost:8080
#
# Usage:
#   kubectl -n lfx port-forward svc/lfx-platform-openfga 8080:8080
#
#   OPENFGA_STORE_ID=<store-id> \
#   SERVICE_CLIENT_IDS="<rs-client-id>" \
#   ./scripts/revoke-crowdfunding-services-team.sh [--dry-run]

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

tuples_to_delete="[]"
while IFS= read -r client_id; do
	fga_user="user:${client_id}@clients"

	echo "  DELETE: $fga_user -> member -> $TEAM_OBJECT"
	tuples_to_delete=$(echo "$tuples_to_delete" | jq \
		--arg u "$fga_user" --arg r "member" --arg o "$TEAM_OBJECT" \
		'. + [{"user":$u,"relation":$r,"object":$o}]')
done < <(echo "$SERVICE_CLIENT_IDS" | tr ',' '\n' | tr -d '[:blank:]' | awk 'NF && !seen[$0]++')
echo ""

delete_count=$(echo "$tuples_to_delete" | jq 'length')
if [[ "$delete_count" -eq 0 ]]; then
	echo "No clients given — nothing to do."
	exit 0
fi

if [[ "$DRY_RUN" == true ]]; then
	echo "[DRY RUN] Would delete:"
	echo "$tuples_to_delete" | jq -r '.[] | "  \(.user) -> \(.relation) -> \(.object)"'
	exit 0
fi

payload=$(jq -n --argjson keys "$tuples_to_delete" '{"deletes":{"tuple_keys":$keys}}')
delete_resp=""
if ! delete_resp=$(curl -sf --show-error -X POST "${BASE_URL}/stores/${STORE_ID}/write" \
	-H 'Content-Type: application/json' \
	-d "$payload" 2>&1); then
	echo "ERROR: curl failed: $delete_resp"
	exit 1
fi
if echo "$delete_resp" | jq -e '.code' >/dev/null 2>&1; then
	echo "ERROR deleting tuples: $(echo "$delete_resp" | jq -r '.message')"
	exit 1
fi

echo "  Done — $delete_count tuple(s) deleted."
echo ""
echo "NOTE: bust fga-sync-cache for this store after this write so cached"
echo "checks don't serve stale results (see lfx-v2-fga-sync)."
