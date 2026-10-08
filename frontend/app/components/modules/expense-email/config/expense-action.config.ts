// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { ExpenseActionConfig } from '~/types/expense.types';

export const EXPENSE_ACTIONS: Record<string, ExpenseActionConfig> = {
  approve: {
    title: 'Approve expense report',
    description:
      'Approving releases the reimbursement from the initiative funds to the beneficiary.',
    confirmLabel: 'Approve',
    verb: 'approve',
    past: 'approved',
  },
  reject: {
    title: 'Reject expense report',
    description: 'Rejecting declines the reimbursement request for this expense report.',
    confirmLabel: 'Reject',
    verb: 'reject',
    past: 'rejected',
  },
};

// Own-key lookup so URL params like 'constructor' or '__proto__' don't resolve to inherited members.
export const getExpenseAction = (action: string): ExpenseActionConfig | undefined =>
  Object.hasOwn(EXPENSE_ACTIONS, action) ? EXPENSE_ACTIONS[action] : undefined;
