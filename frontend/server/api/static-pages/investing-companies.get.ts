// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendOrgDonation } from '../../types/statistics.types';
import type { InvestingCompaniesResponse } from '#shared/types/static-pages.types';

export default defineEventHandler(async (): Promise<InvestingCompaniesResponse> => {
  const { apiBaseUrl } = useRuntimeConfig();
  const res = await $fetch<BackendOrgDonation[]>(`${apiBaseUrl}/v1/statistics/org-donations`);
  return {
    data: (res ?? []).map((d) => ({
      id: d.orgId,
      name: d.name,
      logoUrl: d.avatar_url,
      contributedCents: d.amount_in_cents,
    })),
  };
});
