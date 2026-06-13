// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '../fixtures/auth';
import { E2E_INITIATIVE_SLUG } from '../fixtures/seed';

test.describe('Subscription flow (authenticated)', () => {
  test('monthly donation option visible in donate drawer', async ({ authenticatedPage }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    // Open the donate drawer — use force:true since the button may be inside a
    // tooltip wrapper that Playwright considers non-visible.
    const donateBtn = authenticatedPage
      .getByRole('button', { name: /donate/i })
      .or(authenticatedPage.getByRole('link', { name: /donate/i }))
      .first();
    await donateBtn.click({ force: true });

    // The drawer has a monthly radio option — verify it is present
    const monthlyRadio = authenticatedPage.locator('input[type="radio"][value="monthly"]');
    await expect(monthlyRadio).toBeAttached({ timeout: 10000 });
  });

  test('my subscriptions page loads for authenticated user', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/me/subscriptions');
    await expect(authenticatedPage).not.toHaveURL(/error/);
    await expect(authenticatedPage.locator('body')).toBeVisible();
  });
});
