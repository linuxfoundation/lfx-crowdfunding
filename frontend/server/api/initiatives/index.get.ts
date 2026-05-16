// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { InitiativeBase, InitiativesResponse } from '#shared/types/initiative.types';

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
  financials?: {
    total_raised_cents: number;
    supporters: number;
    goals_total_cents: number;
  };
}

interface BackendResponse {
  data: BackendInitiative[];
  meta: { total: number; limit: number; offset: number };
}

function toInitiativeBase(b: BackendInitiative): InitiativeBase {
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
}

export default defineEventHandler(async (event): Promise<InitiativesResponse> => {
  const { search, type, sort, page, pageSize } = getQuery(event);

  const apiBase = process.env.NUXT_API_BASE_URL ?? 'http://localhost:8080';
  const params = new URLSearchParams();
  if (search) params.set('search', String(search));
  if (type && type !== 'all') params.set('type', String(type));
  if (sort) params.set('sort_by', String(sort));

  const pageSizeNum = typeof pageSize === 'string' ? Math.max(1, parseInt(pageSize, 10) || 12) : 12;
  const pageNum = typeof page === 'string' ? Math.max(1, parseInt(page, 10) || 1) : 1;
  params.set('limit', String(pageSizeNum));
  params.set('offset', String((pageNum - 1) * pageSizeNum));

  const res = await $fetch<BackendResponse>(`${apiBase}/v1/initiatives?${params}`);
  return { data: (res.data ?? []).map(toInitiativeBase), total: res.meta.total };
});
