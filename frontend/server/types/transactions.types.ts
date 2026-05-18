// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface BackendTransaction {
  id: string;
  type: string;
  amount_cents: number;
  date: string;
  category?: string;
  donor_name?: string;
  donor_type?: string;
  donor_logo_url?: string;
  donor_username?: string;
}

export interface BackendTransactionList {
  data: BackendTransaction[];
  total_count: number;
  from: number;
  size: number;
}
