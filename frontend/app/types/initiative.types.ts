// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { Goal, SubscribableDraft, SubscribableEdit } from './subscription.types';
import type { ExpenseCategory } from './transaction.types';
import type { EventSponsorship } from './event-sponsorship.types';
import type { ProjectFundingStatus } from './project.types';

export interface InitiativeStats {
  backers: number;
  sponsors: number;
  totalRaised: number;
}

export interface InitiativeBadge {
  amount: number;
  allocation: string;
}

export interface FundingStatus {
  totalAnnualGoalInCents: number;
  annualSubscriptionAmountInCents?: number;
  annualSubscriptionRemainingAmountInCents?: number;
  amountRaisedCents?: number;
  totalSubscriptionCount?: number;
}

export interface InitiativeGoal extends Goal {
  name: string;
  description?: string;
  fundingStatus?: FundingStatus;
  goalIcon?: File | string;
  goalColor?: string;
  errors?: any;
}

export interface Initiative {
  id: string;
  industry?: string;
  initiativeId: string;
  ownerId: string;
  cocURL?: string;
  name: string;
  status: string;
  initiativeType: string;
  description: string;
  createdOn: string;
  updatedOn: string;
  color: string;
  logoUrl?: string;
  country?: string;
  city?: string;
  initiativeStats?: InitiativeStats;
  fundingStatus?: FundingStatus;
  initiativeDetails?: any;
  goals?: InitiativeGoal[];
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

export interface InitiativeBacker {
  name: string;
  avatarURL: string;
  backerSince: string;
  amountInCents: string;
}

export interface InitiativeBackerResponse {
  entries: InitiativeBacker[];
  link: any;
  totalRecords: number;
}

export interface InitiativeSubscription {
  initiativeId: string;
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
  initiativeType: string;
  ciiProjectID?: string;
  mentee?: any;
}

export interface Beneficiary {
  name: string;
  email: string;
}

export interface DraftInitiative extends SubscribableDraft {
  beneficiaries: Beneficiary[];
  initiativeType: string | null;
  fundingStatus: FundingStatus;
  cocURL?: string;
  ciiProjectID?: string;
  goals?: InitiativeGoal[];
  sponsorshipTiers?: EventSponsorship[];
  websiteURL?: string;
  applicationURL?: string;
  eventbriteId?: string;
  eventStartDate?: string;
  eventEndDate?: string;
  detail?: any;
}

export interface InitiativeEdit extends SubscribableEdit {
  beneficiaries?: Beneficiary[];
  initiativeType?: string | null;
  fundingStatus?: FundingStatus;
  cocURL?: string;
  ciiProjectID?: string;
  goals?: InitiativeGoal[];
  sponsorshipTiers?: EventSponsorship[];
  websiteURL?: string;
  eventbriteId?: string;
  eventStartDate?: string;
  eventEndDate?: string;
  detail?: any;
}
