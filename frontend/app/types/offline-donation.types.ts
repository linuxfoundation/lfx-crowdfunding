// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { LedgerAdjustmentKind } from '#shared/types/ledger-adjustment.types';
import type { OrgClaim } from '#shared/types/organization.types';

/** What the form records: money in (#464), or money out (#489). */
export type OfflineEntryKind = 'donation' | LedgerAdjustmentKind;

/** The offline donation / adjustment form's working state. */
export interface OfflineDonationForm {
  kind: OfflineEntryKind;
  organization: OrgClaim | null;
  /** Payee of a payout. */
  paidTo: string;
  /** Dollars as typed, always positive; separators are stripped before submit. */
  amount: string;
  /** The date received (donation), returned or paid out (adjustment). */
  receivedOn: Date | null;
  category: string;
  /** Why money is taken out; adjustments only. */
  reason: string;
  sourceTxnId: string;
}

/** A choice in the form's type selector, with the labels its fields use. */
export interface OfflineEntryKindOption {
  value: OfflineEntryKind;
  label: string;
  description: string;
  dateLabel: string;
  orgLabel: string;
  orgRequired: boolean;
  submitLabel: string;
  /** The review step's warning about what the entry does. */
  reviewNotice: string;
}

export type OfflineDonationStep = 'form' | 'review' | 'done';

/** What the done step shows after a submit. */
export interface RecordedOutcome {
  title: string;
  message: string;
  icon: string;
  iconClass: string;
}
