-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- lfx-self-serve-ops#95: one donation row per Stripe PaymentIntent. A replayed
-- Idempotency-Key returns Stripe's cached PaymentIntent, and without this each
-- replay inserted another row that the single succeeded webhook then flipped
-- to succeeded. Fails if duplicates already exist; resolve those by hand first.
DROP INDEX IF EXISTS idx_donations_payment_intent;
CREATE UNIQUE INDEX idx_donations_payment_intent
    ON donations(stripe_payment_intent_id) WHERE stripe_payment_intent_id IS NOT NULL;

COMMIT;
