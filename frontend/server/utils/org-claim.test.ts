// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';

vi.mock('./backend-fetch', () => ({
  useBackendFetch: vi.fn(),
}));

import * as backendFetchModule from './backend-fetch';
import { resolveOrgClaim } from './org-claim';

const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);
const mockEvent = {} as H3Event;

describe('resolveOrgClaim', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseBackendFetch.mockResolvedValue({
      data: [{ id: 'org-1', owner_id: 'u1', name: 'Acme Corp', created_on: '', updated_on: '' }],
    });
  });

  it('returns nothing for an individual donation', async () => {
    await expect(resolveOrgClaim(mockEvent, undefined)).resolves.toEqual({});
    expect(mockUseBackendFetch).not.toHaveBeenCalled();
  });

  it('maps the caller-owned org to its name', async () => {
    await expect(resolveOrgClaim(mockEvent, 'org-1')).resolves.toEqual({
      organization: { name: 'Acme Corp' },
    });
    expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/crowdfunding/me/organizations');
  });

  it('rejects an org the caller does not own', async () => {
    await expect(resolveOrgClaim(mockEvent, 'org-2')).rejects.toMatchObject({ statusCode: 404 });
  });
});
