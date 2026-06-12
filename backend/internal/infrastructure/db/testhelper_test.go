// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool is the shared pool used by all DB integration tests.
// Initialised once in TestMain and closed after all tests run.
var testPool *pgxpool.Pool

// TestMain connects to a real Postgres and provides a shared pool to all tests
// in the db package. Set TEST_DATABASE_URL to override the default DSN.
// When -short is passed, skips DB setup entirely — individual tests also guard
// with testing.Short() so they are skipped cleanly without a pool.
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	// Require an explicit TEST_DATABASE_URL so integration tests never
	// accidentally connect to a developer's local database.
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		// Exit cleanly with skip-like output rather than panicking.
		// go test prints "ok" when no tests ran; the intent is communicated by the message.
		_, _ = os.Stderr.WriteString("testhelper: TEST_DATABASE_URL is not set — skipping DB integration tests\n")
		os.Exit(0)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic("testhelper: connect to test DB: " + err.Error())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		panic("testhelper: ping test DB: " + err.Error())
	}
	testPool = pool

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// truncate clears the named tables (schema-qualified, e.g. "crowdfunding.users")
// in order, using RESTART IDENTITY CASCADE. Call at the start of each test that
// inserts rows — this guarantees a clean state regardless of prior test runs.
func truncate(t *testing.T, ctx context.Context, tables ...string) { //nolint:revive // t first is Go test convention
	t.Helper()
	for _, table := range tables {
		if _, err := testPool.Exec(ctx, "TRUNCATE "+table+" RESTART IDENTITY CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
}
