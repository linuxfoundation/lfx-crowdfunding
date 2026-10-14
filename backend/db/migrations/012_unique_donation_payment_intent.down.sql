-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

DROP INDEX IF EXISTS idx_donations_payment_intent;
CREATE INDEX idx_donations_payment_intent
    ON donations(stripe_payment_intent_id) WHERE stripe_payment_intent_id IS NOT NULL;

COMMIT;
