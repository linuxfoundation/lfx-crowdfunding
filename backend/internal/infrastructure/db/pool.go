// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
package db

import (
	"context"
	"fmt"
	"math"
	"time"

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
	config.MaxConns = int32(cfg.MaxConns)
	config.MinConns = int32(cfg.MinConns)
	config.MaxConnLifetime = cfg.ConnMaxLifetime

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
