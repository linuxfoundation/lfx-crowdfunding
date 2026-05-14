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

export type DonorType = 'individual' | 'company';

export interface DonateContactForm {
  donorType: DonorType;
  fullName: string;
  companyName: string;
  contactName: string;
  email: string;
  needsInvoice: boolean;
  poNumber: string;
}

export interface DonatePaymentForm {
  cardNumber: string;
  expiry: string;
  cvc: string;
}

export interface DonateSubmission {
  initiativeId: string;
  tierId: string | null;
  tierName: string | null;
  amountCents: number;
  contact: DonateContactForm;
  payment: DonatePaymentForm;
}

export interface DonationRecord extends DonateSubmission {
  id: string;
  createdAt: string;
}
