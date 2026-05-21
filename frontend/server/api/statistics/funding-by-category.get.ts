// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformDetails } from '../../types/statistics.types';
import { mapToFundingCategory } from '../../services/statistics.service';
import type { FundingByCategoryResponse } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<FundingByCategoryResponse> => {
  const { apiBaseUrl } = useRuntimeConfig();
  try {
    const res = await $fetch<BackendPlatformDetails>(`${apiBaseUrl}/v1/statistics/platform`);
    return { data: (res.categories ?? []).map(mapToFundingCategory) };
  } catch {
    return { data: [] };
  }
});
