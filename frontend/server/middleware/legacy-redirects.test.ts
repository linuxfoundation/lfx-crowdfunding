// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

vi.mock('h3', async (importOriginal) => ({
  ...(await importOriginal<typeof import('h3')>()),
  defineEventHandler: (fn: unknown) => fn,
  getRequestURL: vi.fn(),
  sendRedirect: vi.fn(),
}));

import * as h3 from 'h3';
import type { H3Event } from 'h3';
import legacyRedirects from './legacy-redirects';

const handler = legacyRedirects as unknown as (event: H3Event) => unknown;
const mockGetRequestURL = vi.mocked(h3.getRequestURL);
const mockSendRedirect = vi.mocked(h3.sendRedirect);

const ID = '2f1c9a7e-5b3d-4c8e-9f0a-1b2c3d4e5f60';

// Simulates a request for the given path on the new host.
const request = (path: string, method = 'GET'): H3Event => {
  mockGetRequestURL.mockReturnValue(new URL(path, 'https://crowdfunding.linuxfoundation.org'));
  return { method } as H3Event;
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe('legacy-redirects middleware', () => {
  it.each([
    ['/projects/kubernetes', '/initiatives/kubernetes'],
    ['/projects/kubernetes/financial', '/initiatives/kubernetes'],
    ['/projects/kubernetes/stacks', '/initiatives/kubernetes'],
    [`/projects/${ID}/edit`, `/initiatives/${ID}`],
    [`/projects/${ID}/payments`, `/initiatives/${ID}`],
    [`/details/${ID}`, `/initiatives/${ID}`],
    [`/details/${ID}/financial`, `/initiatives/${ID}`],
    [`/events/${ID}/edit`, `/initiatives/${ID}`],
    [`/initiative/${ID}`, `/initiatives/${ID}`],
    [`/ostif/${ID}`, `/initiatives/${ID}`],
  ])('redirects the legacy page %s to %s', (from, to) => {
    handler(request(from));
    expect(mockSendRedirect).toHaveBeenCalledWith(expect.anything(), to, 301);
  });

  it('keeps the query string when redirecting a legacy page', () => {
    handler(request('/projects/kubernetes?utm_source=newsletter'));
    expect(mockSendRedirect).toHaveBeenCalledWith(
      expect.anything(),
      '/initiatives/kubernetes?utm_source=newsletter',
      301,
    );
  });

  it.each([
    '/projects/create',
    '/projects/create/connect',
    '/email/approve-project?token=abc',
    '/apply',
    '/apply/github',
  ])('sends the legacy flow page %s to the home page without its query', (from) => {
    handler(request(from));
    expect(mockSendRedirect).toHaveBeenCalledWith(expect.anything(), '/', 301);
  });

  it.each([
    '/',
    '/initiatives/kubernetes',
    '/initiatives/kubernetes/process-approval/approve',
    '/expense-email/approve/123',
    '/api/initiatives/kubernetes',
    '/auth/callback',
    '/projects',
    '/projects/',
    '/docs/getting-started',
  ])('leaves %s alone', (path) => {
    handler(request(path));
    expect(mockSendRedirect).not.toHaveBeenCalled();
  });

  it('leaves a non-GET request alone', () => {
    handler(request('/projects/kubernetes', 'POST'));
    expect(mockSendRedirect).not.toHaveBeenCalled();
  });
});
