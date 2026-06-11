// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { H3Event } from 'h3';
import { isValidRedirectUrl } from '../../utils/redirect';

const isLocal = !process.env.NUXT_APP_ENV;

function clearAuthCookies(event: H3Event) {
  const config = useRuntimeConfig();

  const opts = {
    httpOnly: true,
    secure: !isLocal,
    sameSite: 'lax' as const,
    path: '/',
    ...(isLocal ? { domain: 'localhost' } : { domain: config.auth0CookieDomain }),
    maxAge: 0,
  };

  setCookie(event, 'auth_oidc_token', '', opts);
  setCookie(event, 'auth_user_profile', '', opts);
  setCookie(event, 'auth_refresh_token', '', opts);
  setCookie(event, 'auth_pkce', '', opts);
  setCookie(event, 'auth_redirect_to', '', opts);
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();

  const requestOrigin = getRequestURL(event).origin;
  let returnToUrl = `${requestOrigin}?auth=logout`;

  try {
    const body = await readBody(event);
    if (body?.returnTo && isValidRedirectUrl(body.returnTo)) {
      const base = body.returnTo.startsWith('/')
        ? `${requestOrigin}${body.returnTo}`
        : body.returnTo;
      returnToUrl = base.includes('?') ? `${base}&auth=logout` : `${base}?auth=logout`;
    }
  } catch {
    // Body parsing failed — use default returnTo
  }

  try {
    const auth0Domain = config.public.auth0Domain.replace(/^https?:\/\//, '');
    const logoutParams = new URLSearchParams({
      returnTo: returnToUrl,
      client_id: config.public.auth0ClientId,
    });

    const logoutUrl = `https://${auth0Domain}/v2/logout?${logoutParams}`;

    clearAuthCookies(event);
    return { success: true, logoutUrl };
  } catch (error) {
    console.error('Auth logout error:', error);
    clearAuthCookies(event);
    return { success: true, logoutUrl: returnToUrl };
  }
});
