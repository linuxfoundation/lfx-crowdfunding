// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { Goal, SubscribableDraft, SubscribableEdit } from './subscription.types';
import type { ExpenseCategory } from './transaction.types';
import type { EventSponsorship } from './event-sponsorship.types';
import type { ProjectFundingStatus } from './project.types';

export interface EntityStats {
  backers: number;
  sponsors: number;
  totalRaised: number;
}

export interface EntityBadge {
  amount: number;
  allocation: string;
}

export interface FundingStatus {
  totalAnnualGoalInCents: number;
  annualSubscriptionAmountInCents?: number;
  annualSubscriptionRemainingAmountInCents?: number;
  totalDonationsInCents?: number;
  totalSubscriptionCount?: number;
}

export interface EntityGoal extends Goal {
  name: string;
  description?: string;
  fundingStatus?: FundingStatus;
  goalIcon?: File | string;
  goalColor?: string;
  errors?: any;
}

export interface Entity {
  id: string;
  industry?: string;
  entityId: string;
  ownerId: string;
  cocURL?: string;
  name: string;
  status: string;
  entityType: string;
  description: string;
  createdOn: string;
  updatedOn: string;
  color: string;
  logoUrl?: string;
  country?: string;
  city?: string;
  entityStats?: EntityStats;
  fundingStatus?: FundingStatus;
  entityDetails?: any;
  goals?: EntityGoal[];
  sponsors?: any[];
  sponsorshipTiers?: EventSponsorship[];
  websiteURL?: string;
  applicationURL?: string;
  customWebsites?: any[];
  eventbriteId?: string;
  balance?: any;
  beneficiaries?: any[];
  eventStartDate?: string;
  eventEndDate?: string;
  detail?: any;
  amountRaised?: number;
}

export interface EntityBacker {
  name: string;
  avatarURL: string;
  backerSince: string;
  amountInCents: string;
}

export interface EntityBackerResponse {
  entries: EntityBacker[];
  link: any;
  totalRecords: number;
}

export interface EntitySubscription {
  entityId: string;
  createdOn: string;
  amountInCents: number;
  industry: string;
  name: string;
  color: string;
  description: string;
  orgId?: string;
  logoUrl: string;
  category?: ExpenseCategory;
  fundingStatus?: ProjectFundingStatus;
  eventStartDate?: string;
  eventEndDate?: string;
  entityType: string;
  ciiProjectID?: string;
  mentee?: any;
}

export interface Beneficiary {
  name: string;
  email: string;
}

export interface DraftEntity extends SubscribableDraft {
  beneficiaries: Beneficiary[];
  entityType: string | null;
  fundingStatus: FundingStatus;
  cocURL?: string;
  ciiProjectID?: string;
  goals?: EntityGoal[];
  sponsorshipTiers?: EventSponsorship[];
  websiteURL?: string;
  applicationURL?: string;
  eventbriteId?: string;
  eventStartDate?: string;
  eventEndDate?: string;
  detail?: any;
}

export interface EntityEdit extends SubscribableEdit {
  beneficiaries?: Beneficiary[];
  entityType?: string | null;
  fundingStatus?: FundingStatus;
  cocURL?: string;
  ciiProjectID?: string;
  goals?: EntityGoal[];
  sponsorshipTiers?: EventSponsorship[];
  websiteURL?: string;
  eventbriteId?: string;
  eventStartDate?: string;
  eventEndDate?: string;
  detail?: any;
}
