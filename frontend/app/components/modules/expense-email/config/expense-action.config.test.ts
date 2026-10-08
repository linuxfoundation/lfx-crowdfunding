// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, expect, it } from 'vitest';
import { getExpenseAction } from './expense-action.config';

describe('getExpenseAction', () => {
  it('resolves approve and reject', () => {
    expect(getExpenseAction('approve')?.confirmLabel).toBe('Approve');
    expect(getExpenseAction('reject')?.confirmLabel).toBe('Reject');
  });

  it.each(['constructor', '__proto__', 'toString', 'valueOf', 'foo', ''])(
    'rejects %j',
    (action) => {
      expect(getExpenseAction(action)).toBeUndefined();
    },
  );
});
