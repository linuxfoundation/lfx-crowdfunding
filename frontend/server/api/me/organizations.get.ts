// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { OrganizationResponse } from '../../types/organization.types';
import type { Organization } from '#shared/types/organization.types';

export default defineEventHandler(async (event): Promise<Organization[]> => {
  const { data } = await useBackendFetch<{ data: OrganizationResponse[] }>(
    event,
    '/v1/me/organizations',
  );
  return data.map((o) => ({
    id: o.id,
    name: o.name,
    avatarUrl: o.avatar_url,
    status: o.status,
  }));
});
