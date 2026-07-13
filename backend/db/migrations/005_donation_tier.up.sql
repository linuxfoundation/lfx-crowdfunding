-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Track which sponsorship tier (if any) a donor selected when donating.
-- Null means the donor gave without selecting a tier.
ALTER TABLE donations
  ADD COLUMN IF NOT EXISTS donation_tier TEXT NULL
    CHECK (donation_tier IN ('platinum', 'gold', 'silver', 'bronze'));

COMMIT;
