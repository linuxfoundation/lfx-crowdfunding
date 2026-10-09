// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';
import { decodeJwt } from 'jose';
import type {
  EntityDoc,
  OrgSettingsDoc,
  QueryResource,
  QueryResourcesResponse,
} from '../types/query-service.types';
import { getAuthCookie } from '../utils/auth-cookies';
import { useBackendFetch } from '../utils/backend-fetch';
import type { AttributionInput } from '../types/fundraise.types';
import type { AffiliationCandidates, AffiliationEntity } from '#shared/types/affiliation.types';

// Username claim on gateway-audience access tokens (the ID token carries
// `https://sso.linuxfoundation.org/claims/username` instead).
const USERNAME_CLAIM = 'http://lfx.dev/claims/username';

// Upstream filters_or batch limit, and the cap on how many settings docs we read — so the
// name lookup below is always a single call.
const PAGE_SIZE = 100;

// The ROOT pseudo-project is administrative, never an attribution target.
const ROOT_PROJECT_SLUG = 'ROOT';

const uidOf = (id: string) => id.slice(id.indexOf(':') + 1);

const toEntity = (r: QueryResource<EntityDoc>): AffiliationEntity => ({
  id: uidOf(r.id),
  name: r.data.name ?? uidOf(r.id),
  ...(r.data.logo_url ? { logoUrl: r.data.logo_url } : {}),
});

const usernameFromToken = (token: string): string | undefined => {
  try {
    const username = decodeJwt(token)[USERNAME_CLAIM];
    return typeof username === 'string' ? username : undefined;
  } catch {
    return undefined;
  }
};

const query = <T>(event: H3Event, params: string) =>
  useBackendFetch<QueryResourcesResponse<T>>(event, `/query/resources?v=1&${params}`, {
    platform: true,
  });

// Orgs where the caller is an accepted writer or auditor — mirrors Self Serve's org lens
// (`org-role-grants.service.ts`). Both roles pass the save gate (FGA `auditor` on the b2b_org).
const getOrganizations = async (event: H3Event, username: string) => {
  const settings = await query<OrgSettingsDoc>(
    event,
    `type=b2b_org_settings&tags=${encodeURIComponent(`member:${username}`)}&page_size=${PAGE_SIZE}`,
  );
  const uids = settings.resources
    .filter(({ data }) =>
      [...(data.members ?? []), ...(data.writers ?? []), ...(data.auditors ?? [])].some(
        (m) => m.username === username && m.invite_status === 'accepted',
      ),
    )
    .map((r) => uidOf(r.id));

  // An empty tag list makes the b2b_org query unfiltered (arbitrary orgs from the whole
  // index) — a user with no grants must see none.
  if (!uids.length) return [];

  const tags = uids.map((uid) => `tags=${encodeURIComponent(`b2b_org_uid:${uid}`)}`).join('&');
  const orgs = await query<EntityDoc>(event, `type=b2b_org&${tags}&page_size=${PAGE_SIZE}`);
  return orgs.resources.map(toEntity);
};

// Projects the caller holds a direct FGA grant on — mirrors Self Serve's
// `getDirectGrantProjectRows`. Inherited/team grants aren't listed here, but still pass the
// server-side save gate.
// ponytail: first page only (100); paginate via page_token if anyone outgrows it.
const getProjects = async (event: H3Event) => {
  const projects = await query<EntityDoc>(
    event,
    `type=project&filter_grants=direct&page_size=${PAGE_SIZE}`,
  );
  return projects.resources.filter((r) => r.data.slug !== ROOT_PROJECT_SLUG).map(toEntity);
};

// A query failure propagates: the attribution step shows its "couldn't load" state with only
// Personal selectable, so fundraise creation is never blocked.
export const getAffiliations = async (event: H3Event): Promise<AffiliationCandidates> => {
  const username = usernameFromToken(getAuthCookie(event, 'auth_oidc_token') ?? '');
  if (!username) return { organizations: [], projects: [] };

  const [organizations, projects] = await Promise.all([
    getOrganizations(event, username),
    getProjects(event),
  ]);
  return { organizations, projects };
};

// Display values for the public source label (lfx-crowdfunding#346), taken from the caller's own
// candidates rather than the client so an initiative can't carry another entity's name. Empty on a
// miss or a query failure: the attribution still saves (the backend gates it), just unlabeled.
// ponytail: refetches both candidate lists on submit; fine at one call per fundraise.
export const getAttributionDisplay = async (
  event: H3Event,
  { kind, entityId }: AttributionInput,
): Promise<{ name?: string; logo_url?: string }> => {
  try {
    const { organizations, projects } = await getAffiliations(event);
    const entity = (kind === 'organization' ? organizations : projects).find(
      (e) => e.id === entityId,
    );
    if (!entity) return {};
    // The backend rejects non-https logos; drop one rather than fail the whole fundraise.
    const logoUrl = entity.logoUrl?.startsWith('https://') ? entity.logoUrl : undefined;
    return { name: entity.name.slice(0, 255), logo_url: logoUrl };
  } catch {
    return {};
  }
};
