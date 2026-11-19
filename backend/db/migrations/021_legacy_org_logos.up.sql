-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Logos uploaded to the legacy CF organizations (dropped in 017), keyed by the
-- legacy org id. Ledger still buckets those orgs' donations by that id when
-- the rows carry no domain, so sponsor and donor lists can only find their logo
-- here (lfx-self-serve-ops#1647). A one-off snapshot, loaded by hand from the
-- pre-017 export and never refreshed; domain-keyed orgs use b2b_org_logos.
CREATE TABLE IF NOT EXISTS legacy_org_logos (
  legacy_org_id UUID PRIMARY KEY,
  name          TEXT NOT NULL,
  logo_url      TEXT NOT NULL
);

COMMIT;
