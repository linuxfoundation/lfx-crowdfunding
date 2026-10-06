// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { discovery, refreshTokenGrant, ClientSecretPost } from 'openid-client';
import { decodeJwt } from 'jose';
import type { DecodedIdToken } from '~~/types/auth/auth-jwt.types';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();

  const refreshToken = getAuthCookie(event, 'auth_refresh_token');
  if (!refreshToken) {
    // No refresh token stored — client must do a full login
    clearAuthCookies(event);
    throw createError({ statusCode: 401, statusMessage: 'No refresh token available' });
  }

  try {
    const authConfig = await discovery(
      new URL(`${config.public.auth0Domain}`),
      config.public.auth0ClientId,
      undefined,
      config.auth0ClientSecret ? ClientSecretPost(config.auth0ClientSecret) : undefined,
    );

    const tokenResponse = await refreshTokenGrant(authConfig, refreshToken, {
      scope: 'openid profile email access:me',
    });

    if (!tokenResponse.access_token) {
      throw createError({ statusCode: 500, statusMessage: 'No access token in refresh response' });
    }

    const expiresIn = tokenResponse.expires_in || 86400;
    // Replace the access token — this is what the BFF forwards as Authorization: Bearer.
    setAuthCookie(event, 'auth_oidc_token', tokenResponse.access_token, expiresIn);

    // Require the ID token and rebuild the profile cookie on every refresh.
    // auth_user_profile shares the access token's maxAge, so it expires alongside it;
    // /api/auth/user requires both cookies, and without a fresh profile cookie the app
    // appears logged out even though the access token is valid (half-authenticated state).
    if (!tokenResponse.id_token) {
      throw createError({ statusCode: 401, statusMessage: 'No ID token in refresh response' });
    }
    const idTokenClaims = decodeJwt(tokenResponse.id_token) as DecodedIdToken;
    const userProfile = {
      sub: idTokenClaims.sub,
      name: idTokenClaims.name,
      email: idTokenClaims.email,
      picture: idTokenClaims.picture,
      email_verified: idTokenClaims.email_verified,
      updated_at: idTokenClaims.updated_at,
      username: idTokenClaims['https://sso.linuxfoundation.org/claims/username'] as
        | string
        | undefined,
      intercomJwt: idTokenClaims['http://lfx.dev/claims/intercom'] as string | undefined,
    };
    setAuthCookie(
      event,
      'auth_user_profile',
      Buffer.from(JSON.stringify(userProfile)).toString('base64'),
      expiresIn,
    );

    // Persist a rotated refresh token when Auth0 issues one (refresh token rotation).
    if (tokenResponse.refresh_token) {
      setAuthCookie(
        event,
        'auth_refresh_token',
        tokenResponse.refresh_token,
        60 * 60 * 24 * 30, // 30 days
      );
    }

    return { success: true };
  } catch (error) {
    console.error('Auth refresh error:', error);

    // The refresh token is invalid or revoked — clear all auth cookies so the client
    // falls back to a full Auth0 login.
    clearAuthCookies(event);

    throw createError({ statusCode: 401, statusMessage: 'Token refresh failed' });
  }
});
