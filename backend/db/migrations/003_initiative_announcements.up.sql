-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT
-- ============================================
-- Migration: Initiative Announcements
-- Created: 2026-06-30
-- Adds the initiative_announcements table to support
-- owner-published updates visible on the initiative page.
-- ============================================

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- ============================================
-- TABLE: initiative_announcements
-- Stores owner-published announcements for an initiative.
-- created_by stores the LF SSO username of the author.
-- description is TEXT (may contain sanitised HTML).
-- ============================================
CREATE TABLE IF NOT EXISTS initiative_announcements (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  initiative_id  UUID        NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
  created_by     TEXT        NOT NULL,  -- LF SSO username (Principal.Username)
  title          TEXT        NOT NULL,
  description    TEXT        NOT NULL,  -- may contain HTML
  created_on     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_on     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_initiative_announcements_initiative_id
  ON initiative_announcements (initiative_id);

COMMIT;
