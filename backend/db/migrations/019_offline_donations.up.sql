-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Offline org donations (invoice or wire through LF finance) recorded by an
-- approver (#464). They use payment_method 'offline' and the offline account
-- below as user_id.
--   source_txn_id: the Ledger sourceTxnID the approver gives; the Ledger drops
--                  a second row with the same one, so it must be unique here too.
--   received_on:   the date LF finance received the money (the Ledger txnDate).
--   recorded_by:   the approver who recorded it.
ALTER TABLE donations
  ADD COLUMN IF NOT EXISTS source_txn_id TEXT NULL,
  ADD COLUMN IF NOT EXISTS received_on DATE NULL,
  ADD COLUMN IF NOT EXISTS recorded_by UUID NULL REFERENCES users(id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_source_txn_id
  ON donations (source_txn_id) WHERE source_txn_id IS NOT NULL;

-- The account offline donations belong to; the Org lens shows its name as the
-- donor. The username can't be a login principal (':' fails the LF username
-- rule), so a sign-in never updates or claims it. No legacy_user_id: the Ledger
-- side uses the shared auth0|communitybridge subject, and binding it here would
-- block that account's own CF profile sync on the unique legacy_user_id.
INSERT INTO users (username, email, name)
VALUES ('lf:offline-donations', 'admin@communitybridge.org', 'LF Finance (offline donation)')
ON CONFLICT (username) DO NOTHING;

COMMIT;
