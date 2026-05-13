// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { useQuery } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { toValue } from 'vue';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

export function useInitiative(id: MaybeRef<string>) {
  return useQuery<InitiativeDetail>({
    queryKey: ['initiative', id] as const,
    queryFn: () => $fetch<InitiativeDetail>(`/api/initiatives/${toValue(id)}`),
    enabled: computed(() => !!toValue(id)),
  });
}
