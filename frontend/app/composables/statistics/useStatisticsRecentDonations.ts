// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { RecentDonationsResponse } from '#shared/types/statistics.types';

export function useStatisticsRecentDonations() {
  const query = useQuery<RecentDonationsResponse>({
    queryKey: ['statistics', 'recent-donations'] as const,
    queryFn: () => $fetch<RecentDonationsResponse>('/api/statistics/recent-donations'),
  });

  onServerPrefetch(async () => {
    await query.suspense();
  });

  return query;
}
