// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { deleteCookie } from 'h3';
import type { H3Event } from 'h3';
import { isValidRedirectUrl } from '../../utils/redirect';

const isProduction = process.env.NUXT_APP_ENV === 'production';

function clearAuthCookies(event: H3Event) {
  const config = useRuntimeConfig();

  const opts = {
    httpOnly: true,
    secure: isProduction,
    sameSite: 'lax' as const,
    path: '/',
    ...(isProduction ? { domain: config.auth0CookieDomain } : { domain: 'localhost' }),
    maxAge: 0,
  };

  // Explicitly set to empty to ensure clearing works across proxy setups in production
  setCookie(event, 'auth_oidc_token', '', opts);
  deleteCookie(event, 'auth_refresh_token');
  deleteCookie(event, 'auth_pkce');
  deleteCookie(event, 'auth_redirect_to');
}

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();

  let returnToUrl = `${config.public.appUrl}?auth=logout`;

  try {
    const body = await readBody(event);
    if (body?.returnTo && isValidRedirectUrl(body.returnTo)) {
      const base = body.returnTo.startsWith('/')
        ? `${config.public.appUrl}${body.returnTo}`
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
