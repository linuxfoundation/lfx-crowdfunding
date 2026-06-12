// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '../fixtures/auth';

test.describe('Initiative CRUD (authenticated)', () => {
  test('fundraise page loads for authenticated user', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/fundraise');
    await expect(authenticatedPage).not.toHaveURL(/error/);
    await expect(authenticatedPage.locator('body')).toBeVisible();
  });

  test('initiative detail page loads for published initiative', async ({ page }) => {
    await page.goto('/initiatives');
    await page.waitForLoadState('networkidle');

    const firstCard = page.locator('a[href^="/initiatives/"]').first();
    if (await firstCard.count() === 0) {
      test.skip();
      return;
    }

    await firstCard.click();
    await expect(page).not.toHaveURL(/error/);
    await expect(page.locator('body')).toBeVisible();
  });

  test('non-owner cannot see edit controls on an initiative', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/initiatives');
    await authenticatedPage.waitForLoadState('networkidle');

    const firstCard = authenticatedPage.locator('a[href^="/initiatives/"]').first();
    if (await firstCard.count() === 0) {
      test.skip();
      return;
    }
    await firstCard.click();
    await authenticatedPage.waitForLoadState('networkidle');

    // Edit controls should NOT be visible for initiatives not owned by test user
    await expect(authenticatedPage.getByRole('button', { name: /edit/i })).not.toBeVisible();
  });
});
