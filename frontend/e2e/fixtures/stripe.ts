// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import type { Page } from '@playwright/test';

export interface StripeCard {
  number: string;
  expiry: string; // MM/YYYY
  cvc: string;
}

// Stripe test cards used in E2E suites.
// Full list: https://docs.stripe.com/testing#cards
export const STRIPE_CARDS = {
  // Always succeeds
  valid: { number: '4242424242424242', expiry: '12/2030', cvc: '123' },
  // Card is expired
  expired: { number: '4000000000000069', expiry: '12/2030', cvc: '123' },
  // Zip check fails but charge still succeeds (AVS mismatch accepted by default)
  incorrectZip: { number: '4000000000000010', expiry: '12/2030', cvc: '123' },
  // Requires 3D Secure 2 authentication
  threeDSecure: { number: '4000000000003220', expiry: '12/2030', cvc: '123' },
  // Generic decline
  declined: { number: '4000000000000002', expiry: '12/2030', cvc: '123' },
} satisfies Record<string, StripeCard>;

/**
 * Fill the Stripe card element iframes on the donate payment step.
 *
 * All three fields (card number, expiry, CVC) live inside a single cross-origin
 * Stripe iframe titled "Secure card number input frame". They are accessed via
 * Playwright's frameLocator and filled with their internal `name` attributes.
 */
export async function fillStripeCard(page: Page, card: StripeCard): Promise<void> {
  const [month, year] = card.expiry.split('/');
  const frame = page.frameLocator('iframe[title="Secure card number input frame"]');
  await frame.locator('[name="cardnumber"]').fill(card.number);
  await frame.locator('[name="cc-exp-month"]').fill(month);
  await frame.locator('[name="cc-exp-year"]').fill(year);
  await frame.locator('[name="cc-csc"]').fill(card.cvc);
}

/**
 * Complete a Stripe 3DS test challenge.
 * The Stripe test ACS page has COMPLETE and FAIL buttons inside a nested iframe.
 */
export async function complete3DS(page: Page, action: 'complete' | 'fail'): Promise<void> {
  const label = action === 'complete' ? 'COMPLETE' : 'FAIL';
  // The 3DS challenge loads in an iframe on the page; search all frames
  for (const frame of page.frames()) {
    const btn = frame.locator(`button:has-text("${label}")`);
    if ((await btn.count()) > 0) {
      await btn.click();
      return;
    }
  }
  throw new Error(`3DS ${label} button not found in any frame`);
}
