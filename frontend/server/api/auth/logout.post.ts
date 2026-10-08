// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { isValidRedirectUrl } from '../../utils/redirect';

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
