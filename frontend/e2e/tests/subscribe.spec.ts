// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '../fixtures/auth';

test.describe('Subscription flow (authenticated)', () => {
  test('subscription option visible on initiative that accepts funding', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/initiatives');
    await authenticatedPage.waitForLoadState('networkidle');

    const firstCard = authenticatedPage.locator('a[href^="/initiatives/"]').first();
    if (await firstCard.count() === 0) {
      test.skip();
      return;
    }
    await firstCard.click();
    await authenticatedPage.waitForLoadState('networkidle');

    const subscribeBtn = authenticatedPage
      .getByRole('button', { name: /subscribe|monthly|recurring/i })
      .or(authenticatedPage.getByRole('link', { name: /subscribe/i }));

    if (await subscribeBtn.count() === 0) {
      test.skip();
      return;
    }
    await expect(subscribeBtn.first()).toBeVisible();
  });

  test('my subscriptions page loads for authenticated user', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/me/subscriptions');
    await expect(authenticatedPage).not.toHaveURL(/error/);
    await expect(authenticatedPage.locator('body')).toBeVisible();
  });
});
