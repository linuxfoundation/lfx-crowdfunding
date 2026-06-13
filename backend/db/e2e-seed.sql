-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT
--
-- E2E test seed data. Run once after migrations in the e2e CI job.
-- Inserts a known user and a published initiative so Playwright specs
-- can navigate to deterministic URLs without conditional skips.
--
-- Known values referenced by frontend/e2e/fixtures/seed.ts:
--   username: e2e-test-user  (matches DISABLED_MOCK_LOCAL_PRINCIPAL in CI)
--   slug:     e2e-test-initiative

SET search_path TO crowdfunding, public;

INSERT INTO users (username, email, name)
VALUES ('e2e-test-user', 'e2e@example.com', 'E2E Test User')
ON CONFLICT (username) DO NOTHING;

INSERT INTO initiatives (
  id,
  initiative_type,
  owner_id,
  name,
  slug,
  status,
  description,
  accept_funding,
  stripe_product_id
)
SELECT
  'e2e0e2e0-e2e0-e2e0-e2e0-e2e0e2e0e2e0'::uuid,
  'project',
  u.id,
  'E2E Test Initiative',
  'e2e-test-initiative',
  'published',
  'A published initiative seeded for Playwright e2e tests.',
  true,
  'prod_e2e_placeholder'
FROM users u
WHERE u.username = 'e2e-test-user'
ON CONFLICT (id) DO NOTHING;
