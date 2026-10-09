-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- lfx-crowdfunding#346: the attributed entity's name and logo, captured at
-- attribution time for the public source label. b2b_org is non-public in the
-- query service, so anonymous viewers can't look these up live. NULL for
-- personal rows (no backfill). A later rename shows only after re-attribution.
ALTER TABLE initiatives
  ADD COLUMN IF NOT EXISTS attributed_to_name     TEXT,
  ADD COLUMN IF NOT EXISTS attributed_to_logo_url TEXT;

-- Every user-editable column must bump updated_on when it alone changes
-- (see the MAINTENANCE note on the trigger in 001_initial).
-- attributed_to_type/uid, benefit_project_uid and donation_mode were missed
-- when 004 and 007 added them.
DROP TRIGGER IF EXISTS set_updated_on ON initiatives;
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiatives
    FOR EACH ROW
    WHEN (
        OLD.owner_id IS DISTINCT FROM NEW.owner_id OR
        OLD.name IS DISTINCT FROM NEW.name OR
        OLD.slug IS DISTINCT FROM NEW.slug OR
        OLD.status IS DISTINCT FROM NEW.status OR
        OLD.industry IS DISTINCT FROM NEW.industry OR
        OLD.description IS DISTINCT FROM NEW.description OR
        OLD.color IS DISTINCT FROM NEW.color OR
        OLD.logo_url IS DISTINCT FROM NEW.logo_url OR
        OLD.website_url IS DISTINCT FROM NEW.website_url OR
        OLD.coc_url IS DISTINCT FROM NEW.coc_url OR
        OLD.cii_project_id IS DISTINCT FROM NEW.cii_project_id OR
        OLD.stripe_plan_id IS DISTINCT FROM NEW.stripe_plan_id OR
        OLD.stripe_product_id IS DISTINCT FROM NEW.stripe_product_id OR
        OLD.jobspring_project_id IS DISTINCT FROM NEW.jobspring_project_id OR
        OLD.stacks_identifier IS DISTINCT FROM NEW.stacks_identifier OR
        OLD.eventbrite_url IS DISTINCT FROM NEW.eventbrite_url OR
        OLD.application_url IS DISTINCT FROM NEW.application_url OR
        OLD.accept_funding IS DISTINCT FROM NEW.accept_funding OR
        OLD.event_start_date IS DISTINCT FROM NEW.event_start_date OR
        OLD.event_end_date IS DISTINCT FROM NEW.event_end_date OR
        OLD.country IS DISTINCT FROM NEW.country OR
        OLD.city IS DISTINCT FROM NEW.city OR
        OLD.is_online IS DISTINCT FROM NEW.is_online OR
        OLD.updated_by IS DISTINCT FROM NEW.updated_by OR
        OLD.attributed_to_type IS DISTINCT FROM NEW.attributed_to_type OR
        OLD.attributed_to_uid IS DISTINCT FROM NEW.attributed_to_uid OR
        OLD.attributed_to_name IS DISTINCT FROM NEW.attributed_to_name OR
        OLD.attributed_to_logo_url IS DISTINCT FROM NEW.attributed_to_logo_url OR
        OLD.benefit_project_uid IS DISTINCT FROM NEW.benefit_project_uid OR
        OLD.donation_mode IS DISTINCT FROM NEW.donation_mode
    )
    EXECUTE FUNCTION set_updated_on();

COMMIT;
