// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendInitiative } from '../types/initiatives.types';
import type { InitiativeBase } from '#shared/types/initiative.types';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';
import type { SponsorshipTier } from '#shared/types/donate.types';

// TODO: no sponsorship tier data from the backend yet — mocked here until initiatives
// can define their own tiers. Hidden in production via the NUXT_APP_ENV check below.
const MOCK_SPONSORSHIP_TIERS: SponsorshipTier[] = [
  {
    id: 'bronze',
    name: 'Bronze',
    amountCents: 50_000,
    benefits: ['Name on supporters page', 'Quarterly newsletter'],
  },
  {
    id: 'silver',
    name: 'Silver',
    amountCents: 500_000,
    benefits: ['Bronze benefits', 'Logo on project page', 'Early access to audit reports'],
  },
  {
    id: 'gold',
    name: 'Gold',
    amountCents: 2_500_000,
    benefits: [
      'Silver benefits',
      'Logo on homepage',
      'Direct access to audit team',
      'Custom briefing',
    ],
  },
  {
    id: 'platinum',
    name: 'Platinum',
    amountCents: 10_000_000,
    benefits: [
      'Gold benefits',
      'Advisory board seat',
      'Co-branded announcements',
      'Executive briefing',
    ],
  },
];

export const mapToInitiativeBase = (b: BackendInitiative): InitiativeBase => {
  return {
    id: b.id,
    slug: b.slug,
    name: b.name,
    description: b.description ?? '',
    status: b.status,
    initiativeType: b.initiative_type,
    color: b.color ?? '',
    createdOn: b.created_on,
    updatedOn: b.updated_on,
    industry: b.industry,
    logoUrl: b.logo_url,
    country: b.country,
    city: b.city,
    websiteURL: b.website_url,
    applicationURL: b.application_url,
    eventStartDate: b.event_start_date,
    eventEndDate: b.event_end_date,
    acceptFunding: b.accept_funding,
    fundingStatus: b.financials
      ? {
          goalsTotalCents: b.financials.goals_total_cents,
          amountRaisedCents: b.financials.total_raised_cents,
        }
      : undefined,
    initiativeStats: b.financials ? { supporters: b.financials.supporters } : undefined,
  };
};

export const mapToInitiativeDetail = (b: BackendInitiative): InitiativeDetail => {
  const currentBalanceCents = b.balance?.available_cents ?? b.financials?.available_balance_cents;
  const githubURL = (b.custom_websites ?? []).find((w) =>
    ['repository', 'github'].includes((w.name ?? '').toLowerCase()),
  )?.url;

  return {
    ...mapToInitiativeBase(b),
    currentBalanceCents,
    githubURL,
    fundingGoals: (b.goals ?? []).map((g) => ({
      id: g.id,
      name: g.name,
      goalCents: g.goal_amount_cents,
      donatedCents: g.donated_cents ?? 0,
      spentCents: g.spent_cents ?? 0,
    })),
    financialSummary: b.financials
      ? {
          totalReceivedCents: b.financials.total_raised_cents,
          totalExpensesCents: b.financials.total_disbursed_cents ?? 0,
          balanceCents: b.financials.available_balance_cents ?? 0,
        }
      : undefined,
    sponsors: (b.sponsors ?? []).map((s) => ({
      id: s.id,
      name: s.name,
      avatarUrl: s.avatar_url,
      totalCents: s.total_cents,
    })),
    recentDonations: [],
    donationRecords: [],
    expenseRecords: [],
    sponsorshipTiers:
      process.env.NUXT_APP_ENV !== 'production' ? MOCK_SPONSORSHIP_TIERS : undefined,
  };
};
