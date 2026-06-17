// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getRouterParam } from 'h3';
import { useBackendFetch } from '../../../utils/backend-fetch';

export default defineEventHandler(async (event): Promise<void> => {
  const id = getRouterParam(event, 'id');
  await useBackendFetch(event, `/v1/me/organizations/${id}`, { method: 'DELETE' });
});
