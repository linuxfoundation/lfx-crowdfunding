// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { OfflineDonationForm } from '~/types/offline-donation.types';
import type { OfflineDonationRequest } from '#shared/types/offline-donation.types';

// Approvers record org donations LF finance received by invoice or wire (#464).

/** The approver page; ?initiative=<slug> pre-fills the initiative. */
export const OFFLINE_DONATIONS_PATH = '/approvals/offline-donations';

export const offlineDonationLink = (slug: string) =>
  `${OFFLINE_DONATIONS_PATH}?initiative=${encodeURIComponent(slug)}`;

/**
 * Ledger txnCategory finance uses for offline CF donations. Picking a funding goal
 * instead credits that goal's progress (goals match Ledger categories by name).
 */
export const DEFAULT_OFFLINE_CATEGORY = 'general fund';
export const MENTORSHIP_OFFLINE_CATEGORY = 'mentorship';

/** The initiative's funding goals plus the default category, default first. */
export const offlineCategoryOptions = (initiativeType: string, goalNames: string[]): string[] => {
  const fallback =
    initiativeType === 'mentorship' ? MENTORSHIP_OFFLINE_CATEGORY : DEFAULT_OFFLINE_CATEGORY;
  // The Ledger lowercases categories, so "General Fund" and "general fund" are one.
  return [fallback, ...goalNames.filter((n) => n && n.toLowerCase() !== fallback)];
};

/** Same rule as the backend: up to $9,999,999.99, at most two decimals. */
const DOLLAR_AMOUNT = /^\d{1,7}(\.\d{1,2})?$/;

/** Strips "$", commas and spaces so "$1,300.50" is sent as "1300.50". */
export const normalizeDollarAmount = (raw: string): string => raw.replace(/[$,\s]/g, '');

export const isValidDollarAmount = (raw: string): boolean => {
  const v = normalizeDollarAmount(raw);
  return DOLLAR_AMOUNT.test(v) && Number(v) > 0;
};

/** YYYY-MM-DD of the day the approver picked, in their own calendar. */
export const toIsoDate = (d: Date): string =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

export const emptyOfflineDonationForm = (
  category = DEFAULT_OFFLINE_CATEGORY,
): OfflineDonationForm => ({
  organization: null,
  amount: '',
  receivedOn: null,
  category,
  sourceTxnId: '',
});

export const isOfflineDonationFormComplete = (form: OfflineDonationForm): boolean =>
  !!form.organization &&
  isValidDollarAmount(form.amount) &&
  !!form.receivedOn &&
  form.category.trim() !== '' &&
  form.sourceTxnId.trim() !== '';

export const toOfflineDonationRequest = (
  initiativeId: string,
  form: OfflineDonationForm,
): OfflineDonationRequest => ({
  initiativeId,
  organization: form.organization!,
  amount: normalizeDollarAmount(form.amount),
  receivedOn: toIsoDate(form.receivedOn!),
  category: form.category.trim(),
  sourceTxnId: form.sourceTxnId.trim(),
});

/** "$1,300.50" from a typed amount, for the review step. */
export const formatTypedDollars = (raw: string): string =>
  Number(normalizeDollarAmount(raw)).toLocaleString('en-US', {
    style: 'currency',
    currency: 'USD',
  });

export const formatCents = (cents: number): string =>
  (cents / 100).toLocaleString('en-US', { style: 'currency', currency: 'USD' });

/** What to tell the approver for a failed submit, by status code. */
export const offlineDonationErrorMessage = (
  status: number | undefined,
  fallback?: string,
): string => {
  switch (status) {
    case 409:
      return 'This source reference is already recorded in the Ledger, for a different donation or from an earlier manual entry. Check the reference, or the recorded donations below.';
    case 403:
      return 'Only approvers can record offline donations.';
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
