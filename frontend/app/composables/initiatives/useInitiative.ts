// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { useQuery } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { onServerPrefetch, toValue } from 'vue';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

export function useInitiative(slug: MaybeRef<string>) {
  // Forwards the session cookie during SSR so approvers can load non-published initiatives.
  const requestFetch = useRequestFetch();
  const query = useQuery<InitiativeDetail>({
    queryKey: ['initiative', slug] as const,
    queryFn: () => requestFetch<InitiativeDetail>(`/api/initiatives/${toValue(slug)}`),
    enabled: computed(() => !!toValue(slug)),
  });

  onServerPrefetch(async () => {
    await query.suspense().catch(() => {});
  });

  return query;
}
