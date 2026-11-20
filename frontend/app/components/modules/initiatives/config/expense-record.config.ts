// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { ExpenseRecord } from '#shared/types/initiative-detail.types';
import type { Transaction } from '#shared/types/transaction.types';
import { formatShortDate } from '~/utils/date';

/**
 * One line of the Financials expense table. A return (#489) reads "Returned to <org>";
 * a payout or reimbursement shows its payee. LF's reason, when there is one, goes below.
 */
export const toExpenseRecord = (t: Transaction): ExpenseRecord => ({
  id: t.id,
  date: formatShortDate(t.date),
  category: t.category ?? 'Other',
  description:
    t.type === 'return' && t.donorName
      ? `Returned to ${t.donorName}`
      : (t.donorName ?? t.category ?? 'Other'),
  comment: t.comment || undefined,
  amountCents: t.amountCents,
});
