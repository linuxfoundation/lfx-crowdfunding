// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { onServerPrefetch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import type { MaybeRefOrGetter } from 'vue';
import type { DocArticle } from '#shared/types/documentation.types';

export function useDocumentation(slug: MaybeRefOrGetter<string>) {
  const slugRef = toRef(slug);

  const query = useQuery<DocArticle>({
    queryKey: ['docs', 'article', slugRef] as const,
    queryFn: () => $fetch<DocArticle>(`/api/docs/${slugRef.value}`),
    staleTime: 5 * 60 * 1000,
    retry: false,
  });

  onServerPrefetch(async () => {
    await query.suspense();
  });

  return query;
}
