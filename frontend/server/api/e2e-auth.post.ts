// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// This endpoint is ONLY active when NUXT_E2E_TEST_MODE=true.
// It sets the same auth cookies as the real Auth0 callback so Playwright
// tests can authenticate without running a real OAuth flow.
// NEVER enable this in production or staging.

export default defineEventHandler((event) => {
  if (process.env.NUXT_E2E_TEST_MODE !== 'true') {
    throw createError({ statusCode: 404, statusMessage: 'Not found' });
  }

  const cookieOptions = {
    httpOnly: true,
    secure: false,
    sameSite: 'lax' as const,
    path: '/',
    domain: 'localhost',
    maxAge: 3600,
  };

  // The backend runs with DISABLED_MOCK_LOCAL_PRINCIPAL=<username> in e2e mode,
  // so any non-empty token value will be forwarded and accepted.
  setCookie(event, 'auth_oidc_token', 'e2e-test-token', cookieOptions);

  const profile = {
    sub: 'e2e-test-user',
    name: 'E2E Test User',
    email: 'e2e@example.com',
    picture: '',
    email_verified: true,
    username: process.env.NUXT_E2E_TEST_USERNAME ?? 'e2e-test-user',
  };
  setCookie(
    event,
    'auth_user_profile',
    Buffer.from(JSON.stringify(profile)).toString('base64'),
    cookieOptions,
  );

  return { ok: true };
});
