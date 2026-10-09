-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Restore the 009 trigger before dropping columns it would reference.
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
        OLD.updated_by IS DISTINCT FROM NEW.updated_by
    )
    EXECUTE FUNCTION set_updated_on();

ALTER TABLE initiatives
  DROP COLUMN IF EXISTS attributed_to_logo_url,
  DROP COLUMN IF EXISTS attributed_to_name;

COMMIT;
