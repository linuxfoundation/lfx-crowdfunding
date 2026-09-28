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

  // Organization IDs are 18-char Salesforce Account SFIDs, not UUIDs; project
  // IDs are UUIDs. The backend (lfx-crowdfunding#263) validates
  // attribution.entity_uid shape per attribution type accordingly.
  return {
    organizations: [
      { id: '0012M00002qnukOQAQ', name: 'Sample Org — Acme Corp' },
      { id: '0012M00002qnukPQAQ', name: 'Sample Org — Globex' },
    ],
    projects: [
      { id: '0c1d2e3f-4a5b-4c6d-8e9f-0a1b2c3d4e5f', name: 'Sample Project — Kubernetes' },
      { id: '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d', name: 'Sample Foundation — CNCF' },
    ],
  };
};
