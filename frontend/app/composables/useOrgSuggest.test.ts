// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect } from 'vitest';
import { ref } from 'vue';

vi.mock('@vueuse/core', () => ({ refDebounced: (source: unknown) => source }));
vi.mock('@tanstack/vue-query', () => ({ useQuery: () => ({}) }));

import { useOrgSuggest } from './useOrgSuggest';

describe('useOrgSuggest', () => {
  it('only searches once the trimmed query reaches the minimum length', () => {
    const query = ref('');
    const { enabled } = useOrgSuggest(query);
    expect(enabled.value).toBe(false);

    query.value = ' a ';
    expect(enabled.value).toBe(false);

    query.value = 'ac';
    expect(enabled.value).toBe(true);
  });
});
