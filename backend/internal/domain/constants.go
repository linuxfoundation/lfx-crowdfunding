// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package domain

// LedgerSourceTypeStripe is the source type value the Ledger service assigns
// to transactions created by Stripe payments through this service. It is used
// both when posting new Ledger credit transactions (webhook_handler) and when
// filtering the recent-donations feed to exclude non-Stripe allocations such
// as Expensify disbursements (statistics_service).
const LedgerSourceTypeStripe = "stripe"

// Ledger values for offline org donations (#464), copied from the payloads
// approvers posted by hand before CF recorded them. The shared CommunityBridge
// account is "not a person": its credits are org money allocated by LF
// (ledger-service scripts/backfill-organization-id).
const (
	LedgerOfflineUserID          = "auth0|communitybridge"
	LedgerOfflineAccountEmail    = "admin@communitybridge.org"
	LedgerOfflineSourceType      = "expensify"
	LedgerOfflineSourceAccountID = "manual_from_lf"
	LedgerTxnTypeCredit          = "credit"
)

// OfflineAccountUsername is the CF users row offline donations belong to
// (migration 019). No login can produce it, so sign-ins never touch the row.
const OfflineAccountUsername = "lf:offline-donations"
