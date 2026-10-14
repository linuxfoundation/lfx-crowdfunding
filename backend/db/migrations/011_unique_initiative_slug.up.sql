-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- lfx-self-serve-ops#90: the slug is the public identifier, so it must be
-- unique. Otherwise slug lookups pick an arbitrary row and a second initiative
-- can shadow another owner's page, donations and approval links.

-- Empty slugs can't be routed to; store them as NULL so they don't collide.
UPDATE initiatives SET slug = NULL WHERE slug = '';

-- Existing duplicates: the published (then oldest) row keeps the slug, the
-- others get an id suffix so the unique index can be built.
UPDATE initiatives i
SET slug = i.slug || '-' || left(i.id::text, 8)
FROM (
    SELECT id, row_number() OVER (
        PARTITION BY slug
        ORDER BY COALESCE(LOWER(status) = 'published', false) DESC, created_on NULLS LAST, id
    ) AS rn
    FROM initiatives
    WHERE slug IS NOT NULL
) d
WHERE d.id = i.id AND d.rn > 1;

DROP INDEX IF EXISTS idx_initiatives_slug;
CREATE UNIQUE INDEX idx_initiatives_slug ON initiatives(slug);

COMMIT;
