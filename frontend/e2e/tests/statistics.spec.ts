// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '@playwright/test';

test.describe('Statistics page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/statistics');
  });

  test('page loads without error', async ({ page }) => {
    await expect(page).not.toHaveURL(/error/);
    await expect(page.locator('body')).toBeVisible();
  });

  test('shows statistics label', async ({ page }) => {
    // "Statistics" renders as a <span>, not a heading role.
    // The <h1> contains dynamic text (e.g. "$X funds raised by Y supporters").
    await expect(page.getByText('Statistics', { exact: true })).toBeVisible({
      timeout: 10000,
    });
  });

  test('shows platform statistics container', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const statsSection = page.locator('.container').first();
    await expect(statsSection).toBeVisible();
  });
});
