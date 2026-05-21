// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendStatistics } from '../../types/statistics.types';
import type { StatisticsOverview } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<StatisticsOverview> => {
  const { apiBaseUrl } = useRuntimeConfig();
  const res = await $fetch<BackendStatistics>(`${apiBaseUrl}/v1/statistics`);
  return {
    totalRaisedCents: res.total_raised_cents,
    supporterCount: res.total_supporters,
    activeInitiatives: res.total_initiatives,
    annualGoalCents: 1_000_000_000,
  };
});
