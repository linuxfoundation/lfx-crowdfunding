// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendInitiative } from '../../types/initiatives.types';
import { mapToInitiativeDetail } from '../../services/initiatives.services';
import { useBackendFetch } from '../../utils/backend-fetch';

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id');

  if (!id) {
    throw createError({ statusCode: 400, message: 'Missing initiative id' });
  }

  const initiative = await useBackendFetch<BackendInitiative>(event, `/v1/initiatives/${id}`).catch(
    (err) => {
      if (err?.status === 404) throw createError({ statusCode: 404, message: 'Not found' });
      throw err;
    },
  );

  return mapToInitiativeDetail(initiative);
});
