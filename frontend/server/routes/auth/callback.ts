// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { discovery, authorizationCodeGrant, ClientSecretPost } from 'openid-client';
import { H3Error } from 'h3';
import { getSafeRedirectUrl } from '../../utils/redirect';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const query = getQuery(event);

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
      undefined,
      config.auth0ClientSecret ? ClientSecretPost(config.auth0ClientSecret) : undefined,
    );

    const callbackUrl = new URL(config.public.auth0RedirectUri);
    callbackUrl.search = new URL(getRequestURL(event)).search;

    const tokenResponse = await authorizationCodeGrant(authConfig, callbackUrl, {
      expectedState: storedState,
      pkceCodeVerifier: codeVerifier,
    });

    setCookie(event, 'auth_pkce', '', authCookieOptions(0));
    setCookie(event, 'auth_redirect_to', '', authCookieOptions(0));

    if (!tokenResponse.id_token) {
      throw createError({ statusCode: 500, statusMessage: 'No ID token received from Auth0' });
    }

    if (!tokenResponse.access_token) {
      throw createError({ statusCode: 500, statusMessage: 'No access token received from Auth0' });
    }

    setSessionCookies(event, tokenResponse, buildUserProfile(tokenResponse.id_token));

    await sendRedirect(event, buildSuccessRedirect(redirectTo));
  } catch (error) {
    console.error('Auth callback error:', error);

    clearAuthCookies(event);

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
