// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformMonthly } from '../../types/statistics.types';
import { mapToMonthlyBucket } from '../../services/statistics.service';
import type { MonthlyDonations } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<MonthlyDonations> => {
  const { apiBaseUrl } = useRuntimeConfig();
  try {
    const res = await $fetch<BackendPlatformMonthly>(`${apiBaseUrl}/v1/statistics/monthly`);
    return { buckets: (res.buckets ?? []).map(mapToMonthlyBucket) };
  } catch {
    return { buckets: [] };
  }
});
