// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface CardDetails {
  paymentMethodId: string;
  lastFour: string;
  brand: string;
  expiryMonth: number;
  expiryYear: number;
}

export interface SetupIntentResult {
  clientSecret: string;
}

export interface DonationRequest {
  amountInCents: number;
  stripePaymentMethodId: string;
  category?: string;
  organizationId?: string;
}

export interface DonationResult {
  id: string;
  status: string;
  clientSecret?: string;
}
