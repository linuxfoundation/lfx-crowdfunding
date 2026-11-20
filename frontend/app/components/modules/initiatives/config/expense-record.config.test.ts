// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, expect, it } from 'vitest';
import { toExpenseRecord } from './expense-record.config';
import type { Transaction } from '#shared/types/transaction.types';

const base: Transaction = {
  id: 't1',
  type: 'reimbursement',
  amountCents: -150000,
  date: '2026-11-02T00:00:00Z',
  category: 'mentorship',
};

describe('toExpenseRecord', () => {
  it('shows a return as returned to its org, with the reason', () => {
    const r = toExpenseRecord({
      ...base,
      type: 'return',
      donorName: 'CNCF',
      comment: 'Mentee failed',
    });
    expect(r.description).toBe('Returned to CNCF');
    expect(r.comment).toBe('Mentee failed');
    expect(r.amountCents).toBe(-150000);
  });

  it('shows a payout by its payee', () => {
    expect(toExpenseRecord({ ...base, donorName: 'Jane Payee' }).description).toBe('Jane Payee');
  });

  it('falls back to the category and leaves an empty comment out', () => {
    const r = toExpenseRecord({ ...base, comment: '' });
    expect(r.description).toBe('mentorship');
    expect(r.comment).toBeUndefined();
  });
});
