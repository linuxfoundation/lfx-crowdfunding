-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Money an approver takes out of an initiative in the Ledger (#489), the
-- counterpart of the offline donations in the donations table (019):
--   kind 'return': unused funds back to the donor org; posted as a credit
--                  with a negative amount and the org.
--   kind 'payout': paid out from a fund to paid_to; posted as a debit with a
--                  negative amount.
-- amount_cents is the positive amount the approver entered. source_txn_id is
-- the Ledger sourceTxnID, unique ignoring case here and, checked by the app
-- under the same advisory lock, across offline donations too: the Ledger
-- keeps one row per sourceTxnID.
CREATE TABLE IF NOT EXISTS ledger_adjustments (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind          TEXT NOT NULL CHECK (kind IN ('return', 'payout')),
  initiative_id UUID NOT NULL REFERENCES initiatives(id),
  amount_cents  BIGINT NOT NULL CHECK (amount_cents > 0),
  org_name      TEXT NULL,
  org_domain    TEXT NULL,
  b2b_org_sfid  TEXT NULL,
  paid_to       TEXT NULL,
  reason        TEXT NOT NULL,
  category      TEXT NOT NULL,
  adjusted_on   DATE NOT NULL,
  source_txn_id TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'succeeded')),
  recorded_by   UUID NOT NULL REFERENCES users(id),
  created_on    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_on    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (kind <> 'return' OR (org_name IS NOT NULL AND org_domain IS NOT NULL)),
  CHECK (kind <> 'payout' OR paid_to IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ledger_adjustments_source_txn_id_lower
  ON ledger_adjustments (lower(source_txn_id));

CREATE INDEX IF NOT EXISTS idx_ledger_adjustments_initiative
  ON ledger_adjustments (initiative_id, adjusted_on DESC);

DROP TRIGGER IF EXISTS set_updated_on ON ledger_adjustments;
CREATE TRIGGER set_updated_on BEFORE UPDATE ON ledger_adjustments
  FOR EACH ROW EXECUTE FUNCTION set_updated_on();

COMMIT;
