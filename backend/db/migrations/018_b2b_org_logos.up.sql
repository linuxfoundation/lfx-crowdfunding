-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Logos of the b2b_orgs matched to org donations (#472), copied from
-- member-service (Account.Logo_URL__c). Filled when a donation or subscription
-- is matched and refreshed daily by b2b-org-backfill, so donor lists never call
-- member-service. Orgs without a logo have no row.
CREATE TABLE IF NOT EXISTS b2b_org_logos (
  b2b_org_sfid TEXT PRIMARY KEY,
  logo_url     TEXT NOT NULL,
  refreshed_on TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Donor lists map Ledger's org domain to the matched b2b_org.
CREATE INDEX IF NOT EXISTS idx_donations_org_domain_b2b_org
  ON donations (org_domain, b2b_org_sfid) WHERE b2b_org_sfid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_subscriptions_org_domain_b2b_org
  ON subscriptions (org_domain, b2b_org_sfid) WHERE b2b_org_sfid IS NOT NULL;

COMMIT;
