// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { Transaction, TransactionList } from '#shared/types/transaction.types';

interface BackendTransaction {
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

interface BackendTransactionList {
  data: BackendTransaction[];
  total_count: number;
  from: number;
  size: number;
}

function toTransaction(b: BackendTransaction): Transaction {
  return {
    id: b.id,
    type: b.type as Transaction['type'],
    amountCents: b.amount_cents,
    date: b.date,
    category: b.category,
    donorName: b.donor_name,
    donorType: b.donor_type as Transaction['donorType'],
    donorLogoUrl: b.donor_logo_url,
    donorUsername: b.donor_username,
  };
}

export default defineEventHandler(async (event): Promise<TransactionList> => {
  const id = getRouterParam(event, 'id');
  const { type, size, from } = getQuery(event);

  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  const params = new URLSearchParams();
  if (type) params.set('type', String(type));
  if (size) params.set('size', String(size));
  if (from) params.set('from', String(from));

  const res = await $fetch<BackendTransactionList>(
    `${apiBase}/v1/initiatives/${id}/transactions?${params}`,
  );

  return {
    data: (res.data ?? []).map(toTransaction),
    totalCount: res.total_count,
    from: res.from,
    size: res.size,
  };
});
