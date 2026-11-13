// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Wire shapes of the Go backend's offline donation routes (#464).

export interface BackendOfflineDonation {
  id: string;
  initiative_id: string;
  initiative_name?: string;
  initiative_slug?: string;
  organization_name: string;
  organization_domain: string;
  amount_cents: number;
  category: string;
  received_on: string;
  source_txn_id: string;
  status: string;
  recorded_by?: string;
  created_on: string;
}

export interface BackendOfflineDonationList {
  data: BackendOfflineDonation[];
  meta: { total: number; limit: number; offset: number };
}

export interface BackendOfflineDonationInput {
  initiative_id: string;
  organization: { name: string; domain: string };
  amount: string;
  received_on: string;
  category: string;
  source_txn_id: string;
}
