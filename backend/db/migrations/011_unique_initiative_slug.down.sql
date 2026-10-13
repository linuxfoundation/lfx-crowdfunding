-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Slugs suffixed or nulled by the up migration are not restored.
DROP INDEX IF EXISTS idx_initiatives_slug;
CREATE INDEX idx_initiatives_slug ON initiatives(slug);

COMMIT;
