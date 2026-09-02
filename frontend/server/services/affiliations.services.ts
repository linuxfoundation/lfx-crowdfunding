// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';
import type { AffiliationCandidates } from '#shared/types/affiliation.types';

const isProduction = process.env.NUXT_PUBLIC_APP_ENV === 'production';

// ponytail: stubbed candidate source — platform entity enumeration is an open,
// blocking decision (epic LFXV2-2759, "entity enumeration from outside Heimdall").
// This is the single seam the fundraise-form attribution step depends on; swap
// this function's body for a real platform lookup once that decision lands and
// nothing else in the frontend needs to change.
export const getAffiliations = async (_event: H3Event): Promise<AffiliationCandidates> => {
  if (isProduction) return { organizations: [], projects: [] };

  // IDs are UUIDs — the backend (LFXV2-2956) validates attribution.entity_uid as one.
  return {
    organizations: [
      { id: '8b1e2c3d-4f56-4a78-9b0c-1d2e3f4a5b6c', name: 'Sample Org — Acme Corp' },
      { id: '3f4a5b6c-7d8e-4f90-a1b2-c3d4e5f60718', name: 'Sample Org — Globex' },
    ],
    projects: [
      { id: '0c1d2e3f-4a5b-4c6d-8e9f-0a1b2c3d4e5f', name: 'Sample Project — Kubernetes' },
      { id: '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d', name: 'Sample Foundation — CNCF' },
    ],
  };
};
