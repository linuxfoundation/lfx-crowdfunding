// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendTransactionList } from '../../../types/transactions.types';
import { mapToTransaction } from '../../../services/transactions.services';
import type { TransactionList } from '#shared/types/transaction.types';

export default defineEventHandler(async (event): Promise<TransactionList> => {
  const id = getRouterParam(event, 'initiativeId');
  const { type, limit, offset } = getQuery(event);

  const { apiBaseUrl } = useRuntimeConfig();
  const params = new URLSearchParams();
  if (type) params.set('type', String(type));
  if (limit) params.set('limit', String(limit));
  if (offset) params.set('offset', String(offset));

  const res = await $fetch<BackendTransactionList>(
    `${apiBaseUrl}/v1/initiatives/${id}/transactions?${params}`,
  );

  return {
    data: (res.data ?? []).map(mapToTransaction),
    totalCount: res.total_count,
    limit: res.limit,
    offset: res.offset,
  };
});
