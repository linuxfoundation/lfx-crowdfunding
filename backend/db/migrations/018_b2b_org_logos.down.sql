-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

DROP INDEX IF EXISTS idx_subscriptions_org_domain_b2b_org;
DROP INDEX IF EXISTS idx_donations_org_domain_b2b_org;

DROP TABLE IF EXISTS b2b_org_logos;

COMMIT;
