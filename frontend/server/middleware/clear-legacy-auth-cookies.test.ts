// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';

vi.mock('h3', async (importOriginal) => ({
  ...(await importOriginal<typeof import('h3')>()),
  defineEventHandler: (fn: unknown) => fn,
  getCookie: vi.fn(),
  setCookie: vi.fn(),
}));

const runtimeConfig: { auth0CookieDomain?: string } = {};
vi.stubGlobal('useRuntimeConfig', () => runtimeConfig);

import * as h3 from 'h3';
import type { H3Event } from 'h3';
import { AUTH_COOKIE_NAMES } from '../utils/auth-cookies';
import clearLegacyAuthCookies, { LEGACY_CLEARED_COOKIE } from './clear-legacy-auth-cookies';

const handler = clearLegacyAuthCookies as unknown as (event: H3Event) => void;
const mockGetCookie = vi.mocked(h3.getCookie);
const mockSetCookie = vi.mocked(h3.setCookie);
const event = {} as H3Event;

// Simulates the browser sending the given cookie names.
const sendCookies = (...names: string[]) =>
  mockGetCookie.mockImplementation((_e, name) => (names.includes(name) ? 'x' : undefined));

beforeEach(() => {
  vi.clearAllMocks();
  runtimeConfig.auth0CookieDomain = 'example.org';
});

describe('clear-legacy-auth-cookies middleware', () => {
  it('is a no-op when no legacy domain is configured', () => {
    delete runtimeConfig.auth0CookieDomain;
    sendCookies(...AUTH_COOKIE_NAMES);
    handler(event);
    expect(mockSetCookie).not.toHaveBeenCalled();
  });

  it('expires every present cookie on the legacy domain, then sets the marker', () => {
    sendCookies(...AUTH_COOKIE_NAMES);
    handler(event);
    for (const name of AUTH_COOKIE_NAMES) {
      expect(mockSetCookie).toHaveBeenCalledWith(
        event,
        name,
        '',
        expect.objectContaining({ domain: 'example.org', path: '/', maxAge: 0 }),
      );
    }
    const marker = mockSetCookie.mock.calls.find((c) => c[1] === LEGACY_CLEARED_COOKIE)!;
    expect(marker[3]).not.toHaveProperty('domain');
    expect(marker[3]).toMatchObject({ maxAge: 30 * 24 * 60 * 60 });
  });

  it('only expires cookies the browser actually sent', () => {
    sendCookies('auth_oidc_token');
    handler(event);
    expect(mockSetCookie.mock.calls.map((c) => c[1])).toEqual([
      'auth_oidc_token',
      LEGACY_CLEARED_COOKIE,
    ]);
  });

  it('does nothing once the marker is present', () => {
    sendCookies(...AUTH_COOKIE_NAMES, LEGACY_CLEARED_COOKIE);
    handler(event);
    expect(mockSetCookie).not.toHaveBeenCalled();
  });

  it('sets no marker when there were no auth cookies to clear', () => {
    sendCookies();
    handler(event);
    expect(mockSetCookie).not.toHaveBeenCalled();
  });
});
