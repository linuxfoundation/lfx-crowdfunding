// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface CardDetails {
  payment_method_id: string;
  last_four: string;
  brand: string;
  expiry_month: number;
  expiry_year: number;
}

export interface SetupIntentResult {
  client_secret: string;
}

export interface DonationRequest {
  amount_in_cents: number;
  stripe_payment_method_id: string;
  category?: string;
  organization_id?: string;
}

export interface DonationResult {
  id: string;
  status: string;
  client_secret?: string;
  stripe_payment_intent_id?: string;
}
