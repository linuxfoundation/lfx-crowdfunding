// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { StatisticsOverview } from '#shared/types/statistics.types';

export function useStatisticsOverview() {
  const query = useQuery<StatisticsOverview>({
    queryKey: ['statistics', 'overview'] as const,
    queryFn: () => $fetch<StatisticsOverview>('/api/statistics/overview'),
  });

  onServerPrefetch(async () => {
    await query.suspense();
  });

  return query;
}
