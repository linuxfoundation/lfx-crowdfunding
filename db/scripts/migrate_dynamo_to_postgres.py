#!/usr/bin/env python3
# Copyright The Linux Foundation and each contributor to CommunityBridge.
# SPDX-License-Identifier: MIT

"""
DynamoDB → PostgreSQL Migration Script
=======================================
Migrates all lff-prod-* DynamoDB tables into the PostgreSQL schema defined
in db/migrations/001_initial.up.sql (v2 — fully normalised initiatives schema).

Source → Target mapping
-----------------------
  lff-prod-users                                 → users
  lff-prod-organizations                         → organizations
  lff-prod-projects  +  lff-prod-entities        → initiatives  (merged)
                                                 → initiative_goals
                                                 → initiative_beneficiaries
                                                 → initiative_custom_websites
                                                 → initiative_contributors
                                                 → initiative_mentors
                                                 → initiative_program_info_terms
                                                 → initiative_program_info_skills
                                                 → initiative_program_info_config
                                                 → initiative_program_info_custom_term
                                                 → initiative_sponsorship_tiers
                                                 → initiative_ostif_detail
                                                 → initiative_contacts
                                                 → initiative_github_stats
                                                 → initiative_stats
                                                 → initiative_entity_details
  lff-prod-donations + lff-prod-entity-donations → donations     (merged)
  lff-prod-subscriptions
    + lff-prod-entity-subscriptions              → subscriptions (merged)

Key notes
---------
- initiatives.id (UUID PK) is generated deterministically via
  _as_uuid(projectId / entityId) so FK lookups from donations/subscriptions
  remain stable across re-runs.
- DynamoDB entity type quirk: SaveEntity rewrites 'general fund' → 'initiative'
  before every PutItem. Migration reverses this:
    if entityType == 'initiative' → restore to 'general fund'
- Budget.AmountInCents is serialised with json tag "amount" — access as
  budget.get("amount"), NOT budget.get("amountInCents").
- Excluded from schema (no DynamoDB write path / computed at read-time):
    balance, funding_status, entity_stats, sponsors, diversity,
    vulnerability_summary, badges, projectStats.totalRaised
- All INSERTs use ON CONFLICT … DO UPDATE (idempotent; safe to re-run).

Usage
-----
  export AWS_ACCESS_KEY_ID=...
  export AWS_SECRET_ACCESS_KEY=...
  export AWS_SESSION_TOKEN=...          # for temporary/STS credentials
  export AWS_REGION=us-east-1

  export PG_DSN="host=localhost port=5432 dbname=lff user=postgres password=..."

  python3 migrate_dynamo_to_postgres.py

Dependencies
------------
  pip install boto3 psycopg2-binary
"""

import json
import logging
import os
import sys
import uuid
from decimal import Decimal

import boto3
import psycopg2
import psycopg2.extras
from boto3.dynamodb.types import TypeDeserializer as _TypeDeserializer

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
REGION = os.environ.get("AWS_REGION", "us-east-1")
PG_DSN = os.environ.get(
    "PG_DSN",
    "host=localhost port=5432 dbname=lff user=postgres password=postgres",
)

# Stable UUID namespace — must not change between runs to keep IDs deterministic
_UUID_NS = uuid.UUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

# ---------------------------------------------------------------------------
# DynamoDB helpers
# ---------------------------------------------------------------------------
_deser = _TypeDeserializer()


def _deserialize(item: dict) -> dict:
    """Convert DynamoDB typed attribute map → plain Python dict."""
    return {k: _deser.deserialize(v) for k, v in item.items()}


def scan_table(client, table_name: str) -> list:
    """Full table scan with automatic pagination."""
    log.info("Scanning %-48s", table_name + " ...")
    items: list = []
    kwargs: dict = {"TableName": table_name}
    while True:
        resp = client.scan(**kwargs)
        items.extend(_deserialize(raw) for raw in resp.get("Items", []))
        lek = resp.get("LastEvaluatedKey")
        if not lek:
            break
        kwargs["ExclusiveStartKey"] = lek
    log.info("  → %d items", len(items))
    return items


# ---------------------------------------------------------------------------
# General helpers
# ---------------------------------------------------------------------------

def _uuid5(scope: str, *parts) -> str:
    key = "|".join(str(p) for p in parts)
    return str(uuid.uuid5(_UUID_NS, f"{scope}:{key}"))


def _as_uuid(value) -> str | None:
    """Return a valid UUID string or None; coerce non-UUID strings via uuid5."""
    if value is None:
        return None
    s = str(value).strip()
    if not s:
        return None
    try:
        return str(uuid.UUID(s))
    except ValueError:
        return _uuid5("coerce", s)


def _as_int(value, default: int = 0) -> int:
    if value is None:
        return default
    if isinstance(value, Decimal):
        return int(value)
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def _as_nonneg_int(value, default: int = 0) -> int:
    """Like _as_int but clamps the result to 0 — for CHECK (col >= 0) columns."""
    return max(0, _as_int(value, default))


def _to_jsonb(value) -> str | None:
    if value is None:
        return None

    def _default(obj):
        if isinstance(obj, Decimal):
            return float(obj)
        raise TypeError(f"Cannot serialize {type(obj)}")

    return json.dumps(value, default=_default)


def _normalize_status(status: str | None) -> str | None:
    """Normalize status values: 'hide' → 'hidden'; others pass through."""
    if status == "hide":
        return "hidden"
    return status


def _redact_dsn(dsn: str) -> str:
    """Redact password from PostgreSQL DSN for safe logging."""
    import re
    return re.sub(r'password=\S+', 'password=***', dsn)


def _parse_ts(s: str | None):
    """Parse DynamoDB timestamp string → tz-aware datetime (or None).

    Handles:
      - Standard formats: '2024-05-07 09:53:16 +0000', '2023-12-01T20:31:33Z'
      - Go time.String() format with nanoseconds + monotonic suffix:
        '2019-04-02 15:42:26.518360269 +0000 UTC m=+2580.502337766'
    """
    if not s:
        return None
    import re
    from datetime import datetime, timezone

    # Strip Go monotonic clock suffix: ' UTC m=+...' or ' UTC m=-...'
    cleaned = re.sub(r'\s+UTC\s+m=[+-][\d.]+$', '', s)
    # Truncate sub-second precision to at most 6 digits (microseconds)
    cleaned = re.sub(r'(\.\d{6})\d+', r'\1', cleaned)

    for fmt in (
        "%Y-%m-%d %H:%M:%S.%f %z",
        "%Y-%m-%d %H:%M:%S %z",
        "%Y-%m-%dT%H:%M:%S.%f%z",
        "%Y-%m-%dT%H:%M:%S%z",
        "%Y-%m-%d %H:%M:%S.%f",
        "%Y-%m-%d %H:%M:%S",
        "%Y-%m-%dT%H:%M:%S",
        "%Y-%m-%d",
    ):
        try:
            dt = datetime.strptime(cleaned, fmt)
            if dt.tzinfo is None:
                dt = dt.replace(tzinfo=timezone.utc)
            return dt
        except ValueError:
            continue
    log.warning("Could not parse timestamp: %r", s)
    return None


def _trunc(value: str | None, max_len: int, label: str = "") -> str | None:
    """Truncate a string to max_len characters, warning when it happens."""
    if value is None:
        return None
    if len(value) > max_len:
        log.warning("Truncating %s value (%d chars → %d): %r…", label, len(value), max_len, value[:60])
        return value[:max_len]
    return value


# ---------------------------------------------------------------------------
# Migration: user
# ---------------------------------------------------------------------------

def migrate_users(cur, users: list, placeholder_ids: set) -> set:
    """
    Upsert users from lff-prod-users.
    Insert placeholder rows for any user_id referenced elsewhere but absent here.
    Returns the complete set of known user_ids after migration.
    """
    log.info("Migrating users (%d records + %d placeholders) …", len(users), len(placeholder_ids))

    sql = """
        INSERT INTO users (user_id, email, given_name, family_name, name, avatar_url)
        VALUES (%s, %s, %s, %s, %s, %s)
        ON CONFLICT (user_id) DO UPDATE SET
            email       = EXCLUDED.email,
            given_name  = EXCLUDED.given_name,
            family_name = EXCLUDED.family_name,
            name        = EXCLUDED.name,
            avatar_url  = EXCLUDED.avatar_url
    """
    rows: list = []
    known: set = set()

    for u in users:
        uid = u.get("id")
        if not uid:
            continue
        known.add(uid)
        rows.append((uid, u.get("email"), u.get("givenName"), u.get("familyName"), u.get("name"), u.get("avatarUrl")))

    for uid in placeholder_ids:
        if uid and uid not in known:
            known.add(uid)
            rows.append((uid, None, None, None, None, None))

    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("  → %d user rows upserted", len(rows))
    return known


# ---------------------------------------------------------------------------
# Migration: organization
# ---------------------------------------------------------------------------

def migrate_organizations(cur, orgs: list, known_users: set) -> set:
    """
    Upsert organizations from lff-prod-organizations.
    Returns the set of postgres organization UUIDs inserted (for FK resolution).
    """
    log.info("Migrating organizations (%d records) …", len(orgs))

    sql = """
        INSERT INTO organizations (id, owner_id, name, avatar_url, status)
        VALUES (%s, %s, %s, %s, %s)
        ON CONFLICT (id) DO UPDATE SET
            owner_id   = EXCLUDED.owner_id,
            name       = EXCLUDED.name,
            avatar_url = EXCLUDED.avatar_url,
            status     = EXCLUDED.status
    """
    rows: list = []
    known_org_ids: set = set()
    skipped = 0

    for o in orgs:
        owner_id = o.get("ownerId")
        if owner_id not in known_users:
            skipped += 1
            log.debug("Skip org %s — owner %s not found", o.get("organizationId"), owner_id)
            continue

        pg_id = _as_uuid(o.get("organizationId")) or str(uuid.uuid4())
        known_org_ids.add(pg_id)
        rows.append((pg_id, owner_id, o.get("name"), o.get("avatarUrl"), o.get("status")))

    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    if skipped:
        log.warning("  → %d organization(s) skipped (owner not found)", skipped)
    log.info("  → %d organization rows upserted", len(rows))
    return known_org_ids


# ---------------------------------------------------------------------------
# Project goal category ordering (matches convertProjectToDynamoRepresentation)
# ---------------------------------------------------------------------------
_PROJECT_GOAL_CATEGORIES = [
    ("development",   "development",   0),
    ("marketing",     "marketing",     1),
    ("meetups",       "meetups",       2),
    ("travel",        "travel",        3),
    ("bugBounty",     "bugBounty",     4),
    ("documentation", "documentation", 5),
    ("other",         "other",         6),
    # 'mentee' is handled separately (sort_order 7)
]


# ---------------------------------------------------------------------------
# Migration: initiatives  (merged from lff-prod-entities + lff-prod-projects)
# ---------------------------------------------------------------------------

def migrate_initiatives(cur, entities: list, projects: list, known_users: set) -> set:
    """
    Merge lff-prod-entities and lff-prod-projects into the initiatives table
    and all normalised child tables.

    Projects  → initiative_type = 'project';      initiative_id = projectId
    Entities  → initiative_type from entityType;  initiative_id = entityId
                QUIRK: DynamoDB stores 'general fund' as 'initiative'.
                       Migration restores it: 'initiative' → 'general fund'.

    Returns the set of PostgreSQL initiative UUIDs (initiatives.id) that were
    inserted — used as FK targets for the donations/subscriptions migration.
    """
    log.info(
        "Migrating initiatives (%d entities + %d projects = %d total) …",
        len(entities), len(projects), len(entities) + len(projects),
    )

    # ── Row buffers (one list per target table) ───────────────────────────
    initiative_rows:          list = []
    goals_rows:               list = []
    beneficiaries_rows:       list = []
    custom_websites_rows:     list = []
    contributors_rows:        list = []
    mentors_rows:             list = []
    program_info_terms_rows:     list = []
    program_info_skills_rows:    list = []
    program_info_config_rows:    list = []
    program_info_custom_term_rows: list = []
    sponsorship_tiers_rows:   list = []
    ostif_detail_rows:        list = []
    contacts_rows:            list = []
    github_stats_rows:        list = []
    stats_rows:               list = []
    entity_details_rows:      list = []

    known_initiative_ids: set = set()
    skipped = 0

    # ── Projects ─────────────────────────────────────────────────────────
    for p in projects:
        owner_id = p.get("ownerId")
        if owner_id not in known_users:
            skipped += 1
            log.debug("Skip project %s — owner %s not found", p.get("projectId"), owner_id)
            continue

        raw_initiative_id = p.get("projectId") or ""
        if not raw_initiative_id:
            skipped += 1
            log.warning("Skip project with no projectId (name=%s)", p.get("name"))
            continue

        pg_id = _as_uuid(raw_initiative_id) or _uuid5("project", owner_id, p.get("name", ""))
        known_initiative_ids.add(pg_id)

        # projectDetails nested map (ProjectDetails struct serialised via JSON tags)
        pd: dict = p.get("projectDetails") or {}
        # cachedDetails nested map (CachedDetails struct serialised via JSON tags)
        cd: dict = p.get("cachedDetails") or {}

        coc = pd.get("codeOfConduct")
        coc_url = coc.get("link") if isinstance(coc, dict) else None

        # ── Core initiative row ──────────────────────────────────────────
        initiative_rows.append((
            pg_id,                                       # id
            "project",                                   # initiative_type
            "projects",                                  # source_dynamo_table
            owner_id,                                    # owner_id
            p.get("name"),                               # name
            p.get("slug"),                               # slug
            _normalize_status(p.get("status")),          # status
            pd.get("industry"),                          # industry
            pd.get("description"),                       # description
            _trunc(pd.get("color"), 10, "color"),        # color
            p.get("logoUrl"),                            # logo_url
            pd.get("website"),                           # website_url
            coc_url,                                     # coc_url
            pd.get("ciiProjectID"),                      # cii_project_id
            p.get("planId"),                             # stripe_plan_id
            p.get("productId"),                          # stripe_product_id
            p.get("jobspringProjectId") or None,         # jobspring_project_id
            pd.get("stacksIdentifier") or None,          # stacks_identifier
            None,                                        # eventbrite_url  (entity only)
            None,                                        # application_url (entity only)
            _as_int(p.get("amountRaised")),               # amount_raised_in_cents
            False,                                       # accept_funding (entity only)
            None,                                        # event_start_date (entity only)
            None,                                        # event_end_date   (entity only)
            None,                                        # country  (entity only)
            None,                                        # city     (entity only)
            False,                                       # is_online (entity only)
            _parse_ts(p.get("createdOn")),               # created_on
            _parse_ts(p.get("updatedOn")),               # updated_on
        ))

        # ── Budget category goals ────────────────────────────────────────
        for cat_key, goal_name, sort_order in _PROJECT_GOAL_CATEGORIES:
            cat: dict = pd.get(cat_key) or {}
            budget: dict = cat.get("budget") or {}
            # Budget.AmountInCents is serialised with json tag "amount"
            amt    = _as_int(budget.get("amount"))
            alloc  = budget.get("allocation") or None
            repo   = (cat.get("repoLink") or None) if cat_key == "development" else None
            if amt > 0 or alloc or repo:
                goals_rows.append((
                    _uuid5("goal", raw_initiative_id, goal_name),
                    pg_id, goal_name, max(0, amt), alloc, repo,
                    None, None, None,     # description, color, icon (entity only)
                    sort_order,
                ))

        # ── Mentee as a special goal (sort_order 7) ──────────────────────
        mentee: dict = pd.get("mentee") or {}
        if mentee:
            m_budget: dict = mentee.get("budget") or {}
            m_amt   = _as_nonneg_int(m_budget.get("amount"))
            m_alloc = m_budget.get("allocation") or None
            if m_amt > 0 or m_alloc or mentee.get("mentor") or mentee.get("terms") or mentee.get("skills"):
                goals_rows.append((
                    _uuid5("goal", raw_initiative_id, "mentee"),
                    pg_id, "mentee", m_amt, m_alloc, None,
                    None, None, None,
                    7,
                ))

            # Mentors
            for idx, mentor in enumerate(mentee.get("mentor") or []):
                email = mentor.get("email") or None
                mentors_rows.append((
                    _uuid5("mentor", raw_initiative_id, email or mentor.get("name", "") or str(idx)),
                    pg_id,
                    mentor.get("name"),
                    email,
                    mentor.get("avatarURL") or None,
                    mentor.get("introduction") or None,
                ))

            # Mentee config (termsConditions bool — always present once mentee is set)
            program_info_config_rows.append((
                pg_id,
                bool(mentee.get("termsConditions", False)),
            ))

            # Mentee terms ([]string)
            for idx, term in enumerate(mentee.get("terms") or []):
                if term:
                    program_info_terms_rows.append((
                        _uuid5("mentee_term", raw_initiative_id, str(idx)),
                        pg_id, str(term), idx,
                    ))

            # Mentee skills ([]string)
            for skill in (mentee.get("skills") or []):
                if skill:
                    program_info_skills_rows.append((
                        _uuid5("mentee_skill", raw_initiative_id, str(skill)),
                        pg_id, str(skill),
                    ))

            # Mentee custom term — only when termName is non-empty
            ct: dict = mentee.get("customTerm") or {}
            if isinstance(ct, dict) and ct.get("termName"):
                program_info_custom_term_rows.append((
                    pg_id,
                    ct.get("termName"),
                    ct.get("startMonth") or None,
                    ct.get("endMonth") or None,
                    _as_int(ct.get("year")) or None,
                ))

        # ── Contributors ─────────────────────────────────────────────────
        for idx, c in enumerate(pd.get("contributors") or []):
            email = c.get("email") or None
            contributors_rows.append((
                _uuid5("contributor", raw_initiative_id, email or c.get("name", "") or str(idx)),
                pg_id, c.get("name"), email,
            ))

        # ── Beneficiaries ────────────────────────────────────────────────
        for idx, b in enumerate(pd.get("beneficiaries") or []):
            email = b.get("email") or None
            beneficiaries_rows.append((
                _uuid5("beneficiary", raw_initiative_id, email or b.get("name", "") or str(idx)),
                pg_id, b.get("name"), email,
            ))

        # ── Custom websites ──────────────────────────────────────────────
        for cw in (pd.get("customWebsites") or []):
            url = cw.get("url") or ""
            if url:
                custom_websites_rows.append((
                    _uuid5("custom_website", raw_initiative_id, url),
                    pg_id, cw.get("name") or None, url,
                ))

        # ── GitHub stats ─────────────────────────────────────────────────
        gh: dict = cd.get("githubStats") or {}
        if gh.get("forks") or gh.get("stars") or gh.get("openIssues"):
            github_stats_rows.append((
                pg_id,
                _as_int(gh.get("forks")),
                _as_int(gh.get("stars")),
                _as_int(gh.get("openIssues")),
            ))

        # ── Project stats (backers only — totalRaised excluded) ──────────
        ps: dict = cd.get("projectStats") or {}
        stats_rows.append((
            pg_id,
            _as_int(ps.get("backers")),
        ))

    # ── Entities ─────────────────────────────────────────────────────────
    for e in entities:
        owner_id = e.get("ownerId")
        if owner_id not in known_users:
            skipped += 1
            log.debug("Skip entity %s — owner %s not found", e.get("entityId"), owner_id)
            continue

        raw_initiative_id = e.get("entityId") or ""
        if not raw_initiative_id:
            skipped += 1
            log.warning("Skip entity with no entityId (name=%s)", e.get("name"))
            continue

        pg_id = _as_uuid(raw_initiative_id) or _uuid5("entity", owner_id, e.get("name", ""))
        known_initiative_ids.add(pg_id)

        # entityType quirk: SaveEntity rewrites 'general fund' → 'initiative'
        # before every PutItem. Reverse it here.
        entity_type: str = e.get("entityType") or ""
        if entity_type == "initiative":
            entity_type = "general fund"

        # ── Core initiative row ──────────────────────────────────────────
        initiative_rows.append((
            pg_id,                                                # id
            entity_type,                                          # initiative_type
            "entities",                                           # source_dynamo_table
            owner_id,                                             # owner_id
            e.get("name"),                                        # name
            e.get("slug") or None,                                # slug
            _normalize_status(e.get("status")),                   # status
            e.get("industry") or None,                            # industry
            e.get("description") or None,                         # description
            _trunc(e.get("color"), 10, "color"),                  # color
            e.get("logoUrl") or None,                             # logo_url
            e.get("websiteURL") or None,                          # website_url
            e.get("cocURL") or None,                              # coc_url
            None,                                                 # cii_project_id (project only)
            e.get("stripePlanId") or None,                        # stripe_plan_id
            e.get("stripeProductId") or None,                     # stripe_product_id
            None,                                                 # jobspring_project_id (project only)
            None,                                                 # stacks_identifier (project only)
            e.get("eventbriteId") or None,                        # eventbrite_url
            e.get("applicationURL") or None,                      # application_url
            _as_int(e.get("amountRaised")),                        # amount_raised_in_cents
            bool(e.get("acceptFunding", False)),                   # accept_funding
            _parse_ts(e.get("eventStartDate")),                   # event_start_date
            _parse_ts(e.get("eventEndDate")),                     # event_end_date
            e.get("country") or None,                             # country
            e.get("city") or None,                                # city
            bool(e.get("isOnline", False)),                        # is_online
            _parse_ts(e.get("createdOn")),                        # created_on
            _parse_ts(e.get("updatedOn")),                        # updated_on
        ))

        # ── Entity goals[] ───────────────────────────────────────────────
        # Goal{name, description, goalColor, goalIcon, budget{amount, allocation}}
        for idx, goal in enumerate(e.get("goals") or []):
            goal_name = goal.get("name") or f"goal_{idx}"
            budget: dict = goal.get("budget") or {}
            goals_rows.append((
                _uuid5("goal", raw_initiative_id, goal_name),
                pg_id,
                goal_name,
                _as_nonneg_int(budget.get("amount")),
                budget.get("allocation") or None,
                None,                                   # repo_link (project only)
                goal.get("description") or None,
                _trunc(goal.get("goalColor"), 10, "goal_color"),
                goal.get("goalIcon") or None,
                idx,
            ))

        # ── Beneficiaries[] ──────────────────────────────────────────────
        # Entity field: Beneficiary []Beneficiary json:"beneficiaries"
        for idx, b in enumerate(e.get("beneficiaries") or []):
            email = b.get("email") or None
            beneficiaries_rows.append((
                _uuid5("beneficiary", raw_initiative_id, email or b.get("name", "") or str(idx)),
                pg_id, b.get("name") or None, email,
            ))

        # ── CustomWebsites[] ─────────────────────────────────────────────
        for cw in (e.get("customWebsites") or []):
            url = cw.get("url") or ""
            if url:
                custom_websites_rows.append((
                    _uuid5("custom_website", raw_initiative_id, url),
                    pg_id, cw.get("name") or None, url,
                ))

        # ── SponsorshipTiers[] ───────────────────────────────────────────
        # SponsorshipTier{name, description, color, icon, minimum}
        for idx, tier in enumerate(e.get("sponsorshipTiers") or []):
            tier_name = tier.get("name") or str(idx)
            sponsorship_tiers_rows.append((
                _uuid5("sponsorship_tier", raw_initiative_id, tier_name),
                pg_id,
                tier.get("name") or None,
                tier.get("description") or None,
                _trunc(tier.get("color"), 10, "tier_color"),
                tier.get("icon") or None,
                _as_nonneg_int(tier.get("minimum")),
                idx,
            ))

        # ── EntityDetails (map[string]string → JSONB) ────────────────────
        entity_details = e.get("entityDetails")
        if entity_details:
            entity_details_rows.append((
                pg_id,
                _to_jsonb(entity_details),
            ))

        # ── OSTIF-specific detail (ostif entity type only) ───────────────
        # entity.Detail interface{} is a domain.Detail struct when non-nil.
        # TypeDeserializer returns it as a plain dict.
        if entity_type == "ostif":
            raw_detail = e.get("detail")
            if isinstance(raw_detail, dict) and raw_detail:
                ostif_detail_rows.append((
                    pg_id,
                    raw_detail.get("monetizationStrategy") or None,
                    raw_detail.get("currentSecurityStrategy") or None,
                    raw_detail.get("licenseType") or None,
                    _as_nonneg_int(raw_detail.get("totalBudget")),
                    bool(raw_detail.get("termsConditions", False)),
                ))

                # Contacts (primaryContact, secondaryContact, technicalLead)
                for contact_type, detail_key in (
                    ("primary",       "primaryContact"),
                    ("secondary",     "secondaryContact"),
                    ("technical_lead","technicalLead"),
                ):
                    contact = raw_detail.get(detail_key)
                    if isinstance(contact, dict) and contact:
                        contacts_rows.append((
                            _uuid5("contact", raw_initiative_id, contact_type),
                            pg_id,
                            contact_type,
                            contact.get("firstName") or None,
                            contact.get("lastName") or None,
                            contact.get("email") or None,
                            contact.get("phoneNumber") or None,
                            contact.get("otherContactOption") or None,
                            contact.get("preferredContactMethod") or None,
                        ))

    # ── Phase 1: Insert initiatives ──────────────────────────────────────
    _insert_initiatives(cur, initiative_rows)

    # ── Phase 2: Insert child tables (FK → initiatives.initiative_id) ────
    _insert_goals(cur, goals_rows)
    _insert_beneficiaries(cur, beneficiaries_rows)
    _insert_custom_websites(cur, custom_websites_rows)
    _insert_contributors(cur, contributors_rows)
    _insert_mentors(cur, mentors_rows)
    _insert_program_info_terms(cur, program_info_terms_rows)
    _insert_program_info_skills(cur, program_info_skills_rows)
    _insert_program_info_config(cur, program_info_config_rows)
    _insert_program_info_custom_term(cur, program_info_custom_term_rows)
    _insert_sponsorship_tiers(cur, sponsorship_tiers_rows)
    _insert_ostif_detail(cur, ostif_detail_rows)
    _insert_contacts(cur, contacts_rows)
    _insert_github_stats(cur, github_stats_rows)
    _insert_stats(cur, stats_rows)
    _insert_entity_details(cur, entity_details_rows)

    # ── Phase 3: Classify mentorship projects ────────────────────────────
    # Projects that have a 'mentee' goal with amount_in_cents > 0 are
    # mentorship programs; update initiative_type accordingly.
    cur.execute("""
        UPDATE initiatives i
        SET initiative_type = 'mentorship'
        FROM initiative_goals g
        WHERE g.initiative_id = i.id
          AND g.name = 'mentee'
          AND g.amount_in_cents > 0
    """)
    log.info("    mentorship reclassification : %d row(s) updated", cur.rowcount)

    if skipped:
        log.warning("  → %d initiative(s) skipped (owner not found or missing ID)", skipped)
    log.info("  → %d initiative rows upserted", len(initiative_rows))
    return known_initiative_ids


# ---------------------------------------------------------------------------
# initiative row insert
# ---------------------------------------------------------------------------

def _insert_initiatives(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiatives (
            id, initiative_type, source_dynamo_table,
            owner_id,
            name, slug, status, industry, description, color,
            logo_url, website_url, coc_url, cii_project_id,
            stripe_plan_id, stripe_product_id,
            jobspring_project_id, stacks_identifier,
            eventbrite_url, application_url,
            amount_raised_in_cents, accept_funding,
            event_start_date, event_end_date,
            country, city, is_online,
            created_on, updated_on
        ) VALUES (
            %s,%s,%s, %s, %s,%s,%s,%s,%s,%s,
            %s,%s,%s,%s, %s,%s, %s,%s, %s,%s,
            %s,%s, %s,%s, %s,%s,%s, %s,%s
        )
        ON CONFLICT (id) DO UPDATE SET
            initiative_type      = EXCLUDED.initiative_type,
            source_dynamo_table  = EXCLUDED.source_dynamo_table,
            owner_id             = EXCLUDED.owner_id,
            name                 = EXCLUDED.name,
            slug                 = EXCLUDED.slug,
            status               = EXCLUDED.status,
            industry             = EXCLUDED.industry,
            description          = EXCLUDED.description,
            color                = EXCLUDED.color,
            logo_url             = EXCLUDED.logo_url,
            website_url          = EXCLUDED.website_url,
            coc_url              = EXCLUDED.coc_url,
            cii_project_id       = EXCLUDED.cii_project_id,
            stripe_plan_id       = EXCLUDED.stripe_plan_id,
            stripe_product_id    = EXCLUDED.stripe_product_id,
            jobspring_project_id = EXCLUDED.jobspring_project_id,
            stacks_identifier    = EXCLUDED.stacks_identifier,
            eventbrite_url       = EXCLUDED.eventbrite_url,
            application_url      = EXCLUDED.application_url,
            amount_raised_in_cents = EXCLUDED.amount_raised_in_cents,
            accept_funding       = EXCLUDED.accept_funding,
            event_start_date     = EXCLUDED.event_start_date,
            event_end_date       = EXCLUDED.event_end_date,
            country              = EXCLUDED.country,
            city                 = EXCLUDED.city,
            is_online            = EXCLUDED.is_online,
            updated_on           = EXCLUDED.updated_on
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=200)
    log.info("    initiatives        : %d rows", len(rows))


# ---------------------------------------------------------------------------
# Child table inserts
# All use ON CONFLICT (id) DO UPDATE so they are idempotent on re-run.
# UUIDs are generated deterministically via _uuid5 so the same source row
# always produces the same id.
# ---------------------------------------------------------------------------

def _insert_goals(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_goals
            (id, initiative_id, name, amount_in_cents, allocation, repo_link,
             description, color, icon, sort_order)
        VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET
            amount_in_cents = EXCLUDED.amount_in_cents,
            allocation      = EXCLUDED.allocation,
            repo_link       = EXCLUDED.repo_link,
            description     = EXCLUDED.description,
            color           = EXCLUDED.color,
            icon            = EXCLUDED.icon,
            sort_order      = EXCLUDED.sort_order
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_goals   : %d rows", len(rows))


def _insert_beneficiaries(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_beneficiaries (id, initiative_id, name, email)
        VALUES (%s,%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_beneficiaries: %d rows", len(rows))


def _insert_custom_websites(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_custom_websites (id, initiative_id, name, url)
        VALUES (%s,%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_custom_websites: %d rows", len(rows))


def _insert_contributors(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_contributors (id, initiative_id, name, email)
        VALUES (%s,%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_contributors: %d rows", len(rows))


def _insert_mentors(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_mentors
            (id, initiative_id, name, email, avatar_url, introduction)
        VALUES (%s,%s,%s,%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET
            name         = EXCLUDED.name,
            email        = EXCLUDED.email,
            avatar_url   = EXCLUDED.avatar_url,
            introduction = EXCLUDED.introduction
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_mentors : %d rows", len(rows))


def _insert_program_info_terms(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_program_info_terms (id, initiative_id, term, sort_order)
        VALUES (%s,%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET term = EXCLUDED.term, sort_order = EXCLUDED.sort_order
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_program_info_terms: %d rows", len(rows))


def _insert_program_info_skills(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_program_info_skills (id, initiative_id, skill)
        VALUES (%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET skill = EXCLUDED.skill
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_program_info_skills: %d rows", len(rows))


def _insert_program_info_config(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_program_info_config (initiative_id, terms_conditions)
        VALUES (%s,%s)
        ON CONFLICT (initiative_id) DO UPDATE SET
            terms_conditions = EXCLUDED.terms_conditions
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_program_info_config: %d rows", len(rows))


def _insert_program_info_custom_term(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_program_info_custom_term
            (initiative_id, term_name, start_month, end_month, year)
        VALUES (%s,%s,%s,%s,%s)
        ON CONFLICT (initiative_id) DO UPDATE SET
            term_name   = EXCLUDED.term_name,
            start_month = EXCLUDED.start_month,
            end_month   = EXCLUDED.end_month,
            year        = EXCLUDED.year
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_program_info_custom_term: %d rows", len(rows))


def _insert_sponsorship_tiers(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_sponsorship_tiers
            (id, initiative_id, name, description, color, icon, minimum, sort_order)
        VALUES (%s,%s,%s,%s,%s,%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET
            name        = EXCLUDED.name,
            description = EXCLUDED.description,
            color       = EXCLUDED.color,
            icon        = EXCLUDED.icon,
            minimum     = EXCLUDED.minimum,
            sort_order  = EXCLUDED.sort_order
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_sponsorship_tiers: %d rows", len(rows))


def _insert_ostif_detail(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_ostif_detail
            (initiative_id, monetization_strategy, current_security_strategy,
             license_type, total_budget_in_cents, terms_conditions)
        VALUES (%s,%s,%s,%s,%s,%s)
        ON CONFLICT (initiative_id) DO UPDATE SET
            monetization_strategy     = EXCLUDED.monetization_strategy,
            current_security_strategy = EXCLUDED.current_security_strategy,
            license_type              = EXCLUDED.license_type,
            total_budget_in_cents     = EXCLUDED.total_budget_in_cents,
            terms_conditions          = EXCLUDED.terms_conditions
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_ostif_detail: %d rows", len(rows))


def _insert_contacts(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_contacts
            (id, initiative_id, contact_type, first_name, last_name, email,
             phone_number, other_contact_option, preferred_contact_method)
        VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s)
        ON CONFLICT (id) DO UPDATE SET
            first_name               = EXCLUDED.first_name,
            last_name                = EXCLUDED.last_name,
            email                    = EXCLUDED.email,
            phone_number             = EXCLUDED.phone_number,
            other_contact_option     = EXCLUDED.other_contact_option,
            preferred_contact_method = EXCLUDED.preferred_contact_method
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_contacts: %d rows", len(rows))


def _insert_github_stats(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_github_stats
            (initiative_id, forks, stars, open_issues)
        VALUES (%s,%s,%s,%s)
        ON CONFLICT (initiative_id) DO UPDATE SET
            forks       = EXCLUDED.forks,
            stars       = EXCLUDED.stars,
            open_issues = EXCLUDED.open_issues,
            updated_on  = NOW()
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_github_stats: %d rows", len(rows))


def _insert_stats(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_stats (initiative_id, backers)
        VALUES (%s,%s)
        ON CONFLICT (initiative_id) DO UPDATE SET
            backers    = EXCLUDED.backers,
            updated_on = NOW()
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_stats   : %d rows", len(rows))


def _insert_entity_details(cur, rows: list) -> None:
    if not rows:
        return
    sql = """
        INSERT INTO initiative_entity_details (initiative_id, details)
        VALUES (%s,%s::jsonb)
        ON CONFLICT (initiative_id) DO UPDATE SET details = EXCLUDED.details
    """
    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    log.info("    initiative_entity_details: %d rows", len(rows))


# ---------------------------------------------------------------------------
# Migration: donations  (merged from lff-prod-donations + lff-prod-entity-donations)
# ---------------------------------------------------------------------------

def migrate_donations(
    cur,
    proj_donations: list,
    entity_donations: list,
    known_users: set,
    known_initiative_ids: set,
    known_org_ids: set,
):
    """
    Merge lff-prod-donations (initiative_id = projectId) and
    lff-prod-entity-donations (initiative_id = entityId) into donations.

    donations.initiative_id is a HARD FK → rows with unknown initiative IDs
    are skipped.
    donations.organization_id maps from the DynamoDB `orgId` attribute and is a
    HARD FK → resolved only when the UUID is in known_org_ids, else NULL.
    """
    log.info(
        "Migrating donations (%d project + %d entity = %d total) …",
        len(proj_donations), len(entity_donations),
        len(proj_donations) + len(entity_donations),
    )

    sql = """
        INSERT INTO donations (
            id, user_id, initiative_id, organization_id, cached_details, category,
            created_on, current_amount_in_cents, po_number, payment_method, status, stripe_charge_id
        ) VALUES (
            %s,%s,%s,%s,%s::jsonb,%s,%s,%s,%s,%s,%s,%s
        )
        ON CONFLICT (id) DO UPDATE SET
            user_id                 = EXCLUDED.user_id,
            initiative_id           = EXCLUDED.initiative_id,
            organization_id         = EXCLUDED.organization_id,
            cached_details          = EXCLUDED.cached_details,
            category                = EXCLUDED.category,
            created_on              = EXCLUDED.created_on,
            current_amount_in_cents = EXCLUDED.current_amount_in_cents,
            po_number               = EXCLUDED.po_number,
            payment_method          = EXCLUDED.payment_method,
            status                  = EXCLUDED.status,
            stripe_charge_id        = EXCLUDED.stripe_charge_id
    """
    rows: list = []
    skipped_user = 0
    skipped_initiative = 0

    # --- lff-prod-donations: PK = (userId, projectId) ----------------------
    for d in proj_donations:
        user_id = d.get("userId")
        if user_id not in known_users:
            skipped_user += 1
            continue
        initiative_id = _as_uuid(d.get("projectId"))
        if initiative_id not in known_initiative_ids:
            skipped_initiative += 1
            log.debug("Skip project donation — initiative %s not found", initiative_id)
            continue
        pg_id = _uuid5("proj_donation", str(user_id), str(d.get("projectId")))
        org_id = _as_uuid(d.get("orgId"))
        if org_id not in known_org_ids:
            org_id = None                           # orgId absent or not a known org
        rows.append((
            pg_id,
            user_id,
            initiative_id,
            org_id,
            _to_jsonb(d.get("cachedDetails")),
            d.get("category"),
            _parse_ts(d.get("createdOn")),
            _as_nonneg_int(d.get("currentAmountInCents")),
            d.get("ponumber") or None,              # po_number
            d.get("paymentmethod"),
            d.get("status"),
            d.get("stripeChargeId"),
        ))

    # --- lff-prod-entity-donations: PK = (userId, entityId) ----------------
    for d in entity_donations:
        user_id = d.get("userId")
        if user_id not in known_users:
            skipped_user += 1
            continue
        initiative_id = _as_uuid(d.get("entityId"))
        if initiative_id not in known_initiative_ids:
            skipped_initiative += 1
            log.debug("Skip entity donation — initiative %s not found", initiative_id)
            continue
        pg_id = _uuid5("entity_donation", str(user_id), str(d.get("entityId")))
        org_id = _as_uuid(d.get("orgId"))
        if org_id not in known_org_ids:
            org_id = None                           # orgId absent or not a known org
        rows.append((
            pg_id,
            user_id,
            initiative_id,
            org_id,
            _to_jsonb(d.get("cachedDetails")),
            d.get("category"),
            _parse_ts(d.get("createdOn")),
            _as_nonneg_int(d.get("currentAmountInCents")),
            d.get("ponumber") or None,              # po_number
            d.get("paymentmethod"),
            d.get("status"),
            d.get("stripeChargeId"),
        ))

    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    if skipped_user:
        log.warning("  → %d donation(s) skipped (user not found)", skipped_user)
    if skipped_initiative:
        log.warning("  → %d donation(s) skipped (initiative FK not found)", skipped_initiative)
    log.info("  → %d donation rows upserted", len(rows))


# ---------------------------------------------------------------------------
# Migration: subscriptions (merged from lff-prod-subscriptions + lff-prod-entity-subscriptions)
# ---------------------------------------------------------------------------

def migrate_subscriptions(
    cur,
    proj_subs: list,
    entity_subs: list,
    known_users: set,
    known_initiative_ids: set,
    known_org_ids: set,
):
    """
    Merge lff-prod-subscriptions (initiative_id = projectId) and
    lff-prod-entity-subscriptions (initiative_id = entityId) into subscriptions.

    subscriptions.initiative_id is a HARD FK → rows with unknown initiative IDs
    are skipped.
    subscriptions.organization_id maps from the DynamoDB `orgId` attribute and
    is a HARD FK → resolved only when the UUID is in known_org_ids, else NULL.
    """
    log.info(
        "Migrating subscriptions (%d project + %d entity = %d total) …",
        len(proj_subs), len(entity_subs),
        len(proj_subs) + len(entity_subs),
    )

    sql = """
        INSERT INTO subscriptions (
            id, user_id, initiative_id, organization_id, cached_details, category,
            created_on, current_amount_in_cents, frequency, status,
            stripe_subscription_id, stripe_subscription_item_id
        ) VALUES (
            %s,%s,%s,%s,%s::jsonb,%s,%s,%s,%s,%s,%s,%s
        )
        ON CONFLICT (id) DO UPDATE SET
            user_id                     = EXCLUDED.user_id,
            initiative_id               = EXCLUDED.initiative_id,
            organization_id             = EXCLUDED.organization_id,
            cached_details              = EXCLUDED.cached_details,
            category                    = EXCLUDED.category,
            created_on                  = EXCLUDED.created_on,
            current_amount_in_cents     = EXCLUDED.current_amount_in_cents,
            frequency                   = EXCLUDED.frequency,
            status                      = EXCLUDED.status,
            stripe_subscription_id      = EXCLUDED.stripe_subscription_id,
            stripe_subscription_item_id = EXCLUDED.stripe_subscription_item_id
    """
    rows: list = []
    skipped_user = 0
    skipped_initiative = 0

    # --- lff-prod-subscriptions: PK = (userId, projectId) ------------------
    for s in proj_subs:
        user_id = s.get("userId")
        if user_id not in known_users:
            skipped_user += 1
            continue
        initiative_id = _as_uuid(s.get("projectId"))
        if initiative_id not in known_initiative_ids:
            skipped_initiative += 1
            log.debug("Skip project subscription — initiative %s not found", initiative_id)
            continue
        pg_id = _uuid5("proj_subscription", str(user_id), str(s.get("projectId")))
        org_id = _as_uuid(s.get("orgId"))
        if org_id not in known_org_ids:
            org_id = None                           # orgId absent or not a known org
        rows.append((
            pg_id,
            user_id,
            initiative_id,
            org_id,
            _to_jsonb(s.get("cachedDetails")),
            s.get("category"),
            _parse_ts(s.get("createdOn")),
            _as_nonneg_int(s.get("currentAmountInCents")),
            s.get("frequency"),
            s.get("status"),
            s.get("stripeSubscriptionId"),
            s.get("stripeSubscriptionItemId"),
        ))

    # --- lff-prod-entity-subscriptions: PK = (userId, entityId) ------------
    for s in entity_subs:
        user_id = s.get("userId")
        if user_id not in known_users:
            skipped_user += 1
            continue
        initiative_id = _as_uuid(s.get("entityId"))
        if initiative_id not in known_initiative_ids:
            skipped_initiative += 1
            log.debug("Skip entity subscription — initiative %s not found", initiative_id)
            continue
        pg_id = _uuid5("entity_subscription", str(user_id), str(s.get("entityId")))
        org_id = _as_uuid(s.get("orgId"))
        if org_id not in known_org_ids:
            org_id = None                           # orgId absent or not a known org
        rows.append((
            pg_id,
            user_id,
            initiative_id,
            org_id,
            _to_jsonb(s.get("cachedDetails")),
            s.get("category"),
            _parse_ts(s.get("createdOn")),
            _as_nonneg_int(s.get("currentAmountInCents")),
            s.get("frequency"),
            s.get("status"),
            s.get("stripeSubscriptionId"),
            s.get("stripeSubscriptionItemId"),
        ))

    psycopg2.extras.execute_batch(cur, sql, rows, page_size=500)
    if skipped_user:
        log.warning("  → %d subscription(s) skipped (user not found)", skipped_user)
    if skipped_initiative:
        log.warning("  → %d subscription(s) skipped (initiative FK not found)", skipped_initiative)
    log.info("  → %d subscription rows upserted", len(rows))


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    log.info("=" * 60)
    log.info("DynamoDB → PostgreSQL migration starting")
    log.info("Region : %s", REGION)
    log.info("PG DSN : %s", _redact_dsn(PG_DSN))
    log.info("=" * 60)

    dynamo = boto3.client("dynamodb", region_name=REGION)

    # ── 1. Scan all DynamoDB tables ──────────────────────────────────────────
    users            = scan_table(dynamo, "lff-prod-users")
    orgs             = scan_table(dynamo, "lff-prod-organizations")
    entities         = scan_table(dynamo, "lff-prod-entities")
    projects         = scan_table(dynamo, "lff-prod-projects")
    proj_donations   = scan_table(dynamo, "lff-prod-donations")
    entity_donations = scan_table(dynamo, "lff-prod-entity-donations")
    proj_subs        = scan_table(dynamo, "lff-prod-subscriptions")
    entity_subs      = scan_table(dynamo, "lff-prod-entity-subscriptions")

    # ── 2. Collect all user IDs referenced in other tables ──────────────────
    # Insert placeholder user rows for any referenced ID absent from lff-prod-users
    # so FK constraints on dependent tables are never violated.
    extra_user_ids: set = set()
    for o in orgs:
        extra_user_ids.add(o.get("ownerId"))
    for e in entities:
        extra_user_ids.add(e.get("ownerId"))
    for p in projects:
        extra_user_ids.add(p.get("ownerId"))
    for d in proj_donations + entity_donations:
        extra_user_ids.add(d.get("userId"))
    for s in proj_subs + entity_subs:
        extra_user_ids.add(s.get("userId"))

    known_in_dynamo = {u.get("id") for u in users if u.get("id")}
    extra_user_ids -= known_in_dynamo
    extra_user_ids.discard(None)
    if extra_user_ids:
        log.info(
            "%d user ID(s) referenced in other tables but absent from lff-prod-users → placeholders",
            len(extra_user_ids),
        )

    # ── 3. Migrate in FK dependency order ────────────────────────────────────
    log.info("Connecting to PostgreSQL …")
    conn = psycopg2.connect(PG_DSN)
    conn.autocommit = False

    try:
        with conn.cursor() as cur:

            # 1. user — no FK dependencies
            known_users = migrate_users(cur, users, extra_user_ids)
            conn.commit()

            # 2. organization — FK → user
            known_org_ids = migrate_organizations(cur, orgs, known_users)
            conn.commit()

            # 3. initiatives — FK → user  (merged entities + projects)
            known_initiative_ids = migrate_initiatives(cur, entities, projects, known_users)
            conn.commit()

            # 4. donations — FK → user, FK → initiatives, FK → organizations (orgId)
            migrate_donations(cur, proj_donations, entity_donations, known_users, known_initiative_ids, known_org_ids)
            conn.commit()

            # 5. subscriptions — FK → user, FK → initiatives, FK → organizations (orgId)
            migrate_subscriptions(cur, proj_subs, entity_subs, known_users, known_initiative_ids, known_org_ids)
            conn.commit()

        log.info("=" * 60)
        log.info("Migration completed successfully!")
        log.info("=" * 60)

    except Exception:
        conn.rollback()
        log.exception("Migration failed — current phase rolled back (earlier phases already committed)")
        sys.exit(1)

    finally:
        conn.close()


if __name__ == "__main__":
    main()
