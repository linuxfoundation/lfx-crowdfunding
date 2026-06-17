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
    getRouterParam: vi.fn(),
  };
});

vi.mock('../../../utils/backend-fetch', () => ({
  useBackendFetch: vi.fn(),
}));

import * as h3 from 'h3';
import * as backendFetchModule from '../../../utils/backend-fetch';
import handler from './[id].delete';

const mockGetRouterParam = vi.mocked(h3.getRouterParam);
const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);
const mockEvent = {} as H3Event;

describe('DELETE /api/me/organizations/:id BFF handler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetRouterParam.mockReturnValue('org-1');
  });

  it('proxies DELETE /v1/me/organizations/:id and returns void', async () => {
    mockUseBackendFetch.mockResolvedValue(undefined);

    const result = await (handler as (e: unknown) => Promise<unknown>)(mockEvent);

    expect(mockUseBackendFetch).toHaveBeenCalledOnce();
    expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/v1/me/organizations/org-1', {
      method: 'DELETE',
    });
    expect(result).toBeUndefined();
  });

  it('propagates errors thrown by useBackendFetch', async () => {
    const upstreamError = Object.assign(new Error('Not Found'), { statusCode: 404 });
    mockUseBackendFetch.mockRejectedValue(upstreamError);

    await expect((handler as (e: unknown) => Promise<unknown>)(mockEvent)).rejects.toMatchObject({
      statusCode: 404,
    });
  });
});
