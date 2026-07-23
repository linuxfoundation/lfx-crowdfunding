// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { randomBytes } from 'node:crypto';
import { getGithubCallbackUrl } from '../../utils/github';
import { getSafeRedirectUrl } from '../../utils/redirect';

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig();
  const query = getQuery(event);
  const redirectTo = getSafeRedirectUrl(query.redirectTo as string | undefined, '/fundraise');

  const state = randomBytes(16).toString('hex');

  const isLocal = !process.env.NUXT_PUBLIC_APP_ENV;

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
    redirect_uri: getGithubCallbackUrl(event),
    scope: 'public_repo',
    state,
  });

  await sendRedirect(event, `https://github.com/login/oauth/authorize?${params.toString()}`);
});
