// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { getCookie } from 'h3';
import { jwtVerify } from 'jose';
import type { DecodedOidcToken } from '~~/types/auth/auth-jwt.types';

export default defineEventHandler(async (event) => {
  try {
    const config = useRuntimeConfig();
    const oidcToken = getCookie(event, 'auth_oidc_token');

    if (!oidcToken) {
      // Only suggest silent login for browser requests not already in a callback flow
      const userAgent = getHeader(event, 'user-agent') || '';
      const referer = getHeader(event, 'referer') || '';
      const hasPkceCookie = !!getCookie(event, 'auth_pkce');

      const shouldAttemptSilentLogin =
        userAgent.includes('Mozilla') && !referer.includes('/api/auth/callback') && !hasPkceCookie;

      return { isAuthenticated: false, user: null, token: null, shouldAttemptSilentLogin };
    }

    if (!config.jwtSecret) {
      console.error('JWT secret not configured');
      return { isAuthenticated: false, user: null, token: null };
    }

    const secret = new TextEncoder().encode(config.jwtSecret);
    const { payload } = await jwtVerify(oidcToken, secret, { algorithms: ['HS256'] });
    const decoded = payload as unknown as DecodedOidcToken;

    return {
      isAuthenticated: true,
      user: {
        sub: decoded.sub,
        name: decoded.name,
        email: decoded.email,
        picture: decoded.picture,
        email_verified: decoded.email_verified,
        updated_at: decoded.updated_at,
        username: decoded.username,
      },
      token: null,
    };
  } catch (error) {
    console.error('Auth user error:', error);
    return { isAuthenticated: false, user: null, token: null };
  }
});
