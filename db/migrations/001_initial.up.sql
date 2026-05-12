-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT

-- ============================================
-- Migration: Normalised Initiatives Schema
-- Version: 2.0.0
-- Created: 2026-05-07
-- Updated: 2026-05-11 (schema decisions applied — see docs/rewrite/data-design_and_migration.md)
-- Description: Fully normalised schema.
--   initiatives merges lff-prod-projects + lff-prod-entities.
--   users has UUID surrogate PK; user_id is Auth0 subject (unique index).
--   organizations.organization_id dropped (was duplicate of id).
--   initiatives.initiative_id dropped; all child FKs on initiatives(id) UUID.
--
-- Excluded (no DynamoDB write path / computed at read-time):
--   balance          - computed from ledger at read time
--   funding_status   - computed from ledger + subscription summaries
--   entity_stats     - EntityStats.Sponsors computed at read time
--   sponsors         - GetEntitySponsors pushes to ES only; SaveEntity never
--                      called with sponsors populated
--   diversity        - GetDiversity fetches external API at read time
--   vulnerability    - GetVulnerabilitySummary fetches external service
--   badges           - convertProjectToDynamoRepresentation never assigns Badges
--   total_raised     - ProjectStats.TotalRaised has no update path (always 0)
-- ============================================

BEGIN;

CREATE SCHEMA IF NOT EXISTS crowdfunding;
SET search_path TO crowdfunding;

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
  avatar_url          TEXT,
  stripe_customer_id  TEXT,
  github_access_token TEXT,
  created_on          TIMESTAMPTZ  DEFAULT NOW(),
  updated_on          TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: organizations
-- ============================================
CREATE TABLE IF NOT EXISTS organizations (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id   UUID         NOT NULL REFERENCES users(id),
  name       TEXT         NOT NULL,
  avatar_url TEXT,
  status     VARCHAR(50),
  created_on TIMESTAMPTZ  DEFAULT NOW(),
  updated_on TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: initiatives  (merged projects + entities)
--
-- initiative_type values:
--   project, mentorship, general_fund, event, ostif
--
-- source_dynamo_table is migration-only — drop in Phase 5:
--   ALTER TABLE initiatives DROP COLUMN source_dynamo_table;
-- ============================================
CREATE TABLE IF NOT EXISTS initiatives (
  id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_type     VARCHAR(50)  NOT NULL,
  source_dynamo_table VARCHAR(50),           -- 'projects' | 'entities' — drop post-cutover

  -- ownership
  owner_id            UUID         NOT NULL REFERENCES users(id),

  -- core display fields
  name                TEXT         NOT NULL,
  slug                TEXT         NOT NULL UNIQUE,
  status              VARCHAR(50),
  industry            TEXT,
  description         TEXT,
  color               VARCHAR(10),
  logo_url            TEXT,
  website_url         TEXT,
  coc_url             TEXT,

  -- financial / platform IDs
  stripe_plan_id           TEXT,
  stripe_product_id        TEXT,
  amount_raised_in_cents   BIGINT       NOT NULL DEFAULT 0,
  accept_funding           BOOLEAN      DEFAULT false,

  -- project-only fields
  cii_project_id           TEXT,
  mentorship_program_id    TEXT,                -- non-NULL when initiative_type = 'mentorship'
  stacks_identifier        TEXT,

  -- entity-only fields
  eventbrite_id            TEXT,
  application_url          TEXT,
  event_start_date         TIMESTAMPTZ,
  event_end_date           TIMESTAMPTZ,
  country                  VARCHAR(100),
  city                     VARCHAR(100),
  is_online                BOOLEAN      DEFAULT false,

  created_on          TIMESTAMPTZ  DEFAULT NOW(),
  updated_on          TIMESTAMPTZ  DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_goals
-- Projects : projectDetails.{development,marketing,meetups,travel,
--            bugBounty,documentation,programInfo,other} — each carries
--            Budget{amount, allocation}; development also carries repoLink.
-- Entities : entity.Goals[] — each Goal{name,description,goalColor,
--            goalIcon,budget{amount,allocation}}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_goals (
  id             UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id  UUID    NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name           TEXT    NOT NULL,
  amount_in_cents BIGINT NOT NULL DEFAULT 0,
  allocation     TEXT,
  repo_link      TEXT,           -- project 'development' category only
  description    TEXT,           -- entity goals only
  color          VARCHAR(10),    -- entity goals only (goalColor)
  icon           TEXT,           -- entity goals only (goalIcon)
  sort_order     INTEGER DEFAULT 0,
  UNIQUE (initiative_id, name)
);

-- ============================================
-- TABLE: initiative_beneficiaries
-- Projects : projectDetails.beneficiaries[]
-- Entities : entity.Beneficiary[]  (json key: "beneficiaries")
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_beneficiaries (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  email         TEXT
);

-- ============================================
-- TABLE: initiative_custom_websites
-- Projects : projectDetails.customWebsites[]
-- Entities : SE-reflect via EditEntityRequest.CustomWebsites
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_custom_websites (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  url           TEXT NOT NULL
);

-- ============================================
-- TABLE: initiative_contributors  (project only)
-- Projects : projectDetails.Contributors
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_contributors (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  email         TEXT
);

-- ============================================
-- TABLE: initiative_mentors  (project programInfo only)
-- Projects : projectDetails.programInfo.mentor[]
--            Each Mentor{name, email, avatarURL, introduction}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_mentors (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  email         TEXT,
  avatar_url    TEXT,
  introduction  TEXT
);

-- ============================================
-- TABLE: initiative_program_info_terms  (project programInfo only)
-- Projects : projectDetails.programInfo.terms[]  ([]string)
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_program_info_terms (
  id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID    NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  term          TEXT    NOT NULL,
  sort_order    INTEGER DEFAULT 0
);

-- ============================================
-- TABLE: initiative_program_info_skills  (project programInfo only)
-- Projects : projectDetails.programInfo.skills[]  ([]string)
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_program_info_skills (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  skill         TEXT NOT NULL,
  UNIQUE (initiative_id, skill)
);

-- ============================================
-- TABLE: initiative_program_info_config  (project programInfo only — 0-or-1 per initiative)
-- Projects : projectDetails.programInfo.termsConditions  (bool)
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_program_info_config (
  initiative_id    UUID    PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  terms_conditions BOOLEAN DEFAULT false
);

-- ============================================
-- TABLE: initiative_program_info_custom_term  (project programInfo only — 0-or-1 per initiative)
-- Projects : projectDetails.programInfo.customTerm
--            CustomTerm{termName, startMonth, endMonth, year}
--            Only present when termName is non-empty.
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_program_info_custom_term (
  initiative_id UUID        PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  term_name     TEXT,
  start_month   VARCHAR(20),
  end_month     VARCHAR(20),
  year          INTEGER
);

-- ============================================
-- TABLE: initiative_sponsorship_tiers  (entity only)
-- Entities : SE-reflect via EditEntityRequest.SponsorshipTiers
--            SponsorshipTier{name, description, color, icon, minimum}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_sponsorship_tiers (
  id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id UUID    NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  name          TEXT,
  description   TEXT,
  color         VARCHAR(10),
  icon          TEXT,
  minimum       INTEGER NOT NULL DEFAULT 0,
  sort_order    INTEGER DEFAULT 0
);

-- ============================================
-- TABLE: initiative_ostif_detail  (ostif entity type only — 0-or-1 per initiative)
-- Entities : SE-reflect via EditEntityRequest.Detail (domain.Detail struct).
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_ostif_detail (
  initiative_id            UUID        PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  monetization_strategy    TEXT,
  current_security_strategy TEXT,
  license_type             VARCHAR(100),
  total_budget_in_cents    BIGINT      DEFAULT 0,
  terms_conditions         BOOLEAN     DEFAULT false
);

-- ============================================
-- TABLE: initiative_contacts  (ostif entity type only)
-- Entities : entity.detail.{primaryContact, secondaryContact, technicalLead}
--            Contact{firstName, lastName, email, phoneNumber,
--                    otherContactOption, preferredContactMethod}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_contacts (
  id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id            UUID        NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  contact_type             VARCHAR(50) NOT NULL,  -- 'primary' | 'secondary' | 'technical_lead'
  first_name               TEXT,
  last_name                TEXT,
  email                    TEXT,
  phone_number             VARCHAR(50),
  other_contact_option     TEXT,
  preferred_contact_method VARCHAR(50),
  UNIQUE (initiative_id, contact_type)
);

-- ============================================
-- TABLE: initiative_github_stats  (project only — 1-to-1 cached)
-- Updated by UpdateGithubDataCache → targeted UpdateItem on
--   cachedDetails.githubStats.{forks, openIssues, stars}
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_github_stats (
  initiative_id UUID        PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  forks         INTEGER     NOT NULL DEFAULT 0,
  stars         INTEGER     NOT NULL DEFAULT 0,
  open_issues   INTEGER     NOT NULL DEFAULT 0,
  updated_at    TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_stats  (project only — backers count only)
-- total_raised excluded: no active write path (ProjectStats.TotalRaised always 0).
-- Updated by UpdateProjectStats → targeted UpdateItem on
--   cachedDetails.projectStats.backers ± 1
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_stats (
  initiative_id UUID        PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  backers       INTEGER     NOT NULL DEFAULT 0,
  updated_at    TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- TABLE: initiative_entity_details  (entity only — JSONB)
-- entity.EntityDetails is map[string]string with application-defined keys.
-- Serialised by dynamodbattribute.MarshalMap under the 'entityDetails' key.
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_entity_details (
  initiative_id UUID  PRIMARY KEY REFERENCES initiatives(id) ON DELETE CASCADE,
  details       JSONB NOT NULL DEFAULT '{}'
);

-- ============================================
-- TABLE: donations
-- Merged from lff-prod-donations + lff-prod-entity-donations
-- initiative_id references the surrogate UUID PK on initiatives(id)
-- ============================================
CREATE TABLE IF NOT EXISTS donations (
  id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                 UUID        NOT NULL REFERENCES users(id),
  initiative_id           UUID        REFERENCES initiatives(id),
  organization_id         UUID        REFERENCES organizations(id),
  cached_details          JSONB,
  category                TEXT,
  created_on              TIMESTAMPTZ DEFAULT NOW(),
  updated_at              TIMESTAMPTZ DEFAULT NOW(),
  current_amount_in_cents BIGINT      NOT NULL,
  payment_method          VARCHAR(50),
  po_number               TEXT,
  status                  VARCHAR(50),
  stripe_charge_id        VARCHAR(255),
  UNIQUE (stripe_charge_id)             -- partial dedup: Postgres UNIQUE allows multiple NULLs (invoice payments)
);

-- ============================================
-- TABLE: subscriptions
-- Merged from lff-prod-subscriptions + lff-prod-entity-subscriptions
-- initiative_id references the surrogate UUID PK on initiatives(id)
-- ============================================
CREATE TABLE IF NOT EXISTS subscriptions (
  id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id                     UUID        NOT NULL REFERENCES users(id),
  initiative_id               UUID        REFERENCES initiatives(id),
  organization_id             UUID        REFERENCES organizations(id),
  cached_details              JSONB,
  category                    TEXT,
  created_on                  TIMESTAMPTZ DEFAULT NOW(),
  updated_at                  TIMESTAMPTZ DEFAULT NOW(),
  current_amount_in_cents     BIGINT      NOT NULL,
  frequency                   VARCHAR(50),
  status                      VARCHAR(50),
  stripe_subscription_id      VARCHAR(255) UNIQUE,
  stripe_subscription_item_id VARCHAR(255)
);

-- ============================================
-- INDEXES
-- ============================================

-- users
CREATE INDEX IF NOT EXISTS idx_users_user_id             ON users(user_id);

-- organizations
CREATE INDEX IF NOT EXISTS idx_organizations_owner_id    ON organizations(owner_id);

-- initiatives (core)
CREATE INDEX IF NOT EXISTS idx_initiatives_owner_id      ON initiatives(owner_id);
CREATE INDEX IF NOT EXISTS idx_initiatives_slug          ON initiatives(slug);
CREATE INDEX IF NOT EXISTS idx_initiatives_status        ON initiatives(status);
CREATE INDEX IF NOT EXISTS idx_initiatives_type          ON initiatives(initiative_type);
CREATE INDEX IF NOT EXISTS idx_initiatives_amount_raised ON initiatives(amount_raised_in_cents DESC);

-- initiative child tables
CREATE INDEX IF NOT EXISTS idx_initiative_goals_iid                  ON initiative_goals(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_beneficiaries_iid          ON initiative_beneficiaries(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_custom_websites_iid        ON initiative_custom_websites(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_contributors_iid           ON initiative_contributors(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_mentors_iid                ON initiative_mentors(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_program_info_terms_iid     ON initiative_program_info_terms(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_program_info_skills_iid    ON initiative_program_info_skills(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_sponsorship_tiers_iid      ON initiative_sponsorship_tiers(initiative_id);
CREATE INDEX IF NOT EXISTS idx_initiative_contacts_iid               ON initiative_contacts(initiative_id);

-- donations
CREATE INDEX IF NOT EXISTS idx_donations_user_id         ON donations(user_id);
CREATE INDEX IF NOT EXISTS idx_donations_initiative_id   ON donations(initiative_id);
CREATE INDEX IF NOT EXISTS idx_donations_status          ON donations(status);
CREATE INDEX IF NOT EXISTS idx_donations_org_id          ON donations(organization_id);

-- subscriptions
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id     ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_initiative_id ON subscriptions(initiative_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_org_id      ON subscriptions(organization_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status      ON subscriptions(status);

COMMIT;
