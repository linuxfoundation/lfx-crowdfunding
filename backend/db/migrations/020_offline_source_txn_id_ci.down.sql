-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_source_txn_id
  ON donations (source_txn_id) WHERE source_txn_id IS NOT NULL;

DROP INDEX IF EXISTS idx_donations_source_txn_id_lower;

COMMIT;
