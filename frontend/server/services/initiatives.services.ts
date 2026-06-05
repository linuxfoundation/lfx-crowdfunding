// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { BackendInitiative } from '../types/initiatives.types';
import type { InitiativeBase } from '#shared/types/initiative.types';
import type { InitiativeDetail } from '#shared/types/initiative-detail.types';

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

  return {
    ...mapToInitiativeBase(b),
    currentBalanceCents,
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
  };
};
