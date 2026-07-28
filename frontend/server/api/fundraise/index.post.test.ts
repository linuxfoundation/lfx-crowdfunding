// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import type { GoalItemInput } from '../../types/fundraise.types';
import { buildProjectGoals } from './index.post';

const goalItem = (overrides: Partial<GoalItemInput>): GoalItemInput => ({
  category: 'development',
  label: 'Development',
  description: '',
  enabled: true,
  percentage: 0,
  ...overrides,
});

describe('buildProjectGoals', () => {
  it('emits a single goal for the enabled category, not a duplicate Annual Funding Goal row', () => {
    const goals = buildProjectGoals(400000, [goalItem({ label: 'Development', percentage: 100 })]);

    expect(goals).toEqual([
      {
        name: 'Development',
        amount_cents: 400000,
        allocation: 'development',
        description: undefined,
        sort_order: 1,
      },
    ]);
  });

  it('splits the total across enabled categories without a standalone total row', () => {
    const goals = buildProjectGoals(400000, [
      goalItem({ category: 'development', label: 'Development', percentage: 60 }),
      goalItem({ category: 'marketing', label: 'Marketing', percentage: 40 }),
    ]);

    expect(goals).toEqual([
      {
        name: 'Development',
        amount_cents: 240000,
        allocation: 'development',
        description: undefined,
        sort_order: 1,
      },
      {
        name: 'Marketing',
        amount_cents: 160000,
        allocation: 'marketing',
        description: undefined,
        sort_order: 2,
      },
    ]);
  });

  it('falls back to a single Annual Funding Goal row when no categories are enabled', () => {
    const goals = buildProjectGoals(400000, []);

    expect(goals).toEqual([{ name: 'Annual Funding Goal', amount_cents: 400000, sort_order: 0 }]);
  });
});
