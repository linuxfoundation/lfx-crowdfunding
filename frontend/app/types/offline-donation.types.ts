// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { OrgClaim } from '#shared/types/organization.types';

/** The offline donation form's working state (#464). */
export interface OfflineDonationForm {
  organization: OrgClaim | null;
  /** Dollars as typed; separators are stripped before submit. */
  amount: string;
  receivedOn: Date | null;
  category: string;
  sourceTxnId: string;
}

export type OfflineDonationStep = 'form' | 'review' | 'done';
