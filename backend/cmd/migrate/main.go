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
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/db/migrations"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Parsed with pgx rather than net/url: DATABASE_URL is a postgres:// URL
	// locally/in CI but a libpq keyword/value string ("host=... user=...") in
	// staging/prod (see lfx-v2-argocd values), which net/url can't parse.
	connConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("parse DATABASE_URL: %v", err)
	}
	// Lock safety rails required by backend/docs/rewrite/10-database-migrations.md:
	// fail fast and roll back rather than queue behind live traffic for an
	// ACCESS EXCLUSIVE lock. Set on the connection since it isn't derivable
	// from the pool's search_path (internal/infrastructure/db/pool.go), which
	// this separate migrate connection doesn't share.
	connConfig.RuntimeParams["lock_timeout"] = "5s"
	connConfig.RuntimeParams["statement_timeout"] = "30s"

	sqlDB := stdlib.OpenDB(*connConfig)
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		// Pins golang-migrate's own version-tracking table to
		// "public".schema_migrations. Without this, the driver defaults to
		// CURRENT_SCHEMA() — the first schema on the connection's
		// search_path — which is only "public" before migration 001 creates
		// the "crowdfunding" schema. On every run after that, Postgres'
		// default "$user",public search path resolves "crowdfunding" first
		// (it matches the DB username), so the driver would silently create
		// a second, always-empty tracking table there and re-apply every
		// migration from scratch.
		MigrationsTable:       `"public"."schema_migrations"`,
		MigrationsTableQuoted: true,
	})
	if err != nil {
		log.Fatalf("init postgres driver: %v", err)
	}

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatalf("load embedded migrations: %v", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		log.Fatalf("init migrator: %v", err)
	}

	// One-time bootstrap for a database whose schema was already brought
	// up to some migration N by hand (as dev/staging/prod were, before this
	// Job existed) but has no schema_migrations row recording that. Run
	// manually once per such environment: `migrate force N`. Never part of
	// the automatic pre-install/pre-upgrade hook path.
	switch len(os.Args) {
	case 1:
		// no subcommand — normal `m.Up()` path below.
	case 3:
		if os.Args[1] != "force" {
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
	default:
		log.Fatal("usage: migrate force <version>")
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
