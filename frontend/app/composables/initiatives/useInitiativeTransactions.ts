// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { useQuery } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { toValue } from 'vue';
import type { TransactionList } from '#shared/types/transaction.types';

export function useInitiativeTransactions(id: MaybeRef<string>, size = 5) {
  return useQuery<TransactionList>({
    queryKey: ['initiative-transactions', id] as const,
    queryFn: () =>
      $fetch<TransactionList>(`/api/initiatives/${toValue(id)}/transactions`, {
        params: { type: 'donations', size },
      }),
    enabled: computed(() => !!toValue(id)),
  });
}
