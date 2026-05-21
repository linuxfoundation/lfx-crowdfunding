// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { FundingByCategoryResponse } from '#shared/types/statistics.types';

export function useStatisticsFundingByCategory() {
  const query = useQuery<FundingByCategoryResponse>({
    queryKey: ['statistics', 'funding-by-category'] as const,
    queryFn: () => $fetch<FundingByCategoryResponse>('/api/statistics/funding-by-category'),
  });

  onServerPrefetch(async () => {
    await query.suspense().catch(() => {});
  });

  return query;
}
