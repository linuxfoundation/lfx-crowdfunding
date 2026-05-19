// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface StatisticsOverview {
  totalRaisedCents: number;
  supporterCount: number;
  annualGoalCents: number;
  activeInitiatives: number;
}

export interface FundingCategory {
  id: string;
  name: string;
  icon: string;
  raisedCents: number;
  goalCents: number;
  supporterCount: number;
}

export interface FundingByCategoryResponse {
  data: FundingCategory[];
}

export interface DonorBreakdown {
  avgDonationCents: number;
  organizationsCents: number;
  individualsCents: number;
}

export interface MonthlyBucket {
  year: number;
  month: number; // 1–12
  totalCents: number;
  supporters: number;
}

export interface MonthlyDonations {
  buckets: MonthlyBucket[];
}

export interface TopDonor {
  rank: number;
  id: string;
  name: string;
  logoUrl?: string;
  amountCents: number;
}

export interface TopDonorsResponse {
  organizations: TopDonor[];
  individuals: TopDonor[];
}

export type { RecentDonation } from './initiative-detail.types';

export interface RecentDonationsResponse {
  data: import('./initiative-detail.types').RecentDonation[];
}
