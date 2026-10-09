-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

ALTER TABLE initiatives
  DROP COLUMN IF EXISTS attributed_to_logo_url,
  DROP COLUMN IF EXISTS attributed_to_name;

COMMIT;
