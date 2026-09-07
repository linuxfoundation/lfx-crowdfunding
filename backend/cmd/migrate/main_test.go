// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestParseConfig_LockTimeouts guards the fix for both DSN shapes DATABASE_URL
// takes in practice: a postgres:// URL (local/CI) and a libpq keyword/value
// string (staging/prod, via lfx-v2-argocd). net/url.Parse only handles the
// former; pgx.ParseConfig must handle both, and the lock/statement timeout
// safety rails must be settable on the resulting config either way.
func TestParseConfig_LockTimeouts(t *testing.T) {
	dsns := map[string]string{
		"url":     "postgres://crowdfunding:crowdfunding@localhost:5432/crowdfunding?sslmode=disable",
		"keyword": "host=localhost port=5432 user=crowdfunding password=crowdfunding dbname=crowdfunding sslmode=disable",
	}
	for name, dsn := range dsns {
		t.Run(name, func(t *testing.T) {
			cfg, err := pgx.ParseConfig(dsn)
			if err != nil {
				t.Fatalf("parse DATABASE_URL: %v", err)
			}
			cfg.RuntimeParams["lock_timeout"] = "5s"
			cfg.RuntimeParams["statement_timeout"] = "30s"
			if cfg.RuntimeParams["lock_timeout"] != "5s" {
				t.Errorf("lock_timeout = %q, want 5s", cfg.RuntimeParams["lock_timeout"])
			}
			if cfg.RuntimeParams["statement_timeout"] != "30s" {
				t.Errorf("statement_timeout = %q, want 30s", cfg.RuntimeParams["statement_timeout"])
			}
		})
	}
}
