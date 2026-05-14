// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface SponsorshipTier {
  id: string;
  name: string;
  amountCents: number;
  benefits: string[];
}

export interface DonateAmountForm {
  tierId: string | null;
  tierName: string | null;
  customAmountCents: number | null;
  amountCents: number;
}

export interface DonateSubmission {
  initiativeId: string;
  tierId: string | null;
  tierName: string | null;
  amountCents: number;
}

export interface DonationRecord extends DonateSubmission {
  id: string;
  createdAt: string;
}
