// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import {
  discovery,
  randomState,
  randomPKCECodeVerifier,
  calculatePKCECodeChallenge,
  buildAuthorizationUrl,
} from 'openid-client';
import { isValidRedirectUrl, getSafeRedirectUrl } from '../../utils/redirect';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const query = getQuery(event);
  const isLocal = !process.env.NUXT_PUBLIC_APP_ENV;

  try {
    const authConfig = await discovery(
      new URL(`${config.public.auth0Domain}`),
      config.public.auth0ClientId,
    );

    const state = randomState();
    const codeVerifier = randomPKCECodeVerifier();
    const codeChallenge = await calculatePKCECodeChallenge(codeVerifier);

    const cookieOptions = {
      httpOnly: true,
      secure: !isLocal,
      sameSite: 'lax' as const,
      path: '/',
      maxAge: 60 * 15, // 15 minutes
      ...(isLocal ? { domain: 'localhost' } : { domain: config.auth0CookieDomain }),
    };

    // Store state + code verifier together to prevent concurrent login flow races
    setCookie(event, 'auth_pkce', JSON.stringify({ state, codeVerifier }), cookieOptions);

    const redirectTo = query.redirectTo as string;
    if (redirectTo) {
      const safeRedirectTo = getSafeRedirectUrl(redirectTo);
      if (isValidRedirectUrl(redirectTo)) {
        const redirectUrl = new URL(safeRedirectTo, 'http://n');
        redirectUrl.searchParams.delete('auth');
        setCookie(
          event,
          'auth_redirect_to',
          redirectUrl.pathname + redirectUrl.search,
          cookieOptions,
        );
      }
    }

    const authorizationUrl = buildAuthorizationUrl(authConfig, {
      scope: 'openid profile email offline_access access:me',
      state,
      code_challenge: codeChallenge,
      code_challenge_method: 'S256',
      redirect_uri: config.public.auth0RedirectUri,
      ...(config.public.auth0Audience ? { audience: config.public.auth0Audience } : {}),
      ...(query.silent === 'true' && { prompt: 'none' }),
    });

    const userAgent = getHeader(event, 'user-agent') || '';
    const acceptHeader = getHeader(event, 'accept') || '';

    if (userAgent.includes('Mozilla') && acceptHeader.includes('text/html')) {
      await sendRedirect(event, authorizationUrl.toString());
    } else {
      return { success: true, authorizationUrl: authorizationUrl.toString() };
    }
  } catch (error) {
    console.error('Auth login error:', error);
    throw createError({ statusCode: 500, statusMessage: 'Authentication error' });
  }
});
