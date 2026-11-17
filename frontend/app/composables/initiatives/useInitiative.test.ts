// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import { computed, ref } from 'vue';

const prefetch: { fn?: () => Promise<void> } = {};
const suspense = vi.fn();

vi.mock('vue', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue')>()),
  onServerPrefetch: (fn: () => Promise<void>) => {
    prefetch.fn = fn;
  },
}));
// suspense() of a disabled query never settles, like vue-query's.
vi.mock('@tanstack/vue-query', () => ({ useQuery: () => ({ suspense }) }));
vi.stubGlobal('useRequestFetch', () => vi.fn());
vi.stubGlobal('computed', computed);

import { useInitiative } from './useInitiative';

beforeEach(() => {
  prefetch.fn = undefined;
  suspense.mockReset();
});

describe('useInitiative server prefetch', () => {
  it('does not wait on the query when there is no slug', async () => {
    suspense.mockReturnValue(new Promise(() => {}));
    useInitiative(ref(''));

    await expect(prefetch.fn!()).resolves.toBeUndefined();
    expect(suspense).not.toHaveBeenCalled();
  });

  it('waits for the initiative when there is a slug', async () => {
    suspense.mockResolvedValue({});
    useInitiative(ref('example-initiative'));

    await prefetch.fn!();
    expect(suspense).toHaveBeenCalledOnce();
  });
});
