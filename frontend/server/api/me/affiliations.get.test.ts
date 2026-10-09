// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';

// Auth guards are tested in require-auth.test.ts. This file tests handler logic only.

vi.mock('h3', async (importOriginal) => {
  const actual = await importOriginal<typeof import('h3')>();
  return {
    ...actual,
    defineEventHandler: (fn: unknown) => fn,
  };
});

vi.mock('../../utils/backend-fetch', () => ({
  useBackendFetch: vi.fn(),
}));

vi.mock('../../utils/auth-cookies', () => ({
  getAuthCookie: vi.fn(),
}));

import * as backendFetchModule from '../../utils/backend-fetch';
import * as authCookiesModule from '../../utils/auth-cookies';
import handler from './affiliations.get';

const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);
const mockGetAuthCookie = vi.mocked(authCookiesModule.getAuthCookie);
const mockEvent = {} as H3Event;
const run = () => (handler as (e: unknown) => Promise<unknown>)(mockEvent);

// Unsigned JWT — the service only decodes claims; the gateway verifies the signature.
const b64 = (o: object) => Buffer.from(JSON.stringify(o)).toString('base64url');
const tokenFor = (claims: object) => `${b64({ alg: 'none' })}.${b64(claims)}.sig`;

// Routes each query-service call by its `type=` so tests don't depend on call order.
const respond = (byType: Record<string, unknown[]>) =>
  mockUseBackendFetch.mockImplementation(async (_e, path) => {
    const type = new URLSearchParams(path.split('?')[1]).get('type') ?? '';
    return { resources: byType[type] ?? [] };
  });

const calledTypes = () =>
  mockUseBackendFetch.mock.calls.map(([, path]) =>
    new URLSearchParams(path.split('?')[1]).get('type'),
  );

describe('GET /api/me/affiliations BFF handler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetAuthCookie.mockReturnValue(tokenFor({ 'http://lfx.dev/claims/username': 'elim' }));
  });

  it('returns accepted org grants and direct-grant projects with names and logos', async () => {
    respond({
      b2b_org_settings: [
        {
          type: 'b2b_org_settings',
          id: 'b2b_org_settings:0012M00002qnukOQAQ',
          data: { members: [{ username: 'elim', role: 'auditor', invite_status: 'accepted' }] },
        },
        {
          type: 'b2b_org_settings',
          id: 'b2b_org_settings:0012M00002qnukPQAQ',
          data: { members: [{ username: 'elim', role: 'writer', invite_status: 'pending' }] },
        },
      ],
      b2b_org: [
        {
          type: 'b2b_org',
          id: 'b2b_org:0012M00002qnukOQAQ',
          data: { name: 'Acme Corp', logo_url: 'https://example.com/acme.png' },
        },
      ],
      project: [
        { type: 'project', id: 'project:root-uid', data: { name: 'ROOT', slug: 'ROOT' } },
        {
          type: 'project',
          id: 'project:0c1d2e3f-4a5b-4c6d-8e9f-0a1b2c3d4e5f',
          data: { name: 'Kubernetes', slug: 'kubernetes', logo_url: null },
        },
      ],
    });

    expect(await run()).toEqual({
      organizations: [
        { id: '0012M00002qnukOQAQ', name: 'Acme Corp', logoUrl: 'https://example.com/acme.png' },
      ],
      projects: [{ id: '0c1d2e3f-4a5b-4c6d-8e9f-0a1b2c3d4e5f', name: 'Kubernetes' }],
    });

    // Only the accepted grant is looked up — the pending invite is dropped.
    const orgLookup = mockUseBackendFetch.mock.calls.find(([, p]) => p.includes('type=b2b_org&'));
    expect(orgLookup?.[1]).toContain('tags=b2b_org_uid%3A0012M00002qnukOQAQ');
    expect(orgLookup?.[1]).not.toContain('0012M00002qnukPQAQ');
    // Every query-service call targets the platform gateway, not the CF backend.
    expect(mockUseBackendFetch.mock.calls.every(([, , opts]) => opts?.platform)).toBe(true);
  });

  it('shows no orgs and skips the unfiltered b2b_org lookup when the user has no org grants', async () => {
    respond({ b2b_org_settings: [], project: [] });

    expect(await run()).toEqual({ organizations: [], projects: [] });
    expect(calledTypes()).not.toContain('b2b_org');
  });

  it('accepts legacy writers[]/auditors[] settings docs', async () => {
    respond({
      b2b_org_settings: [
        {
          type: 'b2b_org_settings',
          id: 'b2b_org_settings:0012M00002qnukOQAQ',
          data: { auditors: [{ username: 'elim', invite_status: 'accepted' }] },
        },
      ],
      b2b_org: [{ type: 'b2b_org', id: 'b2b_org:0012M00002qnukOQAQ', data: { name: 'Acme Corp' } }],
    });

    expect(await run()).toEqual({
      organizations: [{ id: '0012M00002qnukOQAQ', name: 'Acme Corp' }],
      projects: [],
    });
  });

  it('returns nothing and makes no calls without a username claim', async () => {
    mockGetAuthCookie.mockReturnValue(undefined);

    expect(await run()).toEqual({ organizations: [], projects: [] });
    expect(mockUseBackendFetch).not.toHaveBeenCalled();
  });

  it('propagates a query-service failure so the step shows its error state', async () => {
    mockUseBackendFetch.mockRejectedValue(new Error('Upstream error'));

    await expect(run()).rejects.toThrow('Upstream error');
  });
});
