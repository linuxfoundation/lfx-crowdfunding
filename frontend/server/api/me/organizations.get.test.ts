// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';
import type { OrganizationResponse } from '../../types/organization.types';

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

import * as backendFetchModule from '../../utils/backend-fetch';
import handler from './organizations.get';

const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);
const mockEvent = {} as H3Event;

describe('GET /api/me/organizations BFF handler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('maps backend response to camelCase Organization array', async () => {
    const raw: { data: OrganizationResponse[] } = {
      data: [
        {
          id: 'org-1',
          owner_id: 'user-1',
          name: 'Acme Corp',
          avatar_url: 'https://example.com/logo.png',
          status: 'active',
          created_on: '2024-01-01T00:00:00Z',
          updated_on: '2024-01-02T00:00:00Z',
        },
      ],
    };
    mockUseBackendFetch.mockResolvedValue(raw);

    const result = await (handler as (e: unknown) => Promise<unknown>)(mockEvent);

    expect(mockUseBackendFetch).toHaveBeenCalledOnce();
    expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/v1/me/organizations');
    expect(result).toEqual([
      {
        id: 'org-1',
        name: 'Acme Corp',
        avatarUrl: 'https://example.com/logo.png',
        status: 'active',
      },
    ]);
  });

  it('returns an empty array when data is empty', async () => {
    mockUseBackendFetch.mockResolvedValue({ data: [] });

    const result = await (handler as (e: unknown) => Promise<unknown>)(mockEvent);

    expect(result).toEqual([]);
  });

  it('propagates errors thrown by useBackendFetch', async () => {
    const upstreamError = Object.assign(new Error('Unauthorized'), { statusCode: 401 });
    mockUseBackendFetch.mockRejectedValue(upstreamError);

    await expect((handler as (e: unknown) => Promise<unknown>)(mockEvent)).rejects.toMatchObject({
      statusCode: 401,
    });
  });
});
