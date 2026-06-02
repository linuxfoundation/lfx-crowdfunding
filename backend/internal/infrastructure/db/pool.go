// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
package db

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds connection pool settings, all sourced from environment variables.
type PoolConfig struct {
	DSN             string
	MaxConns        int
	MinConns        int
	ConnMaxLifetime time.Duration
}

// NewPool creates a pgxpool.Pool, pings the database, and returns it.
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}
	if cfg.MaxConns < 0 || cfg.MaxConns > math.MaxInt32 {
		return nil, fmt.Errorf("MaxConns %d is out of valid range [0, %d]", cfg.MaxConns, math.MaxInt32)
	}
	if cfg.MinConns < 0 || cfg.MinConns > math.MaxInt32 {
		return nil, fmt.Errorf("MinConns %d is out of valid range [0, %d]", cfg.MinConns, math.MaxInt32)
	}
	if cfg.MaxConns > 0 && cfg.MinConns > cfg.MaxConns {
		return nil, fmt.Errorf("invalid pool configuration: DB_MIN_CONNS (%d) must be less than or equal to DB_MAX_CONNS (%d)", cfg.MinConns, cfg.MaxConns)
	}
	config.MaxConns = int32(cfg.MaxConns)
	config.MinConns = int32(cfg.MinConns)
	config.MaxConnLifetime = cfg.ConnMaxLifetime
	config.ConnConfig.RuntimeParams["search_path"] = "crowdfunding,public"

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// filterValidUUIDs returns only the elements of ids that are valid UUID strings.
// Non-UUID values (e.g. legacy Auth0 subs like "auth0|...") are silently dropped
// so callers can safely pass the result to a query using ANY($1::uuid[]).
func filterValidUUIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, err := uuid.Parse(id); err == nil {
			out = append(out, id)
		}
	}
	return out
}
