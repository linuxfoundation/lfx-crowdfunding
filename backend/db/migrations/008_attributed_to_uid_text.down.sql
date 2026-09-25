-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

ALTER TABLE initiatives
  ALTER COLUMN attributed_to_uid TYPE UUID USING attributed_to_uid::uuid;

COMMIT;
