// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface CardDetailsWire {
  payment_method_id: string;
  last_four: string;
  brand: string;
  expiry_month: number;
  expiry_year: number;
}

export interface SetupIntentWire {
  client_secret: string;
}

export interface DonationResultWire {
  id: string;
  status: string;
  client_secret?: string;
  stripe_payment_intent_id?: string;
}
