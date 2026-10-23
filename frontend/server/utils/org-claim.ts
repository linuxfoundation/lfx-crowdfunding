// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { OrgClaimWire } from '../types/payment.types';
import type { OrgClaim } from '#shared/types/organization.types';

/** Maps the donor's org claim to the backend wire shape for donation and subscription create (#409). */
export const toOrgClaimBody = (org: OrgClaim | undefined): { organization?: OrgClaimWire } =>
  org ? { organization: { name: org.name, domain: org.domain } } : {};
