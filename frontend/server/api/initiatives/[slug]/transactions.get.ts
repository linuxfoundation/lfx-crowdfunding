// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendTransactionList } from '../../../types/transactions.types';
import type { BackendInitiative } from '../../../types/initiatives.types';
import { mapToTransaction } from '../../../services/transactions.services';
import type { TransactionList } from '#shared/types/transaction.types';

export default defineEventHandler(async (event): Promise<TransactionList> => {
  const slug = getRouterParam(event, 'slug');
  const { type, limit, offset } = getQuery(event);

  const { apiBaseUrl } = useRuntimeConfig();

  const initiative = await $fetch<BackendInitiative>(`${apiBaseUrl}/v1/initiatives/${slug}`).catch(
    () => {
      throw createError({ statusCode: 404, statusMessage: 'Initiative not found' });
    },
  );

  const params = new URLSearchParams();
  if (type) params.set('type', String(type));
  if (limit) params.set('limit', String(limit));
  if (offset) params.set('offset', String(offset));

  const res = await $fetch<BackendTransactionList>(
    `${apiBaseUrl}/v1/initiatives/${initiative.id}/transactions?${params}`,
  );

  return {
    data: (res.data ?? []).map(mapToTransaction),
    totalCount: res.total_count,
    limit: res.limit,
    offset: res.offset,
  };
});
