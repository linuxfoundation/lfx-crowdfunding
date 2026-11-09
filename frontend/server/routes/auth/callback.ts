// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { discovery, authorizationCodeGrant, ClientSecretPost } from 'openid-client';
import type { H3Event } from 'h3';
import { H3Error } from 'h3';
import type { AuthCallbackResult } from '../../types/auth.types';
import { getSafeRedirectUrl } from '../../utils/redirect';
import { isExpectedAuthError, logAuthError } from '../../utils/auth-log';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const query = getQuery(event);

  const redirectTo = getSafeRedirectUrl(getCookie(event, 'auth_redirect_to'));
  // Set when the request doesn't belong to a login started in this browser (bot, back button,
  // second tab). Nothing is exchanged then, so an existing session is left alone.
  let staleCallback = false;

  try {
    // Handle Auth0 errors (e.g. silent auth failure)
    if (query.error) {
      const error = query.error as string;
      const errorDescription = query.error_description as string;

      if (error === 'login_required' || error === 'interaction_required') {
        await sendRedirect(event, buildAuthRedirect(redirectTo, 'success'));
        return;
      }

      // `message`, not `statusMessage`: h3 warns on every statusMessage it has to sanitise, and
      // Auth0's description can contain characters it strips.
      throw createError({
        statusCode: 400,
        message: `Authentication error: ${error} - ${errorDescription}`,
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
      staleCallback = true;
      throw createError({
        statusCode: 400,
        message: 'Missing authorization code or code verifier',
      });
    }

    if (!storedState || storedState !== query.state) {
      staleCallback = true;
      throw createError({ statusCode: 400, message: 'Invalid state parameter' });
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

    clearLoginCookies(event);

    if (!tokenResponse.id_token) {
      throw createError({ statusCode: 500, statusMessage: 'No ID token received from Auth0' });
    }

    if (!tokenResponse.access_token) {
      throw createError({ statusCode: 500, statusMessage: 'No access token received from Auth0' });
    }

    setSessionCookies(event, tokenResponse, buildUserProfile(tokenResponse.id_token));

    await sendRedirect(event, buildAuthRedirect(redirectTo, 'success'));
  } catch (error) {
    logAuthError('Auth callback error', error);

    if (staleCallback) {
      clearLoginCookies(event);
    } else {
      clearAuthCookies(event);
    }

    // Expected failures send the user back to where they started; the client then shows a
    // toast asking them to log in again (unless they still have a session).
    if (isExpectedAuthError(error)) {
      await sendRedirect(event, buildAuthRedirect(redirectTo, 'error'));
      return;
    }

    let statusCode = 500;
    let statusMessage = 'Authentication callback error';
    if (error instanceof H3Error) {
      statusCode = error.statusCode ?? 500;
      statusMessage = error.statusMessage || error.message || statusMessage;
    }

    throw createError({ statusCode, statusMessage });
  }
});

function clearLoginCookies(event: H3Event) {
  setCookie(event, 'auth_pkce', '', authCookieOptions(0));
  setCookie(event, 'auth_redirect_to', '', authCookieOptions(0));
}

function buildAuthRedirect(redirectTo: string, result: AuthCallbackResult): string {
  return redirectTo === '/'
    ? `/?auth=${result}`
    : `${redirectTo}${redirectTo.includes('?') ? '&' : '?'}auth=${result}`;
}
