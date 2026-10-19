// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect } from 'vitest';

import type { BackendGoal, BackendInitiative } from '../types/initiatives.types';
import { isGitHubUrl, mapToInitiativeDetail } from './initiatives.services';

const initiative = (overrides: Partial<BackendInitiative>): BackendInitiative =>
  ({
    id: 'i1',
    slug: 's',
    name: 'n',
    status: 'published',
    initiative_type: 'project',
    ...overrides,
  }) as BackendInitiative;

const devGoal = (repo_link: string): BackendGoal => ({
  id: 'g1',
  name: 'development',
  goal_amount_cents: 0,
  repo_link,
});

describe('isGitHubUrl', () => {
  it.each([
    ['https://github.com/org/repo', true],
    ['https://www.GitHub.com/org/repo', true],
    ['https://git.kernel.org/pub/scm/linux', false],
    ['https://github.com.evil.io/org/repo', false],
    ['github.com/org/repo', false],
    ['N/A', false],
    [undefined, false],
  ])('%s → %s', (url, expected) => {
    expect(isGitHubUrl(url)).toBe(expected);
  });
});

describe('mapToInitiativeDetail githubURL', () => {
  it('prefers a Repository custom website', () => {
    const b = initiative({
      custom_websites: [{ name: 'Repository', url: 'https://github.com/a/site' }],
      goals: [devGoal('https://github.com/a/goal')],
    });
    expect(mapToInitiativeDetail(b).githubURL).toBe('https://github.com/a/site');
  });

  it('falls back to a legacy GitHub repo_link', () => {
    const b = initiative({ goals: [devGoal('https://github.com/a/goal')] });
    expect(mapToInitiativeDetail(b).githubURL).toBe('https://github.com/a/goal');
  });

  it('ignores a non-GitHub repo_link', () => {
    const b = initiative({ goals: [devGoal('https://wiki.example.org/project')] });
    expect(mapToInitiativeDetail(b).githubURL).toBeUndefined();
  });
});
