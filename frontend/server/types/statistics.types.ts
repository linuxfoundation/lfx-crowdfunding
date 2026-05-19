// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface BackendStatistics {
  total_raised_cents: number;
  total_supporters: number;
  total_initiatives: number;
}

export interface BackendCategoryTotal {
  name: string;
  total_cents: number;
  count: number;
}

export interface BackendSponsorEntry {
  id: string;
  name: string;
  avatar_url?: string;
  total_cents: number;
}

export interface BackendPlatformDetails {
  total_raised_cents: number;
  total_supporters: number;
  organizations_cents: number;
  individuals_cents: number;
  categories: BackendCategoryTotal[];
  top_organizations: BackendSponsorEntry[];
  top_individuals: BackendSponsorEntry[];
}

export interface BackendMonthlyBucket {
  year: number;
  month: number;
  total_cents: number;
  supporters: number;
}

export interface BackendPlatformMonthly {
  buckets: BackendMonthlyBucket[];
}

export interface BackendRecentDonation {
  txn_id: string;
  project_id: string;
  donor_name: string;
  donor_avatar_url?: string;
  donor_type: 'organization' | 'individual';
  amount_cents: number;
  txn_date: number;
  category?: string;
}

export interface BackendRecentDonationsResponse {
  data: BackendRecentDonation[];
}
