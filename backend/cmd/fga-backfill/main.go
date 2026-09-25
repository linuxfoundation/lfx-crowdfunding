// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// fga-backfill is the one-time job for lfx-crowdfunding#277: it republishes
// every existing initiative's current owner/attribution/published state to
// fga-sync so crowdfunding_initiative tuples exist for initiatives created
// before CF started emitting them on write.
//
// It is idempotent — it replays current Postgres state through the same
// fga.Publisher.UpdateAccess call the live write path uses (full sync, so
// reruns are safe) — so it can be run as a one-off K8s Job, not a CronJob.
//
// Usage: run once per environment after deploying the emission hooks. Exits
// 0 on success (including partial per-row publish failures, which are
// logged), non-zero if it cannot start at all.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/db"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/fga"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("fga-backfill failed", "error", err)
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

	// No RetryOnFailedConnect: unlike the API server, this is a one-shot job —
	// if fga-sync/NATS is unreachable there is nothing useful to retry into.
	conn, err := nats.Connect(cfg.FGANATSURL, nats.Timeout(cfg.FGATimeout))
	if err != nil {
		return fmt.Errorf("fga-sync nats connect: %w", err)
	}
	defer conn.Close()

	repo := db.NewInitiativeRepository(pool)
	publisher := fga.NewPublisher(conn)
	syncer := newSyncer(repo, publisher, logger)

	logger.Info("fga-backfill starting")

	result, err := syncer.Run(ctx)
	if err != nil {
		return fmt.Errorf("backfill run: %w", err)
	}

	// Publisher.UpdateAccess is fire-and-forget; flush once at the end so the
	// process doesn't exit before the broker acks the batch.
	if err := conn.FlushWithContext(ctx); err != nil {
		return fmt.Errorf("flush fga-sync nats connection: %w", err)
	}

	logger.Info("fga-backfill complete",
		"duration", time.Since(start).String(),
		"total_initiatives", result.total,
		"published", result.published,
		"failed", result.failed,
	)
	return nil
}

// config holds the runtime configuration for fga-backfill.
type config struct {
	DatabaseURL       string
	DBMaxConns        int
	DBMinConns        int
	DBConnMaxLifetime time.Duration
	FGANATSURL        string
	FGATimeout        time.Duration
}

func loadConfig() (*config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	fgaNATSURL := os.Getenv("FGA_NATS_URL")
	if fgaNATSURL == "" {
		return nil, fmt.Errorf("FGA_NATS_URL is required")
	}

	fgaTimeout := 10 * time.Second
	if v := os.Getenv("FGA_ACCESS_CHECK_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("FGA_ACCESS_CHECK_TIMEOUT: invalid duration %q: %w", v, err)
		}
		fgaTimeout = d
	}

	return &config{
		DatabaseURL:       dbURL,
		DBMaxConns:        5,
		DBMinConns:        1,
		DBConnMaxLifetime: 5 * time.Minute,
		FGANATSURL:        fgaNATSURL,
		FGATimeout:        fgaTimeout,
	}, nil
}
