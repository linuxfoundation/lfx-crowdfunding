// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { useInfiniteQuery } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { computed, onServerPrefetch, toValue } from 'vue';
import type { TransactionList } from '#shared/types/transaction.types';

export function useInitiativeTransactions(
  slug: MaybeRef<string>,
  type: 'donations' | 'expenses' = 'donations',
  limit = 5,
) {
  const query = useInfiniteQuery<TransactionList>({
    queryKey: ['initiative-transactions', slug, type, limit] as const,
    queryFn: ({ pageParam }) =>
      $fetch<TransactionList>(`/api/initiatives/${toValue(slug)}/transactions`, {
        params: { type, limit, offset: pageParam },
      }),
    initialPageParam: 0,
    getNextPageParam: (lastPage) => {
      const nextOffset = lastPage.offset + lastPage.limit;
      return nextOffset < lastPage.totalCount ? nextOffset : undefined;
    },
    enabled: computed(() => !!toValue(slug)),
  });

  onServerPrefetch(async () => {
    await query.suspense().catch(() => {});
  });

  return query;
}
