// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { vi, describe, it, expect, beforeEach } from 'vitest';
import type { H3Event } from 'h3';

const cookies: Record<string, string | undefined> = {};
const refreshSession = vi.fn();

vi.stubGlobal('defineEventHandler', (fn: unknown) => fn);
vi.stubGlobal('getAuthCookie', (_e: H3Event, name: string) => cookies[name]);
vi.stubGlobal('refreshSession', refreshSession);

const { default: handler } = await import('./user.get');
const run = handler as unknown as (e: H3Event) => Promise<{ isAuthenticated: boolean }>;
const event = {} as H3Event;
const profile = { sub: 'auth0|1', name: 'Jane' };
const encode = (p: object) => Buffer.from(JSON.stringify(p)).toString('base64');

describe('GET /api/auth/user', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    for (const k of Object.keys(cookies)) delete cookies[k];
  });

  it('returns the profile from cookies without refreshing', async () => {
    cookies.auth_user_profile = encode(profile);
    cookies.auth_oidc_token = 'tok';
    expect(await run(event)).toMatchObject({ isAuthenticated: true, user: profile });
    expect(refreshSession).not.toHaveBeenCalled();
  });

  it('is anonymous with no cookies at all', async () => {
    expect(await run(event)).toMatchObject({ isAuthenticated: false });
    expect(refreshSession).not.toHaveBeenCalled();
  });

  it('recovers the session from the refresh token when the access token expired', async () => {
    cookies.auth_refresh_token = 'rt';
    refreshSession.mockResolvedValue(profile);
    expect(await run(event)).toMatchObject({ isAuthenticated: true, user: profile });
    expect(refreshSession).toHaveBeenCalledWith(event);
  });

  it('is anonymous when the refresh fails', async () => {
    cookies.auth_refresh_token = 'rt';
    refreshSession.mockResolvedValue(null);
    expect(await run(event)).toMatchObject({ isAuthenticated: false });
  });
});
