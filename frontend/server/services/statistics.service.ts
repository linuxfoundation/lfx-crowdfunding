// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type {
  BackendCategoryTotal,
  BackendMonthlyBucket,
  BackendRecentDonation,
  BackendSponsorEntry,
} from '../types/statistics.types';
import type { FundingCategory, MonthlyBucket, TopDonor } from '#shared/types/statistics.types';
import type { RecentDonation } from '#shared/types/initiative-detail.types';

export const mapToFundingCategory = (cat: BackendCategoryTotal): FundingCategory => ({
  id: cat.name,
  name: cat.name,
  icon: '',
  raisedCents: cat.total_cents,
  goalCents: 0,
  supporterCount: cat.count,
});

/**
 * Merges funding categories whose names match case-insensitively (e.g. "mentorship"
 * and "Mentorship") into one entry, summing amounts and supporters. The upstream
 * aggregation is case-sensitive, so the same category can arrive as multiple buckets.
 * The display name from the largest-raising variant is kept; results stay sorted by amount.
 */
export const mergeFundingCategoriesByName = (categories: FundingCategory[]): FundingCategory[] => {
  const merged = new Map<string, { cat: FundingCategory; topRaised: number }>();
  for (const cat of categories) {
    const key = cat.name.trim().toLowerCase();
    const existing = merged.get(key);
    if (!existing) {
      merged.set(key, { cat: { ...cat, id: key }, topRaised: cat.raisedCents });
      continue;
    }
    existing.cat.raisedCents += cat.raisedCents;
    existing.cat.supporterCount += cat.supporterCount;
    existing.cat.goalCents += cat.goalCents;
    if (cat.raisedCents > existing.topRaised) {
      existing.cat.name = cat.name;
      existing.topRaised = cat.raisedCents;
    }
  }
  return [...merged.values()].map((e) => e.cat).sort((a, b) => b.raisedCents - a.raisedCents);
};

export const mapToMonthlyBucket = (b: BackendMonthlyBucket): MonthlyBucket => ({
  year: b.year,
  month: b.month,
  totalCents: b.total_cents,
  supporters: b.supporters,
  newSupporters: b.new_supporters,
});

export const mapToRecentDonation = (d: BackendRecentDonation): RecentDonation => ({
  id: d.txn_id,
  donorName: d.donor_name,
  donorLogoUrl: d.donor_avatar_url,
  donorType: d.donor_type === 'organization' ? 'organization' : 'member',
  amountCents: d.amount_cents,
  date: d.txn_date,
  initiativeId: d.project_id,
  initiativeName: d.project_name,
});

export const mapToTopDonor = (donor: BackendSponsorEntry, index: number): TopDonor => ({
  rank: index + 1,
  id: donor.id,
  name: donor.name,
  logoUrl: donor.avatar_url,
  amountCents: donor.total_cents,
});
