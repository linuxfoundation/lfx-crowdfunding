// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { randomBytes } from 'node:crypto';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const query = getQuery(event);
  const redirectTo = (query.redirectTo as string) || '/fundraise';

  const state = randomBytes(16).toString('hex');

  const isLocal =
    process.env.NUXT_APP_ENV !== 'staging' && process.env.NUXT_APP_ENV !== 'production';

  const cookieOptions = {
    httpOnly: true,
    secure: !isLocal,
    sameSite: 'lax' as const,
    path: '/',
    maxAge: 600, // 10 minutes
  };

  setCookie(event, 'github_oauth_state', state, cookieOptions);
  setCookie(event, 'github_redirect_to', redirectTo, cookieOptions);

  const params = new URLSearchParams({
    client_id: config.public.githubOauthClientId,
    redirect_uri: config.public.githubOauthRedirectUri,
    scope: 'public_repo',
    state,
  });

  await sendRedirect(event, `https://github.com/login/oauth/authorize?${params.toString()}`);
});
