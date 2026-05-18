-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT
-- ============================================
-- Migration: Stripe 3D Secure (3DS) support
-- Version: 2.1.0
-- Created: 2026-05-18
-- ============================================
--
-- Adds columns required for the 3DS payment flow:
--   users         — Stripe Customer ID + default PaymentMethod reference.
--   donations     — PaymentIntent ID for async 3DS reconciliation.
--   subscriptions — Price ID + incomplete-by-default status for 3DS first-invoice flow.
--
-- Status defaults:
--   donations.status     'pending'    — advances to 'succeeded' | 'failed' via webhook
--   subscriptions.status 'incomplete' — advances to 'active' via invoice.payment_succeeded
-- ============================================

BEGIN;

SET LOCAL search_path TO crowdfunding, public;

-- ── users ──────────────────────────────────────────────────────────────────
-- stripe_customer_id:            cus_xxx — persisted on first payment so we
--                                never recreate a customer for the same user.
-- stripe_default_payment_method: pm_xxx  — attached after SetupIntent
--                                confirmation; used for all off-session charges.
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS stripe_customer_id            TEXT,
  ADD COLUMN IF NOT EXISTS stripe_default_payment_method TEXT;

-- ── donations ──────────────────────────────────────────────────────────────
-- stripe_payment_intent_id: pi_xxx — created before the charge attempt.
--   Allows payment_intent.succeeded / .payment_failed webhooks to reconcile
--   the async result when 3DS is required.
ALTER TABLE donations
  ADD COLUMN IF NOT EXISTS stripe_payment_intent_id TEXT;

ALTER TABLE donations
  ALTER COLUMN status SET DEFAULT 'pending';

-- ── subscriptions ──────────────────────────────────────────────────────────
-- stripe_price_id: price_xxx — the recurring Stripe Price used by this
--   subscription. Stored so cancel / update can reference the exact price.
ALTER TABLE subscriptions
  ADD COLUMN IF NOT EXISTS stripe_price_id TEXT;

ALTER TABLE subscriptions
  ALTER COLUMN status SET DEFAULT 'incomplete';

-- ── indexes ────────────────────────────────────────────────────────────────
-- Partial indexes: only rows with non-null Stripe IDs are indexed — migrated
-- rows that pre-date this migration will never have these columns populated.
CREATE INDEX IF NOT EXISTS idx_users_stripe_customer
  ON users(stripe_customer_id)
  WHERE stripe_customer_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_donations_payment_intent
  ON donations(stripe_payment_intent_id)
  WHERE stripe_payment_intent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_sub_id
  ON subscriptions(stripe_subscription_id)
  WHERE stripe_subscription_id IS NOT NULL;

COMMIT;
