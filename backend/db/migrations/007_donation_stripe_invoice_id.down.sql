-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

DROP INDEX IF EXISTS crowdfunding.donations_stripe_invoice_id_key;

ALTER TABLE crowdfunding.donations
    DROP COLUMN IF EXISTS stripe_invoice_id;
