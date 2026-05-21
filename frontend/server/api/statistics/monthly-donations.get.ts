// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformMonthly } from '../../types/statistics.types';
import type { MonthlyDonations } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<MonthlyDonations> => {
  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  try {
    const res = await $fetch<BackendPlatformMonthly>(`${apiBase}/v1/statistics/monthly`);
    return {
      buckets: (res.buckets ?? []).map((b) => ({
        year: b.year,
        month: b.month,
        totalCents: b.total_cents,
        supporters: b.supporters,
        newSupporters: b.new_supporters,
      })),
    };
  } catch {
    return { buckets: [] };
  }
});
