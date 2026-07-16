-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

DROP INDEX IF EXISTS idx_donations_initiative_recent;
DROP INDEX IF EXISTS idx_subscriptions_initiative_recent;

COMMIT;
