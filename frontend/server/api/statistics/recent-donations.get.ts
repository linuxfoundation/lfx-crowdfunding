// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendRecentDonationsResponse } from '../../types/statistics.types';
import type { RecentDonationsResponse } from '#shared/types/statistics.types';
import type { RecentDonation } from '#shared/types/initiative-detail.types';

function timeAgo(unixSeconds: number): string {
  const diffSec = Math.max(0, Math.floor(Date.now() / 1000) - unixSeconds);
  if (diffSec < 60) return `${diffSec}s ago`;
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
}

export default defineEventHandler(async (): Promise<RecentDonationsResponse> => {
  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  try {
    const res = await $fetch<BackendRecentDonationsResponse>(
      `${apiBase}/v1/statistics/recent-donations`,
    );
    const data: RecentDonation[] = (res.data ?? []).map((d) => ({
      id: d.txn_id,
      donorName: d.donor_name,
      donorLogoUrl: d.donor_avatar_url,
      donorType: d.donor_type === 'organization' ? 'organization' : 'member',
      amountCents: d.amount_cents,
      timeAgo: timeAgo(d.txn_date),
      initiativeId: d.project_id,
      initiativeName: d.project_name,
    }));
    return { data };
  } catch {
    return { data: [] };
  }
});
