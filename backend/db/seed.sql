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
--
-- username       — LF SSO username (used by the app for identity lookup)
-- legacy_user_id — the old DynamoDB auth0 subject (kept for ledger JOIN enrichment)
--
-- Local dev mock principal: set DISABLED_MOCK_LOCAL_PRINCIPAL=<username> in backend/.env
-- (e.g. DISABLED_MOCK_LOCAL_PRINCIPAL=dev-user-001).  The value must match a seeded
-- username row or the create-initiative path will fail with ErrForbidden.
-- ============================================
INSERT INTO users (id, username, legacy_user_id, email, given_name, family_name, name, avatar_url) VALUES
  ('a0000000-0000-0000-0000-000000000001', 'dev-user-001', 'auth0|dev-user-001', 'alice@example.com',  'Alice',  'Smith',   'Alice Smith',   'https://i.pravatar.cc/150?u=alice'),
  ('a0000000-0000-0000-0000-000000000002', 'dev-user-002', 'auth0|dev-user-002', 'bob@example.com',    'Bob',    'Johnson', 'Bob Johnson',   'https://i.pravatar.cc/150?u=bob'),
  ('a0000000-0000-0000-0000-000000000003', 'dev-user-003', 'auth0|dev-user-003', 'carol@example.com',  'Carol',  'Williams','Carol Williams','https://i.pravatar.cc/150?u=carol'),
  ('a0000000-0000-0000-0000-000000000004', 'dev-user-004', 'auth0|dev-user-004', 'dave@example.com',   'Dave',   'Brown',   'Dave Brown',    'https://i.pravatar.cc/150?u=dave'),
  -- Ledger dev auth0 subjects — legacy_user_id values match transaction UserIDs in the dev Ledger instance
  -- so JOIN enrichment returns real donor names. Emails are synthetic (example.com) — do not use real addresses.
  ('a0000000-0000-0000-0000-000000000010', 'lewisoj',        'auth0|lewisoj',        'dev-lewisoj@example.com',        'Lewis',  'O',      'Lewis O',       'https://i.pravatar.cc/150?u=lewisoj'),
  ('a0000000-0000-0000-0000-000000000011', 'lewisojile',     'auth0|lewisojile',     'dev-lewisojile@example.com',     'Lewis',  'Ojile',  'Lewis Ojile',   'https://i.pravatar.cc/150?u=lewisojile'),
  ('a0000000-0000-0000-0000-000000000012', 'kelo',           'auth0|kelo',           'dev-kelo@example.com',           'Kelo',   'O',      'Kelo O',        'https://i.pravatar.cc/150?u=kelo'),
  ('a0000000-0000-0000-0000-000000000013', 'simk68',         'auth0|simk68',         'dev-simk68@example.com',         'Simrit', 'K',      'Simrit K',      'https://i.pravatar.cc/150?u=simk68'),
  ('a0000000-0000-0000-0000-000000000014', 'simk61',         'auth0|simk61',         'dev-simk61@example.com',         'Simrit', 'K',      'Simrit K',      'https://i.pravatar.cc/150?u=simk61'),
  ('a0000000-0000-0000-0000-000000000015', 'simk.ment.admin','auth0|simk.ment.admin','dev-simkadmin@example.com',      'Simrit', 'Admin',  'Simrit Admin',  'https://i.pravatar.cc/150?u=simkadmin'),
  ('a0000000-0000-0000-0000-000000000016', 'simk43',         'auth0|simk43',         'dev-simk43@example.com',         'Simrit', 'K',      'Simrit K',      'https://i.pravatar.cc/150?u=simk43'),
  -- Local mock principal — set DISABLED_MOCK_LOCAL_PRINCIPAL=local-dev-user in backend/.env
  ('a0000000-0000-0000-0000-000000000099', 'local-dev-user', NULL,                   'local-dev-user@local.dev',       'Local',  'Dev',    'Local Dev User','https://i.pravatar.cc/150?u=localdev')
ON CONFLICT (username) DO NOTHING;

-- ============================================
-- Organizations
-- owner_id references users.id (UUID)
-- ============================================
INSERT INTO organizations (id, owner_id, name, avatar_url, status) VALUES
  ('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Acme Corp',       'https://i.pravatar.cc/150?u=acme',  'Active'),
  ('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000002', 'Open Source Inc', 'https://i.pravatar.cc/150?u=ossinc','Active'),
  -- Real Ledger dev org UUID — matches organizationID on Kubernetes transactions from auth0|kelo.
  ('09b68fe3-12ae-4a7f-b021-7b522e87ae3d', 'a0000000-0000-0000-0000-000000000012', 'Google Cloud',    'https://ui-avatars.com/api/?name=Google+Cloud&background=4285F4&color=fff&size=128&bold=true', 'Active'),
  -- Real Ledger dev org UUIDs — match organizationID on Prometheus transactions.
  ('a5df9992-9374-445c-8b88-545f6178bb11', 'a0000000-0000-0000-0000-000000000001', 'Grafana Labs',     'https://ui-avatars.com/api/?name=Grafana+Labs&background=F46800&color=fff&size=128&bold=true', 'Active'),
  ('da78ebed-3c32-49ca-80db-234761b01979', 'a0000000-0000-0000-0000-000000000001', 'Weaveworks',       'https://ui-avatars.com/api/?name=Weaveworks&background=0077CC&color=fff&size=128&bold=true',   'Active')
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiatives — Projects
-- Real Ledger project IDs so ledger-stats-sync can pull live financials.
-- IDs sourced from GET /balance on the dev Ledger instance.
-- owner_id references users.id (UUID)
-- ============================================
INSERT INTO initiatives (
  id, initiative_type, source_dynamo_table, owner_id,
  name, slug, status, industry, description, color, logo_url, website_url,
  stripe_plan_id, stripe_product_id, amount_raised_in_cents, accept_funding,
  cii_project_id, stacks_identifier
) VALUES
  (
    'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'project', 'projects', 'a0000000-0000-0000-0000-000000000001',
    'Kubernetes', 'kubernetes', 'published', 'Technology',
    'Kubernetes (K8s) is an open-source system for automating deployment, scaling, and management of containerized applications. It groups containers that make up an application into logical units for easy management and discovery. Kubernetes builds upon 15 years of experience of running production workloads at Google, combined with best-of-breed ideas and practices from the community.',
    '#326CE5', 'https://jobspring-prod-uploads.s3.amazonaws.com/97f183f1-157e-4dd9-981d-fd3712ffe66c-.png', 'https://kubernetes.io',
    'plan_dev_kubernetes', 'prod_dev_kubernetes', 478500, true,
    'cii-001', 'kubernetes'
  ),
  (
    '57135156-cb73-4896-bbd3-8d503b568b3b', 'project', 'projects', 'a0000000-0000-0000-0000-000000000002',
    'Prometheus', 'prometheus', 'published', 'Technology',
    'Prometheus is an open-source systems monitoring and alerting toolkit originally built at SoundCloud. Since its inception in 2012, many companies and organizations have adopted Prometheus, and the project has a very active developer and user community. It is now a standalone open source project and maintained independently of any company. To emphasize this, and to clarify the project''s governance structure, Prometheus joined the Cloud Native Computing Foundation in 2016 as the second hosted project, after Kubernetes.',
    '#E6522C', 'https://jobspring-prod-uploads.s3.amazonaws.com/8b76b332-0137-44b3-bd6f-d2c8db04101d-.png', 'https://prometheus.io',
    'plan_dev_prometheus', 'prod_dev_prometheus', 99000000, true,
    'cii-002', 'prometheus'
  ),
  (
    '5f478c13-d72b-4f25-960a-a09249a5fc16', 'project', 'projects', 'a0000000-0000-0000-0000-000000000003',
    'OpenTelemetry', 'opentelemetry', 'published', 'Technology',
    'High-quality, ubiquitous, and portable telemetry to enable effective observability.',
    '#425CC7', NULL, 'https://opentelemetry.io',
    NULL, NULL, 156500, true,
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
    'c0000000-0000-0000-0000-000000000010', 'event', 'entities', 'a0000000-0000-0000-0000-000000000001',
    'KubeCon NA 2026', 'kubecon-na-2026', 'published', 'Technology',
    'The Cloud Native Computing Foundation flagship conference for adopters and technologists.',
    '#326CE5', NULL, 'https://events.linuxfoundation.org/kubecon-cloudnativecon-north-america/',
    750000, true,
    'https://eventbrite.com/e/kubecon-na-2026', 'https://events.linuxfoundation.org/kubecon/register',
    '2026-11-10 08:00:00+00', '2026-11-13 18:00:00+00',
    'US', 'Atlanta', false
  ),
  (
    'c0000000-0000-0000-0000-000000000011', 'event', 'entities', 'a0000000-0000-0000-0000-000000000002',
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
    'c0000000-0000-0000-0000-000000000020', 'mentorship', 'projects', 'a0000000-0000-0000-0000-000000000001',
    'Linux Kernel Bug Fixing', 'linux-kernel-bug-fixing', 'published', 'Technology',
    'Help new contributors fix real bugs in the Linux kernel under the guidance of experienced maintainers.',
    '#4CAF50', NULL, 'https://mentorship.lfx.linuxfoundation.org',
    'plan_dev_lk_mentorship', 'prod_dev_lk_mentorship', 980000, true,
    'jobspring-001'
  ),
  (
    'c0000000-0000-0000-0000-000000000021', 'mentorship', 'projects', 'a0000000-0000-0000-0000-000000000003',
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
    'c0000000-0000-0000-0000-000000000030', 'security_audit', 'projects', 'a0000000-0000-0000-0000-000000000002',
    'Kubernetes Security Audit', 'kubernetes-security-audit', 'published', 'Security',
    'Comprehensive third-party security audit of the Kubernetes codebase facilitated by OSTIF.',
    '#E05C00', NULL, 'https://ostif.org',
    1250000, true,
    'cii-003'
  ),
  (
    'c0000000-0000-0000-0000-000000000031', 'security_audit', 'projects', 'a0000000-0000-0000-0000-000000000004',
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
    'c0000000-0000-0000-0000-000000000040', 'general_fund', 'projects', 'a0000000-0000-0000-0000-000000000001',
    'CNCF General Fund', 'cncf-general-fund', 'published', 'Technology',
    'General funding pool for Cloud Native Computing Foundation projects — covers infrastructure, travel grants, and community programs.',
    '#9C27B0', 'https://cncf.io',
    3200000, true
  )
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiative Goals
-- ============================================
-- Remove legacy 'mentee' goal name (renamed to 'mentorship' to match Ledger txnCategory).
DELETE FROM initiative_goals WHERE initiative_id = 'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b' AND name = 'mentee';

INSERT INTO initiative_goals (initiative_id, name, amount_in_cents, allocation, repo_link, sort_order) VALUES
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'development',   2000000, 'Core development work',   'https://github.com/kubernetes/kubernetes', 0),
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'documentation', 500000,  'Docs and tutorials',       NULL, 1),
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'travel',        300000,  'Conference travel grants', NULL, 2),
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'mentorship',    800000,  'Mentorship stipends',      NULL, 3),
  ('57135156-cb73-4896-bbd3-8d503b568b3b', 'development',   800000,  'Exporter development',    'https://github.com/prometheus/prometheus', 0),
  ('57135156-cb73-4896-bbd3-8d503b568b3b', 'marketing',     250000,  'Community outreach',       NULL, 1),
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
INSERT INTO initiative_beneficiaries (id, initiative_id, name, email) VALUES
  ('f1000000-0000-0000-0000-000000000001', 'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'Alice Smith',   'alice@example.com'),
  ('f1000000-0000-0000-0000-000000000002', 'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'Bob Johnson',   'bob@example.com'),
  ('f1000000-0000-0000-0000-000000000003', '57135156-cb73-4896-bbd3-8d503b568b3b', 'Carol Williams','carol@example.com')
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- Initiative Custom Websites
-- ============================================
INSERT INTO initiative_custom_websites (initiative_id, name, url) VALUES
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'GitHub',     'https://github.com/kubernetes'),
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'Slack',      'https://slack.k8s.io'),
  ('57135156-cb73-4896-bbd3-8d503b568b3b', 'GitHub',     'https://github.com/prometheus')
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiative Contributors
-- ============================================
INSERT INTO initiative_contributors (initiative_id, name, email) VALUES
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'Dave Brown',  'dave@example.com'),
  ('57135156-cb73-4896-bbd3-8d503b568b3b', 'Alice Smith', 'alice@example.com')
ON CONFLICT DO NOTHING;

-- ============================================
-- Initiative Mentors
-- ============================================
INSERT INTO initiative_mentors (initiative_id, name, email, avatar_url, introduction) VALUES
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'Bob Johnson', 'bob@example.com',
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
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 41200, 108000, 2300),
  ('57135156-cb73-4896-bbd3-8d503b568b3b', 9800,  54000,  780),
  ('5f478c13-d72b-4f25-960a-a09249a5fc16', 3100,  21000,  410),
  ('c0000000-0000-0000-0000-000000000021', 2800,  12000,  190),
  ('c0000000-0000-0000-0000-000000000030', 41200, 108000, 2300)
ON CONFLICT (initiative_id) DO UPDATE
  SET forks = EXCLUDED.forks, stars = EXCLUDED.stars, open_issues = EXCLUDED.open_issues;

-- ============================================
-- Initiative Ledger Stats
-- ============================================
INSERT INTO initiative_ledger_stats (
  initiative_id, total_raised_cents, total_debited_cents,
  total_balance_cents, available_balance_cents, fee_balance_cents, supporters, sponsors
) VALUES
  -- Real Ledger data (synced from dev Ledger instance).
  -- sponsors JSONB individuals[].id retains the auth0 subject — ledger-stats-sync
  -- matches these against users.legacy_user_id to resolve names.
  ('c3ca17ca-edbc-4f26-aad0-d119e0af4c8b',  478500,       0,  478500,  478500,      0,   6,
   '{"orgs":[{"id":"09b68fe3-12ae-4a7f-b021-7b522e87ae3d","name":"Google Cloud","avatarUrl":"https://ui-avatars.com/api/?name=Google+Cloud&background=4285F4&color=fff&size=128&bold=true","total":250000}],"individuals":[{"id":"auth0|simk68","name":"Siim Kallas","avatarUrl":"https://i.pravatar.cc/128?u=simk68","total":113000},{"id":"auth0|simk.ment.admin","name":"Siim Admin","avatarUrl":"https://i.pravatar.cc/128?u=simkadmin","total":110500},{"id":"auth0|simk61","name":"Siim K","avatarUrl":"https://i.pravatar.cc/128?u=simk61","total":2500},{"id":"auth0|lewisoj","name":"Lewis O","avatarUrl":"https://i.pravatar.cc/128?u=lewisoj","total":2000},{"id":"auth0|lewisojile","name":"Lewis Ojile","avatarUrl":"https://i.pravatar.cc/128?u=lewisojile","total":500}]}'::jsonb),
  ('57135156-cb73-4896-bbd3-8d503b568b3b', 99000000,      0, 99000000, 99000000,    0,   2,
   '{"orgs":[{"id":"a5df9992-9374-445c-8b88-545f6178bb11","name":"Grafana Labs","avatarUrl":"https://ui-avatars.com/api/?name=Grafana+Labs&background=F46800&color=fff&size=128&bold=true","total":90000000},{"id":"da78ebed-3c32-49ca-80db-234761b01979","name":"Weaveworks","avatarUrl":"https://ui-avatars.com/api/?name=Weaveworks&background=0077CC&color=fff&size=128&bold=true","total":9000000}],"individuals":[]}'::jsonb),
  ('5f478c13-d72b-4f25-960a-a09249a5fc16',  156500,       0,  156500,  156500,      0,   1,
   '{"orgs":[],"individuals":[{"id":"auth0|aj.maintainer","name":"AJ Maintainer","avatarUrl":"https://i.pravatar.cc/128?u=ajmaintainer","total":156500}]}'::jsonb),
  -- Fabricated data for non-Ledger initiatives
  ('c0000000-0000-0000-0000-000000000010',  800000,   50000,  750000,  720000,  30000,  44,
   '{"orgs":[{"id":"org-001","name":"Red Hat","avatarUrl":"https://ui-avatars.com/api/?name=Red+Hat&background=EE0000&color=fff&size=128&bold=true","total":400000},{"id":"org-002","name":"IBM","avatarUrl":"https://ui-avatars.com/api/?name=IBM&background=006699&color=fff&size=128&bold=true","total":200000}],"individuals":[]}'::jsonb),
  ('c0000000-0000-0000-0000-000000000011',  350000,   30000,  320000,  305000,  15000,  28,
   '{"orgs":[{"id":"org-003","name":"VMware","avatarUrl":"https://ui-avatars.com/api/?name=VMware&background=607078&color=fff&size=128&bold=true","total":200000}],"individuals":[]}'::jsonb),
  ('c0000000-0000-0000-0000-000000000020', 1050000,   70000,  980000,  940000,  40000,  63,
   '{"orgs":[{"id":"org-004","name":"Microsoft","avatarUrl":"https://ui-avatars.com/api/?name=Microsoft&background=00A4EF&color=fff&size=128&bold=true","total":500000}],"individuals":[]}'::jsonb),
  ('c0000000-0000-0000-0000-000000000021',  580000,   40000,  540000,  510000,  30000,  41,
   '{"orgs":[],"individuals":[]}'::jsonb),
  ('c0000000-0000-0000-0000-000000000030', 1320000,   70000, 1250000, 1200000,  50000,  98,
   '{"orgs":[{"id":"org-005","name":"Snyk","avatarUrl":"https://ui-avatars.com/api/?name=Snyk&background=4C4A73&color=fff&size=128&bold=true","total":600000},{"id":"org-006","name":"Trail of Bits","avatarUrl":"https://ui-avatars.com/api/?name=Trail+of+Bits&background=1A1A2E&color=fff&size=128&bold=true","total":400000}],"individuals":[]}'::jsonb),
  ('c0000000-0000-0000-0000-000000000031',  920000,   50000,  870000,  840000,  30000,  74,
   '{"orgs":[],"individuals":[]}'::jsonb),
  ('c0000000-0000-0000-0000-000000000040', 3380000,  180000, 3200000, 3080000, 120000, 215,
   '{"orgs":[{"id":"org-007","name":"Amazon Web Services","avatarUrl":"https://ui-avatars.com/api/?name=AWS&background=FF9900&color=fff&size=128&bold=true","total":1500000},{"id":"org-008","name":"Google Cloud","avatarUrl":"https://ui-avatars.com/api/?name=Google+Cloud&background=4285F4&color=fff&size=128&bold=true","total":900000}],"individuals":[]}'::jsonb)
ON CONFLICT (initiative_id) DO UPDATE
  SET total_raised_cents      = EXCLUDED.total_raised_cents,
      total_debited_cents     = EXCLUDED.total_debited_cents,
      total_balance_cents     = EXCLUDED.total_balance_cents,
      available_balance_cents = EXCLUDED.available_balance_cents,
      fee_balance_cents       = EXCLUDED.fee_balance_cents,
      supporters              = EXCLUDED.supporters,
      sponsors                = EXCLUDED.sponsors;

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
-- user_id and organization_id reference users.id / organizations.id (UUID)
-- ============================================
INSERT INTO donations (
  id, user_id, initiative_id, organization_id,
  category, current_amount_in_cents, payment_method, status, stripe_charge_id,
  cached_details
) VALUES
  (
    'd0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000002', 'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', NULL,
    'development', 10000, 'card', 'Processed', 'ch_dev_001',
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000003', 'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', NULL,
    'travel', 5000, 'card', 'Processed', 'ch_dev_002',
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000004', '57135156-cb73-4896-bbd3-8d503b568b3b', NULL,
    'development', 2500, 'card', 'Processed', 'ch_dev_003',
    '{"initiative_name": "Prometheus", "initiative_slug": "prometheus"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000004',
    'a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000001',
    'Diversity Scholarships', 50000, 'invoice', 'Processed', NULL,
    '{"initiative_name": "KubeCon NA 2026", "initiative_slug": "kubecon-na-2026"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000005',
    'a0000000-0000-0000-0000-000000000002', 'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', NULL,
    'mentee', 7500, 'card', 'Pending', NULL,
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000006',
    'a0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000020', NULL,
    'stipends', 15000, 'card', 'Processed', 'ch_dev_006',
    '{"initiative_name": "Linux Kernel Bug Fixing", "initiative_slug": "linux-kernel-bug-fixing"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000007',
    'a0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000030', 'b0000000-0000-0000-0000-000000000002',
    'audit', 100000, 'invoice', 'Processed', NULL,
    '{"initiative_name": "Kubernetes Security Audit", "initiative_slug": "kubernetes-security-audit"}'
  ),
  (
    'd0000000-0000-0000-0000-000000000008',
    'a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000040', NULL,
    'infrastructure', 25000, 'card', 'Processed', 'ch_dev_008',
    '{"initiative_name": "CNCF General Fund", "initiative_slug": "cncf-general-fund"}'
  )
ON CONFLICT DO NOTHING;

-- ============================================
-- Subscriptions
-- user_id and organization_id reference users.id / organizations.id (UUID)
-- ============================================
INSERT INTO subscriptions (
  id, user_id, initiative_id, organization_id,
  category, current_amount_in_cents, frequency, status,
  stripe_subscription_id, stripe_subscription_item_id,
  cached_details
) VALUES
  (
    'e0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000003', 'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', NULL,
    'development', 1000, 'monthly', 'Active',
    'sub_dev_001', 'si_dev_001',
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'e0000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000004', '57135156-cb73-4896-bbd3-8d503b568b3b', NULL,
    'development', 500, 'monthly', 'Active',
    'sub_dev_002', 'si_dev_002',
    '{"initiative_name": "Prometheus", "initiative_slug": "prometheus"}'
  ),
  (
    'e0000000-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000002', 'c3ca17ca-edbc-4f26-aad0-d119e0af4c8b', 'b0000000-0000-0000-0000-000000000002',
    'travel', 2500, 'monthly', 'Cancelled',
    'sub_dev_003', 'si_dev_003',
    '{"initiative_name": "Kubernetes", "initiative_slug": "kubernetes"}'
  ),
  (
    'e0000000-0000-0000-0000-000000000004',
    'a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000040', NULL,
    'infrastructure', 5000, 'monthly', 'Active',
    'sub_dev_004', 'si_dev_004',
    '{"initiative_name": "CNCF General Fund", "initiative_slug": "cncf-general-fund"}'
  )
ON CONFLICT DO NOTHING;
