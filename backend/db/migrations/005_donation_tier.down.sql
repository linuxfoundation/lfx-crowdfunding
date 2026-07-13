-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

ALTER TABLE donations
  DROP COLUMN IF EXISTS donation_tier;

COMMIT;