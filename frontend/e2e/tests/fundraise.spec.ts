// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

import { test, expect } from '../fixtures/auth';

// The fundraise drawer is opened via the "Start Fundraise" button in the global header.
// General Fund is the simplest flow: name + elevator pitch + topic → compliance checkboxes → submit.
// Validation: details step requires only a non-empty name; logo/beneficiaries/goal are optional.

test.describe('Fundraise form — General Fund (authenticated)', () => {
  test.beforeEach(async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/');
    await authenticatedPage.waitForLoadState('networkidle');
  });

  test('opens the fundraise drawer when Start Fundraise is clicked', async ({
    authenticatedPage,
  }) => {
    await authenticatedPage.getByRole('button', { name: 'Start Fundraise' }).first().click();
    await expect(authenticatedPage.getByText('Choose initiative type')).toBeVisible();
  });

  test('Continue button disabled until an initiative type is selected', async ({
    authenticatedPage,
  }) => {
    await authenticatedPage.getByRole('button', { name: 'Start Fundraise' }).first().click();

    const continueBtn = authenticatedPage.getByRole('button', { name: 'Continue' });
    await expect(continueBtn).toBeDisabled();

    await authenticatedPage.getByRole('button', { name: 'General Fund' }).click();
    await expect(continueBtn).toBeEnabled();
  });

  test('submits a General Fund initiative and shows the success screen', async ({
    authenticatedPage,
  }) => {
    // Open drawer
    await authenticatedPage.getByRole('button', { name: 'Start Fundraise' }).first().click();

    // Step 0: select General Fund
    await authenticatedPage.getByRole('button', { name: 'General Fund' }).click();
    await authenticatedPage.getByRole('button', { name: 'Continue' }).click();

    // Step 1a: initiative details — only name is required
    await expect(authenticatedPage.getByText('General Fund Details')).toBeVisible();

    await authenticatedPage
      .locator('input[placeholder="My project"]')
      .fill('E2E Test General Fund');
    await authenticatedPage
      .locator('textarea[placeholder="Briefly introduce your project..."]')
      .fill('An initiative created by the Playwright e2e suite.');

    // Select a topic from the custom dropdown (multi-select — clicking an option keeps it open).
    // After a topic is selected the trigger label changes to the chip text, so locate it by
    // the container's relative position rather than by accessible name.
    const topicContainer = authenticatedPage.locator('[class*="relative"]').filter({
      has: authenticatedPage.getByText('Select topic(s)'),
    });
    await topicContainer.getByRole('button').first().click(); // open
    await authenticatedPage.getByRole('button', { name: '3D' }).click();
    await topicContainer.getByRole('button').first().click(); // close (toggle)

    await authenticatedPage.getByRole('button', { name: 'Continue' }).click();

    // Step 1b: compliance — scope checkboxes to each section to avoid matching unrelated inputs
    await expect(authenticatedPage.getByText('Compliance Confirmation')).toBeVisible();

    const ofacSection = authenticatedPage
      .locator('.border')
      .filter({ hasText: 'Compliance Confirmation' });
    const termsSection = authenticatedPage
      .locator('.border')
      .filter({ hasText: 'Terms and Conditions' });
    await ofacSection.locator('input[type="checkbox"]').check({ force: true });
    await termsSection.locator('input[type="checkbox"]').check({ force: true });

    // Submit
    await authenticatedPage.getByRole('button', { name: 'Submit initiative' }).click();

    // Success screen
    await expect(authenticatedPage.getByText('Initiative submitted with success!')).toBeVisible({
      timeout: 15000,
    });
  });

  test('Continue button disabled on details step until name is filled', async ({
    authenticatedPage,
  }) => {
    await authenticatedPage.getByRole('button', { name: 'Start Fundraise' }).first().click();
    await authenticatedPage.getByRole('button', { name: 'General Fund' }).click();
    await authenticatedPage.getByRole('button', { name: 'Continue' }).click();

    // Name is empty — Continue should be disabled
    const continueBtn = authenticatedPage.getByRole('button', { name: 'Continue' });
    await expect(continueBtn).toBeDisabled();

    await authenticatedPage.locator('input[placeholder="My project"]').fill('My Fund');
    await expect(continueBtn).toBeEnabled();
  });

  test('Submit button disabled until both compliance checkboxes are checked', async ({
    authenticatedPage,
  }) => {
    await authenticatedPage.getByRole('button', { name: 'Start Fundraise' }).first().click();
    await authenticatedPage.getByRole('button', { name: 'General Fund' }).click();
    await authenticatedPage.getByRole('button', { name: 'Continue' }).click();

    await authenticatedPage.locator('input[placeholder="My project"]').fill('My Fund');
    await authenticatedPage.getByRole('button', { name: 'Continue' }).click();

    await expect(authenticatedPage.getByText('Compliance Confirmation')).toBeVisible();

    const submitBtn = authenticatedPage.getByRole('button', { name: 'Submit initiative' });
    await expect(submitBtn).toBeDisabled();

    const ofacSection = authenticatedPage
      .locator('.border')
      .filter({ hasText: 'Compliance Confirmation' });
    const termsSection = authenticatedPage
      .locator('.border')
      .filter({ hasText: 'Terms and Conditions' });
    await ofacSection.locator('input[type="checkbox"]').check({ force: true });
    await expect(submitBtn).toBeDisabled(); // still disabled — need both

    await termsSection.locator('input[type="checkbox"]').check({ force: true });
    await expect(submitBtn).toBeEnabled();
  });

  test('Cancel closes the drawer', async ({ authenticatedPage }) => {
    await authenticatedPage.getByRole('button', { name: 'Start Fundraise' }).first().click();
    await expect(authenticatedPage.getByText('Choose initiative type')).toBeVisible();

    await authenticatedPage.getByRole('button', { name: 'Cancel' }).click();
    await expect(authenticatedPage.getByText('Choose initiative type')).not.toBeVisible();
  });
});
