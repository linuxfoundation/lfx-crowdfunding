// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// ledger-stats-sync pulls balance data from the Ledger HTTP API and upserts
// rows into initiative_ledger_stats in CF Postgres.
//
// See docs/rewrite/02-decisions.md § ledger-stats-sync CronJob for the full
// specification, column mapping, and ID constraints.
//
// Usage: run as a K8s CronJob (schedule: hourly). Exits 0 on success,
// non-zero on any error — K8s uses the exit code to track CronJob health.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("ledger-stats-sync failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	_ = context.Background() // TODO(lewis): pass ctx into DB and HTTP calls
	start := time.Now()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// TODO(lewis): connect to CF Postgres using cfg.DatabaseURL.
	// Recommended: use pgx/v5 pgxpool, same as the initiatives API.
	_ = cfg

	// Step 1: Load all non-archived, non-draft initiative IDs from CF DB.
	//
	// SELECT id FROM initiatives WHERE status NOT IN ('archived', 'draft')
	//
	// Sync all active initiatives (not just published) so that when an
	// initiative is published its stats are already populated.
	logger.Info("loading initiatives from CF DB")
	// TODO(lewis): implement — query CF DB, collect []string of initiative IDs.

	// Step 2: Fetch all balances from the Ledger service in one bulk call.
	//
	// GET <LEDGER_BASE_URL>/balance
	// Authorization: <LEDGER_API_KEY>
	//
	// Returns AllBalances{Balances: []Balance{...}} — one entry per project_id.
	// See ledger-service/balance/balance.go for the Balance struct.
	logger.Info("fetching all balances from Ledger")
	// TODO(lewis): implement — call Ledger GET /balance, decode into AllBalances.

	// Step 3: Build a map[projectID]Balance for O(1) lookup.
	// TODO(lewis): implement.

	// Step 4: For each initiative, look up its Ledger entry and upsert.
	//
	// Column mapping (see 02-decisions.md for rationale):
	//   total_raised_cents      = totalCredit           (always positive)
	//   total_debited_cents     = ABS(totalDebit)       (Ledger stores as negative)
	//   total_balance_cents     = totalBalance
	//   available_balance_cents = availableBalance      (Ledger computes this)
	//   fee_balance_cents       = ABS(feeBalance)       (Ledger stores as negative)
	//   supporters              = supporters            (distinct user_id count)
	//
	// Use INSERT ... ON CONFLICT (initiative_id) DO UPDATE SET ...
	// Always set updated_on = NOW() on upsert.
	// Skip initiatives with no Ledger entry — do not delete or zero out rows.
	logger.Info("upserting initiative_ledger_stats")
	// TODO(lewis): implement upsert loop. Collect upserted/skipped counts.

	// Step 5: Log summary.
	logger.Info("ledger-stats-sync complete",
		"duration", time.Since(start).String(),
		// TODO(lewis): add total, matched, upserted, skipped counts.
	)
	return nil
}

// config holds the runtime configuration for ledger-stats-sync.
type config struct {
	DatabaseURL   string
	LedgerBaseURL string
	LedgerAPIKey  string
	LedgerTimeout time.Duration
}

func loadConfig() (*config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	ledgerBaseURL := os.Getenv("LEDGER_BASE_URL")
	if ledgerBaseURL == "" {
		return nil, fmt.Errorf("LEDGER_BASE_URL is required")
	}

	ledgerTimeout := 30 * time.Second
	if v := os.Getenv("LEDGER_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("LEDGER_TIMEOUT: invalid duration %q: %w", v, err)
		}
		ledgerTimeout = d
	}

	return &config{
		DatabaseURL:   dbURL,
		LedgerBaseURL: ledgerBaseURL,
		LedgerAPIKey:  os.Getenv("LEDGER_API_KEY"),
		LedgerTimeout: ledgerTimeout,
	}, nil
}
