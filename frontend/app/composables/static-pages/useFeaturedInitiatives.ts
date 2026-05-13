// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { FeaturedInitiativesResponse } from '#shared/types/static-pages.types';

export function useFeaturedInitiatives() {
  const query = useQuery<FeaturedInitiativesResponse>({
    queryKey: ['static-pages', 'featured-initiatives'] as const,
    queryFn: () => $fetch<FeaturedInitiativesResponse>('/api/static-pages/featured-initiatives'),
  });

  onServerPrefetch(async () => {
    await query.suspense();
  });

  return query;
}
