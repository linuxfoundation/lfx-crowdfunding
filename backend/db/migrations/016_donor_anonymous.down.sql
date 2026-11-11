-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

DROP INDEX IF EXISTS idx_subscriptions_anonymous_initiative;
DROP INDEX IF EXISTS idx_donations_anonymous_initiative;
DROP INDEX IF EXISTS idx_donations_stripe_charge_id;

ALTER TABLE subscriptions
  DROP COLUMN IF EXISTS anonymous;

ALTER TABLE donations
  DROP COLUMN IF EXISTS anonymous;

COMMIT;
