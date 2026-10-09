// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

vi.mock('h3', async (importOriginal) => {
  const actual = await importOriginal<typeof import('h3')>();
  return {
    ...actual,
    defineEventHandler: (fn: unknown) => fn,
    getRequestURL: vi.fn(),
    getRequestHeader: vi.fn(),
  };
});

import * as h3 from 'h3';
import csrf from './csrf';

const mockGetRequestURL = vi.mocked(h3.getRequestURL);
const mockGetRequestHeader = vi.mocked(h3.getRequestHeader);

const run = (method: string, path: string, headers: Record<string, string> = {}) => {
  mockGetRequestURL.mockReturnValue(new URL(`http://crowdfunding.lfx.dev${path}`));
  mockGetRequestHeader.mockImplementation((_e, name) => headers[name.toLowerCase()]);
  return () => (csrf as unknown as (e: unknown) => void)({ method });
};

const statusOf = (fn: () => void) => {
  try {
    fn();
    return undefined;
  } catch (e) {
    return (e as { statusCode: number }).statusCode;
  }
};

const JSON_TYPE = { 'content-type': 'application/json' };

describe('csrf middleware', () => {
  beforeEach(() => vi.clearAllMocks());

  it('allows same-origin JSON POST (Origin scheme may differ behind the ingress)', () => {
    expect(
      statusOf(
        run('POST', '/api/me/organizations', {
          origin: 'https://crowdfunding.lfx.dev',
          ...JSON_TYPE,
        }),
      ),
    ).toBeUndefined();
  });

  it('rejects a POST from a sibling linuxfoundation host', () => {
    expect(
      statusOf(run('POST', '/api/fundraise', { origin: 'https://evil.lfx.dev', ...JSON_TYPE })),
    ).toBe(403);
  });

  it('rejects the opaque null origin', () => {
    expect(statusOf(run('DELETE', '/api/me/organizations/1', { origin: 'null' }))).toBe(403);
  });

  it('falls back to Sec-Fetch-Site when Origin is absent', () => {
    expect(statusOf(run('POST', '/api/payment/method', { 'sec-fetch-site': 'same-site' }))).toBe(
      403,
    );
    expect(
      statusOf(run('POST', '/api/payment/method', { 'sec-fetch-site': 'same-origin' })),
    ).toBeUndefined();
  });

  it('allows requests with neither header (non-browser clients)', () => {
    expect(statusOf(run('POST', '/api/e2e-auth'))).toBeUndefined();
  });

  it('rejects same-origin form-encoded bodies', () => {
    const origin = 'https://crowdfunding.lfx.dev';
    for (const type of [
      'application/x-www-form-urlencoded',
      'multipart/form-data; boundary=x',
      'text/plain',
    ]) {
      expect(
        statusOf(
          run('POST', '/api/initiatives/x/process-approval/approve', {
            origin,
            'content-type': type,
          }),
        ),
      ).toBe(415);
    }
  });

  it('ignores safe methods and non-/api paths', () => {
    expect(statusOf(run('GET', '/api/me', { origin: 'https://evil.lfx.dev' }))).toBeUndefined();
    expect(
      statusOf(run('POST', '/auth/callback', { origin: 'https://evil.lfx.dev' })),
    ).toBeUndefined();
  });
});
