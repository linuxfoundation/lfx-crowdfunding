-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- lfx-crowdfunding#346: the attributed entity's name and logo, captured at
-- attribution time for the public source label. b2b_org is non-public in the
-- query service, so anonymous viewers can't look these up live. NULL for
-- personal rows (no backfill). A later rename shows only after re-attribution.
ALTER TABLE initiatives
  ADD COLUMN IF NOT EXISTS attributed_to_name     TEXT,
  ADD COLUMN IF NOT EXISTS attributed_to_logo_url TEXT;

COMMIT;
