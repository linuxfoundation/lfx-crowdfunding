// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface FeaturedInitiative {
  id: string;
  name: string;
  logoUrl?: string;
  raisedCents: number;
  goalCents: number;
  supporterCount: number;
}

export interface FeaturedInitiativesResponse {
  data: FeaturedInitiative[];
}
