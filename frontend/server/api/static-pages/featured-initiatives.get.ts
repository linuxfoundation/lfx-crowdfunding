// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendResponse } from '../../types/initiatives.types';
import type { FeaturedInitiativesResponse } from '#shared/types/static-pages.types';

// TODO: stop-gap — returns the 6 most recently created initiatives. Replace once the backend
// supports a proper "featured" concept (e.g. a curated list or a featured flag on initiatives).
export default defineEventHandler(async (): Promise<FeaturedInitiativesResponse> => {
  const { apiBaseUrl } = useRuntimeConfig();

  const res = await $fetch<BackendResponse>(
    `${apiBaseUrl}/v1/initiatives?status=published&limit=6&offset=0`,
  );

  return {
    data: (res.data ?? []).map((i) => ({
      id: i.id,
      slug: i.slug,
      name: i.name,
      logoUrl: i.logo_url,
      raisedCents: i.financials?.total_raised_cents ?? 0,
      goalCents: i.financials?.goals_total_cents ?? 0,
      supporterCount: i.financials?.supporters ?? 0,
    })),
  };
});
