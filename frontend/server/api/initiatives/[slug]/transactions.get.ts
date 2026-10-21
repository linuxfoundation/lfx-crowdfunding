// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendTransactionList } from '../../../types/transactions.types';
import type { BackendInitiative } from '../../../types/initiatives.types';
import { mapToTransaction } from '../../../services/transactions.services';
import { useBackendFetch } from '../../../utils/backend-fetch';
import type { TransactionList } from '#shared/types/transaction.types';

export default defineEventHandler(async (event): Promise<TransactionList> => {
  const slug = getRouterParam(event, 'slug');
  const { type, limit, offset } = getQuery(event);

  // Authenticated lookup so approvers and owners can resolve non-published initiatives.
  const initiative = await useBackendFetch<BackendInitiative>(
    event,
    `/crowdfunding/initiatives/${slug}`,
  ).catch(() => {
    throw createError({ statusCode: 404, statusMessage: 'Initiative not found' });
  });

  // The backend's transactions endpoint is published-only, and an initiative can't take
  // donations before it's published, so there's nothing to fetch.
  if (initiative.status?.toLowerCase() !== 'published') {
    return { data: [], totalCount: 0, limit: Number(limit) || 0, offset: Number(offset) || 0 };
  }

  const params = new URLSearchParams();
  if (type) params.set('type', String(type));
  if (limit) params.set('limit', String(limit));
  if (offset) params.set('offset', String(offset));

  const res = await useBackendFetch<BackendTransactionList>(
    event,
    `/crowdfunding/initiatives/${initiative.id}/transactions?${params}`,
  );

  return {
    data: (res.data ?? []).map(mapToTransaction),
    totalCount: res.total_count,
    limit: res.limit,
    offset: res.offset,
  };
});
