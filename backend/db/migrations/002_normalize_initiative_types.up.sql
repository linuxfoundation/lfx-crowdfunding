-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT
-- ============================================
-- Migration: Normalize legacy initiative_type values
-- Created: 2026-06-14
--
-- Legacy DynamoDB rows imported with initiative_type = 'general fund'
-- (space-separated display string). The canonical value used by all new
-- code is 'general_fund' (underscore). Normalize the 2 legacy rows so
-- that the General Funds filter tab and type validation work correctly.
-- ============================================

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

UPDATE initiatives
SET    initiative_type = 'general_fund',
       updated_on      = NOW()
WHERE  initiative_type = 'general fund';

COMMIT;
