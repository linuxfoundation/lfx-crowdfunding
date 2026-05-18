// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { discovery, authorizationCodeGrant } from 'openid-client';
import { SignJWT, decodeJwt } from 'jose';
import { H3Error } from 'h3';
import { getSafeRedirectUrl } from '../../utils/redirect';
import type { DecodedIdToken } from '~~/types/auth/auth-jwt.types';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const query = getQuery(event);
  const isProduction = process.env.NUXT_APP_ENV === 'production';

  const redirectTo = getSafeRedirectUrl(getCookie(event, 'auth_redirect_to'));

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
    );

    // Reconstruct the callback URL with original query params to avoid
    // mismatch from the /auth/callback -> /api/auth/callback route redirect
    const callbackUrl = new URL(config.public.auth0RedirectUri);
    callbackUrl.search = new URL(getRequestURL(event)).search;

    const tokenResponse = await authorizationCodeGrant(authConfig, callbackUrl, {
      expectedState: storedState,
      pkceCodeVerifier: codeVerifier,
    });

    deleteCookie(event, 'auth_pkce');
    deleteCookie(event, 'auth_redirect_to');

    if (!config.jwtSecret) {
      throw createError({ statusCode: 500, statusMessage: 'JWT secret not configured' });
    }

    if (!tokenResponse.id_token) {
      throw createError({ statusCode: 500, statusMessage: 'No ID token received from Auth0' });
    }

    const idTokenClaims = decodeJwt(tokenResponse.id_token) as DecodedIdToken;

    const jwtPayload = {
      sub: idTokenClaims.sub,
      name: idTokenClaims.name,
      email: idTokenClaims.email,
      picture: idTokenClaims.picture,
      email_verified: idTokenClaims.email_verified,
      updated_at: idTokenClaims.updated_at,
      username: idTokenClaims['https://sso.linuxfoundation.org/claims/username'] as
        | string
        | undefined,
      iss: config.public.auth0Domain,
    };

    const secret = new TextEncoder().encode(config.jwtSecret);
    const expiresIn = tokenResponse.expires_in || 86400;

    const oidcToken = await new SignJWT(jwtPayload)
      .setProtectedHeader({ alg: 'HS256' })
      .setIssuedAt()
      .setExpirationTime(Math.floor(Date.now() / 1000) + expiresIn)
      .sign(secret);

    const tokenCookieOptions = {
      httpOnly: true,
      secure: isProduction,
      sameSite: 'lax' as const,
      path: '/',
      ...(isProduction ? { domain: config.auth0CookieDomain } : { domain: 'localhost' }),
      maxAge: expiresIn,
    };

    setCookie(event, 'auth_oidc_token', oidcToken, tokenCookieOptions);

    if (tokenResponse.refresh_token) {
      setCookie(event, 'auth_refresh_token', tokenResponse.refresh_token, {
        ...tokenCookieOptions,
        maxAge: 60 * 60 * 24 * 30, // 30 days
      });
    }

    await sendRedirect(event, buildSuccessRedirect(redirectTo));
  } catch (error) {
    console.error('Auth callback error:', error);

    deleteCookie(event, 'auth_pkce');
    deleteCookie(event, 'auth_redirect_to');
    deleteCookie(event, 'auth_oidc_token');
    deleteCookie(event, 'auth_refresh_token');

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
