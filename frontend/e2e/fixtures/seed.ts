// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Known values matching backend/db/e2e-seed.sql.
// These are inserted before the e2e suite runs (CI: "Seed e2e data" step;
// local: run `psql $DATABASE_URL -f backend/db/e2e-seed.sql` once).
export const E2E_INITIATIVE_SLUG = 'e2e-test-initiative';

// Dedicated DEV initiative for payment E2E tests.
// This initiative has a real Stripe test-mode product ID so full card flows
// (charge, 3DS, declines) can be exercised against the DEV environment.
// Run payment tests against DEV with:
//   E2E_BASE_URL=https://crowdfunding.dev.lfx.dev pnpm test:e2e --grep "@dev"
export const DEV_PAYMENT_INITIATIVE_SLUG = 'test-html-text';
