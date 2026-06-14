// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '../fixtures/auth';
import { fillStripeCard, complete3DS, STRIPE_CARDS } from '../fixtures/stripe';
import { E2E_INITIATIVE_SLUG, DEV_PAYMENT_INITIATIVE_SLUG } from '../fixtures/seed';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const isDevRun = process.env.E2E_BASE_URL?.includes('dev.lfx.dev');

/**
 * Open the donate drawer for the given slug, pick $10 one-time, fill contact,
 * and land on the payment step. Leaves the page ready for card entry.
 */
async function openDonateToPayment(
  page: Parameters<typeof fillStripeCard>[0],
  initiativeSlug: string,
  opts: { email?: string } = {},
) {
  await page.goto(`/initiatives/${initiativeSlug}`);
  await page.waitForLoadState('networkidle');

  // The Donate button has a <span> child that intercepts pointer events,
  // so use a JS dispatch to open the drawer reliably.
  await page.evaluate(() => {
    const btns = document.querySelectorAll('button');
    for (const b of btns) {
      if (b.textContent?.trim() === 'Donate') {
        b.click();
        return;
      }
    }
  });

  // Step 1 — amount
  await page.getByRole('button', { name: '$10', exact: true }).click();
  await page.getByRole('button', { name: 'Continue' }).click();

  // Step 2 — contact
  await page.locator('input[type="text"]').fill('E2E Test User');
  await page.locator('input[type="email"]').fill(opts.email ?? 'e2e@example.com');
  await page.getByRole('button', { name: 'Continue to Payment' }).click();
}

/** Click the primary Donate submit button inside the drawer. */
async function clickDrawerDonate(page: Parameters<typeof fillStripeCard>[0]) {
  await page.evaluate(() => {
    const donateBtn = Array.from(document.querySelectorAll('button')).find(
      (b) =>
        b.textContent?.trim() === 'Donate' &&
        b.className.includes('p-button-primary') &&
        !b.className.includes('pill'),
    );
    donateBtn?.click();
  });
}

// ---------------------------------------------------------------------------
// CI suite — runs in CI against the local stack (placeholder Stripe keys).
// Tests the drawer UI: visibility, step validation, Cancel.
// No real Stripe charges are made.
// ---------------------------------------------------------------------------

test.describe('Donate — UI flows (CI)', () => {
  test('Donate button is visible on a published initiative', async ({ authenticatedPage }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('button', { name: /donate/i }).first()).toBeVisible();
  });

  test('Donation drawer opens when Donate is clicked', async ({ authenticatedPage }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    await authenticatedPage.evaluate(() => {
      const btns = document.querySelectorAll('button');
      for (const b of btns) {
        if (b.textContent?.trim() === 'Donate') {
          b.click();
          return;
        }
      }
    });

    await expect(authenticatedPage.getByText('Enter a custom amount')).toBeVisible({
      timeout: 8000,
    });
  });

  test('Continue is disabled until an amount is selected', async ({ authenticatedPage }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    await authenticatedPage.evaluate(() => {
      const btns = document.querySelectorAll('button');
      for (const b of btns) {
        if (b.textContent?.trim() === 'Donate') {
          b.click();
          return;
        }
      }
    });

    const continueBtn = authenticatedPage.getByRole('button', { name: 'Continue' });
    await expect(continueBtn).toBeDisabled();
    await authenticatedPage.getByRole('button', { name: '$10', exact: true }).click();
    await expect(continueBtn).toBeEnabled();
  });

  test('Continue to Payment is disabled until name and email are filled', async ({
    authenticatedPage,
  }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`, {
      waitUntil: 'domcontentloaded',
    });
    await authenticatedPage.waitForLoadState('networkidle');

    await authenticatedPage.evaluate(() => {
      const btns = document.querySelectorAll('button');
      for (const b of btns) {
        if (b.textContent?.trim() === 'Donate') {
          b.click();
          return;
        }
      }
    });

    await authenticatedPage.getByRole('button', { name: '$10', exact: true }).click();
    await authenticatedPage.getByRole('button', { name: 'Continue' }).click();

    const continueToPayment = authenticatedPage.getByRole('button', {
      name: 'Continue to Payment',
    });
    await expect(continueToPayment).toBeDisabled();

    await authenticatedPage.locator('input[type="text"]').fill('Test User');
    await expect(continueToPayment).toBeDisabled(); // still needs email

    await authenticatedPage.locator('input[type="email"]').fill('test@example.com');
    await expect(continueToPayment).toBeEnabled();
  });

  test('Cancel closes the drawer', async ({ authenticatedPage }) => {
    await authenticatedPage.goto(`/initiatives/${E2E_INITIATIVE_SLUG}`);
    await authenticatedPage.waitForLoadState('networkidle');

    await authenticatedPage.evaluate(() => {
      const btns = document.querySelectorAll('button');
      for (const b of btns) {
        if (b.textContent?.trim() === 'Donate') {
          b.click();
          return;
        }
      }
    });

    await expect(authenticatedPage.getByText('Enter a custom amount')).toBeVisible({
      timeout: 8000,
    });
    await authenticatedPage.getByRole('button', { name: 'Cancel' }).click();
    await expect(authenticatedPage.getByText('Enter a custom amount')).not.toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// DEV payment suite — tagged @dev, skipped in CI.
// Runs against crowdfunding.dev.lfx.dev with real Stripe test-mode keys.
// The DEV frontend must have NUXT_E2E_TEST_MODE=true for the e2e-auth
// endpoint to be available (so the authenticatedPage fixture works).
//
// Run locally:
//   E2E_BASE_URL=https://crowdfunding.dev.lfx.dev pnpm test:e2e --grep "@dev"
// ---------------------------------------------------------------------------

test.describe('Donate — payment flows @dev', () => {
  test.skip(!isDevRun, 'Payment tests only run against the DEV environment (@dev)');

  test('valid card completes donation and shows thank-you screen @dev', async ({
    authenticatedPage,
  }) => {
    await openDonateToPayment(authenticatedPage, DEV_PAYMENT_INITIATIVE_SLUG);

    await authenticatedPage.getByText('Use a different card').click();
    await fillStripeCard(authenticatedPage, STRIPE_CARDS.valid);
    await clickDrawerDonate(authenticatedPage);

    await expect(authenticatedPage.getByText('Thank you for your donation!')).toBeVisible({
      timeout: 15000,
    });
  });

  test('expired card shows "Your card is expired" error @dev', async ({ authenticatedPage }) => {
    await openDonateToPayment(authenticatedPage, DEV_PAYMENT_INITIATIVE_SLUG);

    await authenticatedPage.getByText('Use a different card').click();
    await fillStripeCard(authenticatedPage, STRIPE_CARDS.expired);
    await clickDrawerDonate(authenticatedPage);

    await expect(authenticatedPage.getByText(/your card is expired/i)).toBeVisible({
      timeout: 10000,
    });
  });

  test('declined card shows error message @dev', async ({ authenticatedPage }) => {
    await openDonateToPayment(authenticatedPage, DEV_PAYMENT_INITIATIVE_SLUG);

    await authenticatedPage.getByText('Use a different card').click();
    await fillStripeCard(authenticatedPage, STRIPE_CARDS.declined);
    await clickDrawerDonate(authenticatedPage);

    await expect(authenticatedPage.getByText(/card.*declined|declined|do not honor/i)).toBeVisible({
      timeout: 10000,
    });
  });

  test('3DS card — approve — completes donation @dev', async ({ authenticatedPage }) => {
    await openDonateToPayment(authenticatedPage, DEV_PAYMENT_INITIATIVE_SLUG);

    await authenticatedPage.getByText('Use a different card').click();
    await fillStripeCard(authenticatedPage, STRIPE_CARDS.threeDSecure);
    await clickDrawerDonate(authenticatedPage);

    // Stripe test ACS may show the challenge twice
    for (let i = 0; i < 2; i++) {
      await authenticatedPage.waitForTimeout(2000);
      const frames = authenticatedPage.frames();
      let found = false;
      for (const frame of frames) {
        const btn = frame.locator('button:has-text("COMPLETE")');
        if ((await btn.count()) > 0) {
          await btn.click();
          found = true;
          break;
        }
      }
      if (!found) break;
    }

    await expect(authenticatedPage.getByText('Thank you for your donation!')).toBeVisible({
      timeout: 15000,
    });
  });

  test('3DS card — fail — shows authentication error @dev', async ({ authenticatedPage }) => {
    await openDonateToPayment(authenticatedPage, DEV_PAYMENT_INITIATIVE_SLUG);

    await authenticatedPage.getByText('Use a different card').click();
    await fillStripeCard(authenticatedPage, STRIPE_CARDS.threeDSecure);
    await clickDrawerDonate(authenticatedPage);

    await authenticatedPage.waitForTimeout(2000);
    await complete3DS(authenticatedPage, 'fail');

    await expect(
      authenticatedPage.getByText(/unable to authenticate|authentication.*failed/i),
    ).toBeVisible({ timeout: 10000 });
  });
});
