// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendResponse } from '../../types/initiatives.types';
import { mapToInitiativeBase } from '../../services/initiatives.services';
import type { InitiativesResponse } from '#shared/types/initiative.types';

export default defineEventHandler(async (event): Promise<InitiativesResponse> => {
  const { search, type, sort, page, pageSize } = getQuery(event);

  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  const params = new URLSearchParams();
  if (search) params.set('search', String(search));
  if (type && type !== 'all') params.set('type', String(type));
  if (sort) params.set('sort_by', String(sort));

  const pageSizeNum = typeof pageSize === 'string' ? Math.max(1, parseInt(pageSize, 10) || 12) : 12;
  const pageNum = typeof page === 'string' ? Math.max(1, parseInt(page, 10) || 1) : 1;
  params.set('limit', String(pageSizeNum));
  params.set('offset', String((pageNum - 1) * pageSizeNum));

  const res = await $fetch<BackendResponse>(`${apiBase}/v1/initiatives?${params}`);

  return {
    data: (res.data ?? []).map(mapToInitiativeBase),
    total: res.meta.total,
    page: pageNum,
    pageSize: pageSizeNum,
  };
});
