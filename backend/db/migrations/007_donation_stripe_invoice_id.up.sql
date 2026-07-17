-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

ALTER TABLE crowdfunding.donations
    ADD COLUMN IF NOT EXISTS stripe_invoice_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS donations_stripe_invoice_id_key
    ON crowdfunding.donations (stripe_invoice_id)
    WHERE stripe_invoice_id IS NOT NULL;
