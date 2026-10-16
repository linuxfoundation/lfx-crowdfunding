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

test.describe('Initiative detail', () => {
  // Non-published and unknown slugs both 404 for anonymous users. The server used to render an
  // empty page that the client hydrated as a spinner, logging a hydration mismatch.
  test('unknown slug renders a 404 without a hydration mismatch', async ({ page }) => {
    const hydrationErrors: string[] = [];
    page.on('console', (msg) => {
      if (/hydration/i.test(msg.text())) hydrationErrors.push(msg.text());
    });

    const response = await page.goto('/initiatives/e2e-does-not-exist');
    // Pages hydrate async inside <Suspense>; Nuxt clears isHydrating once that finishes.
    await page.waitForFunction(() => {
      const el = document.querySelector('#__nuxt') as {
        __vue_app__?: { config: { globalProperties: { $nuxt?: { isHydrating?: boolean } } } };
      } | null;
      return el?.__vue_app__?.config.globalProperties.$nuxt?.isHydrating === false;
    });

    expect(response?.status()).toBe(404);
    expect(hydrationErrors).toEqual([]);
  });
});
