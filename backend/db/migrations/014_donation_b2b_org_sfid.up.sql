-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- b2b_org (Salesforce Account SFID) matched from org_domain after save (#411).
-- NULL means not matched yet: no match, or a failed lookup that #412 retries.
ALTER TABLE donations
  ADD COLUMN IF NOT EXISTS b2b_org_sfid TEXT NULL;

ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS b2b_org_sfid TEXT NULL;

COMMIT;
