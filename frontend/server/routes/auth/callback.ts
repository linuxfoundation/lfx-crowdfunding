// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { discovery, authorizationCodeGrant, ClientSecretPost } from 'openid-client';
import { decodeJwt } from 'jose';
import { H3Error } from 'h3';
import { getSafeRedirectUrl } from '../../utils/redirect';
import type { DecodedIdToken } from '~~/types/auth/auth-jwt.types';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const query = getQuery(event);
  const isLocal = !process.env.NUXT_PUBLIC_APP_ENV;

  const redirectTo = getSafeRedirectUrl(getCookie(event, 'auth_redirect_to'));

  const clearCookieOptions = {
    httpOnly: true,
    secure: !isLocal,
    sameSite: 'lax' as const,
    path: '/',
    ...(isLocal ? { domain: 'localhost' } : { domain: config.auth0CookieDomain }),
    maxAge: 0,
  };

  try {
    // Handle Auth0 errors (e.g. silent auth failure)
    if (query.error) {
      const error = query.error as string;
      const errorDescription = query.error_description as string;

      if (error === 'login_required' || error === 'interaction_required') {
        await sendRedirect(event, buildSuccessRedirect(redirectTo));
        return;
      }

      throw createError({
        statusCode: 400,
        statusMessage: `Authentication error: ${error} - ${errorDescription}`,
      });
    }

    // Parse PKCE cookie
    const pkceCookie = getCookie(event, 'auth_pkce');
    let storedState: string | undefined;
    let codeVerifier: string | undefined;

    if (pkceCookie) {
      try {
        const pkceData = JSON.parse(pkceCookie);
        storedState = pkceData.state;
        codeVerifier = pkceData.codeVerifier;
      } catch {
        // Invalid cookie format — handled below
      }
    }

    if (!query.code || !codeVerifier) {
      throw createError({
        statusCode: 400,
        statusMessage: 'Missing authorization code or code verifier',
      });
    }

    if (!storedState || storedState !== query.state) {
      throw createError({
        statusCode: 400,
        statusMessage: 'Invalid state parameter — please try logging in again',
      });
    }

    const authConfig = await discovery(
      new URL(`${config.public.auth0Domain}`),
      config.public.auth0ClientId,
      undefined,
      config.auth0ClientSecret ? ClientSecretPost(config.auth0ClientSecret) : undefined,
    );

    const callbackUrl = new URL(config.public.auth0RedirectUri);
    callbackUrl.search = new URL(getRequestURL(event)).search;

    const tokenResponse = await authorizationCodeGrant(authConfig, callbackUrl, {
      expectedState: storedState,
      pkceCodeVerifier: codeVerifier,
    });

    setCookie(event, 'auth_pkce', '', clearCookieOptions);
    setCookie(event, 'auth_redirect_to', '', clearCookieOptions);

    if (!tokenResponse.id_token) {
      throw createError({ statusCode: 500, statusMessage: 'No ID token received from Auth0' });
    }

    if (!tokenResponse.access_token) {
      throw createError({ statusCode: 500, statusMessage: 'No access token received from Auth0' });
    }

    const idTokenClaims = decodeJwt(tokenResponse.id_token) as DecodedIdToken;
    const expiresIn = tokenResponse.expires_in || 86400;

    const tokenCookieOptions = {
      httpOnly: true,
      secure: !isLocal,
      sameSite: 'lax' as const,
      path: '/',
      ...(isLocal ? { domain: 'localhost' } : { domain: config.auth0CookieDomain }),
      maxAge: expiresIn,
    };

    // Store the Auth0 access token — forwarded by the BFF as Authorization: Bearer to the Go backend.
    setCookie(event, 'auth_oidc_token', tokenResponse.access_token, tokenCookieOptions);

    // Store display-only profile claims for the /api/auth/user endpoint.
    // IMPORTANT: this cookie is base64-encoded JSON with no HMAC signature — treat as
    // display-only. Never use it for identity decisions or authorization; use auth_oidc_token
    // (forwarded as Authorization: Bearer to the Go backend) for that.
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
    setCookie(
      event,
      'auth_user_profile',
      Buffer.from(JSON.stringify(userProfile)).toString('base64'),
      tokenCookieOptions,
    );

    if (tokenResponse.refresh_token) {
      setCookie(event, 'auth_refresh_token', tokenResponse.refresh_token, {
        ...tokenCookieOptions,
        maxAge: 60 * 60 * 24 * 30, // 30 days
      });
    }

    await sendRedirect(event, buildSuccessRedirect(redirectTo));
  } catch (error) {
    console.error('Auth callback error:', error);

    setCookie(event, 'auth_pkce', '', clearCookieOptions);
    setCookie(event, 'auth_redirect_to', '', clearCookieOptions);
    setCookie(event, 'auth_oidc_token', '', clearCookieOptions);
    setCookie(event, 'auth_user_profile', '', clearCookieOptions);
    setCookie(event, 'auth_refresh_token', '', clearCookieOptions);

    let statusCode = 500;
    let statusMessage = 'Authentication callback error';
    if (error instanceof H3Error) {
      statusCode = error.statusCode ?? 500;
      statusMessage = error.statusMessage || error.message || statusMessage;
    }

    throw createError({ statusCode, statusMessage });
  }
});

function buildSuccessRedirect(redirectTo: string): string {
  return redirectTo === '/'
    ? '/?auth=success'
    : `${redirectTo}${redirectTo.includes('?') ? '&' : '?'}auth=success`;
}
