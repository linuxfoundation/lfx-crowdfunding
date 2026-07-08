// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendInitiative } from '../../../types/initiatives.types';
import type { BackendAnnouncementList } from '../../../types/announcements.types';
import type { AnnouncementList } from '#shared/types/announcement.types';

export default defineEventHandler(async (event): Promise<AnnouncementList> => {
  const slug = getRouterParam(event, 'slug');
  const { limit, offset } = getQuery(event);
  const { apiBaseUrl } = useRuntimeConfig();

  const initiative = await $fetch<BackendInitiative>(`${apiBaseUrl}/v1/initiatives/${slug}`).catch(
    () => {
      throw createError({ statusCode: 404, statusMessage: 'Initiative not found' });
    },
  );

  const params = new URLSearchParams();
  if (limit) params.set('limit', String(limit));
  if (offset) params.set('offset', String(offset));
  const qs = params.toString() ? `?${params}` : '';

  const raw = await $fetch<BackendAnnouncementList>(
    `${apiBaseUrl}/v1/initiatives/${initiative.id}/announcements${qs}`,
  );

  return {
    data: raw.data.map((a) => ({
      id: a.id,
      initiativeId: a.initiative_id,
      createdBy: a.created_by,
      title: a.title,
      description: a.description,
      createdOn: a.created_on,
      updatedOn: a.updated_on,
    })),
    totalCount: raw.meta.total,
  };
});
