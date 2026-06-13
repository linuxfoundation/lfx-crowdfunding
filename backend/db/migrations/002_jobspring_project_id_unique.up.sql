-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

-- Add unique constraint on jobspring_project_id so that the mentorship-sync
-- CronJob's ON CONFLICT (jobspring_project_id) upsert works correctly.
ALTER TABLE initiatives
    ADD CONSTRAINT initiatives_jobspring_project_id_key UNIQUE (jobspring_project_id);
