-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

-- Add unique constraint on jobspring_project_id to enforce data integrity —
-- prevents duplicate Jobspring program IDs from being inserted into the
-- initiatives table during mentorship-sync runs.
BEGIN;

SET LOCAL search_path TO crowdfunding, public;

ALTER TABLE initiatives
    ADD CONSTRAINT initiatives_jobspring_project_id_key UNIQUE (jobspring_project_id);

COMMIT;
