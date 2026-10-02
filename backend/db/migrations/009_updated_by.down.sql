-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

ALTER TABLE initiative_announcements DROP COLUMN IF EXISTS updated_by;
ALTER TABLE initiatives DROP COLUMN IF EXISTS updated_by;

COMMIT;
