// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Wire shapes of the Go backend's ledger adjustment routes (#489).

export interface BackendLedgerAdjustment {
  id: string;
  kind: string;
  initiative_id: string;
  initiative_name?: string;
  initiative_slug?: string;
  organization_name: string;
  organization_domain: string;
  paid_to: string;
  amount_cents: number;
  category: string;
  reason: string;
  adjusted_on: string;
  source_txn_id: string;
  status: string;
  recorded_by?: string;
  created_on: string;
}

export interface BackendLedgerAdjustmentList {
  data: BackendLedgerAdjustment[];
  meta: { total: number; limit: number; offset: number };
}

export interface BackendLedgerAdjustmentInput {
  kind: string;
  initiative_id: string;
  organization: { name: string; domain: string } | null;
  paid_to: string;
  amount: string;
  adjusted_on: string;
  category: string;
  reason: string;
  source_txn_id: string;
}
