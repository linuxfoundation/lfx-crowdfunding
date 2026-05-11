// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
import { useQuery } from '@tanstack/vue-query';
import type { Entity } from '~/types/entity.types';

export interface EntitiesResponse {
  data: Entity[];
  total: number;
}

export function useEntities() {
  return useQuery<EntitiesResponse>({
    queryKey: ['entities'],
    queryFn: () => $fetch<EntitiesResponse>('/api/entities'),
  });
}
