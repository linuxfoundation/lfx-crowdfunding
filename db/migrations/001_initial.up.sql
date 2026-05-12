-- Copyright The Linux Foundation and each contributor to CommunityBridge.
-- SPDX-License-Identifier: MIT
-- ============================================
-- Migration: Normalised Initiatives Schema
-- Version: 2.0.0
-- Created: 2026-05-07
-- Description: Fully normalised schema.
--   initiatives merges lff-prod-projects + lff-prod-entities.
--
-- Excluded (no DynamoDB write path / computed at read-time):
--   balance        — computed from ledger at read time
--   funding_status — computed from ledger + subscription summaries
--   entity_stats   — EntityStats.Sponsors computed at read time
--   sponsors       — GetEntitySponsors pushes to ES only; SaveEntity never
--                    called with sponsors populated
--   diversity      — GetDiversity fetches external API at read time
--   vulnerability  — GetVulnerabilitySummary fetches external service
--   badges         — convertProjectToDynamoRepresentation never assigns Badges
--   total_raised   — ProjectStats.TotalRaised has no update path (always 0)
-- ============================================

CREATE SCHEMA IF NOT EXISTS crowdfunding;

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- Enable pgcrypto extension for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================
-- TABLE: users
-- ============================================
CREATE TABLE IF NOT EXISTS users (
  id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     VARCHAR(255) NOT NULL UNIQUE,  -- Auth0 subject (e.g. auth0|abc123)
  email       TEXT,
  given_name  TEXT,
  family_name TEXT,
  name        TEXT,
  avatar_url  TEXT,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: organizations
-- ============================================
CREATE TABLE IF NOT EXISTS organizations (
  id              UUID     PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id        VARCHAR(255) NOT NULL REFERENCES users(user_id),
  name            TEXT         NOT NULL,
  avatar_url      TEXT,
  status          VARCHAR(50),
  created_on      TIMESTAMPTZ  DEFAULT NOW(),
  updated_on      TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: initiatives  (merged projects + entities)
--
-- source_dynamo_table is "migration only" and will be dropped in Phase 5
--   after the application no longer writes to DynamoDB.
-- ============================================
CREATE TABLE IF NOT EXISTS initiatives (
  id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_type      VARCHAR(50)  NOT NULL,
  source_dynamo_table  VARCHAR(50),            -- 'projects' | 'entities' — drop post-cutover

  -- ownership
  owner_id             VARCHAR(255) NOT NULL REFERENCES users(user_id),

  -- core display fields
  name                 TEXT         NOT NULL,
  slug                 TEXT,
  status               VARCHAR(50),
  industry             TEXT,
  description          TEXT,
  color                VARCHAR(10),
  logo_url             TEXT,
  website_url          TEXT,
  coc_url              TEXT,

  -- financial / platform IDs
  stripe_plan_id       TEXT,
  stripe_product_id    TEXT,
  amount_raised_in_cents BIGINT       NOT NULL DEFAULT 0,
  accept_funding        BOOLEAN      DEFAULT false,

  -- project-only fields (SP write path)
  cii_project_id       TEXT,
  jobspring_project_id TEXT,                   -- top-level jobspringProjectId; present when mentee goal exists
  stacks_identifier    TEXT,

  -- entity-only fields (SE / SE-reflect write path)
  eventbrite_url       TEXT,
  application_url      TEXT,
  event_start_date     TIMESTAMPTZ,
  event_end_date       TIMESTAMPTZ,
  country              VARCHAR(100),
  city                 VARCHAR(100),
  is_online            BOOLEAN      DEFAULT false,
  created_on           TIMESTAMPTZ  DEFAULT NOW(),
  updated_on           TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_goals
-- Projects : projectDetails.{development,marketing,meetups,travel,
--            bugBounty,documentation,mentee,other} — each carries
--            Budget{amount, allocation}; development also carries repoLink.
-- Entities : entity.Goals[] — each Goal{name,description,goalColor,
--            goalIcon, budget{amount,allocation}}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_goals (
  id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id   UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name            TEXT         NOT NULL,
  amount_in_cents BIGINT       NOT NULL DEFAULT 0 CHECK (amount_in_cents >= 0),
  allocation      TEXT,
  repo_link       TEXT,         -- project 'development' category only
  description     TEXT,         -- entity goals only
  color           VARCHAR(10),  -- entity goals only (goalColor)
  icon            TEXT,         -- entity goals only (goalIcon)
  sort_order      INTEGER       DEFAULT 0,
  UNIQUE (initiative_id, name),
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_beneficiaries
-- Projects : projectDetails.beneficiaries[]
-- Entities : entity.Beneficiary[]  (json key: "beneficiaries")
--            Also updated by UpdateEntityBeneficiaries (sets entity.Beneficiary
--            then calls SaveEntity).
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_beneficiaries (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  email         TEXT,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_custom_websites
-- Projects : projectDetails.customWebsites[]
-- Entities : SE-reflect via EditEntityRequest.CustomWebsites
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_custom_websites (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  url           TEXT         NOT NULL,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_contributors  (project only)
-- Projects : projectDetails.Contributors
--            convertProjectToDynamoRepresentation line ~927
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_contributors (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  email         TEXT,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_mentors  (mentorship projects only)
-- Projects : projectDetails.mentee.mentor[]
--            Each Mentor{name, email, avatarURL, introduction}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_mentors (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  email         TEXT,
  avatar_url    TEXT,
  introduction  TEXT,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_program_info_terms  (mentorship projects only)
-- Projects : projectDetails.mentee.terms[]  ([]string)
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_program_info_terms (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  term          TEXT         NOT NULL,
  sort_order    INTEGER      DEFAULT 0,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_program_info_skills  (mentorship projects only)
-- Projects : projectDetails.mentee.skills[]  ([]string)
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_program_info_skills (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  skill         TEXT         NOT NULL,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (initiative_id, skill)
);

-- ============================================
-- TABLE: initiative_program_info_config  (mentorship projects only — 0-or-1 per initiative)
-- Projects : projectDetails.mentee.termsConditions  (bool)
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_program_info_config (
  initiative_id    UUID PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  terms_conditions BOOLEAN      DEFAULT false,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_program_info_custom_term  (mentorship projects only — 0-or-1 per initiative)
-- Projects : projectDetails.mentee.customTerm
--            CustomTerm{termName, startMonth, endMonth, year}
--            Only present when termName is non-empty.
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_program_info_custom_term (
  initiative_id UUID PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  term_name     TEXT,
  start_month   VARCHAR(20),
  end_month     VARCHAR(20),
  year          INTEGER,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_sponsorship_tiers  (entity only)
-- Entities : SE-reflect via EditEntityRequest.SponsorshipTiers
--            SponsorshipTier{name, description, color, icon, minimum}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_sponsorship_tiers (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  description   TEXT,
  color         VARCHAR(10),
  icon          TEXT,
  minimum       BIGINT       NOT NULL DEFAULT 0 CHECK (minimum >= 0),
  sort_order    INTEGER      DEFAULT 0,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_ostif_detail  (ostif entity type only — 0-or-1 per initiative)
-- Entities : SE-reflect via EditEntityRequest.Detail (domain.Detail struct).
--            detail.totalBudget is also consumed explicitly:
--            entity.FundingStatus.TotalAnnualGoalInCents = int64(editEntityRequest.Detail.TotalBudget)
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_ostif_detail (
  initiative_id             UUID PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  monetization_strategy     TEXT,
  current_security_strategy TEXT,
  license_type              VARCHAR(100),
  total_budget_in_cents     BIGINT       NOT NULL DEFAULT 0 CHECK (total_budget_in_cents >= 0),
  terms_conditions          BOOLEAN      DEFAULT false,
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_contacts  (ostif entity type only)
-- Entities : entity.Detail.PrimaryContact, .SecondaryContact, .TechnicalLead
--            Contact{firstName, lastName, email, phoneNumber,
--                    otherContactOption, preferredContactMethod}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_contacts (
  id                       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id            UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  contact_type             VARCHAR(50)  NOT NULL,  -- 'primary' | 'secondary' | 'technical_lead'
  first_name               TEXT,
  last_name                TEXT,
  email                    TEXT,
  phone_number             VARCHAR(50),
  other_contact_option     TEXT,
  preferred_contact_method VARCHAR(50),
  created_on  TIMESTAMPTZ DEFAULT NOW(),
  updated_on  TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (initiative_id, contact_type)
);

-- ============================================
-- TABLE: initiative_github_stats  (project only — 1-to-1 cached)
-- Updated by UpdateGithubDataCache → targeted UpdateItem on
--   cachedDetails.githubStats.{forks, openIssues, stars}
-- Also initialised at project creation via CachedDetails.GithubStats.
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_github_stats (
  initiative_id UUID PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  forks         INTEGER      NOT NULL DEFAULT 0,
  stars         INTEGER      NOT NULL DEFAULT 0,
  open_issues   INTEGER      NOT NULL DEFAULT 0,
  created_on    TIMESTAMPTZ  DEFAULT NOW(),
  updated_on    TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_stats  (project only — backers count only)
-- total_raised excluded: no active write path (ProjectStats.TotalRaised always 0).
-- Updated by UpdateProjectStats → targeted UpdateItem on
--   cachedDetails.projectStats.backers ± 1
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_stats (
  initiative_id UUID PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  backers       INTEGER      NOT NULL DEFAULT 0,
  created_on    TIMESTAMPTZ  DEFAULT NOW(),
  updated_on    TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_entity_details  (entity only — JSONB)
-- entity.EntityDetails is map[string]string with application-defined keys.
-- Serialised by dynamodbattribute.MarshalMap under the 'entityDetails' key.
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_entity_details (
  initiative_id UUID PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  details       JSONB        NOT NULL DEFAULT '{}',
  created_on    TIMESTAMPTZ  DEFAULT NOW(),
  updated_on    TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: donations
-- Merged from lff-prod-donations + lff-prod-entity-donations
-- initiative_id references the surrogate UUID PK on initiatives(id)
-- ============================================
CREATE TABLE IF NOT EXISTS donations (
  id                      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                 VARCHAR(255) NOT NULL REFERENCES users(user_id),
  initiative_id           UUID         REFERENCES initiatives(id) ON DELETE SET NULL,
  organization_id         UUID         REFERENCES organizations(id) ON DELETE SET NULL,
  cached_details          JSONB,
  category                TEXT,
  current_amount_in_cents BIGINT       NOT NULL CHECK (current_amount_in_cents >= 0),
  po_number               TEXT,
  payment_method          VARCHAR(50),
  status                  VARCHAR(50),
  stripe_charge_id        VARCHAR(255),
  created_on              TIMESTAMPTZ  DEFAULT NOW(),
  updated_on              TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: subscriptions
-- Merged from lff-prod-subscriptions + lff-prod-entity-subscriptions
-- initiative_id references the surrogate UUID PK on initiatives(id)
-- ============================================
CREATE TABLE IF NOT EXISTS subscriptions (
  id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                     VARCHAR(255) NOT NULL REFERENCES users(user_id),
  initiative_id               UUID         REFERENCES initiatives(id) ON DELETE SET NULL,
  organization_id             UUID         REFERENCES organizations(id) ON DELETE SET NULL,
  cached_details              JSONB,
  category                    TEXT,
  current_amount_in_cents     BIGINT       NOT NULL CHECK (current_amount_in_cents >= 0),
  frequency                   VARCHAR(50),
  status                      VARCHAR(50),
  stripe_subscription_id      VARCHAR(255),
  stripe_subscription_item_id VARCHAR(255),
  created_on                  TIMESTAMPTZ  DEFAULT NOW(),
  updated_on                  TIMESTAMPTZ  DEFAULT NOW()
);


-- ============================================
-- TRIGGER: auto-update updated_on on every UPDATE
-- Note: initiatives trigger skips updates when only cache fields change
-- ============================================
CREATE OR REPLACE FUNCTION set_updated_on()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_on = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER set_updated_on BEFORE UPDATE ON users                               FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON organizations                       FOR EACH ROW EXECUTE FUNCTION set_updated_on();
-- Skip updated_on change when only cache fields (amount_raised_in_cents) are updated
-- MAINTENANCE: This WHEN clause lists all non-cache columns. When adding new columns:
--   - Include them here if they are user-editable fields (trigger should fire)
--   - Omit them if they are cache/computed fields (trigger should NOT fire)
--   - Current cache fields: amount_raised_in_cents, source_dynamo_table
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiatives                         FOR EACH ROW 
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
        OLD.is_online IS DISTINCT FROM NEW.is_online
    )
    EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_goals                    FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_beneficiaries            FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_custom_websites          FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_contributors             FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_mentors                  FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_program_info_terms       FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_program_info_skills      FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_program_info_config      FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_program_info_custom_term FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_sponsorship_tiers        FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_ostif_detail             FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_contacts                 FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_github_stats             FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_stats                    FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON initiative_entity_details           FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON donations                           FOR EACH ROW EXECUTE FUNCTION set_updated_on();
CREATE TRIGGER set_updated_on BEFORE UPDATE ON subscriptions                       FOR EACH ROW EXECUTE FUNCTION set_updated_on();

-- ============================================
-- INDEXES
-- ============================================

-- organizations
CREATE INDEX IF NOT EXISTS idx_organizations_owner_id               ON organizations(owner_id);

-- initiatives (core)

CREATE INDEX IF NOT EXISTS idx_initiatives_owner_id                 ON initiatives(owner_id);
CREATE INDEX IF NOT EXISTS idx_initiatives_slug                     ON initiatives(slug);
CREATE INDEX IF NOT EXISTS idx_initiatives_status                   ON initiatives(status);
CREATE INDEX IF NOT EXISTS idx_initiatives_type                     ON initiatives(initiative_type);
CREATE INDEX IF NOT EXISTS idx_initiatives_amount_raised            ON initiatives(amount_raised_in_cents DESC);

-- initiative child tables
CREATE INDEX IF NOT EXISTS idx_initiative_goals_iid                 ON initiative_goals(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_beneficiaries_iid         ON initiative_beneficiaries(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_custom_websites_iid       ON initiative_custom_websites(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_contributors_iid          ON initiative_contributors(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_mentors_iid               ON initiative_mentors(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_program_info_terms_iid    ON initiative_program_info_terms(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_program_info_skills_iid   ON initiative_program_info_skills(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_sponsorship_tiers_iid     ON initiative_sponsorship_tiers(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_contacts_iid              ON initiative_contacts(initiative_id);

-- donations
CREATE INDEX IF NOT EXISTS idx_donations_user_id                    ON donations(user_id);
CREATE INDEX IF NOT EXISTS idx_donations_initiative_id              ON donations(initiative_id);
CREATE INDEX IF NOT EXISTS idx_donations_status                     ON donations(status);
CREATE INDEX IF NOT EXISTS idx_donations_org_id                     ON donations(organization_id);

-- subscriptions
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id                ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_initiative_id          ON subscriptions(initiative_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_org_id                 ON subscriptions(organization_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status                 ON subscriptions(status);

COMMIT;
