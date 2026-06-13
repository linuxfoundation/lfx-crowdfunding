// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Known values matching backend/db/e2e-seed.sql.
// These are inserted before the e2e suite runs (CI: "Seed e2e data" step;
// local: run `psql $DATABASE_URL -f backend/db/e2e-seed.sql` once).
export const E2E_INITIATIVE_SLUG = 'e2e-test-initiative';
