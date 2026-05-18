// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendInitiative } from '../../types/initiatives.types';
import { mapToInitiativeDetail } from '../../services/initiatives.services';

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id');

  if (!id) {
    throw createError({ statusCode: 400, message: 'Missing initiative id' });
  }

  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  const initiative = await $fetch<BackendInitiative>(`${apiBase}/v1/initiatives/${id}`).catch(
    (err) => {
      if (err?.status === 404) throw createError({ statusCode: 404, message: 'Not found' });
      throw err;
    },
  );

  return mapToInitiativeDetail(initiative);
});
