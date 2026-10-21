// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { UserProfile } from '../../types/auth.types';

const anonymous = () => ({ isAuthenticated: false, user: null, token: null });

export default defineEventHandler(async (event) => {
  try {
    const profileCookie = getAuthCookie(event, 'auth_user_profile');
    const tokenCookie = getAuthCookie(event, 'auth_oidc_token');

    let user: UserProfile | null;
    if (profileCookie && tokenCookie) {
      user = JSON.parse(Buffer.from(profileCookie, 'base64').toString('utf-8'));
    } else if (getAuthCookie(event, 'auth_refresh_token')) {
      // The access token and profile cookies expire with the access token (~1h), but the
      // refresh token lives 30 days. Recover the session here so a returning user isn't
      // shown as logged out.
      user = await refreshSession(event);
    } else {
      return anonymous();
    }
    if (!user) return anonymous();

    return {
      isAuthenticated: true,
      user: {
        sub: user.sub,
        name: user.name,
        email: user.email,
        picture: user.picture,
        email_verified: user.email_verified,
        updated_at: user.updated_at,
        username: user.username,
        intercomJwt: user.intercomJwt,
      },
      token: null,
    };
  } catch (error) {
    console.error('Auth user error:', error);
    return anonymous();
  }
});
