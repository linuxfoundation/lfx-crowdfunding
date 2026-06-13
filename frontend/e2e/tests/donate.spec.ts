// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '../fixtures/auth';
import { E2E_INITIATIVE_SLUG } from '../fixtures/seed';

test.describe('One-time donation flow (authenticated)', () => {
  test('donate button visible on published initiative with accept_funding=true', async ({
    authenticatedPage,
  }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    const donateBtn = authenticatedPage
      .getByRole('button', { name: /donate/i })
      .or(authenticatedPage.getByRole('link', { name: /donate/i }));

    await expect(donateBtn.first()).toBeVisible();
  });

  test('donation form opens when donate button clicked', async ({ authenticatedPage }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    const donateBtn = authenticatedPage.getByRole('button', { name: /donate/i }).first();
    await donateBtn.click();

    const amountInput = authenticatedPage
      .getByRole('spinbutton')
      .or(authenticatedPage.locator('input[type="number"]'))
      .first();
    await expect(amountInput).toBeVisible({ timeout: 5000 });
  });
});
