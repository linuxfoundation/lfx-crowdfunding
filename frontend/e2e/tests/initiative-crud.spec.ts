// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '../fixtures/auth';
import { E2E_INITIATIVE_SLUG } from '../fixtures/seed';

test.describe('Initiative CRUD (authenticated)', () => {
  test('fundraise page loads for authenticated user', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/fundraise');
    await expect(authenticatedPage).not.toHaveURL(/error/);
    await expect(authenticatedPage.locator('body')).toBeVisible();
  });

  test('initiative detail page loads for published initiative', async ({ page }) => {
    await page.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await expect(page).not.toHaveURL(/error/);
    await expect(page.locator('body')).toBeVisible();
  });

  test('edit controls are not shown on initiative detail page', async ({ authenticatedPage }) => {
    // Edit controls (if any) live in the owner dashboard, not the public detail page.
    // Verify no edit button is rendered on the detail view regardless of ownership.
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('button', { name: /edit/i })).not.toBeVisible();
  });
});
