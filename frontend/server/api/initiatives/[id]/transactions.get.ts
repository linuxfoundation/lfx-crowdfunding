// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendTransactionList } from '../../../types/transactions.types';
import { mapToTransaction } from '../../../services/transactions.services';
import type { TransactionList } from '#shared/types/transaction.types';

export default defineEventHandler(async (event): Promise<TransactionList> => {
  const id = getRouterParam(event, 'id');
  const { type, size, from } = getQuery(event);

  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  const params = new URLSearchParams();
  if (type) params.set('type', String(type));
  if (size) params.set('size', String(size));
  if (from) params.set('from', String(from));

  const res = await $fetch<BackendTransactionList>(
    `${apiBase}/v1/initiatives/${id}/transactions?${params}`,
  );

  return {
    data: (res.data ?? []).map(mapToTransaction),
    totalCount: res.total_count,
    from: res.from,
    size: res.size,
  };
});
