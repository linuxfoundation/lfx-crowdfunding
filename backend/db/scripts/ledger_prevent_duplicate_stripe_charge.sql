-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT
--
-- Trigger: ledger_no_duplicate_charge_txn
--
-- Prevents a second row being inserted (or updated into) the ledger table
-- with the same source_txn_id when that ID is a Stripe charge ID (ch_...).
--
-- Rationale: the Ledger service has no UNIQUE constraint on source_txn_id.
-- Stripe charge IDs are globally unique so two rows sharing the same ch_*
-- source_txn_id always represent a double-post bug. Numeric IDs (Expensify,
-- Jobspring) are not in scope here because they can legitimately appear
-- multiple times representing separate line items.
--
-- Apply to: ledger DB (both dev and prod)
-- Idempotent: uses CREATE OR REPLACE FUNCTION / TRIGGER … OR REPLACE

CREATE OR REPLACE FUNCTION ledger_prevent_duplicate_charge_txn()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF LEFT(NEW.source_txn_id, 3) = 'ch_' THEN
        IF EXISTS (
            SELECT 1
            FROM ledger
            WHERE source_txn_id = NEW.source_txn_id
              AND txn_id != NEW.txn_id
        ) THEN
            RAISE EXCEPTION
                'duplicate ledger entry rejected: source_txn_id % already exists (txn_id=%)',
                NEW.source_txn_id,
                (SELECT txn_id FROM ledger WHERE source_txn_id = NEW.source_txn_id LIMIT 1)
                USING ERRCODE = '23505'; -- unique_violation
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

-- OR REPLACE requires Postgres 14+. The Ledger RDS instance is >= 14.
CREATE OR REPLACE TRIGGER ledger_no_duplicate_charge_txn
BEFORE INSERT OR UPDATE ON ledger
FOR EACH ROW EXECUTE FUNCTION ledger_prevent_duplicate_charge_txn();
