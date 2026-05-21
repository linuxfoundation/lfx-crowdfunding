// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { DonorBreakdown } from '#shared/types/statistics.types';

export function useStatisticsDonorBreakdown() {
  const query = useQuery<DonorBreakdown>({
    queryKey: ['statistics', 'donor-breakdown'] as const,
    queryFn: () => $fetch<DonorBreakdown>('/api/statistics/donor-breakdown'),
  });

  onServerPrefetch(async () => {
    await query.suspense().catch(() => {});
  });

  return query;
}
