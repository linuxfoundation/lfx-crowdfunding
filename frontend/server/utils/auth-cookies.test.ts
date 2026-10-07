// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('h3', async (importOriginal) => ({
  ...(await importOriginal<typeof import('h3')>()),
  getCookie: vi.fn(),
  setCookie: vi.fn(),
}));

const runtimeConfig: { auth0ClientSecret: string; auth0CookieDomain?: string } = {
  auth0ClientSecret: 'test-secret',
};
vi.stubGlobal('useRuntimeConfig', () => runtimeConfig);

import * as h3 from 'h3';
import type { H3Event } from 'h3';
import {
  AUTH_COOKIE_NAMES,
  authCookieOptions,
  clearAuthCookies,
  decryptValue,
  encryptValue,
  getAuthCookie,
  setAuthCookie,
} from './auth-cookies';

const mockSetCookie = vi.mocked(h3.setCookie);
const mockGetCookie = vi.mocked(h3.getCookie);
const event = {} as H3Event;

beforeEach(() => {
  vi.clearAllMocks();
  delete runtimeConfig.auth0CookieDomain;
});

describe('auth cookies', () => {
  it('round-trips and does not expose the plaintext', () => {
    const enc = encryptValue('raw-access-token');
    expect(enc).not.toContain('raw-access-token');
    expect(decryptValue(enc)).toBe('raw-access-token');
  });

  it('rejects tampered, legacy plaintext and missing values', () => {
    const enc = encryptValue('t');
    const tampered = enc.slice(0, -2) + (enc.endsWith('AA') ? 'BB' : 'AA');
    expect(decryptValue(tampered)).toBeUndefined();
    expect(decryptValue('eyJhbGciOi.legacy.jwt')).toBeUndefined();
    expect(decryptValue(undefined)).toBeUndefined();
  });

  it('is host-only', () => {
    expect(authCookieOptions(60)).not.toHaveProperty('domain');
  });

  it('setAuthCookie stores an encrypted value that getAuthCookie reads back', () => {
    setAuthCookie(event, 'auth_oidc_token', 'secret-token', 60);
    const [, name, stored, opts] = mockSetCookie.mock.calls[0]!;
    expect(name).toBe('auth_oidc_token');
    expect(stored).not.toContain('secret-token');
    expect(opts).toMatchObject({ httpOnly: true, maxAge: 60 });
    expect(opts).not.toHaveProperty('domain');

    mockGetCookie.mockReturnValue(stored);
    expect(getAuthCookie(event, 'auth_oidc_token')).toBe('secret-token');
  });

  it('clearAuthCookies expires host-only cookies only when no legacy domain is set', () => {
    clearAuthCookies(event);
    expect(mockSetCookie).toHaveBeenCalledTimes(AUTH_COOKIE_NAMES.length);
    for (const call of mockSetCookie.mock.calls) {
      expect(call[3]).toMatchObject({ maxAge: 0 });
      expect(call[3]).not.toHaveProperty('domain');
    }
  });

  it('clearAuthCookies also expires the legacy Domain copy of every cookie', () => {
    runtimeConfig.auth0CookieDomain = 'example.org';
    clearAuthCookies(event);
    expect(mockSetCookie).toHaveBeenCalledTimes(AUTH_COOKIE_NAMES.length * 2);
    for (const name of AUTH_COOKIE_NAMES) {
      expect(mockSetCookie).toHaveBeenCalledWith(
        event,
        name,
        '',
        expect.objectContaining({ domain: 'example.org', maxAge: 0 }),
      );
    }
  });
});
