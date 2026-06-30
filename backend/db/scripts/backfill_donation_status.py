#!/usr/bin/env python3
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
"""
Backfill status='succeeded' on legacy migrated donations whose status is NULL,
using the Ledger DB's credit transactions as the source of truth.

Matching key: donations.stripe_charge_id = ledger.source_txn_id
Verification: donations.initiative_id    = ledger.project_id   (must also match)

A donation is marked 'succeeded' only when BOTH conditions are met.

Environment variables (all optional — CLI flags take precedence):

  Ledger DB  (matches the LEDGER_DB_* names in backend/.env)
  ─────────────────────────────────────────────────────────────
  LEDGER_DB_URL       Ledger DB hostname
  LEDGER_DB_PORT      Port  (default: 5432)
  LEDGER_DB_USER      DB user  (default: postgres)
  LEDGER_DB_PASSWORD  DB password
  LEDGER_DB_NAME      DB name  (default: ledger)

  CF (crowdfunding) DB
  ─────────────────────────────────────────────────────────────
  CF_DB_URL           Full postgres DSN for the CF DB.
                      If unset, falls back to the dev-tunnel defaults (127.0.0.1:15432).
                      CF_DB_PASSWORD must be set when using the dev tunnel.
                      Example: postgres://crowdfunding:<pass>@<host>:5432/crowdfunding

Usage (dry-run — default, no changes written):
    python3 backfill_donation_status.py

Usage (apply using env vars for both DBs):
    export LEDGER_DB_URL=<host>
    export LEDGER_DB_PASSWORD=<password>
    python3 backfill_donation_status.py --apply

Usage (apply to a specific CF DB, overriding CF_DB_URL):
    python3 backfill_donation_status.py --apply \\
      --cf-db-url "postgres://crowdfunding:<pass>@<host>:5432/crowdfunding"

Usage (apply using explicit flags for both DBs):
    python3 backfill_donation_status.py --apply \\
      --ledger-db-url "postgres://postgres:<pass>@<ledger-host>:5432/ledger" \\
      --cf-db-url     "postgres://crowdfunding:<pass>@<cf-host>:5432/crowdfunding"

Requires: psycopg2-binary
    pip install psycopg2-binary
"""

import argparse
import os
from dataclasses import dataclass
from typing import TYPE_CHECKING
from urllib.parse import parse_qs, urlencode, urlparse, urlunparse

if TYPE_CHECKING:
    import psycopg2.extensions

import psycopg2
import psycopg2.extras

# ---------------------------------------------------------------------------
# Defaults — overridden by env vars, then by CLI flags (highest priority)
# ---------------------------------------------------------------------------
DEFAULT_CF_DB = {
    "host": "127.0.0.1",
    "port": 15432,
    "user": "crowdfunding",
    "password": os.environ.get("CF_DB_PASSWORD", ""),
    "dbname": "crowdfunding",
    "connect_timeout": 10,
    "options": "-c search_path=crowdfunding",
}

# Ledger DB — reads LEDGER_DB_* env vars (same names as backend/.env).
# LEDGER_DB_URL and LEDGER_DB_PASSWORD must be set; there are no hardcoded defaults.
LEDGER_DB = {
    "host": os.environ.get("LEDGER_DB_URL", ""),
    "port": int(os.environ.get("LEDGER_DB_PORT", "5432")),
    "user": os.environ.get("LEDGER_DB_USER", "postgres"),
    "password": os.environ.get("LEDGER_DB_PASSWORD", ""),
    "dbname": os.environ.get("LEDGER_DB_NAME", "ledger"),
    "connect_timeout": 10,
}


# ---------------------------------------------------------------------------
# Data types
# ---------------------------------------------------------------------------
@dataclass
class LedgerCredit:
    txn_id: str
    project_id: str
    source_txn_id: str
    amount: int


@dataclass
class NullDonation:
    id: str
    initiative_id: str
    stripe_charge_id: str
    amount: int


# ---------------------------------------------------------------------------
# Ledger: build index of all credit txns keyed by source_txn_id
# ---------------------------------------------------------------------------
def load_ledger_credits(conn: "psycopg2.extensions.connection") -> dict[str, LedgerCredit]:
    """
    Returns {source_txn_id: LedgerCredit} for all credit rows in the Ledger.
    Rows with empty/null source_txn_id are skipped.
    """
    sql = """
        SELECT txn_id, project_id, source_txn_id, amount
        FROM public.ledger
        WHERE txn_type = 'credit'
          AND source_txn_id IS NOT NULL
          AND source_txn_id != ''
    """
    index: dict[str, LedgerCredit] = {}
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute(sql)
        for row in cur.fetchall():
            stid = row["source_txn_id"]
            if stid in index:
                # Duplicate — keep first seen (shouldn't happen in practice)
                continue
            index[stid] = LedgerCredit(
                txn_id=row["txn_id"],
                project_id=row["project_id"],
                source_txn_id=stid,
                amount=row["amount"],
            )
    return index


# ---------------------------------------------------------------------------
# CF DB: load all NULL-status donations that have a stripe_charge_id
# ---------------------------------------------------------------------------
def load_null_donations(conn: "psycopg2.extensions.connection") -> list[NullDonation]:
    sql = """
        SELECT id, initiative_id::text, stripe_charge_id, current_amount_in_cents
        FROM donations
        WHERE status IS NULL
          AND stripe_charge_id IS NOT NULL
          AND stripe_charge_id != ''
    """
    rows = []
    with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
        cur.execute(sql)
        for row in cur.fetchall():
            rows.append(NullDonation(
                id=str(row["id"]),
                initiative_id=str(row["initiative_id"]),
                stripe_charge_id=row["stripe_charge_id"],
                amount=row["current_amount_in_cents"],
            ))
    return rows


# ---------------------------------------------------------------------------
# Matching logic
# ---------------------------------------------------------------------------
@dataclass
class MatchResult:
    donation_id: str
    stripe_charge_id: str
    initiative_id: str
    ledger_txn_id: str
    amount_match: bool


def match_donations(
    donations: list[NullDonation],
    ledger_index: dict[str, LedgerCredit],
) -> tuple[list[MatchResult], list[NullDonation], list[NullDonation]]:
    """
    Returns:
      matched          — donations with a Ledger credit (initiative_id also agrees)
      initiative_mismatch — found in Ledger by charge ID but initiative_id differs
      not_found        — no Ledger credit for the stripe_charge_id at all
    """
    matched = []
    initiative_mismatch = []
    not_found = []

    for d in donations:
        credit = ledger_index.get(d.stripe_charge_id)
        if credit is None:
            not_found.append(d)
            continue

        if credit.project_id != d.initiative_id:
            initiative_mismatch.append(d)
            print(
                f"  [MISMATCH] donation {d.id}: charge {d.stripe_charge_id} "
                f"found in Ledger under project {credit.project_id!r} "
                f"but donation has initiative {d.initiative_id!r} — SKIP",
                flush=True,
            )
            continue

        matched.append(MatchResult(
            donation_id=d.id,
            stripe_charge_id=d.stripe_charge_id,
            initiative_id=d.initiative_id,
            ledger_txn_id=credit.txn_id,
            amount_match=(credit.amount == d.amount),
        ))

    return matched, initiative_mismatch, not_found


# ---------------------------------------------------------------------------
# CF DB: apply the update
# ---------------------------------------------------------------------------
def apply_updates(conn: "psycopg2.extensions.connection", matched: list[MatchResult]) -> int:
    """Sets status='succeeded' for all matched donation IDs. Returns row count."""
    ids = [m.donation_id for m in matched]
    sql = """
        UPDATE donations
        SET status = 'succeeded'
        WHERE id = ANY(%s::uuid[])
          AND status IS NULL
    """
    with conn.cursor() as cur:
        cur.execute(sql, (ids,))
        updated = cur.rowcount
    conn.commit()
    return updated


def _connect_cf(dsn: str):
    """
    Connect to the CF DB from a DSN URL.
    Strips unsupported query params (search_path) and passes them via options.
    """
    parsed = urlparse(dsn)
    query = parse_qs(parsed.query, keep_blank_values=True)
    schema = query.pop("search_path", ["crowdfunding"])[0]
    clean = urlunparse(parsed._replace(query=urlencode({k: v[0] for k, v in query.items()})))
    return psycopg2.connect(clean, options=f"-c search_path={schema}")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def parse_args():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument(
        "--apply",
        action="store_true",
        default=False,
        help="Write the UPDATE to the CF DB. Without this flag the script is read-only.",
    )
    p.add_argument(
        "--cf-db-url",
        default=None,
        help="PostgreSQL DSN for the CF DB (overrides the default dev tunnel). "
             "Format: postgres://user:pass@host:port/dbname",
    )
    p.add_argument(
        "--ledger-db-url",
        default=None,
        help="PostgreSQL DSN for the Ledger DB (overrides the default prod Ledger). "
             "Format: postgres://user:pass@host:port/dbname",
    )
    return p.parse_args()


def main():
    args = parse_args()

    # --- Validate required config ---
    if not args.ledger_db_url and not LEDGER_DB["host"]:
        print(
            "ERROR: Ledger DB host is required. "
            "Set LEDGER_DB_URL env var or pass --ledger-db-url.",
            flush=True,
        )
        raise SystemExit(1)

    cf_db_url_env = args.cf_db_url or os.environ.get("CF_DB_URL")
    if not cf_db_url_env and not DEFAULT_CF_DB["password"]:
        print(
            "ERROR: CF DB password is required. "
            "Set CF_DB_PASSWORD / CF_DB_URL env var or pass --cf-db-url.",
            flush=True,
        )
        raise SystemExit(1)

    # --- Connect to Ledger DB ---
    if args.ledger_db_url:
        print(f"Connecting to Ledger DB: {args.ledger_db_url.split('@')[-1]}…", flush=True)
        ledger_conn = psycopg2.connect(args.ledger_db_url)
    else:
        print("Connecting to Ledger prod DB…", flush=True)
        ledger_conn = psycopg2.connect(**LEDGER_DB)
    ledger_conn.set_session(readonly=True, autocommit=True)

    print("Loading Ledger credit transactions…", flush=True)
    ledger_index = load_ledger_credits(ledger_conn)
    ledger_conn.close()
    print(f"  {len(ledger_index):,} Ledger credit txns indexed by source_txn_id", flush=True)

    # --- Connect to CF DB --- (priority: --cf-db-url flag > CF_DB_URL env var > dev tunnel defaults)
    cf_db_url = cf_db_url_env
    if cf_db_url:
        print(f"Connecting to CF DB: {cf_db_url.split('@')[-1]}…", flush=True)
        cf_conn = _connect_cf(cf_db_url)
    else:
        print("Connecting to CF DB (dev tunnel)…", flush=True)
        cf_conn = psycopg2.connect(**DEFAULT_CF_DB)

    print("Loading NULL-status donations…", flush=True)
    donations = load_null_donations(cf_conn)
    print(f"  {len(donations):,} donations with status=NULL and a stripe_charge_id", flush=True)

    if not donations:
        print("Nothing to backfill.")
        cf_conn.close()
        return

    # --- Match ---
    print("\nMatching donations against Ledger…", flush=True)
    matched, mismatched, not_found = match_donations(donations, ledger_index)

    amount_mismatches = [m for m in matched if not m.amount_match]

    print(f"\nResults:")
    print(f"  Matched (both charge ID + initiative agree):  {len(matched):,}")
    print(f"    of which amount differs between DBs:        {len(amount_mismatches):,}")
    print(f"  Initiative ID mismatch (charge found, wrong initiative): {len(mismatched):,}")
    print(f"  Not found in Ledger at all:                   {len(not_found):,}")

    donations_by_id = {d.id: d for d in donations} if amount_mismatches else {}
    if amount_mismatches:
        print("\n  [WARN] Amount mismatches (donation_id | charge_id | cf_cents | ledger_cents):")
        for m in amount_mismatches:
            d = donations_by_id[m.donation_id]
            credit = ledger_index[m.stripe_charge_id]
            print(f"    {m.donation_id}  {m.stripe_charge_id}  {d.amount}  {credit.amount}")

    if not_found:
        print(f"\n  [INFO] First 10 charge IDs not in Ledger:")
        for d in not_found[:10]:
            print(f"    {d.stripe_charge_id}  (donation {d.id}, initiative {d.initiative_id})")

    if not matched:
        print("\nNo matches to apply.")
        cf_conn.close()
        return

    # --- Apply or dry-run ---
    if args.apply:
        print(f"\nApplying {len(matched):,} updates to donations.status = 'succeeded'…", flush=True)
        try:
            updated = apply_updates(cf_conn, matched)
            print(f"  Done — {updated:,} rows updated.")
        except Exception as exc:
            cf_conn.rollback()
            cf_conn.close()
            print(f"ERROR: update failed — {exc}", flush=True)
            raise SystemExit(1) from exc
    else:
        print(f"\nDRY RUN — {len(matched):,} donations would be set to 'succeeded'.")
        print("Re-run with --apply to commit the changes.")

    cf_conn.close()


if __name__ == "__main__":
    main()
