-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- When the first invoice was paid (invoice.payment_succeeded). A canceled
-- subscription with no value here never took a payment, e.g. an abandoned 3DS
-- that Stripe expired as incomplete_expired (#413).
ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS first_paid_at TIMESTAMPTZ NULL;

-- ponytail: rows canceled before this column existed can't be told apart, so
-- they count as paid; a few never-paid ones stay on the Org lens list. Check
-- Stripe invoices per row if exact history is ever needed.
UPDATE subscriptions SET first_paid_at = created_on
WHERE status = 'canceled' AND first_paid_at IS NULL;

-- Org lens donations list filters on the matched b2b_org (#413).
CREATE INDEX IF NOT EXISTS idx_donations_b2b_org_sfid
  ON donations (b2b_org_sfid) WHERE b2b_org_sfid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_subscriptions_b2b_org_sfid
  ON subscriptions (b2b_org_sfid) WHERE b2b_org_sfid IS NOT NULL;

COMMIT;
