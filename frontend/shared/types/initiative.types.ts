// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
//
// Types in this file are accessible to both the Nuxt app (~/types) and the
// Nitro server layer (server/api). Keep this file free of app-only imports
// (Vue, vue-query, browser APIs, etc.).
import type { Pagination } from './pagination';

export interface InitiativeStats {
  supporters: number;
}

export interface FundingStatus {
  goalsTotalCents: number;
  annualSubscriptionAmountInCents?: number;
  annualSubscriptionRemainingAmountInCents?: number;
  amountRaisedCents?: number;
  totalSubscriptionCount?: number;
}

/** Core initiative fields constructed and returned by the server. */
export interface InitiativeBase {
  id: string;
  slug: string;
  name: string;
  description: string;
  status: string;
  initiativeType: string;
  color: string;
  createdOn: string;
  updatedOn: string;
  industry?: string;
  logoUrl?: string;
  country?: string;
  city?: string;
  websiteURL?: string;
  applicationURL?: string;
  eventStartDate?: string;
  eventEndDate?: string;
  acceptFunding: boolean;
  initiativeStats?: InitiativeStats;
  fundingStatus?: FundingStatus;
}

export interface InitiativesParams {
  search?: string;
  type?: string;
  sort?: string;
  page?: number;
  pageSize?: number;
}

export type { Pagination };
export type InitiativesResponse = Pagination<InitiativeBase>;
