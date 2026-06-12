// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// mentorship-sync pulls Mentorship program data from Snowflake (or a JSON
// fixture in DEV) and upserts initiative_type=mentorship rows into CF Postgres.
//
// See backend/docs/rewrite/10-mentorship-sync-dev-testing.md for the DEV testing strategy.
//
// Usage: run as a K8s CronJob (daily schedule). Exits 0 on success,
// non-zero on any error. K8s uses the exit code to track CronJob health.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/db"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/snowflake"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("mentorship-sync failed", "error", err)
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
		DSN:      cfg.DatabaseURL,
		MaxConns: 5,
		MinConns: 1,
	})
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	defer pool.Close()

	var src mentorshipSource
	if cfg.FixtureFile != "" {
		logger.Info("using fixture source", "path", cfg.FixtureFile)
		src = snowflake.NewFixtureSource(cfg.FixtureFile)
	} else {
		client, err := snowflake.NewClient(snowflake.ClientConfig{
			Account:    cfg.SnowflakeAccount,
			User:       cfg.SnowflakeUser,
			Warehouse:  cfg.SnowflakeWarehouse,
			Database:   cfg.SnowflakeDatabase,
			Role:       cfg.SnowflakeRole,
			PrivateKey: cfg.SnowflakePrivateKey,
		})
		if err != nil {
			return fmt.Errorf("snowflake client: %w", err)
		}
		defer client.Close()
		src = client
	}

	repo := db.NewMentorshipRepository(pool)
	syncer := newSyncer(repo, src, logger)

	logger.Info("mentorship-sync starting")

	result, err := syncer.Run(ctx)
	if err != nil {
		return fmt.Errorf("sync run: %w", err)
	}

	logger.Info("mentorship-sync complete",
		"duration", time.Since(start).String(),
		"total", result.total,
		"upserted", result.upserted,
		"errors", result.errors,
	)
	return nil
}

type config struct {
	DatabaseURL         string
	FixtureFile         string
	SnowflakeAccount    string
	SnowflakeUser       string
	SnowflakeWarehouse  string
	SnowflakeDatabase   string
	SnowflakeRole       string
	SnowflakePrivateKey string
}

func loadConfig() (*config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	cfg := &config{
		DatabaseURL: dbURL,
		FixtureFile: os.Getenv("MENTORSHIP_SYNC_FIXTURE_FILE"),
	}

	if cfg.FixtureFile == "" {
		cfg.SnowflakeAccount = os.Getenv("SNOWFLAKE_ACCOUNT")
		if cfg.SnowflakeAccount == "" {
			return nil, fmt.Errorf("SNOWFLAKE_ACCOUNT is required when MENTORSHIP_SYNC_FIXTURE_FILE is not set")
		}
		cfg.SnowflakeUser = os.Getenv("SNOWFLAKE_USER")
		if cfg.SnowflakeUser == "" {
			return nil, fmt.Errorf("SNOWFLAKE_USER is required when MENTORSHIP_SYNC_FIXTURE_FILE is not set")
		}
		cfg.SnowflakeWarehouse = os.Getenv("SNOWFLAKE_WAREHOUSE")
		if cfg.SnowflakeWarehouse == "" {
			return nil, fmt.Errorf("SNOWFLAKE_WAREHOUSE is required when MENTORSHIP_SYNC_FIXTURE_FILE is not set")
		}
		cfg.SnowflakeDatabase = os.Getenv("SNOWFLAKE_DATABASE")
		if cfg.SnowflakeDatabase == "" {
			return nil, fmt.Errorf("SNOWFLAKE_DATABASE is required when MENTORSHIP_SYNC_FIXTURE_FILE is not set")
		}
		cfg.SnowflakeRole = os.Getenv("SNOWFLAKE_ROLE")
		if cfg.SnowflakeRole == "" {
			return nil, fmt.Errorf("SNOWFLAKE_ROLE is required when MENTORSHIP_SYNC_FIXTURE_FILE is not set")
		}
		cfg.SnowflakePrivateKey = os.Getenv("SNOWFLAKE_PRIVATE_KEY")
		if cfg.SnowflakePrivateKey == "" {
			return nil, fmt.Errorf("SNOWFLAKE_PRIVATE_KEY is required when MENTORSHIP_SYNC_FIXTURE_FILE is not set")
		}
	}

	return cfg, nil
}
