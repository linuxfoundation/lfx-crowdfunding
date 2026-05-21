// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformDetails } from '../../types/statistics.types';
import { mapToTopDonor } from '../../services/statistics.service';
import type { TopDonorsResponse } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<TopDonorsResponse> => {
  const { apiBaseUrl } = useRuntimeConfig();
  try {
    const res = await $fetch<BackendPlatformDetails>(`${apiBaseUrl}/v1/statistics/platform`);
    return {
      organizations: (res.top_organizations ?? []).map(mapToTopDonor),
      individuals: (res.top_individuals ?? []).map(mapToTopDonor),
    };
  } catch {
    return { organizations: [], individuals: [] };
  }
});
