// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { OrganizationResponse } from '../../types/organization.types';
import type { Organization } from '#shared/types/organization.types';

interface CreateOrganizationBody {
  name: string;
  avatar_url?: string;
}

export default defineEventHandler(async (event): Promise<Organization> => {
  const body = await readBody<CreateOrganizationBody>(event);
  const raw = await useBackendFetch<OrganizationResponse>(event, '/v1/me/organizations', {
    method: 'POST',
    body,
  });
  return {
    id: raw.id,
    name: raw.name,
    avatarUrl: raw.avatar_url,
    status: raw.status,
  };
});
