-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- A legacy org's newer donations can be keyed in the Ledger by its website
-- domain instead of the legacy org id. When that domain has no matched
-- b2b_org logo, donor lists fall back to the legacy org's uploaded logo
-- (lfx-self-serve-ops#1647). Set by hand, one vouched-for domain per org.
ALTER TABLE legacy_org_logos ADD COLUMN IF NOT EXISTS org_domain TEXT
  CHECK (org_domain = lower(btrim(org_domain)) AND org_domain <> '');

CREATE UNIQUE INDEX IF NOT EXISTS idx_legacy_org_logos_org_domain
  ON legacy_org_logos (org_domain) WHERE org_domain IS NOT NULL;

COMMIT;
