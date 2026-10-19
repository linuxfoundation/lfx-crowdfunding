// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';
import { discovery, refreshTokenGrant, ClientSecretPost } from 'openid-client';
import { decodeJwt } from 'jose';
import type { UserProfile } from '../types/auth.types';
import { clearAuthCookies, getAuthCookie, setAuthCookie } from './auth-cookies';
import type { DecodedIdToken } from '~~/types/auth/auth-jwt.types';

const REFRESH_TOKEN_MAX_AGE = 60 * 60 * 24 * 30; // 30 days

export function buildUserProfile(idToken: string): UserProfile {
  const claims = decodeJwt(idToken) as DecodedIdToken;
  return {
    sub: claims.sub,
    name: claims.name,
    email: claims.email,
    picture: claims.picture,
    email_verified: claims.email_verified,
    updated_at: claims.updated_at,
    username: claims['https://sso.linuxfoundation.org/claims/username'] as string | undefined,
    intercomJwt: claims['http://lfx.dev/claims/intercom'] as string | undefined,
  };
}

// Stores the access token, profile and (rotated) refresh token cookies from a token response.
// auth_oidc_token and auth_user_profile share the access token's maxAge so they expire together.
export function setSessionCookies(
  event: H3Event,
  tokens: { access_token: string; expires_in?: number; refresh_token?: string },
  profile: UserProfile,
) {
  const expiresIn = tokens.expires_in || 86400;
  // Forwarded by the BFF as Authorization: Bearer to the Go backend.
  setAuthCookie(event, 'auth_oidc_token', tokens.access_token, expiresIn);
  // IMPORTANT: base64 JSON of unverified id-token claims — display-only. Never use it for
  // identity or authorization decisions; use auth_oidc_token for that.
  setAuthCookie(
    event,
    'auth_user_profile',
    Buffer.from(JSON.stringify(profile)).toString('base64'),
    expiresIn,
  );
  if (tokens.refresh_token) {
    setAuthCookie(event, 'auth_refresh_token', tokens.refresh_token, REFRESH_TOKEN_MAX_AGE);
  }
}

// Exchanges the stored refresh token for a fresh session. Returns the new profile, or null
// (with all auth cookies cleared) when there is no refresh token or Auth0 rejects it.
export async function refreshSession(event: H3Event): Promise<UserProfile | null> {
  const refreshToken = getAuthCookie(event, 'auth_refresh_token');
  if (!refreshToken) {
    clearAuthCookies(event);
    return null;
  }

  try {
    const config = useRuntimeConfig();
    const authConfig = await discovery(
      new URL(`${config.public.auth0Domain}`),
      config.public.auth0ClientId,
      undefined,
      config.auth0ClientSecret ? ClientSecretPost(config.auth0ClientSecret) : undefined,
    );

    const tokenResponse = await refreshTokenGrant(authConfig, refreshToken, {
      scope: 'openid profile email access:me',
    });

    // Both are required: /api/auth/user needs the profile and token cookies together, otherwise
    // the app is half-authenticated.
    if (!tokenResponse.access_token || !tokenResponse.id_token) {
      throw new Error('Refresh response missing access or ID token');
    }

    const profile = buildUserProfile(tokenResponse.id_token);
    setSessionCookies(event, tokenResponse, profile);
    return profile;
  } catch (error) {
    console.error('Auth refresh error:', error);
    // Refresh token invalid or revoked — fall back to a full Auth0 login.
    clearAuthCookies(event);
    return null;
  }
}
