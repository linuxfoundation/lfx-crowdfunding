-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT
-- ============================================
-- Migration: Normalize legacy initiative_type values
-- Created: 2026-06-14
--
-- migrate_dynamo_to_postgres.py previously restored DynamoDB's entityType
-- quirk as 'general fund' (space) instead of the canonical 'general_fund'
-- (underscore) used by all new backend code. The script has been fixed, but
-- any environment that ran the old script (e.g. DEV) still has the wrong value.
-- This migration normalises those rows so the General Funds filter tab and
-- type validation work correctly.
--
-- Safe to re-run: WHERE clause is a no-op if already normalised.
-- ============================================

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

UPDATE initiatives
SET    initiative_type = 'general_fund',
       updated_on      = NOW()
WHERE  initiative_type = 'general fund';

COMMIT;
