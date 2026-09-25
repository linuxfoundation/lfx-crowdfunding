-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- organization attribution values are SFIDs and don't fit UUID — clear them
-- (and their type, to satisfy initiatives_attribution_uid_ck) before the
-- cast. This rollback is for reverting an unreleased migration in dev, not
-- for use once real org attribution data exists.
UPDATE initiatives
  SET attributed_to_type = 'personal', attributed_to_uid = NULL
  WHERE attributed_to_type = 'organization';

ALTER TABLE initiatives
  ALTER COLUMN attributed_to_uid TYPE UUID USING attributed_to_uid::uuid;

COMMIT;
