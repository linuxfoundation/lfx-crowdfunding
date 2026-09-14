// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';
import type { GoalItemInput, GeneralFundFundraisePayload } from '../../types/fundraise.types';
import { buildProjectGoals, buildBackendPayload } from './index.post';

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

describe('buildBackendPayload — attribution', () => {
  const basePayload = (): GeneralFundFundraisePayload => ({
    initiativeType: 'general_fund',
    name: 'General Fund',
    description: 'A general fund',
  });

  it('omits attribution when no attribution is sent', () => {
    const backendPayload = buildBackendPayload(basePayload());

    expect(backendPayload.attribution).toBeUndefined();
  });

  it('maps kind + entityId to attribution.type + .entity_uid', () => {
    const backendPayload = buildBackendPayload({
      ...basePayload(),
      attribution: { kind: 'organization', entityId: '8b1e2c3d-4f56-4a78-9b0c-1d2e3f4a5b6c' },
    });

    expect(backendPayload.attribution).toEqual({
      type: 'organization',
      entity_uid: '8b1e2c3d-4f56-4a78-9b0c-1d2e3f4a5b6c',
    });
  });
});
