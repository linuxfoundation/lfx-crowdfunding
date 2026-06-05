// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendInitiative } from '../../types/initiatives.types';
import { mapToInitiativeDetail } from '../../services/initiatives.services';
import { useBackendFetch } from '../../utils/backend-fetch';

export default defineEventHandler(async (event) => {
  const slug = getRouterParam(event, 'slug');

  if (!slug) {
    throw createError({ statusCode: 400, message: 'Missing initiative slug' });
  }

  const initiative = await useBackendFetch<BackendInitiative>(
    event,
    `/v1/initiatives/${slug}`,
  ).catch((err) => {
    const status = err?.statusCode ?? err?.status;
    if (status === 404) throw createError({ statusCode: 404, message: 'Not found' });
    throw err;
  });

  return mapToInitiativeDetail(initiative);
});
