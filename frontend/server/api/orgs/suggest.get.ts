// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { defineEventHandler, getQuery, type H3Event } from 'h3';
import { useBackendFetch } from '../../utils/backend-fetch';
import type { OrgSuggestResponse } from '../../types/query-service.types';
import type { OrgLogosWire } from '../../types/org-logos.types';
import type { OrgSuggestion } from '#shared/types/organization.types';

// The backend answers for at most this many domains per lookup.
const MAX_LOGO_DOMAINS = 20;

// Organization typeahead for the donate flow: query-service `/query/orgs/suggest`
// (Clearbit), through the gateway like the affiliations lookup. Each suggestion
// gets the b2b_org logo CF has stored for its domain, if any (#472).
export default defineEventHandler(async (event): Promise<OrgSuggestion[]> => {
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
  const claims = (suggestions ?? []).map((s) => ({ name: s.name, domain: s.domain }));
  if (claims.length === 0) return [];

  const logos = await storedOrgLogos(
    event,
    claims.map((c) => c.domain),
  );
  return claims.map((c) => (logos[c.domain] ? { ...c, logoUrl: logos[c.domain] } : c));
});

// Reads only CF's own logo table, never member-service. Logos are decoration:
// a failed lookup (logged by the backend) leaves the suggestions without them.
const storedOrgLogos = async (
  event: H3Event,
  domains: string[],
): Promise<Record<string, string>> => {
  const unique = [...new Set(domains.filter(Boolean))].slice(0, MAX_LOGO_DOMAINS);
  if (unique.length === 0) return {};
  try {
    const { logos } = await useBackendFetch<OrgLogosWire>(
      event,
      `/crowdfunding/me/org-logos?domains=${encodeURIComponent(unique.join(','))}`,
    );
    return logos ?? {};
  } catch {
    return {};
  }
};
