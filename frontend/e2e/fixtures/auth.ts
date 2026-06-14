// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test as base, type Page } from '@playwright/test';

// Suppress the Osano cookie consent banner by setting its localStorage consent record.
// The banner reads this key on every page load; pre-setting it prevents the overlay
// from appearing and intercepting pointer events on footer-area buttons.
async function dismissCookieBanner(page: Page): Promise<void> {
  await page.addInitScript(() => {
    window.localStorage.setItem(
      'osano_consentmanager',
      JSON.stringify({ analytics: 'ACCEPT', marketing: 'ACCEPT', essential: 'ACCEPT' }),
    );
  });
}

// loginAsTestUser calls the e2e-auth endpoint to set auth cookies,
// then navigates home to establish the session.
export async function loginAsTestUser(page: Page): Promise<void> {
  await dismissCookieBanner(page);
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
