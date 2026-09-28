-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- lfx-crowdfunding#263: b2b_org uids are 18-char Salesforce Account SFIDs, not
-- UUIDs. attributed_to_uid must hold both organization (SFID) and project
-- (UUID) values, so widen it to TEXT. benefit_project_uid stays UUID — it is
-- always a real v2 project UUID.
ALTER TABLE initiatives
  ALTER COLUMN attributed_to_uid TYPE TEXT USING attributed_to_uid::text;

COMMIT;
