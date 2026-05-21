// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendRecentDonationsResponse } from '../../types/statistics.types';
import { mapToRecentDonation } from '../../services/statistics.service';
import type { RecentDonationsResponse } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<RecentDonationsResponse> => {
  const { apiBaseUrl } = useRuntimeConfig();
  try {
    const res = await $fetch<BackendRecentDonationsResponse>(
      `${apiBaseUrl}/v1/statistics/recent-donations`,
    );
    return { data: (res.data ?? []).map(mapToRecentDonation) };
  } catch {
    return { data: [] };
  }
});
