-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

ALTER TABLE initiative_sponsorship_tiers
  DROP COLUMN IF EXISTS benefits,
  DROP COLUMN IF EXISTS enabled;

ALTER TABLE initiatives
  DROP COLUMN IF EXISTS donation_mode;

COMMIT;
