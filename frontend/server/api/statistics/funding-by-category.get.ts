// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformDetails } from '../../types/statistics.types';
import {
  mapToFundingCategory,
  mergeFundingCategoriesByName,
} from '../../services/statistics.service';
import type { FundingByCategoryResponse } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<FundingByCategoryResponse> => {
  const { apiBaseUrl } = useRuntimeConfig();
  const res = await $fetch<BackendPlatformDetails>(`${apiBaseUrl}/v1/statistics/platform`);
  // Hide categories with no positive net funding (e.g. refund-only or $0 buckets).
  const categories = mergeFundingCategoriesByName(
    (res.categories ?? []).map(mapToFundingCategory),
  ).filter((c) => c.raisedCents > 0);
  return { data: categories };
});
