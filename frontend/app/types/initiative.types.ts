// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { Goal } from './subscription.types';
import type { EventSponsorship } from './event-sponsorship.types';

export type {
  InitiativeStats,
  FundingStatus,
  InitiativeBase,
  InitiativesParams,
  InitiativesResponse,
} from '#shared/types/initiative.types';

import type { InitiativeBase, FundingStatus } from '#shared/types/initiative.types';

export interface InitiativeGoal extends Goal {
  name: string;
  description?: string;
  fundingStatus?: FundingStatus;
  goalIcon?: File | string;
  goalColor?: string;
}

/** Full initiative shape used in the app — extends the server-safe base type. */
export interface Initiative extends InitiativeBase {
  cocURL?: string;
  initiativeDetails?: string;
  goals?: InitiativeGoal[];
  sponsors?: string[];
  sponsorshipTiers?: EventSponsorship[];
  customWebsites?: string[];
  eventbriteId?: string;
  balance?: string;
  beneficiaries?: { name: string; email: string }[];
  detail?: string;
  amountRaised?: number;
}
