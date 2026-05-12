// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { InitiativeBase } from './initiative.types';

export interface SponsorEntry {
  name: string;
  logoUrl?: string;
}

export interface RecentDonation {
  id: string;
  donorName: string;
  donorLogoUrl?: string;
  donorType: 'organization' | 'member';
  amountCents: number;
  timeAgo: string;
  initiativeId?: string;
  initiativeName?: string;
}

export interface FundingGoal {
  id: string;
  name: string;
  donatedCents: number;
  spentCents: number;
  goalCents: number;
}

export interface FinancialSummary {
  totalReceivedCents: number;
  totalExpensesCents: number;
  balanceCents: number;
}

export interface DonationRecord {
  id: string;
  date: string;
  supporterName: string;
  supporterLogoUrl?: string;
  supporterType: 'organization' | 'member';
  donorCategory: 'Company' | 'Individual';
  amountCents: number;
}

export interface ExpenseRecord {
  id: string;
  date: string;
  category: string;
  description: string;
  amountCents: number;
}

export interface ImpactStat {
  value: string;
  label: string;
}

export interface ProjectHealthStat {
  icon: string;
  label: string;
  value: string;
}

export interface InitiativeDetail extends InitiativeBase {
  websiteURL?: string;
  githubURL?: string;
  currentBalanceCents?: number;
  sponsors?: SponsorEntry[];
  recentDonations?: RecentDonation[];
  impactStats?: ImpactStat[];
  projectHealthStats?: ProjectHealthStat[];
  projectHealthRating?: string;
  fundingGoals?: FundingGoal[];
  financialSummary?: FinancialSummary;
  donationRecords?: DonationRecord[];
  expenseRecords?: ExpenseRecord[];
}
