// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { FundingGoal, InitiativeDetail } from '#shared/types/initiative-detail.types';

interface BackendGoal {
  id: string;
  name: string;
  goal_amount_cents: number;
  description?: string;
  donated_cents?: number;
  spent_cents?: number;
}

interface BackendSponsor {
  id: string;
  name: string;
  avatar_url?: string;
  total_cents: number;
}

interface BackendInitiative {
  id: string;
  initiative_type: string;
  owner_id: string;
  name: string;
  slug: string;
  status: string;
  industry?: string;
  description?: string;
  color?: string;
  logo_url?: string;
  website_url?: string;
  country?: string;
  city?: string;
  application_url?: string;
  event_start_date?: string;
  event_end_date?: string;
  created_on: string;
  updated_on: string;
  goals?: BackendGoal[];
  financials?: {
    total_raised_cents: number;
    total_disbursed_cents: number;
    available_balance_cents: number;
    supporters: number;
    goals_total_cents: number;
  };
  balance?: {
    total_raised_cents: number;
    total_disbursed_cents: number;
    available_cents: number;
  };
  sponsors?: BackendSponsor[];
}

function toInitiativeDetail(b: BackendInitiative): InitiativeDetail {
  const fundingGoals: FundingGoal[] = (b.goals ?? []).map((g) => ({
    id: g.id,
    name: g.name,
    goalCents: g.goal_amount_cents,
    donatedCents: g.donated_cents ?? 0,
    spentCents: g.spent_cents ?? 0,
  }));

  const currentBalanceCents = b.balance?.available_cents ?? b.financials?.available_balance_cents;

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
    websiteURL: b.website_url,
    country: b.country,
    city: b.city,
    applicationURL: b.application_url,
    eventStartDate: b.event_start_date,
    eventEndDate: b.event_end_date,
    currentBalanceCents,
    fundingGoals,
    financialSummary: b.financials
      ? {
          totalReceivedCents: b.financials.total_raised_cents,
          totalExpensesCents: b.financials.total_disbursed_cents,
          balanceCents: b.financials.available_balance_cents,
        }
      : undefined,
    fundingStatus: b.financials
      ? {
          goalsTotalCents: b.financials.goals_total_cents,
          amountRaisedCents: b.financials.total_raised_cents,
        }
      : undefined,
    initiativeStats: b.financials ? { supporters: b.financials.supporters } : undefined,
    sponsors: (b.sponsors ?? []).map((s) => ({
      id: s.id,
      name: s.name,
      avatarUrl: s.avatar_url,
      totalCents: s.total_cents,
    })),
    recentDonations: [],
    donationRecords: [],
    expenseRecords: [],
  };
}

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id');

  if (!id) {
    throw createError({ statusCode: 400, message: 'Missing initiative id' });
  }

  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  const initiative = await $fetch<BackendInitiative>(`${apiBase}/v1/initiatives/${id}`).catch(
    (err) => {
      if (err?.status === 404) throw createError({ statusCode: 404, message: 'Not found' });
      throw err;
    },
  );

  return toInitiativeDetail(initiative);
});
