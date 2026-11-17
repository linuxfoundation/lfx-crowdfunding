-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Offline donation source references are unique ignoring case: finance treats
-- "Invoice 12" and "invoice 12" as one reference, but 019's index let both in,
-- so the same donation was recorded (and posted to the Ledger) twice.
--
-- Rows that already clash must be resolved by hand first (they are separate
-- Ledger rows, so deleting the CF row alone doesn't fix the totals). Stop with
-- a clear message rather than a bare unique violation.
DO $$
DECLARE
  clashes TEXT;
BEGIN
  SELECT string_agg(refs, '; ') INTO clashes
  FROM (
    SELECT string_agg(source_txn_id, ' / ' ORDER BY created_on) AS refs
    FROM donations
    WHERE source_txn_id IS NOT NULL
    GROUP BY lower(source_txn_id)
    HAVING count(*) > 1
  ) c;
  IF clashes IS NOT NULL THEN
    RAISE EXCEPTION 'offline donations whose source_txn_id differs only in case: %. '
      'Remove the extra CF row and its Ledger entry, then rerun.', clashes;
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_source_txn_id_lower
  ON donations (lower(source_txn_id)) WHERE source_txn_id IS NOT NULL;

DROP INDEX IF EXISTS idx_donations_source_txn_id;

COMMIT;
