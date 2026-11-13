// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { computed } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { ApproverStatus } from '#shared/types/offline-donation.types';

/** Whether the signed-in user is an approver, to show approver-only actions (#464). */
export function useIsApprover() {
  const { isAuthenticated } = useAuth();
  const query = useQuery<ApproverStatus>({
    queryKey: ['me', 'approver'] as const,
    queryFn: () => $fetch<ApproverStatus>('/api/me/approver'),
    enabled: computed(() => isAuthenticated.value),
    staleTime: 10 * 60_000,
  });
  const isApprover = computed(() => isAuthenticated.value && query.data.value?.approver === true);
  return { ...query, isApprover };
}
