-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

ALTER TABLE donations
  DROP COLUMN IF EXISTS donation_tier;

COMMIT;