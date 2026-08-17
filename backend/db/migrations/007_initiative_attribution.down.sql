-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

DROP INDEX IF EXISTS idx_initiatives_attributed_to;

ALTER TABLE initiatives
  DROP CONSTRAINT IF EXISTS initiatives_attribution_uid_ck;

ALTER TABLE initiatives
  DROP COLUMN IF EXISTS attributed_to_type,
  DROP COLUMN IF EXISTS attributed_to_uid,
  DROP COLUMN IF EXISTS benefit_project_uid;

COMMIT;
