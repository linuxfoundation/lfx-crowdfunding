-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- LFXV2-2956 M1: attribution foundation (design doc
-- backend/docs/rewrite/11-initiative-attribution-and-access.md, §2.1, §5).
-- attributed_to_type/uid: who an initiative is run on behalf of. Existing rows
-- default to 'personal' with a NULL uid — no backfill, behavior unchanged.
-- benefit_project_uid: separate, independent field (PM decision, 2026-08,
-- open question 1) — nullable, requiredness is a form-validation decision.
ALTER TABLE initiatives
  ADD COLUMN IF NOT EXISTS attributed_to_type VARCHAR(20) NOT NULL DEFAULT 'personal'
    CHECK (attributed_to_type IN ('personal', 'organization', 'project')),
  ADD COLUMN IF NOT EXISTS attributed_to_uid   UUID,
  ADD COLUMN IF NOT EXISTS benefit_project_uid UUID;

ALTER TABLE initiatives
  ADD CONSTRAINT initiatives_attribution_uid_ck
    CHECK ((attributed_to_type = 'personal') = (attributed_to_uid IS NULL));

-- Supports the M2 ListForUser writable-set filter, matched as
-- (attribution type, entity uid) pairs (design §5.1 step 2).
CREATE INDEX IF NOT EXISTS idx_initiatives_attributed_to
  ON initiatives (attributed_to_type, attributed_to_uid)
  WHERE attributed_to_uid IS NOT NULL;

COMMIT;
