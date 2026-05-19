// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
// Uses LEFT JOIN so published initiatives without a stats row (before first cron run)
// are counted in total_initiatives but contribute 0 to financial totals.
func (r *StatisticsRepository) GetPlatformStatistics(ctx context.Context) (*models.PlatformStatistics, error) {
	ctx, span := statisticsTracer.Start(ctx, "db.statistics.GetPlatformStatistics")
	defer span.End()

	const q = `
		SELECT
			COALESCE(SUM(ls.total_raised_cents), 0)::bigint AS total_raised_cents,
			COALESCE(SUM(ls.supporters), 0)::bigint         AS total_supporters,
			COUNT(i.id)::bigint                             AS total_initiatives
		FROM initiatives i
		LEFT JOIN initiative_ledger_stats ls ON ls.initiative_id = i.id
			WHERE LOWER(i.status) = $1`

	var s models.PlatformStatistics
	if err := r.pool.QueryRow(ctx, q, models.StatusPublished).Scan(
		&s.TotalRaisedCents,
		&s.TotalSupporters,
		&s.TotalInitiatives,
	); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get platform statistics: %w", err)
	}
	return &s, nil
}

// GetOrganizationsByIDs returns a map of org UUID → Organization for the given IDs.
// Missing IDs are absent from the map.
func (r *StatisticsRepository) GetOrganizationsByIDs(ctx context.Context, ids []string) (map[string]models.Organization, error) {
	ctx, span := statisticsTracer.Start(ctx, "db.statistics.GetOrganizationsByIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("db.id_count", len(ids)))

	result := make(map[string]models.Organization, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	const q = `
		SELECT id, owner_id, name, avatar_url, status, created_on, updated_on
		FROM organizations
		WHERE id = ANY($1::uuid[])`

	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get organizations by IDs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var (
			o         models.Organization
			avatarURL *string
			status    *string
			createdOn *time.Time
			updatedOn *time.Time
		)
		if err := rows.Scan(&o.ID, &o.OwnerID, &o.Name, &avatarURL, &status, &createdOn, &updatedOn); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		if avatarURL != nil {
			o.AvatarURL = *avatarURL
		}
		if status != nil {
			o.Status = *status
		}
		if createdOn != nil {
			o.CreatedOn = *createdOn
		}
		if updatedOn != nil {
			o.UpdatedOn = *updatedOn
		}
		result[o.ID] = o
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organizations: %w", err)
	}
	return result, nil
}

// GetUsersByIDs returns a map of Auth0 user_id → User for the given IDs.
// Missing IDs are absent from the map.
func (r *StatisticsRepository) GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]models.User, error) {
	ctx, span := statisticsTracer.Start(ctx, "db.statistics.GetUsersByIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("db.id_count", len(userIDs)))

	result := make(map[string]models.User, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	const q = `
		SELECT id, user_id, name, avatar_url, created_on, updated_on
		FROM users
		WHERE user_id = ANY($1)`

	rows, err := r.pool.Query(ctx, q, userIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get users by IDs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var (
			u         models.User
			name      *string
			avatarURL *string
			createdOn *time.Time
			updatedOn *time.Time
		)
		if err := rows.Scan(&u.ID, &u.UserID, &name, &avatarURL, &createdOn, &updatedOn); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if name != nil {
			u.Name = *name
		}
		if avatarURL != nil {
			u.AvatarURL = *avatarURL
		}
		if createdOn != nil {
			u.CreatedOn = *createdOn
		}
		if updatedOn != nil {
			u.UpdatedOn = *updatedOn
		}
		result[u.UserID] = u
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return result, nil
}
