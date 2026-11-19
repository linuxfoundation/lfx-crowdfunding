-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

DROP INDEX IF EXISTS idx_legacy_org_logos_org_domain;
ALTER TABLE legacy_org_logos DROP COLUMN IF EXISTS org_domain;

COMMIT;
