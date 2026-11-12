// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, expect, it } from 'vitest';
import { formatShortDate } from './date';

describe('formatShortDate', () => {
  it('formats the UTC day regardless of the runtime timezone', () => {
    // 02:00 UTC on Mar 1 is still Feb 28 in the Americas; SSR (UTC) and the browser must agree.
    expect(formatShortDate('2026-03-01T02:00:00Z')).toBe('Mar 1, 2026');
    expect(formatShortDate('2026-02-28T23:30:00Z')).toBe('Feb 28, 2026');
  });
});
