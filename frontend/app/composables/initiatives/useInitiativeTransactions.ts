// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { useQuery } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { toValue } from 'vue';
import type { TransactionList } from '#shared/types/transaction.types';

export function useInitiativeTransactions(
  id: MaybeRef<string>,
  type: 'donations' | 'expenses' = 'donations',
  limit = 5,
  offset = 0,
) {
  return useQuery<TransactionList>({
    queryKey: ['initiative-transactions', id, type, limit, offset] as const,
    queryFn: () =>
      $fetch<TransactionList>(`/api/initiatives/${toValue(id)}/transactions`, {
        params: { type, limit, offset },
      }),
    enabled: computed(() => !!toValue(id)),
  });
}
