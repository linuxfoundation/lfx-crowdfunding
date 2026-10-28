#!/usr/bin/env python3
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
"""
Backfill the org claim (name and domain) onto legacy org donations (#408).

Rows created before #405/#406 carry only organization_id. This copies the CF
org name, plus a domain from org_domains.csv where one was hand-curated, onto:

  cf      donations and subscriptions (org_name, org_domain)
  stripe  live org subscriptions: subscription item and subscription metadata
          (org_name, org_domain), so future charges reach Ledger with the org

Steps run in that order; stripe reads the names written by cf. Every step
only fills empty values, so re-running is safe.

Ledger transactions are backfilled from ledger-service
(scripts/backfill-organization-claim), which owns writes to the Ledger DB.
--export FILE writes the org_id,name,domain CSV that script reads.

Environment variables (CLI flags take precedence):

  CF_DB_URL / CF_DB_PASSWORD   CF DB (falls back to the dev tunnel 127.0.0.1:15432)
  STRIPE_SECRET_KEY            needed for the stripe step

Usage (dry-run — default, nothing written):
    python3 backfill_org_claim.py

Usage (apply one step at a time):
    python3 backfill_org_claim.py --apply --only cf
    python3 backfill_org_claim.py --apply --only stripe

Usage (export the claims for the ledger-service backfill; reads only):
    python3 backfill_org_claim.py --export org_claims.csv

Requires: psycopg2-binary, stripe
"""

import argparse
import csv
import os
import re
from pathlib import Path

import psycopg2
import psycopg2.extras
import stripe
from backfill_donation_status import DEFAULT_CF_DB, _connect_cf

CSV_PATH = Path(__file__).with_name("org_domains.csv")
STEPS = ("cf", "stripe")
LIVE_SUBSCRIPTION_STATUSES = ("active", "past_due")

# Same shape Ledger's util.NormalizeDomain accepts: lowercase labels, no www.
DOMAIN_RE = re.compile(r"^(?!www\.)([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]*[a-z0-9])?$")
STRIPE_METADATA_VALUE_MAX = 500


def load_domains(path: Path = CSV_PATH) -> dict[str, str]:
    """Returns {org_id: domain} from the curated CSV, failing on a bad domain."""
    domains: dict[str, str] = {}
    with path.open(newline="") as f:
        for row in csv.DictReader(f):
            domain = row["domain"].strip().lower()
            if not DOMAIN_RE.match(domain):
                raise SystemExit(f"ERROR: invalid domain {domain!r} for org {row['org_id']} in {path.name}")
            domains[row["org_id"].strip()] = domain
    return domains


# ---------------------------------------------------------------------------
# cf: donations and subscriptions
# ---------------------------------------------------------------------------
def backfill_cf(conn, domains: dict[str, str]) -> None:
    with conn.cursor() as cur:
        cur.execute("CREATE TEMP TABLE org_domains (org_id UUID PRIMARY KEY, domain TEXT NOT NULL) ON COMMIT DROP")
        psycopg2.extras.execute_values(cur, "INSERT INTO org_domains VALUES %s", list(domains.items()))
        for table in ("donations", "subscriptions"):
            cur.execute(f"""
                UPDATE {table} t
                SET org_name   = COALESCE(NULLIF(t.org_name, ''), o.name),
                    org_domain = COALESCE(NULLIF(t.org_domain, ''), d.domain)
                FROM organizations o
                LEFT JOIN org_domains d ON d.org_id = o.id
                WHERE t.organization_id = o.id
                  AND (NULLIF(t.org_name, '') IS NULL
                       OR (NULLIF(t.org_domain, '') IS NULL AND d.domain IS NOT NULL))
            """)
            print(f"  {table}: {cur.rowcount:,} rows", flush=True)


# ---------------------------------------------------------------------------
# stripe: live org subscriptions
# ---------------------------------------------------------------------------
def backfill_stripe(conn, apply: bool) -> None:
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute("""
            SELECT id, stripe_subscription_id, stripe_subscription_item_id, org_name, org_domain
            FROM subscriptions
            WHERE organization_id IS NOT NULL
              AND NULLIF(org_name, '') IS NOT NULL
              AND NULLIF(stripe_subscription_id, '') IS NOT NULL
              AND status = ANY(%s)
        """, (list(LIVE_SUBSCRIPTION_STATUSES),))
        rows = cur.fetchall()
    print(f"  {len(rows):,} live org subscriptions", flush=True)

    for r in rows:
        # Same keys and limit the API writes for new subscriptions (stripe_client.go).
        meta = {"org_name": r["org_name"][:STRIPE_METADATA_VALUE_MAX]}
        if r["org_domain"]:
            meta["org_domain"] = r["org_domain"]
        if not stripe.api_key:
            # Dry run without a key: show the CF claim; what Stripe already holds is unknown.
            print(f"    {r['stripe_subscription_id']}  {meta}", flush=True)
            continue
        sub = stripe.Subscription.retrieve(r["stripe_subscription_id"])
        items = sub["items"]["data"]
        item = next((i for i in items if i["id"] == r["stripe_subscription_item_id"]), items[0])
        # Fill only empty keys, so a claim already on Stripe is never replaced.
        item_fill = missing_metadata(item["metadata"], meta)
        sub_fill = missing_metadata(sub["metadata"], meta)
        print(f"    {sub['id']}  item {item_fill or '-'}  subscription {sub_fill or '-'}", flush=True)
        if not apply:
            continue
        # The Ledger charge.succeeded hook reads the invoice line, which carries the item metadata.
        if item_fill:
            stripe.SubscriptionItem.modify(item["id"], metadata=item_fill)
        if sub_fill:
            stripe.Subscription.modify(sub["id"], metadata=sub_fill)


def missing_metadata(current, meta: dict[str, str]) -> dict[str, str]:
    """Returns the keys of meta that are empty or absent in current."""
    return {k: v for k, v in meta.items() if not (current or {}).get(k)}


# ---------------------------------------------------------------------------
# export: claims for the ledger-service backfill
# ---------------------------------------------------------------------------
def export_claims(conn, domains: dict[str, str], path: Path) -> None:
    with conn.cursor() as cur:
        cur.execute("SELECT id::text, name FROM organizations")
        names = dict(cur.fetchall())

    # CSV ids missing from CF (Ledger-only typos) get a domain and keep their Ledger name.
    with path.open("w", newline="") as f:
        w = csv.writer(f)
        w.writerow(("org_id", "name", "domain"))
        for org_id in sorted(names.keys() | domains.keys()):
            w.writerow((org_id, names.get(org_id) or "", domains.get(org_id, "")))
    print(f"  {len(names.keys() | domains.keys()):,} org claims written to {path.resolve()}", flush=True)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def parse_args():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--apply", action="store_true", help="Write changes. Without this flag nothing is written.")
    p.add_argument("--only", choices=STEPS, help="Run a single step.")
    p.add_argument("--cf-db-url", help="PostgreSQL DSN for the CF DB (overrides CF_DB_URL and the dev tunnel).")
    p.add_argument("--export", type=Path, metavar="FILE",
                   help="Write the org_id,name,domain CSV for ledger-service's backfill. Runs nothing else.")
    return p.parse_args()


def finish(conn, apply: bool) -> None:
    """Commits on --apply; otherwise rolls back so the dry run's counts are all it leaves behind."""
    if apply:
        conn.commit()
    else:
        conn.rollback()
    conn.close()


def main():
    args = parse_args()
    steps = [args.only] if args.only else list(STEPS)
    domains = load_domains()
    print(f"{len(domains):,} curated domains in {CSV_PATH.name}", flush=True)
    print("APPLY" if args.apply else "DRY RUN — counts are what --apply would change", flush=True)

    cf_db_url = args.cf_db_url or os.environ.get("CF_DB_URL")
    if not cf_db_url and not DEFAULT_CF_DB["password"]:
        raise SystemExit("ERROR: set CF_DB_PASSWORD / CF_DB_URL or pass --cf-db-url.")
    cf_conn = _connect_cf(cf_db_url) if cf_db_url else psycopg2.connect(**DEFAULT_CF_DB)

    if args.export:
        print("\n[export] org claims for ledger-service", flush=True)
        export_claims(cf_conn, domains, args.export)
        finish(cf_conn, apply=False)
        return

    if "cf" in steps:
        print("\n[cf] donations and subscriptions", flush=True)
        # On a dry run the writes stay uncommitted, so the stripe step still sees them.
        backfill_cf(cf_conn, domains)
        if args.apply:
            cf_conn.commit()

    if "stripe" in steps:
        print("\n[stripe] live org subscriptions", flush=True)
        # Optional on a dry run: with a key it reports what Stripe is actually missing.
        stripe.api_key = os.environ.get("STRIPE_SECRET_KEY", "")
        if args.apply and not stripe.api_key:
            raise SystemExit("ERROR: STRIPE_SECRET_KEY is required for --apply on the stripe step.")
        backfill_stripe(cf_conn, args.apply)

    finish(cf_conn, args.apply)
    if not args.apply:
        print("\nRe-run with --apply to commit the changes.")


if __name__ == "__main__":
    main()
