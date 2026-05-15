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

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("ledger-stats-sync failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx := context.Background()
	start := time.Now()

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Database pool — shared pgxpool, same pattern as the initiatives API.
	pool, err := db.NewPool(ctx, db.PoolConfig{
		DSN:             cfg.DatabaseURL,
		MaxConns:        cfg.DBMaxConns,
		MinConns:        cfg.DBMinConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
	})
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	defer pool.Close()

	// Ledger HTTP client.
	ledgerClient := clients.NewLedgerClient(clients.LedgerConfig{
		BaseURL: cfg.LedgerBaseURL,
		APIKey:  cfg.LedgerAPIKey,
		Timeout: cfg.LedgerTimeout,
	})

	// Repository and syncer.
	repo := db.NewLedgerStatsRepository(pool)
	syncer := newSyncer(repo, ledgerClient, logger)

	logger.Info("ledger-stats-sync starting")

	result, err := syncer.Run(ctx)
	if err != nil {
		return fmt.Errorf("sync run: %w", err)
	}

	logger.Info("ledger-stats-sync complete",
		"duration", time.Since(start).String(),
		"total_initiatives", result.total,
		"matched", result.matched,
		"upserted", result.upserted,
		"skipped", result.skipped,
	)
	return nil
}

// config holds the runtime configuration for ledger-stats-sync.
type config struct {
	DatabaseURL       string
	DBMaxConns        int
	DBMinConns        int
	DBConnMaxLifetime time.Duration
	LedgerBaseURL     string
	LedgerAPIKey      string
	LedgerTimeout     time.Duration
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
	ledgerAPIKey := os.Getenv("LEDGER_API_KEY")
	if ledgerAPIKey == "" {
		return nil, fmt.Errorf("LEDGER_API_KEY is required")
	}

	ledgerTimeout := 30 * time.Second
	if v := os.Getenv("LEDGER_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("LEDGER_TIMEOUT: invalid duration %q: %w", v, err)
		}
		ledgerTimeout = d
	}

	dbMaxConns := 5
	dbMinConns := 1
	dbConnMaxLifetime := 5 * time.Minute

	return &config{
		DatabaseURL:       dbURL,
		DBMaxConns:        dbMaxConns,
		DBMinConns:        dbMinConns,
		DBConnMaxLifetime: dbConnMaxLifetime,
		LedgerBaseURL:     ledgerBaseURL,
		LedgerAPIKey:      ledgerAPIKey,
		LedgerTimeout:     ledgerTimeout,
	}, nil
}
