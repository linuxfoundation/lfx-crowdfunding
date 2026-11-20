// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

export interface Transaction {
  id: string;
  /** `return`: unused funds sent back to the donor org (#489), listed with expenses. */
  type: 'donation' | 'reimbursement' | 'return';
  amountCents: number;
  date: string;
  category?: string;
  donorName?: string;
  donorType?: 'organization' | 'individual';
  donorLogoUrl?: string;
  donorUsername?: string;
  /** The approver's reason on a return or payout LF recorded (#489). */
  comment?: string;
}

export interface TransactionList {
  data: Transaction[];
  totalCount: number;
  limit: number;
  offset: number;
}
