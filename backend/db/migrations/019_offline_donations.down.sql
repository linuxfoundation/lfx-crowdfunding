-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

DROP INDEX IF EXISTS idx_donations_source_txn_id;

ALTER TABLE donations
  DROP COLUMN IF EXISTS recorded_by,
  DROP COLUMN IF EXISTS received_on,
  DROP COLUMN IF EXISTS source_txn_id;

-- Kept while offline donations still point at it.
DELETE FROM users u
WHERE u.username = 'lf:offline-donations'
  AND NOT EXISTS (SELECT 1 FROM donations d WHERE d.user_id = u.id);

COMMIT;
