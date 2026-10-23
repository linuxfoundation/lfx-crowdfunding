// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { computed, type Ref } from 'vue';
import { refDebounced } from '@vueuse/core';
import { useQuery } from '@tanstack/vue-query';
import type { OrgClaim } from '#shared/types/organization.types';
import {
  ORG_SEARCH_DEBOUNCE_MS,
  ORG_SEARCH_MIN_CHARS,
} from '~/components/modules/donate/config/org-search.config';

/** Organization typeahead (Clearbit via query-service) for the donate flow. */
export function useOrgSuggest(query: Ref<string>) {
  const term = refDebounced(
    computed(() => query.value.trim()),
    ORG_SEARCH_DEBOUNCE_MS,
  );
  const enabled = computed(() => term.value.length >= ORG_SEARCH_MIN_CHARS);

  const result = useQuery<OrgClaim[]>({
    queryKey: ['org-suggest', term] as const,
    queryFn: () => $fetch<OrgClaim[]>('/api/orgs/suggest', { query: { query: term.value } }),
    enabled,
    staleTime: 5 * 60_000,
  });

  return { ...result, enabled };
}
