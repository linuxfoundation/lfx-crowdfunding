// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package domain

// LedgerSourceTypeStripe is the source type value the Ledger service assigns
// to transactions created by Stripe payments through this service. It is used
// both when posting new Ledger credit transactions (webhook_handler) and when
// filtering the recent-donations feed to exclude non-Stripe allocations such
// as Expensify disbursements (statistics_service).
const LedgerSourceTypeStripe = "stripe"
