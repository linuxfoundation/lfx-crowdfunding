// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { OrgClaim } from './organization.types';

/**
 * Money an approver takes out of an initiative (#489): `return` sends unused funds
 * back to the donor org, `payout` pays a person out of a fund.
 */
export type LedgerAdjustmentKind = 'return' | 'payout';

/** POST /api/ledger-adjustments body. */
export interface LedgerAdjustmentRequest {
  kind: LedgerAdjustmentKind;
  initiativeId: string;
  /** The donor whose money it is: required for a return, optional for a payout. */
  organization: OrgClaim | null;
  /** The payee of a payout; shown on the public Financials tab. */
  paidTo: string;
  /** Positive dollars as typed, e.g. "1500"; the Ledger row gets it negated. */
  amount: string;
  /** YYYY-MM-DD */
  adjustedOn: string;
  category: string;
  /** Why the money is taken out; shown on the public Financials tab. */
  reason: string;
  /** The Ledger sourceTxnID. */
  sourceTxnId: string;
}

/** A recorded return or payout. */
export interface LedgerAdjustment {
  id: string;
  kind: LedgerAdjustmentKind;
  initiativeId: string;
  initiativeName: string;
  initiativeSlug: string;
  organizationName: string;
  organizationDomain: string;
  paidTo: string;
  amountCents: number;
  category: string;
  reason: string;
  adjustedOn: string;
  sourceTxnId: string;
  status: string;
  recordedBy: string;
  createdOn: string;
}

/** POST /api/ledger-adjustments reply. */
export interface LedgerAdjustmentResult extends LedgerAdjustment {
  /** False when the same adjustment was already recorded: this is the stored entry, unchanged. */
  created: boolean;
}

export interface LedgerAdjustmentList {
  data: LedgerAdjustment[];
  total: number;
}
