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
    createError: vi.fn().mockImplementation((input: unknown) => {
      const opts = input as { statusCode: number; statusMessage?: string };
      return Object.assign(new Error(opts.statusMessage ?? 'error'), {
        statusCode: opts.statusCode,
      });
    }),
  };
});

vi.mock('../../../utils/backend-fetch', () => ({
  useBackendFetch: vi.fn(),
}));

import * as h3 from 'h3';
import * as backendFetchModule from '../../../utils/backend-fetch';
import handler from './[reportId].post';

const mockGetRouterParam = vi.mocked(h3.getRouterParam);
const mockCreateError = vi.mocked(h3.createError);
const mockUseBackendFetch = vi.mocked(backendFetchModule.useBackendFetch);

const mockEvent = {} as H3Event;

const setupParams = (action: string, reportId: string) => {
  mockGetRouterParam.mockImplementation((_event, key) => {
    if (key === 'action') return action;
    if (key === 'reportId') return reportId;
    return undefined;
  });
};

describe('POST /api/expense-email/:action/:reportId BFF handler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCreateError.mockImplementation((input: unknown) => {
      const opts = input as { statusCode: number; statusMessage?: string };
      return Object.assign(new Error(opts.statusMessage ?? 'error'), {
        statusCode: opts.statusCode,
      }) as never;
    });
  });

  describe('valid actions', () => {
    it('proxies approve action to POST /v1/expense/approve/:reportId', async () => {
      setupParams('approve', 'R-001');
      mockUseBackendFetch.mockResolvedValue(undefined);

      await (handler as (e: unknown) => Promise<void>)(mockEvent);

      expect(mockUseBackendFetch).toHaveBeenCalledOnce();
      expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/v1/expense/approve/R-001', {
        method: 'POST',
      });
    });

    it('proxies reject action to POST /v1/expense/reject/:reportId', async () => {
      setupParams('reject', 'R-002');
      mockUseBackendFetch.mockResolvedValue(undefined);

      await (handler as (e: unknown) => Promise<void>)(mockEvent);

      expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/v1/expense/reject/R-002', {
        method: 'POST',
      });
    });

    it('URL-encodes special characters in action and reportId', async () => {
      setupParams('approve', 'R 001');
      mockUseBackendFetch.mockResolvedValue(undefined);

      await (handler as (e: unknown) => Promise<void>)(mockEvent);

      expect(mockUseBackendFetch).toHaveBeenCalledWith(mockEvent, '/v1/expense/approve/R%20001', {
        method: 'POST',
      });
    });
  });

  describe('invalid action', () => {
    it('throws 400 for an unknown action', async () => {
      setupParams('banana', 'R-001');

      await expect((handler as (e: unknown) => Promise<void>)(mockEvent)).rejects.toMatchObject({
        statusCode: 400,
      });

      expect(mockCreateError).toHaveBeenCalledWith(expect.objectContaining({ statusCode: 400 }));
      expect(mockUseBackendFetch).not.toHaveBeenCalled();
    });
  });

  describe('upstream errors', () => {
    it('propagates errors from useBackendFetch', async () => {
      setupParams('approve', 'R-003');
      const upstreamError = Object.assign(new Error('Not Found'), { statusCode: 404 });
      mockUseBackendFetch.mockRejectedValue(upstreamError);

      await expect((handler as (e: unknown) => Promise<void>)(mockEvent)).rejects.toMatchObject({
        statusCode: 404,
      });
    });
  });
});
