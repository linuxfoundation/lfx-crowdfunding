// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';
import type { UserResponse } from '../types/user.types';

// Authentication guards are enforced by server/middleware/require-auth.ts and
// are tested in require-auth.test.ts.  This file tests the handler logic only.

vi.mock('h3', async (importOriginal) => {
  const actual = await importOriginal<typeof import('h3')>();
  return {
    ...actual,
    defineEventHandler: (fn: unknown) => fn,
  };
});

vi.mock('../utils/backend-fetch', () => ({
  useBackendFetch: vi.fn(),
}));

import * as backendFetchModule from '../utils/backend-fetch';
import syncProfileHandler from './me.patch';

const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);

const mockEvent = {} as H3Event;

describe('PATCH /api/me BFF handler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('proxies to PATCH /v1/me and returns the backend response', async () => {
    const fakeResponse: UserResponse = {
      id: 'uuid-1',
      username: 'jdoe',
      email: 'jdoe@example.com',
      name: 'John Doe',
      created_on: '2024-01-01T00:00:00Z',
      updated_on: '2024-01-01T00:00:00Z',
    };
    mockUseBackendFetch.mockResolvedValue(fakeResponse);

    const result = await (syncProfileHandler as (e: unknown) => Promise<UserResponse>)(mockEvent);

    expect(mockUseBackendFetch).toHaveBeenCalledOnce();
    expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/v1/me', {
      method: 'PATCH',
    });
    expect(result).toEqual(fakeResponse);
  });

  it('propagates errors thrown by useBackendFetch', async () => {
    const upstreamError = Object.assign(new Error('Bad Gateway'), { statusCode: 502 });
    mockUseBackendFetch.mockRejectedValue(upstreamError);

    await expect(
      (syncProfileHandler as (e: unknown) => Promise<unknown>)(mockEvent),
    ).rejects.toMatchObject({ statusCode: 502 });
  });
});
