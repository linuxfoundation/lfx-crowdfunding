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

// A fake token string — only valid when backend runs with DISABLED_MOCK_LOCAL_PRINCIPAL=true.
// The Nuxt BFF require-auth middleware checks for cookie presence only; full JWT validation
// is delegated to the Go backend and skipped with the mock flag.
const MOCK_TOKEN = 'e2e-mock-token.placeholder.signature';

async function injectAuthCookies(page: Page): Promise<void> {
  const profileBase64 = Buffer.from(JSON.stringify(MOCK_USER)).toString('base64');
  await page.context().addCookies([
    {
      name: 'auth_oidc_token',
      value: MOCK_TOKEN,
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      secure: false,
      sameSite: 'Lax',
    },
    {
      name: 'auth_user_profile',
      value: profileBase64,
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      secure: false,
      sameSite: 'Lax',
    },
  ]);
}

export const test = base.extend<{ authenticatedPage: Page }>({
  authenticatedPage: async ({ page }, use) => {
    await page.goto('/');
    await injectAuthCookies(page);
    await use(page);
  },
});

export { expect } from '@playwright/test';
