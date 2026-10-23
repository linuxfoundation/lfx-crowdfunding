// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { OrgClaim } from './organization.types';

export interface SponsorshipTier {
  id: string;
  name: string;
  enabled: boolean;
  amountCents: number;
  benefits: string[];
}

export type DonationType = 'one-time' | 'monthly';

export interface DonateAmountForm {
  tierId: string | null;
  tierName: string | null;
  customAmountCents: number | null;
  amountCents: number;
  donationType: DonationType;
  category: string | null;
}

export type DonorType = 'individual' | 'organization';

export interface DonateContactForm {
  donorType: DonorType;
  organization: OrgClaim | null;
}

export interface DonateSubmission {
  initiativeId: string;
  tierId: string | null;
  tierName: string | null;
  amountCents: number;
  contact: DonateContactForm;
  paymentMethodId: string;
}

export interface DonationRecord extends DonateSubmission {
  id: string;
  createdAt: string;
}
