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
