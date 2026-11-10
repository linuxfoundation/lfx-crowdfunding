-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Restores the schema from 001_initial only. The rows and the organization_id
-- values are not recoverable from here.
CREATE TABLE IF NOT EXISTS organizations (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id   UUID        NOT NULL REFERENCES users(id),
  name       TEXT        NOT NULL,
  avatar_url TEXT,
  status     VARCHAR(50),
  created_on TIMESTAMPTZ DEFAULT NOW(),
  updated_on TIMESTAMPTZ DEFAULT NOW()
);

DROP TRIGGER IF EXISTS set_updated_on ON organizations;
CREATE TRIGGER set_updated_on BEFORE UPDATE ON organizations
  FOR EACH ROW EXECUTE FUNCTION set_updated_on();

CREATE INDEX IF NOT EXISTS idx_organizations_owner_id ON organizations(owner_id);

ALTER TABLE donations
  ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_donations_org_id ON donations(organization_id);

ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_subscriptions_org_id ON subscriptions(organization_id);

COMMIT;
