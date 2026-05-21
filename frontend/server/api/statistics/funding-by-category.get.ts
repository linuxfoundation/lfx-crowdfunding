// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendPlatformDetails } from '../../types/statistics.types';
import type { FundingByCategoryResponse, FundingCategory } from '#shared/types/statistics.types';

const CATEGORY_ICONS: Record<string, string> = {
  'security audit': 'shield',
  security: 'shield',
  infrastructure: 'server',
  community: 'users',
  events: 'calendar',
  travel: 'plane',
  mentorship: 'graduation-cap',
  'general fund': 'hand-holding-dollar',
  other: 'circle-dot',
  uncategorised: 'circle-question',
  uncategorized: 'circle-question',
};

function categoryIcon(name: string): string {
  return CATEGORY_ICONS[name.toLowerCase()] ?? 'tag';
}

export default defineEventHandler(async (): Promise<FundingByCategoryResponse> => {
  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  try {
    const res = await $fetch<BackendPlatformDetails>(`${apiBase}/v1/statistics/platform`);
    const data: FundingCategory[] = (res.categories ?? []).map((cat) => ({
      id: cat.name,
      name: cat.name,
      icon: categoryIcon(cat.name),
      raisedCents: cat.total_cents,
      goalCents: 0,
      supporterCount: cat.count,
    }));
    return { data };
  } catch {
    return { data: [] };
  }
});
