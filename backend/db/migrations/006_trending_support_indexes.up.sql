-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Support LFXV2-2533 "trending" sort: counts succeeded donations / active
-- subscriptions per initiative in the last 30 days.
CREATE INDEX IF NOT EXISTS idx_donations_initiative_recent
  ON donations (initiative_id, created_on) WHERE status = 'succeeded';

CREATE INDEX IF NOT EXISTS idx_subscriptions_initiative_recent
  ON subscriptions (initiative_id, created_on) WHERE status = 'active';

COMMIT;
