// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { OrgClaim } from './organization.types';

/** POST /api/offline-donations body: an offline org donation an approver records (#464). */
export interface OfflineDonationRequest {
  initiativeId: string;
  organization: OrgClaim;
  /** Dollars as typed, e.g. "1300" or "1300.50"; the backend converts to cents. */
  amount: string;
  /** YYYY-MM-DD */
  receivedOn: string;
  category: string;
  /** The Ledger sourceTxnID, e.g. "LFX Mentorship CNCF Fall 2026 Row 52". */
  sourceTxnId: string;
}

/** A recorded offline org donation. */
export interface OfflineDonation {
  id: string;
  initiativeId: string;
  initiativeName: string;
  initiativeSlug: string;
  organizationName: string;
  organizationDomain: string;
  amountCents: number;
  category: string;
  receivedOn: string;
  sourceTxnId: string;
  status: string;
  recordedBy: string;
  createdOn: string;
}

/** POST /api/offline-donations reply. */
export type OfflineDonationResult = OfflineDonation;

export interface OfflineDonationList {
  data: OfflineDonation[];
  total: number;
}

export interface ApproverStatus {
  approver: boolean;
}
