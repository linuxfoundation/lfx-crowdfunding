// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
)

var statisticsTracer = otel.Tracer("statistics-db")

// StatisticsRepository implements domain.StatisticsRepository against PostgreSQL.
type StatisticsRepository struct {
	pool *pgxpool.Pool
}

// NewStatisticsRepository creates a new StatisticsRepository.
func NewStatisticsRepository(pool *pgxpool.Pool) *StatisticsRepository {
	return &StatisticsRepository{pool: pool}
}

// GetPlatformStatistics returns platform-wide aggregates from initiative_ledger_stats.
// total_initiatives counts only published initiatives that have a ledger stats row.
func (r *StatisticsRepository) GetPlatformStatistics(ctx context.Context) (*models.PlatformStatistics, error) {
	ctx, span := statisticsTracer.Start(ctx, "db.statistics.GetPlatformStatistics")
	defer span.End()

	const q = `
		SELECT
			COALESCE(SUM(ls.total_raised_cents), 0)::bigint AS total_raised_cents,
			COALESCE(SUM(ls.supporters), 0)::bigint         AS total_supporters,
			COUNT(i.id)::bigint                             AS total_initiatives
		FROM initiatives i
		INNER JOIN initiative_ledger_stats ls ON ls.initiative_id = i.id
		WHERE i.status = 'published'`

	var s models.PlatformStatistics
	if err := r.pool.QueryRow(ctx, q).Scan(
		&s.TotalRaisedCents,
		&s.TotalSupporters,
		&s.TotalInitiatives,
	); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get platform statistics: %w", err)
	}
	return &s, nil
}
