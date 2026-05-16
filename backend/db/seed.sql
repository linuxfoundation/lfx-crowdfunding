-- Copyright The Linux Foundation and each contributor to LFX.
-- SPDX-License-Identifier: MIT
--
-- Local development seed data.
-- Run after migrations:
--   psql "$DATABASE_URL" -f db/seed.sql
-- Or via make:
--   make db-seed

SET search_path TO crowdfunding, public;

-- ============================================
-- Users
-- ============================================
INSERT INTO users (id, user_id, email, given_name, family_name, name, avatar_url) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'auth0|dev-user-001', 'alice@example.com',  'Alice',  'Smith',   'Alice Smith',   'https://i.pravatar.cc/150?u=alice'),
  ('a0000000-0000-0000-0000-000000000002', 'auth0|dev-user-002', 'bob@example.com',    'Bob',    'Johnson', 'Bob Johnson',   'https://i.pravatar.cc/150?u=bob'),
  ('a0000000-0000-0000-0000-000000000003', 'auth0|dev-user-003', 'carol@example.com',  'Carol',  'Williams','Carol Williams','https://i.pravatar.cc/150?u=carol'),
  ('a0000000-0000-0000-0000-000000000004', 'auth0|dev-user-004', 'dave@example.com',   'Dave',   'Brown',   'Dave Brown',    'https://i.pravatar.cc/150?u=dave')
ON CONFLICT (user_id) DO NOTHING;

-- ============================================
-- Organizations
-- ============================================
INSERT INTO organizations (id, owner_id, name, avatar_url, status) VALUES
  ('b0000000-0000-0000-0000-000000000001', 'auth0|dev-user-001', 'Acme Corp',       'https://i.pravatar.cc/150?u=acme',  'Active'),
  ('b0000000-0000-0000-0000-000000000002', 'auth0|dev-user-002', 'Open Source Inc', 'https://i.pravatar.cc/150?u=ossinc','Active')
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiatives — Projects
-- ============================================
INSERT INTO initiatives (
  id, initiative_type, source_dynamo_table, owner_id,
  name, slug, status, industry, description, color, logo_url, website_url,
  stripe_plan_id, stripe_product_id, amount_raised_in_cents, accept_funding,
  cii_project_id, stacks_identifier
) VALUES
  (
    'c0000000-0000-0000-0000-000000000001', 'project', 'projects', 'auth0|dev-user-001',
    'Kubernetes', 'kubernetes', 'published', 'Technology',
    'Production-Grade Container Orchestration — automate deployment, scaling, and management of containerized applications.',
    '#326CE5', 'https://kubernetes.io/images/favicon.png', 'https://kubernetes.io',
    'plan_dev_kubernetes', 'prod_dev_kubernetes', 4850000, true,
    'cii-001', 'kubernetes'
  ),
  (
    'c0000000-0000-0000-0000-000000000002', 'project', 'projects', 'auth0|dev-user-002',
    'Prometheus', 'prometheus', 'published', 'Technology',
    'An open-source systems monitoring and alerting toolkit originally built at SoundCloud.',
    '#E6522C', 'https://prometheus.io/assets/favicons/favicon.ico', 'https://prometheus.io',
    'plan_dev_prometheus', 'prod_dev_prometheus', 1230000, true,
    'cii-002', 'prometheus'
  ),
  (
    'c0000000-0000-0000-0000-000000000003', 'project', 'projects', 'auth0|dev-user-003',
    'OpenTelemetry', 'opentelemetry', 'Pending', 'Technology',
    'High-quality, ubiquitous, and portable telemetry to enable effective observability.',
    '#425CC7', NULL, 'https://opentelemetry.io',
    NULL, NULL, 0, false,
    NULL, 'opentelemetry'
  )
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiatives — Events
-- ============================================
INSERT INTO initiatives (
  id, initiative_type, source_dynamo_table, owner_id,
  name, slug, status, industry, description, color, logo_url, website_url,
  amount_raised_in_cents, accept_funding,
  eventbrite_url, application_url, event_start_date, event_end_date,
  country, city, is_online
) VALUES
  (
    'c0000000-0000-0000-0000-000000000010', 'event', 'entities', 'auth0|dev-user-001',
    'KubeCon NA 2026', 'kubecon-na-2026', 'published', 'Technology',
    'The Cloud Native Computing Foundation flagship conference for adopters and technologists.',
    '#326CE5', NULL, 'https://events.linuxfoundation.org/kubecon-cloudnativecon-north-america/',
    750000, true,
    'https://eventbrite.com/e/kubecon-na-2026', 'https://events.linuxfoundation.org/kubecon/register',
    '2026-11-10 08:00:00+00', '2026-11-13 18:00:00+00',
    'US', 'Atlanta', false
  ),
  (
    'c0000000-0000-0000-0000-000000000011', 'event', 'entities', 'auth0|dev-user-002',
    'Open Source Summit 2026', 'open-source-summit-2026', 'published', 'Technology',
    'Connecting the open source ecosystem under one roof for collaboration and education.',
    '#F9A825', NULL, 'https://events.linuxfoundation.org/open-source-summit-north-america/',
    320000, true,
    NULL, NULL,
    '2026-06-23 08:00:00+00', '2026-06-25 18:00:00+00',
    'US', 'Denver', false
  )
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiatives — Mentorships
-- ============================================
INSERT INTO initiatives (
  id, initiative_type, source_dynamo_table, owner_id,
  name, slug, status, industry, description, color, logo_url, website_url,
  stripe_plan_id, stripe_product_id, amount_raised_in_cents, accept_funding,
  jobspring_project_id
) VALUES
  (
    'c0000000-0000-0000-0000-000000000020', 'mentorship', 'projects', 'auth0|dev-user-001',
    'Linux Kernel Bug Fixing', 'linux-kernel-bug-fixing', 'published', 'Technology',
    'Help new contributors fix real bugs in the Linux kernel under the guidance of experienced maintainers.',
    '#4CAF50', NULL, 'https://mentorship.lfx.linuxfoundation.org',
    'plan_dev_lk_mentorship', 'prod_dev_lk_mentorship', 980000, true,
    'jobspring-001'
  ),
  (
    'c0000000-0000-0000-0000-000000000021', 'mentorship', 'projects', 'auth0|dev-user-003',
    'CNCF — Thanos', 'cncf-thanos', 'published', 'Technology',
    'Highly available Prometheus setup with long-term storage capabilities. Mentees work on core Thanos components.',
    '#4CAF50', NULL, 'https://mentorship.lfx.linuxfoundation.org',
    NULL, NULL, 540000, true,
    'jobspring-002'
  )
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiatives — Security Audits
-- ============================================
INSERT INTO initiatives (
  id, initiative_type, source_dynamo_table, owner_id,
  name, slug, status, industry, description, color, logo_url, website_url,
  amount_raised_in_cents, accept_funding,
  cii_project_id
) VALUES
  (
    'c0000000-0000-0000-0000-000000000030', 'security_audit', 'projects', 'auth0|dev-user-002',
    'Kubernetes Security Audit', 'kubernetes-security-audit', 'published', 'Security',
    'Comprehensive third-party security audit of the Kubernetes codebase facilitated by OSTIF.',
    '#E05C00', NULL, 'https://ostif.org',
    1250000, true,
    'cii-003'
  ),
  (
    'c0000000-0000-0000-0000-000000000031', 'security_audit', 'projects', 'auth0|dev-user-004',
    'Linux Kernel Vulnerability Remediation', 'linux-kernel-vuln-remediation', 'published', 'Security',
    'Review of practices and policies around how security vulnerabilities are reported, processed, and disclosed in the Linux kernel.',
    '#E05C00', NULL, 'https://ostif.org',
    870000, true,
    'cii-004'
  )
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiatives — General Funds
-- ============================================
INSERT INTO initiatives (
  id, initiative_type, source_dynamo_table, owner_id,
  name, slug, status, industry, description, color, website_url,
  amount_raised_in_cents, accept_funding
) VALUES
  (
    'c0000000-0000-0000-0000-000000000040', 'general_fund', 'projects', 'auth0|dev-user-001',
    'CNCF General Fund', 'cncf-general-fund', 'published', 'Technology',
    'General funding pool for Cloud Native Computing Foundation projects — covers infrastructure, travel grants, and community programs.',
    '#9C27B0', 'https://cncf.io',
    3200000, true
  )
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiative Goals
-- ============================================
INSERT INTO initiative_goals (initiative_id, name, amount_in_cents, allocation, repo_link, sort_order) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'development',   2000000, 'Core development work',   'https://github.com/kubernetes/kubernetes', 0),
  ('c0000000-0000-0000-0000-000000000001', 'documentation', 500000,  'Docs and tutorials',       NULL, 1),
  ('c0000000-0000-0000-0000-000000000001', 'travel',        300000,  'Conference travel grants', NULL, 2),
  ('c0000000-0000-0000-0000-000000000001', 'mentee',        800000,  'Mentorship stipends',      NULL, 3),
  ('c0000000-0000-0000-0000-000000000002', 'development',   800000,  'Exporter development',    'https://github.com/prometheus/prometheus', 0),
  ('c0000000-0000-0000-0000-000000000002', 'marketing',     250000,  'Community outreach',       NULL, 1),
  ('c0000000-0000-0000-0000-000000000020', 'stipends',      600000,  'Mentee stipends',          NULL, 0),
  ('c0000000-0000-0000-0000-000000000020', 'honorariums',   200000,  'Mentor honorariums',       NULL, 1),
  ('c0000000-0000-0000-0000-000000000021', 'stipends',      400000,  'Mentee stipends',          NULL, 0),
  ('c0000000-0000-0000-0000-000000000030', 'audit',        1000000,  'Security audit fees',      NULL, 0),
  ('c0000000-0000-0000-0000-000000000030', 'remediation',   250000,  'Fix prioritisation fund',  NULL, 1),
  ('c0000000-0000-0000-0000-000000000031', 'audit',         700000,  'Security audit fees',      NULL, 0),
  ('c0000000-0000-0000-0000-000000000040', 'infrastructure',1500000, 'CI/CD and hosting',        NULL, 0),
  ('c0000000-0000-0000-0000-000000000040', 'travel',         800000, 'Travel grants',            NULL, 1),
  ('c0000000-0000-0000-0000-000000000040', 'community',      500000, 'Community programs',       NULL, 2)
ON CONFLICT (initiative_id, name) DO NOTHING;

INSERT INTO initiative_goals (initiative_id, name, amount_in_cents, description, color, icon, sort_order) VALUES
  ('c0000000-0000-0000-0000-000000000010', 'Diversity Scholarships', 300000, 'Travel and registration scholarships for underrepresented groups', '#5E35B1', 'heart', 0),
  ('c0000000-0000-0000-0000-000000000010', 'Speaker Support',        200000, 'Support for first-time speakers', '#1976D2', 'mic', 1),
  ('c0000000-0000-0000-0000-000000000011', 'Student Scholarships',   150000, 'Scholarships for students and recent graduates', '#2E7D32', 'school', 0)
ON CONFLICT (initiative_id, name) DO NOTHING;

-- ============================================
-- Initiative Beneficiaries
-- ============================================
INSERT INTO initiative_beneficiaries (initiative_id, name, email) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'Alice Smith',   'alice@example.com'),
  ('c0000000-0000-0000-0000-000000000001', 'Bob Johnson',   'bob@example.com'),
  ('c0000000-0000-0000-0000-000000000002', 'Carol Williams','carol@example.com')
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiative Custom Websites
-- ============================================
INSERT INTO initiative_custom_websites (initiative_id, name, url) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'GitHub',     'https://github.com/kubernetes'),
  ('c0000000-0000-0000-0000-000000000001', 'Slack',      'https://slack.k8s.io'),
  ('c0000000-0000-0000-0000-000000000002', 'GitHub',     'https://github.com/prometheus')
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiative Contributors
-- ============================================
INSERT INTO initiative_contributors (initiative_id, name, email) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'Dave Brown',  'dave@example.com'),
  ('c0000000-0000-0000-0000-000000000002', 'Alice Smith', 'alice@example.com')
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiative Mentors
-- ============================================
INSERT INTO initiative_mentors (initiative_id, name, email, avatar_url, introduction) VALUES
  ('c0000000-0000-0000-0000-000000000001', 'Bob Johnson', 'bob@example.com',
   'https://i.pravatar.cc/150?u=bob',
   'Kubernetes contributor since 2016. Focused on scheduler and autoscaler.'),
  ('c0000000-0000-0000-0000-000000000020', 'Carol Williams', 'carol@example.com',
   'https://i.pravatar.cc/150?u=carol',
   'Linux kernel developer for 8 years. Specialises in memory management and scheduling.'),
  ('c0000000-0000-0000-0000-000000000021', 'Dave Brown', 'dave@example.com',
   'https://i.pravatar.cc/150?u=dave',
   'Thanos maintainer and Prometheus contributor. Works on long-term storage and query engine.')
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiative GitHub Stats
-- ============================================
INSERT INTO initiative_github_stats (initiative_id, forks, stars, open_issues) VALUES
  ('c0000000-0000-0000-0000-000000000001', 41200, 108000, 2300),
  ('c0000000-0000-0000-0000-000000000002', 9800,  54000,  780),
  ('c0000000-0000-0000-0000-000000000003', 3100,  21000,  410),
  ('c0000000-0000-0000-0000-000000000021', 2800,  12000,  190),
  ('c0000000-0000-0000-0000-000000000030', 41200, 108000, 2300)
ON CONFLICT (initiative_id) DO UPDATE
  SET forks = EXCLUDED.forks, stars = EXCLUDED.stars, open_issues = EXCLUDED.open_issues;

-- ============================================
-- Initiative Ledger Stats
-- ============================================
INSERT INTO initiative_ledger_stats (
  initiative_id, total_raised_cents, total_debited_cents,
  total_balance_cents, available_balance_cents, fee_balance_cents, supporters
) VALUES
  ('c0000000-0000-0000-0000-000000000001', 5200000,  350000, 4850000, 4700000, 150000, 312),
  ('c0000000-0000-0000-0000-000000000002', 1400000,  170000, 1230000, 1180000,  50000,  87),
  ('c0000000-0000-0000-0000-000000000010',  800000,   50000,  750000,  720000,  30000,  44),
  ('c0000000-0000-0000-0000-000000000011',  350000,   30000,  320000,  305000,  15000,  28),
  ('c0000000-0000-0000-0000-000000000020', 1050000,   70000,  980000,  940000,  40000,  63),
  ('c0000000-0000-0000-0000-000000000021',  580000,   40000,  540000,  510000,  30000,  41),
  ('c0000000-0000-0000-0000-000000000030', 1320000,   70000, 1250000, 1200000,  50000,  98),
  ('c0000000-0000-0000-0000-000000000031',  920000,   50000,  870000,  840000,  30000,  74),
  ('c0000000-0000-0000-0000-000000000040', 3380000,  180000, 3200000, 3080000, 120000, 215)
ON CONFLICT (initiative_id) DO UPDATE
  SET total_raised_cents      = EXCLUDED.total_raised_cents,
      total_debited_cents     = EXCLUDED.total_debited_cents,
      total_balance_cents     = EXCLUDED.total_balance_cents,
      available_balance_cents = EXCLUDED.available_balance_cents,
      fee_balance_cents       = EXCLUDED.fee_balance_cents,
      supporters              = EXCLUDED.supporters;

-- ============================================
-- Initiative Sponsorship Tiers (events)
-- ============================================
INSERT INTO initiative_sponsorship_tiers (initiative_id, name, description, color, icon, minimum, sort_order) VALUES
  ('c0000000-0000-0000-0000-000000000010', 'Platinum', 'Top-tier visibility across all event materials', '#B0BEC5', 'star',   5000000, 0),
  ('c0000000-0000-0000-0000-000000000010', 'Gold',     'Logo on website and printed programme',          '#FDD835', 'trophy', 2500000, 1),
  ('c0000000-0000-0000-0000-000000000010', 'Silver',   'Logo on event website',                          '#90A4AE', 'award',  1000000, 2),
  ('c0000000-0000-0000-0000-000000000011', 'Platinum', 'Premier sponsorship with speaking slot',         '#B0BEC5', 'star',   3000000, 0),
  ('c0000000-0000-0000-0000-000000000011', 'Gold',     'Booth and logo visibility',                      '#FDD835', 'trophy', 1500000, 1)
ON CONFLICT DO NOTHING;

-- ============================================
-- Donations
-- ============================================
INSERT INTO donations (
  id, user_id, initiative_id, organization_id,
  category, current_amount_in_cents, payment_method, status, stripe_charge_id,
  cached_details
) VALUES
  (
    'd0000000-0000-0000-0000-000000000001',
    'auth0|dev-user-002', 'c0000000-0000-0000-0000-000000000001', NULL,
    'development', 10000, 'card', 'Processed', 'ch_dev_001',
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000002',
    'auth0|dev-user-003', 'c0000000-0000-0000-0000-000000000001', NULL,
    'travel', 5000, 'card', 'Processed', 'ch_dev_002',
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000003',
    'auth0|dev-user-004', 'c0000000-0000-0000-0000-000000000002', NULL,
    'development', 2500, 'card', 'Processed', 'ch_dev_003',
    '{"initiative_name": "Prometheus", "initiative_slug": "prometheus"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000004',
    'auth0|dev-user-001', 'c0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000001',
    'Diversity Scholarships', 50000, 'invoice', 'Processed', NULL,
    '{"initiative_name": "KubeCon NA 2026", "initiative_slug": "kubecon-na-2026"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000005',
    'auth0|dev-user-002', 'c0000000-0000-0000-0000-000000000001', NULL,
    'mentee', 7500, 'card', 'Pending', NULL,
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000006',
    'auth0|dev-user-003', 'c0000000-0000-0000-0000-000000000020', NULL,
    'stipends', 15000, 'card', 'Processed', 'ch_dev_006',
    '{"initiative_name": "Linux Kernel Bug Fixing", "initiative_slug": "linux-kernel-bug-fixing"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000007',
    'auth0|dev-user-004', 'c0000000-0000-0000-0000-000000000030', 'b0000000-0000-0000-0000-000000000002',
    'audit', 100000, 'invoice', 'Processed', NULL,
    '{"initiative_name": "Kubernetes Security Audit", "initiative_slug": "kubernetes-security-audit"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000008',
    'auth0|dev-user-001', 'c0000000-0000-0000-0000-000000000040', NULL,
    'infrastructure', 25000, 'card', 'Processed', 'ch_dev_008',
    '{"initiative_name": "CNCF General Fund", "initiative_slug": "cncf-general-fund"}'
  )
ON CONFLICT DO NOTHING;

-- ============================================
-- Subscriptions
-- ============================================
INSERT INTO subscriptions (
  id, user_id, initiative_id, organization_id,
  category, current_amount_in_cents, frequency, status,
  stripe_subscription_id, stripe_subscription_item_id,
  cached_details
) VALUES
  (
    'e0000000-0000-0000-0000-000000000001',
    'auth0|dev-user-003', 'c0000000-0000-0000-0000-000000000001', NULL,
    'development', 1000, 'monthly', 'Active',
    'sub_dev_001', 'si_dev_001',
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'e0000000-0000-0000-0000-000000000002',
    'auth0|dev-user-004', 'c0000000-0000-0000-0000-000000000002', NULL,
    'development', 500, 'monthly', 'Active',
    'sub_dev_002', 'si_dev_002',
    '{"initiative_name": "Prometheus", "initiative_slug": "prometheus"}'
  ),
  (
    'e0000000-0000-0000-0000-000000000003',
    'auth0|dev-user-002', 'c0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000002',
    'travel', 2500, 'monthly', 'Cancelled',
    'sub_dev_003', 'si_dev_003',
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'e0000000-0000-0000-0000-000000000004',
    'auth0|dev-user-001', 'c0000000-0000-0000-0000-000000000040', NULL,
    'infrastructure', 5000, 'monthly', 'Active',
    'sub_dev_004', 'si_dev_004',
    '{"initiative_name": "CNCF General Fund", "initiative_slug": "cncf-general-fund"}'
  )
ON CONFLICT DO NOTHING;
