// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test as base, type Page } from '@playwright/test';

const MOCK_USER = {
  sub: 'auth0|test-user-e2e',
  name: 'E2E Test User',
  email: 'e2e-test@linuxfoundation.org',
  picture: '',
  email_verified: true,
  updated_at: new Date().toISOString(),
  username: 'e2e-test',
};

// A fake token string — only valid when the backend runs with both:
//   DISABLED_MOCK_LOCAL_PRINCIPAL=<any non-empty string>
//   ALLOW_MOCK_LOCAL_PRINCIPAL_BYPASS=true
// The Nuxt BFF require-auth middleware checks for cookie presence only; full JWT validation
// is delegated to the Go backend and bypassed with those flags.
const MOCK_TOKEN = 'e2e-mock-token.placeholder.signature';

async function injectAuthCookies(page: Page): Promise<void> {
  const baseURL = process.env.E2E_BASE_URL ?? 'http://localhost:3000';
  const url = new URL(baseURL);
  const domain = url.hostname;
  const secure = url.protocol === 'https:';

  const profileBase64 = Buffer.from(JSON.stringify(MOCK_USER)).toString('base64');
  await page.context().addCookies([
    {
      name: 'auth_oidc_token',
      value: MOCK_TOKEN,
      domain,
      path: '/',
      httpOnly: true,
      secure,
      sameSite: 'Lax',
    },
    {
      name: 'auth_user_profile',
      value: profileBase64,
      domain,
      path: '/',
      httpOnly: true,
      secure,
      sameSite: 'Lax',
    },
  ]);
}

export const test = base.extend<{ authenticatedPage: Page }>({
  authenticatedPage: async ({ page }, use) => {
    // Inject cookies before navigating so the first request is authenticated.
    await injectAuthCookies(page);
    await page.goto('/');
    await use(page);
  },
});

export { expect } from '@playwright/test';
