// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface FeaturedInitiative {
  id: string;
  slug: string;
  name: string;
  logoUrl?: string;
  raisedCents: number;
  goalCents: number;
  supporterCount: number;
}

export interface FeaturedInitiativesResponse {
  data: FeaturedInitiative[];
}

export interface InvestingCompany {
  id: string;
  name: string;
  logoUrl?: string;
  contributedCents: number;
}

export interface InvestingCompaniesResponse {
  data: InvestingCompany[];
}
