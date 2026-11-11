-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- The donor asked to hide their name and email on donor lists (#466). Only
-- the person is hidden; an org donation keeps the org name. A subscription's
-- flag covers every charge it makes.
ALTER TABLE donations
  ADD COLUMN IF NOT EXISTS anonymous BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS anonymous BOOLEAN NOT NULL DEFAULT false;

-- Ledger rows carry the Stripe charge ID (sourceTxnID); match them back here.
CREATE INDEX IF NOT EXISTS idx_donations_stripe_charge_id
  ON donations (stripe_charge_id) WHERE stripe_charge_id IS NOT NULL;

-- Public lists ask which donors on an initiative chose to hide their name.
CREATE INDEX IF NOT EXISTS idx_donations_anonymous_initiative
  ON donations (initiative_id, user_id) WHERE anonymous;
CREATE INDEX IF NOT EXISTS idx_subscriptions_anonymous_initiative
  ON subscriptions (initiative_id, user_id) WHERE anonymous;

COMMIT;
