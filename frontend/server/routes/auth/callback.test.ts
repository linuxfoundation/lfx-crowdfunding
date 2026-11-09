// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';
import { createError } from 'h3';
import { ResponseBodyError } from 'openid-client';

vi.mock('openid-client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('openid-client')>()),
  discovery: vi.fn().mockResolvedValue({}),
  authorizationCodeGrant: vi.fn(),
}));

const { authorizationCodeGrant, discovery } = await import('openid-client');

let query: Record<string, string> = {};
const cookies: Record<string, string | undefined> = {};
const sendRedirect = vi.fn();
const setCookie = vi.fn();
const clearAuthCookies = vi.fn();
const setSessionCookies = vi.fn();

vi.stubGlobal('defineEventHandler', (fn: unknown) => fn);
vi.stubGlobal('useRuntimeConfig', () => ({
  public: {
    auth0Domain: 'https://auth.example.com',
    auth0ClientId: 'client',
    auth0RedirectUri: 'http://localhost:3000/auth/callback',
  },
  auth0ClientSecret: 'secret',
}));
vi.stubGlobal('getQuery', () => query);
vi.stubGlobal('getCookie', (_e: H3Event, name: string) => cookies[name]);
vi.stubGlobal('getRequestURL', () => new URL('http://localhost:3000/auth/callback?code=c&state=s'));
vi.stubGlobal('sendRedirect', sendRedirect);
vi.stubGlobal('setCookie', setCookie);
vi.stubGlobal('createError', createError);
vi.stubGlobal('authCookieOptions', (maxAge: number) => ({ maxAge }));
vi.stubGlobal('clearAuthCookies', clearAuthCookies);
vi.stubGlobal('setSessionCookies', setSessionCookies);
vi.stubGlobal('buildUserProfile', () => ({ sub: 'auth0|1' }));

const { default: handler } = await import('./callback');
const run = handler as unknown as (e: H3Event) => Promise<void>;
const event = {} as H3Event;

const pkce = (state: string) => JSON.stringify({ state, codeVerifier: 'v' });
const clearedLoginCookies = () =>
  setCookie.mock.calls.filter(([, , value]) => value === '').map(([, name]) => name);

describe('GET /auth/callback', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.spyOn(console, 'error').mockImplementation(() => {});
    query = {};
    for (const k of Object.keys(cookies)) delete cookies[k];
  });

  it('logs in and redirects with auth=success', async () => {
    query = { code: 'c', state: 's' };
    cookies.auth_pkce = pkce('s');
    cookies.auth_redirect_to = '/initiatives/x';
    vi.mocked(authorizationCodeGrant).mockResolvedValue({
      access_token: 'at',
      id_token: 'id',
    } as Awaited<ReturnType<typeof authorizationCodeGrant>>);

    await run(event);

    expect(setSessionCookies).toHaveBeenCalled();
    expect(sendRedirect).toHaveBeenCalledWith(event, '/initiatives/x?auth=success');
  });

  it('keeps the session and redirects with auth=error when there is no login cookie', async () => {
    cookies.auth_redirect_to = '/initiatives/x?tab=updates';

    await run(event);

    expect(clearAuthCookies).not.toHaveBeenCalled();
    expect(clearedLoginCookies()).toEqual(['auth_pkce', 'auth_redirect_to']);
    expect(sendRedirect).toHaveBeenCalledWith(event, '/initiatives/x?tab=updates&auth=error');
    expect(console.warn).toHaveBeenCalledTimes(1);
    expect(console.error).not.toHaveBeenCalled();
  });

  it('keeps the session and redirects with auth=error when the state does not match', async () => {
    query = { code: 'c', state: 's' };
    cookies.auth_pkce = pkce('other');

    await run(event);

    expect(clearAuthCookies).not.toHaveBeenCalled();
    expect(sendRedirect).toHaveBeenCalledWith(event, '/?auth=error');
    expect(discovery).not.toHaveBeenCalled();
    // One line from logAuthError; h3 must not add its statusMessage-sanitising warning.
    expect(console.warn).toHaveBeenCalledTimes(1);
  });

  it('clears the session and redirects with auth=error on an Auth0 error', async () => {
    query = { error: 'access_denied', error_description: 'User did not authorize — sorry' };

    await run(event);

    expect(clearAuthCookies).toHaveBeenCalled();
    expect(sendRedirect).toHaveBeenCalledWith(event, '/?auth=error');
    expect(console.warn).toHaveBeenCalledTimes(1);
  });

  it('still redirects with auth=success on login_required', async () => {
    query = { error: 'login_required' };

    await run(event);

    expect(sendRedirect).toHaveBeenCalledWith(event, '/?auth=success');
    expect(clearAuthCookies).not.toHaveBeenCalled();
  });

  it('clears the session and redirects with auth=error when the code is rejected', async () => {
    query = { code: 'c', state: 's' };
    cookies.auth_pkce = pkce('s');
    vi.mocked(authorizationCodeGrant).mockRejectedValue(
      new ResponseBodyError('server responded with an error in the response body', {
        cause: { error: 'invalid_grant', error_description: 'Invalid authorization code' },
        response: new Response(null, { status: 403 }),
      }),
    );

    await run(event);

    expect(clearAuthCookies).toHaveBeenCalled();
    expect(sendRedirect).toHaveBeenCalledWith(event, '/?auth=error');
  });

  it('throws a 500 with the full error logged on an unexpected failure', async () => {
    query = { code: 'c', state: 's' };
    cookies.auth_pkce = pkce('s');
    vi.mocked(discovery).mockRejectedValueOnce(new Error('network down'));

    await expect(run(event)).rejects.toMatchObject({ statusCode: 500 });

    expect(clearAuthCookies).toHaveBeenCalled();
    expect(sendRedirect).not.toHaveBeenCalled();
    expect(console.error).toHaveBeenCalledWith('Auth callback error:', expect.any(Error));
  });
});
