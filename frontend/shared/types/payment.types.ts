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
  donationTier?: string;
}

export interface DonationResult {
  id: string;
  status: string;
  clientSecret?: string;
}

export type SubscriptionFrequency = 'monthly' | 'yearly' | 'weekly' | 'daily';

export interface SubscriptionRequest {
  amountInCents: number;
  frequency: SubscriptionFrequency;
  stripePaymentMethodId: string;
  category?: string;
  organizationId?: string;
}

export interface SubscriptionResult {
  id: string;
  status: string;
  clientSecret?: string;
}
