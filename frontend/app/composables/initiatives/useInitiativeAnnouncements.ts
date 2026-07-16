// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { useQuery } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { onServerPrefetch, toValue } from 'vue';
import type { AnnouncementList } from '#shared/types/announcement.types';

export function useInitiativeAnnouncements(slug: MaybeRef<string>) {
  const query = useQuery<AnnouncementList>({
    queryKey: ['initiative-announcements', slug] as const,
    queryFn: () => $fetch<AnnouncementList>(`/api/initiatives/${toValue(slug)}/announcements`),
    enabled: computed(() => !!toValue(slug)),
  });

  onServerPrefetch(async () => {
    await query.suspense().catch(() => {});
  });

  return query;
}
