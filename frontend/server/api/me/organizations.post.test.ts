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
    readBody: vi.fn(),
  };
});

vi.mock('../../utils/backend-fetch', () => ({
  useBackendFetch: vi.fn(),
}));

import * as h3 from 'h3';
import * as backendFetchModule from '../../utils/backend-fetch';
import handler from './organizations.post';

const mockReadBody = vi.mocked(h3.readBody);
const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);
const mockEvent = {} as H3Event;

describe('POST /api/me/organizations BFF handler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('proxies body to POST /v1/me/organizations and maps the response', async () => {
    const body = { name: 'Acme Corp', avatar_url: 'https://example.com/logo.png' };
    const raw: OrganizationResponse = {
      id: 'org-new',
      owner_id: 'user-1',
      name: 'Acme Corp',
      avatar_url: 'https://example.com/logo.png',
      status: 'active',
      created_on: '2024-01-01T00:00:00Z',
      updated_on: '2024-01-01T00:00:00Z',
    };
    mockReadBody.mockResolvedValue(body);
    mockUseBackendFetch.mockResolvedValue(raw);

    const result = await (handler as (e: unknown) => Promise<unknown>)(mockEvent);

    expect(mockUseBackendFetch).toHaveBeenCalledOnce();
    expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/v1/me/organizations', {
      method: 'POST',
      body,
    });
    expect(result).toEqual({
      id: 'org-new',
      name: 'Acme Corp',
      avatarUrl: 'https://example.com/logo.png',
      status: 'active',
    });
  });

  it('propagates errors thrown by useBackendFetch', async () => {
    mockReadBody.mockResolvedValue({ name: 'Acme' });
    const upstreamError = Object.assign(new Error('Bad Request'), { statusCode: 400 });
    mockUseBackendFetch.mockRejectedValue(upstreamError);

    await expect((handler as (e: unknown) => Promise<unknown>)(mockEvent)).rejects.toMatchObject({
      statusCode: 400,
    });
  });
});
