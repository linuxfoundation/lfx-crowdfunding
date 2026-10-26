// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { createError } from 'h3';
import type { OrgClaimWire } from '../types/payment.types';
import type { OrgClaim } from '#shared/types/organization.types';

/** Maps the donor's org claim to the backend wire shape for donation and subscription create (#409). */
export const toOrgClaimBody = (org: OrgClaim | undefined): { organization?: OrgClaimWire } =>
  org ? { organization: { name: org.name, domain: org.domain } } : {};

/**
 * A drawer loaded before #409 still sends `organizationId`, which the routes no longer
 * forward; reject it so the donation isn't silently recorded as individual.
 * ponytail: drop with the backend organization_id bridge.
 */
export const rejectLegacyOrgId = (body: object): void => {
  if (
    'organizationId' in body &&
    body.organizationId &&
    !('organization' in body && body.organization)
  ) {
    throw createError({
      statusCode: 409,
      statusMessage: 'Please reload the page and select your organization again',
    });
  }
};
