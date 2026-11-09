-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

DROP INDEX IF EXISTS idx_subscriptions_b2b_org_sfid;
DROP INDEX IF EXISTS idx_donations_b2b_org_sfid;

ALTER TABLE subscriptions
  DROP COLUMN IF EXISTS first_paid_at;

COMMIT;
