// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';
import type { OrganizationResponse } from '../../../types/organization.types';

// Auth guards are tested in require-auth.test.ts. This file tests handler logic only.

vi.mock('h3', async (importOriginal) => {
  const actual = await importOriginal<typeof import('h3')>();
  return {
    ...actual,
    defineEventHandler: (fn: unknown) => fn,
    readBody: vi.fn(),
    getRouterParam: vi.fn(),
  };
});

vi.mock('../../../utils/backend-fetch', () => ({
  useBackendFetch: vi.fn(),
}));

import * as h3 from 'h3';
import * as backendFetchModule from '../../../utils/backend-fetch';
import handler from './[id].patch';

const mockReadBody = vi.mocked(h3.readBody);
const mockGetRouterParam = vi.mocked(h3.getRouterParam);
const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);
const mockEvent = {} as H3Event;

describe('PATCH /api/me/organizations/:id BFF handler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetRouterParam.mockReturnValue('org-1');
  });

  it('proxies body to PATCH /v1/me/organizations/:id and maps the response', async () => {
    const body = { name: 'Updated Corp', avatar_url: 'https://example.com/new.png' };
    const raw: OrganizationResponse = {
      id: 'org-1',
      owner_id: 'user-1',
      name: 'Updated Corp',
      avatar_url: 'https://example.com/new.png',
      status: 'active',
      created_on: '2024-01-01T00:00:00Z',
      updated_on: '2024-06-01T00:00:00Z',
    };
    mockReadBody.mockResolvedValue(body);
    mockUseBackendFetch.mockResolvedValue(raw);

    const result = await (handler as (e: unknown) => Promise<unknown>)(mockEvent);

    expect(mockUseBackendFetch).toHaveBeenCalledOnce();
    expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/v1/me/organizations/org-1', {
      method: 'PATCH',
      body,
    });
    expect(result).toEqual({
      id: 'org-1',
      name: 'Updated Corp',
      avatarUrl: 'https://example.com/new.png',
      status: 'active',
    });
  });

  it('propagates errors thrown by useBackendFetch', async () => {
    mockReadBody.mockResolvedValue({ name: 'Updated Corp' });
    const upstreamError = Object.assign(new Error('Not Found'), { statusCode: 404 });
    mockUseBackendFetch.mockRejectedValue(upstreamError);

    await expect((handler as (e: unknown) => Promise<unknown>)(mockEvent)).rejects.toMatchObject({
      statusCode: 404,
    });
  });
});
