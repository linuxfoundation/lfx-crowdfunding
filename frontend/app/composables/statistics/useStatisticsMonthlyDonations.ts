// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { MonthlyDonations } from '#shared/types/statistics.types';

export function useStatisticsMonthlyDonations() {
  const query = useQuery<MonthlyDonations>({
    queryKey: ['statistics', 'monthly-donations'] as const,
    queryFn: () => $fetch<MonthlyDonations>('/api/statistics/monthly-donations'),
  });

  onServerPrefetch(async () => {
    await query.suspense();
  });

  return query;
}
