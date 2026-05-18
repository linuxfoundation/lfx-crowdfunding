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
  const isProduction = process.env.NUXT_APP_ENV === 'production';

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
      secure: isProduction,
      sameSite: 'lax' as const,
      path: '/',
      maxAge: 60 * 15, // 15 minutes
      ...(!isProduction && { domain: 'localhost' }),
    };

    // Store state + code verifier together to prevent concurrent login flow races
    setCookie(event, 'auth_pkce', JSON.stringify({ state, codeVerifier }), cookieOptions);

    const redirectTo = query.redirectTo as string;
    if (redirectTo) {
      const safeRedirectTo = getSafeRedirectUrl(redirectTo);
      if (isValidRedirectUrl(redirectTo)) {
        setCookie(event, 'auth_redirect_to', safeRedirectTo, cookieOptions);
      }
    }

    const authorizationUrl = buildAuthorizationUrl(authConfig, {
      scope: 'openid profile email',
      state,
      code_challenge: codeChallenge,
      code_challenge_method: 'S256',
      redirect_uri: config.public.auth0RedirectUri,
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
