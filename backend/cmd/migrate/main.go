// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Command migrate applies pending db/migrations to DATABASE_URL. It is run
// as a Helm pre-install/pre-upgrade hook Job so schema changes land before
// the new application image starts serving traffic.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/db/migrations"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	dsn, err := withPinnedMigrationsTable(dsn)
	if err != nil {
		log.Fatalf("parse DATABASE_URL: %v", err)
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("load embedded migrations: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		log.Fatalf("init migrator: %v", err)
	}

	// One-time bootstrap for a database whose schema was already brought
	// up to some migration N by hand (as dev/staging/prod were, before this
	// Job existed) but has no schema_migrations row recording that. Run
	// manually once per such environment: `migrate force N`. Never part of
	// the automatic pre-install/pre-upgrade hook path.
	if len(os.Args) > 1 && os.Args[1] == "force" {
		if len(os.Args) != 3 {
			log.Fatal("usage: migrate force <version>")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("invalid version %q: %v", os.Args[2], err)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("force version: %v", err)
		}
		fmt.Printf("schema_migrations forced to version %d\n", version)
		return
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("apply migrations: %v", err)
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		log.Fatalf("read schema version: %v", err)
	}
	fmt.Printf("migrations applied: schema version %d (dirty=%v)\n", version, dirty)
}

// withPinnedMigrationsTable pins golang-migrate's own version-tracking table
// to "public".schema_migrations. Without this, the postgres driver defaults
// to CURRENT_SCHEMA() — the first schema on the connection's search_path —
// which is only "public" before migration 001 creates the "crowdfunding"
// schema. On every run after that, Postgres' default "$user",public search
// path resolves "crowdfunding" first (it matches the DB username), so the
// driver silently creates a second, always-empty tracking table there and
// re-applies every migration from scratch.
func withPinnedMigrationsTable(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("x-migrations-table", `"public"."schema_migrations"`)
	q.Set("x-migrations-table-quoted", "true")
	u.RawQuery = q.Encode()
	return u.String(), nil
}
