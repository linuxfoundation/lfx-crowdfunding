// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { InvestingCompaniesResponse } from '#shared/types/static-pages.types';

export function useInvestingCompanies() {
  const query = useQuery<InvestingCompaniesResponse>({
    queryKey: ['static-pages', 'investing-companies'] as const,
    queryFn: () => $fetch<InvestingCompaniesResponse>('/api/static-pages/investing-companies'),
  });

  onServerPrefetch(async () => {
    await query.suspense();
  });

  return query;
}
