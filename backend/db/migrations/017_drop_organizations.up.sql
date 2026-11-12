-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Donations and subscriptions own their org claim (org_name, org_domain) since
-- #365, and the prod backfill (#408) copied it onto every legacy row. Nothing
-- reads the per-user organizations model any more (#441).
-- Dropping the columns also drops their FKs and idx_*_org_id indexes.
ALTER TABLE donations
  DROP COLUMN IF EXISTS organization_id;

ALTER TABLE subscriptions
  DROP COLUMN IF EXISTS organization_id;

DROP TABLE IF EXISTS organizations;

COMMIT;
