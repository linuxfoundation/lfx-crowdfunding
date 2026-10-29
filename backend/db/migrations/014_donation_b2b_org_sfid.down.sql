-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

ALTER TABLE donations
  DROP COLUMN IF EXISTS b2b_org_sfid;

ALTER TABLE subscriptions
  DROP COLUMN IF EXISTS b2b_org_sfid;

COMMIT;
