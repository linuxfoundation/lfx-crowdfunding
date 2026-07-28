// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { DocSectionsResponse } from '#shared/types/documentation.types';

export function useDocumentationNav() {
  const query = useQuery<DocSectionsResponse>({
    queryKey: ['docs', 'nav'] as const,
    queryFn: () => $fetch<DocSectionsResponse>('/api/docs'),
    staleTime: 5 * 60 * 1000,
  });

  onServerPrefetch(async () => {
    await query.suspense();
  });

  return query;
}
