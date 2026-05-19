// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformDetails } from '../../types/statistics.types';
import type { TopDonorsResponse, TopDonor } from '#shared/types/statistics.types';

export default defineEventHandler(async (): Promise<TopDonorsResponse> => {
  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  const res = await $fetch<BackendPlatformDetails>(`${apiBase}/v1/statistics/platform`);

  const organizations: TopDonor[] = res.top_organizations.map((org, i) => ({
    rank: i + 1,
    id: org.id,
    name: org.name,
    logoUrl: org.avatar_url,
    amountCents: org.total_cents,
  }));

  const individuals: TopDonor[] = res.top_individuals.map((ind, i) => ({
    rank: i + 1,
    id: ind.id,
    name: ind.name,
    logoUrl: ind.avatar_url,
    amountCents: ind.total_cents,
  }));

  return { organizations, individuals };
});
