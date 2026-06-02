// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { getSafeRedirectUrl } from '../../utils/redirect';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const query = getQuery(event);
  const isLocal = !process.env.NUXT_APP_ENV;

  const redirectTo = getSafeRedirectUrl(getCookie(event, 'github_redirect_to'), '/fundraise');

  deleteCookie(event, 'github_redirect_to');

  if (query.error) {
    const sep = redirectTo.includes('?') ? '&' : '?';
    const errorVal = encodeURIComponent(
      Array.isArray(query.error) ? query.error[0]! : String(query.error),
    );
    await sendRedirect(event, `${redirectTo}${sep}github_error=${errorVal}`);
    return;
  }

  const storedState = getCookie(event, 'github_oauth_state');
  deleteCookie(event, 'github_oauth_state');

  if (!query.code || !storedState || storedState !== query.state) {
    throw createError({ statusCode: 400, statusMessage: 'Invalid OAuth state or missing code' });
  }

  const tokenResponse = await $fetch<{ access_token?: string; error?: string }>(
    'https://github.com/login/oauth/access_token',
    {
      method: 'POST',
      headers: { Accept: 'application/json' },
      body: {
        client_id: config.public.githubOauthClientId,
        client_secret: config.githubOauthClientSecret,
        code: query.code,
        redirect_uri: config.public.githubOauthRedirectUri,
      },
    },
  );

  if (tokenResponse.error || !tokenResponse.access_token) {
    throw createError({
      statusCode: 401,
      statusMessage: tokenResponse.error || 'Failed to exchange GitHub OAuth code for token',
    });
  }

  setCookie(event, 'github_oauth_token', tokenResponse.access_token, {
    httpOnly: true,
    secure: !isLocal,
    sameSite: 'lax' as const,
    path: '/',
    maxAge: 60 * 60 * 8, // 8 hours
  });

  const sep = redirectTo.includes('?') ? '&' : '?';
  await sendRedirect(event, `${redirectTo}${sep}github_connected=true`);
});
