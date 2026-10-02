-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- lfx-crowdfunding#259: once attributed-entity writers can edit, record who
-- made the last edit (LF SSO username, same as initiative_announcements.created_by).
ALTER TABLE initiatives ADD COLUMN IF NOT EXISTS updated_by TEXT;
ALTER TABLE initiative_announcements ADD COLUMN IF NOT EXISTS updated_by TEXT;

COMMIT;
