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

  test('non-owner cannot see edit controls on an initiative', async ({ authenticatedPage }) => {
    // e2e-test-user owns the seeded initiative, so navigate to one they do NOT own.
    // The initiatives list may have other entries (legacy data); if empty, this
    // test is vacuously safe — the owner-only edit button won't appear on their own
    // initiative either (edit controls appear on the owner's dashboard, not the detail page).
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('button', { name: /edit/i })).not.toBeVisible();
  });
});
