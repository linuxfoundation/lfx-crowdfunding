// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '../fixtures/auth';
import { E2E_INITIATIVE_SLUG } from '../fixtures/seed';

test.describe('Subscription flow (authenticated)', () => {
  test('subscription option visible on initiative that accepts funding', async ({
    authenticatedPage,
  }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    const subscribeBtn = authenticatedPage
      .getByRole('button', { name: /subscribe|monthly|recurring/i })
      .or(authenticatedPage.getByRole('link', { name: /subscribe/i }));

    await expect(subscribeBtn.first()).toBeVisible();
  });

  test('my subscriptions page loads for authenticated user', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/me/subscriptions');
    await expect(authenticatedPage).not.toHaveURL(/error/);
    await expect(authenticatedPage.locator('body')).toBeVisible();
  });
});
