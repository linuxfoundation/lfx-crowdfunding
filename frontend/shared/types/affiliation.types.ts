// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface AffiliationEntity {
  id: string;
  name: string;
  logoUrl?: string;
}

export interface AffiliationCandidates {
  organizations: AffiliationEntity[];
  projects: AffiliationEntity[];
}
