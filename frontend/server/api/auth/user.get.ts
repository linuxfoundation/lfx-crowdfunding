// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { getCookie } from 'h3';

export default defineEventHandler((event) => {
  try {
    const profileCookie = getCookie(event, 'auth_user_profile');
    const tokenCookie = getCookie(event, 'auth_oidc_token');

    if (!profileCookie || !tokenCookie) {
      return { isAuthenticated: false, user: null, token: null };
    }

    const user = JSON.parse(Buffer.from(profileCookie, 'base64').toString('utf-8'));

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
      },
      token: null,
    };
  } catch (error) {
    console.error('Auth user error:', error);
    return { isAuthenticated: false, user: null, token: null };
  }
});
