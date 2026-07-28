-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Add donation_mode to initiatives.
ALTER TABLE initiatives
  ADD COLUMN IF NOT EXISTS donation_mode TEXT NOT NULL DEFAULT 'open'
    CHECK (donation_mode IN ('tiers', 'open'));

-- Add enabled + benefits to initiative_sponsorship_tiers.
ALTER TABLE initiative_sponsorship_tiers
  ADD COLUMN IF NOT EXISTS enabled  BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS benefits TEXT[]  NOT NULL DEFAULT '{}';

COMMIT;
