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
    const hasCards = await page.locator('[data-testid="initiative-card"]').count();
    const hasEmpty = await page.getByText(/no initiatives/i).count();
    expect(hasCards + hasEmpty).toBeGreaterThan(0);
  });
});
