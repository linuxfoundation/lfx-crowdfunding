// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformDetails } from '../../types/statistics.types';
import type { DonorBreakdown } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<DonorBreakdown> => {
  const { apiBaseUrl } = useRuntimeConfig();
  try {
    const res = await $fetch<BackendPlatformDetails>(`${apiBaseUrl}/v1/statistics/platform`);

    const totalCents = res.organizations_cents + res.individuals_cents;
    const totalDonations = (res.categories ?? []).reduce((sum, c) => sum + c.count, 0);
    const avgDonationCents = totalDonations > 0 ? Math.round(totalCents / totalDonations) : 0;

    return {
      avgDonationCents,
      organizationsCents: res.organizations_cents,
      individualsCents: res.individuals_cents,
    };
  } catch {
    return { avgDonationCents: 0, organizationsCents: 0, individualsCents: 0 };
  }
});
