// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface Transaction {
  id: string;
  type: 'donation' | 'reimbursement';
  amountCents: number;
  date: string;
  category?: string;
  donorName?: string;
  donorType?: 'organization' | 'individual';
  donorLogoUrl?: string;
  donorUsername?: string;
}

export interface TransactionList {
  data: Transaction[];
  totalCount: number;
  from: number;
  size: number;
}
