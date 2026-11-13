#!/usr/bin/env python3
# Copyright The Linux Foundation and each contributor to LFX.
# SPDX-License-Identifier: MIT
"""
Repair users rows keyed on an Auth0 subject instead of the LF username (#485).

A brand-new account's first token could carry its Auth0 subject as the
username (auth0|<lfid>), so SyncProfile created a users row for it. On the next
login the user got a second row under the plain username and lost access to
anything owned by the first. The API now refuses those principals; this script
fixes the rows already written.

For every users row whose username contains '|':

  merge   a plain row exists for the same person: its legacy_user_id is the
          auth0| username (bound on a later login), or its username is the
          auth0|<lfid> name without the prefix. Every users(id) foreign key and
          the username text columns (initiatives.updated_by,
          initiative_announcements.created_by/updated_by) move to the plain row,
          Stripe ids move over when the plain row has none, and the auth0| row
          is deleted.
  rename  auth0|<lfid> with no plain row: the row becomes <lfid>, and its
          legacy_user_id becomes the subject when it has none.
  review  nothing is changed and the row is listed for a human:
            - a hashed subject (auth0|<base58>) with no row bound to it; plain
              rows with the same email are listed as candidates
            - both rows have a stripe_customer_id
            - the two match rules point at different rows
            - the row carries a legacy_user_id other than its own subject

Everything runs in one transaction. Without --apply it is rolled back, so the
dry run executes the same statements and prints exactly what --apply commits.

After --apply, rerun the fga-reconcile CronJob (cmd/fga-backfill) so the
crowdfunding_initiative owner tuples point at user:<lfid>:

    kubectl -n crowdfunding-backend create job \\
      --from=cronjob/lfx-crowdfunding-backend-fga-reconcile fga-reconcile-485

Environment variables (CLI flags take precedence):

  CF_DB_URL / CF_DB_PASSWORD   CF DB (falls back to the dev tunnel 127.0.0.1:15432)

Usage:
    python3 repair_auth0_principal_users.py            # dry run
    python3 repair_auth0_principal_users.py --apply

Requires: psycopg2-binary
"""

import argparse
import os
import re
import sys
from dataclasses import dataclass, field

import psycopg2
import psycopg2.extras
from backfill_donation_status import DEFAULT_CF_DB, _connect_cf

# Inverse of auth-service's username_to_sub for "safe" usernames, the same rule
# as usernameFromAuth0Subject in backend/internal/infrastructure/auth/jwt.go.
SAFE_USERNAME_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,58}[A-Za-z0-9]$")
HEX_USERNAME_RE = re.compile(r"^[0-9a-f]{24,60}$")

# Columns that store a username as text rather than a users(id) reference.
USERNAME_TEXT_COLUMNS = (
    ("initiatives", "updated_by"),
    ("initiative_announcements", "created_by"),
    ("initiative_announcements", "updated_by"),
)

USER_COLUMNS = "id, username, legacy_user_id, email, stripe_customer_id, stripe_default_payment_method"


@dataclass
class Plan:
    action: str  # merge | rename | review
    row: dict
    target: dict | None = None
    new_username: str = ""
    reasons: list[str] = field(default_factory=list)
    candidates: list[dict] = field(default_factory=list)


def username_from_subject(sub: str) -> str | None:
    """auth0|jdoe -> jdoe; None for a hashed subject or another connection's."""
    if not sub.startswith("auth0|"):
        return None
    name = sub[len("auth0|"):]
    if not SAFE_USERNAME_RE.match(name) or HEX_USERNAME_RE.match(name):
        return None
    return name


def user_fk_columns(cur) -> list[tuple[str, str]]:
    """Every (table, column) with a foreign key to users(id), read from the catalog."""
    cur.execute(
        """
        SELECT c.conrelid::regclass::text AS tbl, a.attname AS col
        FROM pg_constraint c
        JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
        WHERE c.contype = 'f' AND c.confrelid = 'users'::regclass
        ORDER BY 1, 2
        """
    )
    return [(r["tbl"], r["col"]) for r in cur.fetchall()]


def fetch_user(cur, where: str, value: str) -> dict | None:
    cur.execute(f"SELECT {USER_COLUMNS} FROM users WHERE {where} = %s", (value,))
    return cur.fetchone()


def plan_row(cur, row: dict) -> Plan:
    sub = row["username"]
    lfid = username_from_subject(sub)

    by_subject = fetch_user(cur, "legacy_user_id", sub)
    if by_subject and by_subject["id"] == row["id"]:
        by_subject = None
    by_name = fetch_user(cur, "username", lfid) if lfid else None

    if by_subject and by_name and by_subject["id"] != by_name["id"]:
        return Plan("review", row, reasons=[
            f"legacy_user_id match {by_subject['username']!r} and username match {by_name['username']!r} are different rows"
        ])

    target = by_subject or by_name
    if row["legacy_user_id"] and row["legacy_user_id"] != sub:
        return Plan("review", row, target=target, reasons=[
            f"row carries legacy_user_id {row['legacy_user_id']!r}, not its own subject"
        ])

    if target:
        if row["stripe_customer_id"] and target["stripe_customer_id"]:
            return Plan("review", row, target=target, reasons=["both rows have a stripe_customer_id"])
        if target["legacy_user_id"] and target["legacy_user_id"] != sub:
            return Plan("review", row, target=target, reasons=[
                f"plain row is bound to another subject {target['legacy_user_id']!r}"
            ])
        return Plan("merge", row, target=target)

    if lfid:
        return Plan("rename", row, new_username=lfid)

    plan = Plan("review", row, reasons=["hashed subject with no row bound to it"])
    if row["email"]:
        cur.execute(
            f"SELECT {USER_COLUMNS} FROM users WHERE lower(email) = lower(%s) AND username NOT LIKE '%%|%%'",
            (row["email"],),
        )
        plan.candidates = cur.fetchall()
    return plan


def move_username_text(cur, old: str, new: str) -> dict[str, int]:
    moved = {}
    for table, column in USERNAME_TEXT_COLUMNS:
        cur.execute(f"UPDATE {table} SET {column} = %s WHERE {column} = %s", (new, old))
        if cur.rowcount:
            moved[f"{table}.{column}"] = cur.rowcount
    return moved


def owned_initiatives(cur, user_id: str) -> list[str]:
    cur.execute("SELECT slug FROM initiatives WHERE owner_id = %s ORDER BY slug", (user_id,))
    return [r["slug"] for r in cur.fetchall()]


def apply_merge(cur, plan: Plan, fk_columns: list[tuple[str, str]]) -> dict:
    old, target = plan.row, plan.target
    assert target is not None
    sub = old["username"]
    initiatives = owned_initiatives(cur, old["id"])
    moved = {}
    for table, column in fk_columns:
        cur.execute(f"UPDATE {table} SET {column} = %s WHERE {column} = %s", (target["id"], old["id"]))
        if cur.rowcount:
            moved[f"{table}.{column}"] = cur.rowcount
    moved.update(move_username_text(cur, sub, target["username"]))

    # Delete first: legacy_user_id is unique, and the old row may hold the subject.
    cur.execute("DELETE FROM users WHERE id = %s", (old["id"],))
    cur.execute(
        """
        UPDATE users SET
          legacy_user_id = COALESCE(legacy_user_id, %s),
          stripe_customer_id = COALESCE(stripe_customer_id, %s),
          stripe_default_payment_method = CASE WHEN stripe_customer_id IS NULL
            THEN %s ELSE stripe_default_payment_method END,
          updated_on = NOW()
        WHERE id = %s
        """,
        (sub, old["stripe_customer_id"], old["stripe_default_payment_method"], target["id"]),
    )
    return {"moved": moved, "initiatives": initiatives}


def apply_rename(cur, plan: Plan) -> dict:
    old = plan.row
    sub = old["username"]
    initiatives = owned_initiatives(cur, old["id"])
    cur.execute(
        """
        UPDATE users SET username = %s, legacy_user_id = COALESCE(legacy_user_id, %s), updated_on = NOW()
        WHERE id = %s
        """,
        (plan.new_username, sub, old["id"]),
    )
    moved = move_username_text(cur, sub, plan.new_username)
    return {"moved": moved, "initiatives": initiatives}


def describe(user: dict) -> str:
    stripe = user["stripe_customer_id"] or "-"
    return f"{user['username']} (id {user['id']}, legacy {user['legacy_user_id'] or '-'}, stripe {stripe})"


def run(conn) -> tuple[int, int]:
    """Plans and applies every row inside the caller's transaction. Returns (changed, review)."""
    cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
    # Nobody else may create or bind a users row while the plans are made and applied.
    cur.execute("LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE")
    fk_columns = user_fk_columns(cur)
    print("users(id) references: " + ", ".join(f"{t}.{c}" for t, c in fk_columns))

    cur.execute(f"SELECT {USER_COLUMNS} FROM users WHERE username LIKE '%%|%%' ORDER BY created_on, id")
    rows = cur.fetchall()
    print(f"{len(rows)} users row(s) with an Auth0-subject username\n")

    changed = review = 0
    touched_initiatives: list[str] = []
    for row in rows:
        plan = plan_row(cur, row)
        print(f"[{plan.action}] {describe(row)}")
        if plan.action == "review":
            review += 1
            for reason in plan.reasons:
                print(f"    reason: {reason}")
            if plan.target:
                print(f"    plain row: {describe(plan.target)}")
            for cand in plan.candidates:
                print(f"    candidate by email: {describe(cand)}")
            if not plan.candidates and plan.reasons[0].startswith("hashed"):
                print("    candidate by email: none")
            continue

        if plan.action == "merge":
            assert plan.target is not None
            print(f"    into: {describe(plan.target)}")
            result = apply_merge(cur, plan, fk_columns)
        else:
            print(f"    new username: {plan.new_username}")
            result = apply_rename(cur, plan)
        changed += 1
        for column, count in result["moved"].items():
            print(f"    moved {count} row(s) in {column}")
        if result["initiatives"]:
            print(f"    owned initiatives: {', '.join(result['initiatives'])}")
            touched_initiatives += result["initiatives"]

    cur.execute("SELECT count(*) AS n FROM users WHERE username LIKE '%%|%%'")
    left = cur.fetchone()["n"]
    print(f"\n{changed} repaired, {review} need review, {left} Auth0-subject row(s) left after this run")
    if touched_initiatives:
        print("Rerun fga-reconcile for: " + ", ".join(touched_initiatives))
    return changed, review


def parse_args():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--apply", action="store_true", help="Commit the changes. Without this flag they are rolled back.")
    p.add_argument("--cf-db-url", help="PostgreSQL DSN for the CF DB (overrides CF_DB_URL and the dev tunnel).")
    return p.parse_args()


def main():
    args = parse_args()
    cf_db_url = args.cf_db_url or os.environ.get("CF_DB_URL")
    if not cf_db_url and not DEFAULT_CF_DB["password"]:
        raise SystemExit("ERROR: set CF_DB_PASSWORD / CF_DB_URL or pass --cf-db-url.")
    conn = _connect_cf(cf_db_url) if cf_db_url else psycopg2.connect(**DEFAULT_CF_DB)
    print("APPLY" if args.apply else "DRY RUN: every change below is rolled back", flush=True)
    try:
        _, review = run(conn)
        if args.apply:
            conn.commit()
            print("Committed.")
        else:
            conn.rollback()
            print("Rolled back. Re-run with --apply to commit.")
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()
    sys.exit(1 if review else 0)


if __name__ == "__main__":
    main()
