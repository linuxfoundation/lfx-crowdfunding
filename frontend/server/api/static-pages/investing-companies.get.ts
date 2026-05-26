// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformDetails } from '../../types/statistics.types';
import type { InvestingCompaniesResponse } from '#shared/types/static-pages.types';

export default defineEventHandler(async (): Promise<InvestingCompaniesResponse> => {
  const { apiBaseUrl } = useRuntimeConfig();
  const res = await $fetch<BackendPlatformDetails>(`${apiBaseUrl}/v1/statistics/platform?top_limit=20`);
  return {
    data: (res.top_organizations ?? []).map((org) => ({
      id: org.id,
      name: org.name,
      logoUrl: org.avatar_url,
      contributedCents: org.total_cents,
    })),
  };
});
