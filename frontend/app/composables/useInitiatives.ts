// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { useQuery } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { toValue } from 'vue';
import type { InitiativesParams, InitiativesResponse } from '#shared/types/initiative.types';

export type { InitiativesParams, InitiativesResponse };

export function useInitiatives(params?: { [K in keyof InitiativesParams]: MaybeRef<string> }) {
  return useQuery<InitiativesResponse>({
    queryKey: [
      'initiatives',
      params?.search ?? '',
      params?.type ?? '',
      params?.sort ?? '',
    ] as const,
    queryFn: () => {
      const query = new URLSearchParams();
      const search = toValue(params?.search ?? '');
      const type = toValue(params?.type ?? '');
      const sort = toValue(params?.sort ?? '');
      if (search) query.set('search', search);
      if (type) query.set('type', type);
      if (sort) query.set('sort', sort);
      const qs = query.toString();
      return $fetch<InitiativesResponse>(`/api/initiatives${qs ? `?${qs}` : ''}`);
    },
  });
}
