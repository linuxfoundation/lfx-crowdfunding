// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

// h3 is mocked before the module under test is imported so that
// defineEventHandler becomes an identity function and the other helpers are
// controllable stubs.
vi.mock('h3', async (importOriginal) => {
  const actual = await importOriginal<typeof import('h3')>();
  return {
    ...actual,
    // Return the handler function directly so we can call it in tests.
    defineEventHandler: (fn: unknown) => fn,
    getCookie: vi.fn(),
    getRequestURL: vi.fn(),
    createError: vi.fn().mockImplementation((input: unknown) => {
      const opts = input as { statusCode: number; statusMessage?: string };
      return Object.assign(new Error(opts.statusMessage ?? 'error'), {
        statusCode: opts.statusCode,
      });
    }),
  };
});

import * as h3 from 'h3';
import requireAuth from './require-auth';

const mockGetRequestURL = vi.mocked(h3.getRequestURL);
const mockGetCookie = vi.mocked(h3.getCookie);
const mockCreateError = vi.mocked(h3.createError);

// Helper: set up the URL + method for a mock event.
const makeEvent = (method: string, path: string, token?: string) => {
  mockGetRequestURL.mockReturnValue(new URL(`http://localhost${path}`));
  mockGetCookie.mockReturnValue(token as ReturnType<typeof h3.getCookie>);
  return { method } as unknown as h3.H3Event;
};

describe('require-auth middleware', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // vi.clearAllMocks() resets call history but not mock implementations.
    // Re-apply the createError implementation so each test gets the throwing stub.
    mockCreateError.mockImplementation((input: unknown) => {
      const opts = input as { statusCode: number; statusMessage?: string };
      return Object.assign(new Error(opts.statusMessage ?? 'error'), {
        statusCode: opts.statusCode,
      }) as never;
    });
  });

  describe('PATCH /api/me', () => {
    it('throws 401 when no auth token is present', () => {
      const event = makeEvent('PATCH', '/api/me');
      expect(() => (requireAuth as (e: unknown) => void)(event)).toThrow();
      expect(mockCreateError).toHaveBeenCalledWith({
        statusCode: 401,
        statusMessage: 'Authentication required',
      });
    });

    it('passes through when a valid auth token is present', () => {
      const event = makeEvent('PATCH', '/api/me', 'eyJtoken');
      expect(() => (requireAuth as (e: unknown) => void)(event)).not.toThrow();
      expect(mockCreateError).not.toHaveBeenCalled();
    });

    it('passes through for GET — wrong method is not protected', () => {
      const event = makeEvent('GET', '/api/me');
      expect(() => (requireAuth as (e: unknown) => void)(event)).not.toThrow();
      expect(mockCreateError).not.toHaveBeenCalled();
    });
  });

  describe('existing protected routes (regression)', () => {
    it('blocks /api/payment/* without token', () => {
      const event = makeEvent('GET', '/api/payment/account');
      expect(() => (requireAuth as (e: unknown) => void)(event)).toThrow();
      expect(mockCreateError).toHaveBeenCalledWith({
        statusCode: 401,
        statusMessage: 'Authentication required',
      });
    });

    it('passes /api/payment/* with token', () => {
      const event = makeEvent('GET', '/api/payment/account', 'bearer-token');
      expect(() => (requireAuth as (e: unknown) => void)(event)).not.toThrow();
    });

    it('passes public routes without token', () => {
      const event = makeEvent('GET', '/api/statistics/platform');
      expect(() => (requireAuth as (e: unknown) => void)(event)).not.toThrow();
      expect(mockCreateError).not.toHaveBeenCalled();
    });
  });
});
