// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { createError, type H3Event } from 'h3';
import type { OrganizationResponse } from '../types/organization.types';
import type { OrgClaimWire } from '../types/payment.types';
import { useBackendFetch } from './backend-fetch';

/**
 * Resolves the donor's selected per-user org to the `{ name }` org claim the
 * backend now stores on the donation (#405). Only the caller's own orgs resolve,
 * so the old ownership check still holds for the per-user picker.
 * ponytail: bridge until the org search picker sends name and domain itself (#409).
 */
export async function resolveOrgClaim(
  event: H3Event,
  organizationId: string | undefined,
): Promise<{ organization?: OrgClaimWire }> {
  if (!organizationId) return {};
  const { data } = await useBackendFetch<{ data: OrganizationResponse[] }>(
    event,
    '/crowdfunding/me/organizations',
  );
  const org = data.find((o) => o.id === organizationId);
  if (!org) {
    throw createError({ statusCode: 404, statusMessage: 'Organization not found' });
  }
  return { organization: { name: org.name } };
}
