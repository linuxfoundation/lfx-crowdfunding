// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { TopDonorsResponse } from '#shared/types/statistics.types';

export function useStatisticsTopDonors() {
  const query = useQuery<TopDonorsResponse>({
    queryKey: ['statistics', 'top-donors'] as const,
    queryFn: () => $fetch<TopDonorsResponse>('/api/statistics/top-donors'),
  });

  onServerPrefetch(async () => {
    await query.suspense().catch(() => {});
  });

  return query;
}
