// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

vi.mock('h3', async (importOriginal) => {
  const actual = await importOriginal<typeof import('h3')>();
  return {
    ...actual,
    getRequestURL: vi.fn(),
    getRequestProtocol: vi.fn(),
  };
});

import * as h3 from 'h3';
import { getGithubCallbackUrl } from './github';

const mockGetRequestURL = vi.mocked(h3.getRequestURL);
const mockGetRequestProtocol = vi.mocked(h3.getRequestProtocol);

const makeEvent = (host: string, proto: string) => {
  mockGetRequestURL.mockReturnValue(
    new URL(`http://${host}/`) as ReturnType<typeof h3.getRequestURL>,
  );
  mockGetRequestProtocol.mockReturnValue(proto);
  return {} as h3.H3Event;
};

describe('getGithubCallbackUrl', () => {
  beforeEach(() => vi.clearAllMocks());

  it('returns http callback URL for local dev (no forwarded headers)', () => {
    const event = makeEvent('localhost:3000', 'http');
    expect(getGithubCallbackUrl(event)).toBe('http://localhost:3000/api/github/callback');
  });

  it('uses X-Forwarded-Host to build the callback host', () => {
    const event = makeEvent('app.lfx.dev', 'http');
    expect(getGithubCallbackUrl(event)).toBe('http://app.lfx.dev/api/github/callback');
  });

  it('uses X-Forwarded-Proto to upgrade the protocol to https behind TLS ingress', () => {
    const event = makeEvent('app.lfx.dev', 'https');
    expect(getGithubCallbackUrl(event)).toBe('https://app.lfx.dev/api/github/callback');
  });
});
