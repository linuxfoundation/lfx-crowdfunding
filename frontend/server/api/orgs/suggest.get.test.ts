// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';

// Auth guards are tested in require-auth.test.ts. This file tests handler logic only.

const mockGetQuery = vi.fn();

vi.mock('h3', async (importOriginal) => {
  const actual = await importOriginal<typeof import('h3')>();
  return {
    ...actual,
    defineEventHandler: (fn: unknown) => fn,
    getQuery: () => mockGetQuery(),
  };
});

vi.mock('../../utils/backend-fetch', () => ({
  useBackendFetch: vi.fn(),
}));

import * as backendFetchModule from '../../utils/backend-fetch';
import handler from './suggest.get';

const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);
const mockEvent = {} as H3Event;
const run = () => (handler as (e: unknown) => Promise<unknown>)(mockEvent);

describe('GET /api/orgs/suggest BFF handler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('maps query-service suggestions to org claims, without the logo', async () => {
    mockGetQuery.mockReturnValue({ query: ' acme ' });
    mockUseBackendFetch.mockResolvedValue({
      suggestions: [
        { name: 'Acme Inc.', domain: 'acme.com', logo: null },
        { name: 'Acme Labs', domain: 'acmelabs.io' },
      ],
    });

    await expect(run()).resolves.toEqual([
      { name: 'Acme Inc.', domain: 'acme.com' },
      { name: 'Acme Labs', domain: 'acmelabs.io' },
    ]);
    expect(mockUseBackendFetch).toHaveBeenCalledWith(
      mockEvent,
      '/query/orgs/suggest?v=1&query=acme',
      {
        platform: true,
      },
    );
  });

  it('encodes the query', async () => {
    mockGetQuery.mockReturnValue({ query: 'AT&T' });
    mockUseBackendFetch.mockResolvedValue({ suggestions: [] });

    await run();
    expect(mockUseBackendFetch).toHaveBeenCalledWith(
      mockEvent,
      '/query/orgs/suggest?v=1&query=AT%26T',
      {
        platform: true,
      },
    );
  });

  it('skips the upstream call for queries under 2 characters', async () => {
    mockGetQuery.mockReturnValue({ query: 'a' });

    await expect(run()).resolves.toEqual([]);
    expect(mockUseBackendFetch).not.toHaveBeenCalled();
  });

  it('caps the query at 100 characters', async () => {
    mockGetQuery.mockReturnValue({ query: 'a'.repeat(150) });
    mockUseBackendFetch.mockResolvedValue({ suggestions: [] });

    await run();
    expect(mockUseBackendFetch).toHaveBeenCalledWith(
      mockEvent,
      `/query/orgs/suggest?v=1&query=${'a'.repeat(100)}`,
      { platform: true },
    );
  });

  it('returns no suggestions when upstream omits them', async () => {
    mockGetQuery.mockReturnValue({ query: 'acme' });
    mockUseBackendFetch.mockResolvedValue({ suggestions: null });

    await expect(run()).resolves.toEqual([]);
  });
});
