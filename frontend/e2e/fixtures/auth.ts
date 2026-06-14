// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test as base, type Page } from '@playwright/test';

// loginAsTestUser calls the e2e-auth endpoint to set auth cookies,
// then navigates home to establish the session.
export async function loginAsTestUser(page: Page): Promise<void> {
  // Block the Osano cookie consent script entirely so the banner never renders.
  // The banner sits at the bottom of the viewport and intercepts pointer events on
  // footer-area buttons (Continue, Submit initiative). Blocking via route interception
  // is the only reliable approach — localStorage/cookie tricks don't work because
  // Osano fetches its config from its own CDN and ignores pre-set local state.
  await page.route('**osano.com**', (route) => route.abort());

  const response = await page.request.post('/api/e2e-auth');
  if (!response.ok()) {
    throw new Error(
      `e2e-auth endpoint returned ${response.status()} — is NUXT_E2E_TEST_MODE=true set?`,
    );
  }
  await page.goto('/');
}

// test fixture that provides an authenticated page
export const test = base.extend<{ authenticatedPage: Page }>({
  authenticatedPage: async ({ page }, use) => {
    await loginAsTestUser(page);
    await use(page);
  },
});

export { expect } from '@playwright/test';
