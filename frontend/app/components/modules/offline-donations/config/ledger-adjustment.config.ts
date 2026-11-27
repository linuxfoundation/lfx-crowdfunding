// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import {
  RECORDED_OUTCOME_STYLE,
  formatCents,
  isValidDollarAmount,
  normalizeDollarAmount,
  toIsoDate,
} from './offline-donation.config';
import type {
  OfflineDonationForm,
  OfflineEntryKind,
  OfflineEntryKindOption,
  RecordedOutcome,
} from '~/types/offline-donation.types';
import type {
  LedgerAdjustment,
  LedgerAdjustmentRequest,
  LedgerAdjustmentResult,
} from '#shared/types/ledger-adjustment.types';
import { formatShortDate } from '~/utils/date';

// Money approvers take out of an initiative (#489), recorded on the offline donations page.

/** Same limits as the backend. */
export const REASON_MAX_LENGTH = 500;
export const PAID_TO_MAX_LENGTH = 255;

/** The form's type selector, donation first (the page's original purpose). */
export const OFFLINE_ENTRY_KIND_OPTIONS: OfflineEntryKindOption[] = [
  {
    value: 'donation',
    label: 'Donation',
    description: 'An organization donation LF finance received by invoice or wire.',
    dateLabel: 'Date received',
    orgLabel: 'Organization',
    orgRequired: true,
    submitLabel: 'Record donation',
    reviewNotice:
      "This posts the donation to the Ledger. It can't be edited or removed from here afterwards.",
  },
  {
    value: 'return',
    label: 'Return to donor',
    description:
      "Unused funds sent back to the organization that donated them, e.g. a mentee's unpaid stipend.",
    dateLabel: 'Date returned',
    orgLabel: 'Returned to organization',
    orgRequired: true,
    submitLabel: 'Record return',
    reviewNotice:
      "This takes the amount out of the initiative's Ledger balance and shows it on the public Financials tab as returned to the organization, with the reason. It can't be edited or removed from here afterwards.",
  },
  {
    value: 'payout',
    label: 'Payout',
    description: 'Money paid to a person out of the initiative\u2019s fund, e.g. a reward payout.',
    dateLabel: 'Date paid',
    orgLabel: 'Funds from organization',
    orgRequired: false,
    submitLabel: 'Record payout',
    reviewNotice:
      "This takes the amount out of the initiative's Ledger balance and shows it on the public Financials tab as an expense paid to the payee, with the reason. It can't be edited or removed from here afterwards.",
  },
];

export const entryKindOption = (kind: OfflineEntryKind): OfflineEntryKindOption =>
  OFFLINE_ENTRY_KIND_OPTIONS.find((o) => o.value === kind) ?? OFFLINE_ENTRY_KIND_OPTIONS[0]!;

const filled = (v: string, max: number) => v.trim() !== '' && v.trim().length <= max;

/** Whether the form has everything its kind needs. */
export const isOfflineEntryFormComplete = (form: OfflineDonationForm): boolean => {
  const common =
    isValidDollarAmount(form.amount) &&
    !!form.receivedOn &&
    form.category.trim() !== '' &&
    form.sourceTxnId.trim() !== '';
  switch (form.kind) {
    case 'donation':
      return common && !!form.organization;
    case 'return':
      return common && !!form.organization && filled(form.reason, REASON_MAX_LENGTH);
    case 'payout':
      return (
        common && filled(form.paidTo, PAID_TO_MAX_LENGTH) && filled(form.reason, REASON_MAX_LENGTH)
      );
  }
};

export const toLedgerAdjustmentRequest = (
  initiativeId: string,
  form: OfflineDonationForm,
): LedgerAdjustmentRequest => {
  if (form.kind === 'donation') throw new Error('a donation is not a ledger adjustment');
  return {
    kind: form.kind,
    initiativeId,
    organization: form.organization,
    // A return goes back to the organization; it has no payee.
    paidTo: form.kind === 'payout' ? form.paidTo.trim() : '',
    amount: normalizeDollarAmount(form.amount),
    adjustedOn: toIsoDate(form.receivedOn!),
    category: form.category.trim(),
    reason: form.reason.trim(),
    sourceTxnId: form.sourceTxnId.trim(),
  };
};

/** "Returned to CNCF" or "Paid to Jane Doe", for lists and the done step. */
export const adjustmentSummary = (
  a: Pick<LedgerAdjustment, 'kind' | 'organizationName' | 'paidTo'>,
): string => (a.kind === 'return' ? `Returned to ${a.organizationName}` : `Paid to ${a.paidTo}`);

/** Heading and text of the done step for an adjustment. */
export const adjustmentOutcome = (
  a: LedgerAdjustmentResult,
  initiativeName: string,
): RecordedOutcome =>
  a.created
    ? {
        title: a.kind === 'return' ? 'Return recorded' : 'Payout recorded',
        ...RECORDED_OUTCOME_STYLE.created,
        message: `${adjustmentSummary(a)}: ${formatCents(a.amountCents)} out of ${initiativeName}. The initiative's balance updates within the hour.`,
      }
    : {
        title: 'Already recorded',
        ...RECORDED_OUTCOME_STYLE.alreadyRecorded,
        message: `This ${a.kind} was already recorded on ${formatShortDate(a.createdOn)}${a.recordedBy ? ` by ${a.recordedBy}` : ''}. No new entry was added to the Ledger.`,
      };

/** What to tell the approver for a failed adjustment submit, by status code. */
export const adjustmentErrorMessage = (status: number | undefined, fallback?: string): string => {
  switch (status) {
    case 409:
      return 'This source reference is already recorded in the Ledger, for a different entry or from an earlier manual one. Check the reference, or the recorded entries below.';
    case 403:
      return 'Only approvers can record returns and payouts.';
    case 404:
      return 'The initiative was not found, or it is not published.';
    case 503:
      return "The Ledger isn't reachable right now. Submit the same details again in a few minutes; it won't create a second entry.";
    case 400:
      return fallback ? `Check the details: ${fallback}` : 'Check the details and try again.';
    default:
      return 'Something went wrong. Submit the same details again; it won\u2019t create a second entry.';
  }
};
