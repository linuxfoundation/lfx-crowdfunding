// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '@playwright/test';

test.describe('Initiatives list', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/initiatives');
  });

  test('page loads without error', async ({ page }) => {
    await expect(page).not.toHaveURL(/error/);
    await expect(page.locator('body')).toBeVisible();
  });

  test('shows initiatives header with search', async ({ page }) => {
    const searchInput = page.getByRole('searchbox').or(page.getByPlaceholder(/search/i));
    await expect(searchInput).toBeVisible({ timeout: 10000 });
  });

  test('shows initiative cards or empty state', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    // Cards render as <NuxtLink> anchors with hrefs like /initiatives/<slug>
    const hasCards = await page.locator('a[href^="/initiatives/"]').count();
    const hasEmpty = await page.getByText('No initiatives found.').count();
    expect(hasCards + hasEmpty).toBeGreaterThan(0);
  });
});
