// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendTransaction } from '../types/transactions.types';
import type { Transaction } from '#shared/types/transaction.types';

export const mapToTransaction = (b: BackendTransaction): Transaction => ({
  id: b.id,
  type: b.type as Transaction['type'],
  amountCents: b.amount_cents,
  date: b.date,
  category: b.category,
  donorName: b.donor_name,
  donorType: b.donor_type as Transaction['donorType'],
  donorLogoUrl: b.donor_logo_url,
  donorUsername: b.donor_username,
});
