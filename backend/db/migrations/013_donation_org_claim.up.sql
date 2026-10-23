-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- The donation owns its org claim (option D, #365): name and normalized domain
-- as the donor stated them. Replaces organization_id for new rows.
ALTER TABLE donations
  ADD COLUMN IF NOT EXISTS org_name TEXT NULL,
  ADD COLUMN IF NOT EXISTS org_domain TEXT NULL;

ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS org_name TEXT NULL,
  ADD COLUMN IF NOT EXISTS org_domain TEXT NULL;

COMMIT;
