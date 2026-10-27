// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getQuery } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { OrgSuggestResponse } from '../../types/query-service.types';
import type { OrgClaim } from '#shared/types/organization.types';

// Organization typeahead for the donate flow: query-service `/query/orgs/suggest`
// (Clearbit), through the gateway like the affiliations lookup.
export default defineEventHandler(async (event): Promise<OrgClaim[]> => {
  // Capped so a logged-in user can't push arbitrarily long strings upstream.
  const query = String(getQuery(event).query ?? '')
    .trim()
    .slice(0, 100);
  if (query.length < 2) return [];

  const { suggestions } = await useBackendFetch<OrgSuggestResponse>(
    event,
    `/query/orgs/suggest?v=1&query=${encodeURIComponent(query)}`,
    { platform: true },
  );
  return (suggestions ?? []).map((s) => ({ name: s.name, domain: s.domain }));
});
