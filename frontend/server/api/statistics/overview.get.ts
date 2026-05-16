// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { StatisticsOverview } from '#shared/types/statistics.types';

interface BackendStatistics {
  total_raised_cents: number;
  total_supporters: number;
  total_initiatives: number;
}

export default defineEventHandler(async (): Promise<StatisticsOverview> => {
  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  const res = await $fetch<BackendStatistics>(`${apiBase}/v1/statistics`);
  return {
    totalRaisedCents: res.total_raised_cents,
    supporterCount: res.total_supporters,
    activeInitiatives: res.total_initiatives,
    annualGoalCents: 0,
  };
});
