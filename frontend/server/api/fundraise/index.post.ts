// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, readBody, createError } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { BackendInitiative } from '../../types/initiatives.types';
import type {
  FundraisePayload,
  FundraiseResult,
  FundraiseContactInput,
  SecurityAuditFundraisePayload,
  GoalItemInput,
} from '../../types/fundraise.types';

export default defineEventHandler(async (event): Promise<FundraiseResult> => {
  const body = await readBody<FundraisePayload>(event);

  if (!body.initiativeType || !body.name || !body.description) {
    throw createError({
      statusCode: 400,
      statusMessage: 'initiativeType, name, and description are required',
    });
  }

  const initiative = await useBackendFetch<BackendInitiative>(event, '/v1/me/initiatives', {
    method: 'POST',
    body: buildBackendPayload(body),
  });

  return {
    id: initiative.id,
    slug: initiative.slug,
    name: initiative.name,
    status: initiative.status,
  };
});

function buildBackendPayload(payload: FundraisePayload): Record<string, unknown> {
  const base = {
    initiative_type: payload.initiativeType,
    name: payload.name,
    description: payload.description,
    industry: payload.industry || undefined,
    website_url: payload.websiteUrl || undefined,
    accept_funding: true,
  };

  switch (payload.initiativeType) {
    case 'project': {
      return {
        ...base,
        coc_url: payload.cocUrl || undefined,
        logo_url: payload.logoUrl || undefined,
        custom_websites: payload.repositoryUrl
          ? [{ name: 'Repository', url: payload.repositoryUrl }]
          : undefined,
        beneficiaries: payload.beneficiaries?.length
          ? payload.beneficiaries.map((b) => ({ name: b.name, email: b.email }))
          : undefined,
        goals: buildProjectGoals(payload.annualFundingGoalCents, payload.goals),
      };
    }

    case 'security_audit': {
      return {
        ...base,
        logo_url: payload.logoUrl || undefined,
        custom_websites: payload.repositoryUrl
          ? [{ name: 'Repository', url: payload.repositoryUrl }]
          : undefined,
        goals: payload.fundingGoalCents
          ? [
              {
                name: 'Audit Funding Goal',
                amount_cents: payload.fundingGoalCents,
                sort_order: 0,
              },
            ]
          : undefined,
        contacts: buildContacts(payload),
      };
    }

    case 'event': {
      return {
        ...base,
        logo_url: payload.logoUrl || undefined,
        eventbrite_url: payload.registrationUrl || undefined,
        event_start_date: payload.startDate ? new Date(payload.startDate).toISOString() : undefined,
        event_end_date: payload.endDate ? new Date(payload.endDate).toISOString() : undefined,
        city: payload.city || undefined,
        country: payload.country || undefined,
        is_online: payload.isOnline ?? false,
        beneficiaries: payload.beneficiaries?.length
          ? payload.beneficiaries.map((b) => ({ name: b.name, email: b.email }))
          : undefined,
        goals: payload.sponsorshipGoalCents
          ? [
              {
                name: 'Sponsorship Goal',
                amount_cents: payload.sponsorshipGoalCents,
                sort_order: 0,
              },
            ]
          : undefined,
        budget_distribution: payload.budgetDistribution?.length
          ? payload.budgetDistribution.map(buildBudgetDistributionItem)
          : undefined,
      };
    }

    case 'general_fund': {
      return {
        ...base,
        logo_url: payload.logoUrl || undefined,
        beneficiaries: payload.beneficiaries?.length
          ? payload.beneficiaries.map((b) => ({ name: b.name, email: b.email }))
          : undefined,
        goals: payload.annualFundingGoalCents
          ? [
              {
                name: 'Annual Funding Goal',
                amount_cents: payload.annualFundingGoalCents,
                sort_order: 0,
              },
            ]
          : undefined,
      };
    }
  }
}

// buildProjectGoals merges the top-level annual goal (if set) with the
// fund distribution items (enabled only) into a flat GoalInput array.
// Distribution items are mapped to initiative_goals: label → name,
// category → allocation, description → description, and amount_cents is
// derived from the item's percentage share of annualFundingGoalCents.
function buildProjectGoals(
  annualFundingGoalCents: number | undefined,
  goals: GoalItemInput[] | undefined,
): Array<Record<string, unknown>> | undefined {
  const result: Array<Record<string, unknown>> = [];

  if (annualFundingGoalCents) {
    result.push({
      name: 'Annual Funding Goal',
      amount_cents: annualFundingGoalCents,
      sort_order: 0,
    });
  }

  const enabledItems = goals?.filter((item) => item.enabled) ?? [];
  enabledItems.forEach((item, index) => {
    result.push({
      name: item.label,
      amount_cents: Math.round((item.percentage / 100) * (annualFundingGoalCents ?? 0)),
      allocation: item.category || undefined,
      description: item.description || undefined,
      sort_order: index + 1,
    });
  });

  return result.length > 0 ? result : undefined;
}

function buildBudgetDistributionItem(item: GoalItemInput): Record<string, unknown> {
  return {
    category: item.category,
    label: item.label,
    description: item.description,
    enabled: item.enabled,
    percentage: item.percentage,
  };
}

function buildContacts(
  payload: SecurityAuditFundraisePayload,
): Array<Record<string, unknown>> | undefined {
  const contacts: Array<Record<string, unknown>> = [];

  const add = (contactType: string, contact: FundraiseContactInput | undefined) => {
    if (!contact?.email && !contact?.firstName) return;
    contacts.push({
      contact_type: contactType,
      first_name: contact.firstName || undefined,
      last_name: contact.lastName || undefined,
      email: contact.email || undefined,
      phone_number: contact.phone || undefined,
      preferred_contact_method: contact.preferredContact || undefined,
    });
  };

  add('primary', payload.primaryContact);
  add('secondary', payload.secondaryContact);
  add('technical_lead', payload.technicalLead);

  return contacts.length > 0 ? contacts : undefined;
}
