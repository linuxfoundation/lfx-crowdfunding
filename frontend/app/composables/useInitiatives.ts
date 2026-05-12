// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { useQuery } from '@tanstack/vue-query';
import type { Initiative } from '~/types/initiative.types';

export interface InitiativesResponse {
  data: Initiative[];
  total: number;
}

export function useInitiatives() {
  return useQuery<InitiativesResponse>({
    queryKey: ['initiatives'],
    queryFn: () => $fetch<InitiativesResponse>('/api/initiatives'),
  });
}
