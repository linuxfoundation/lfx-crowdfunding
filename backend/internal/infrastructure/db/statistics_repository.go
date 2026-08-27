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

// ListOrgContributions returns each organization's total succeeded-donation
// amount, ordered by amount descending. When ids is non-empty, results are
// restricted to those organization IDs; otherwise all organizations with at
// least one succeeded donation are considered. When limit > 0, results are
// capped to the top `limit` organizations by amount.
func (r *StatisticsRepository) ListOrgContributions(ctx context.Context, ids []string, limit int) ([]models.OrgContribution, error) {
	ctx, span := statisticsTracer.Start(ctx, "db.statistics.ListOrgContributions")
	defer span.End()
	span.SetAttributes(attribute.Int("db.id_count", len(ids)), attribute.Int("db.limit", limit))

	if ids == nil {
		// pgx sends a nil slice as SQL NULL, which would make
		// `cardinality($1::uuid[]) = 0` evaluate to NULL instead of true.
		ids = []string{}
	}

	const q = `
		SELECT CAST(o.id AS text), o.name, COALESCE(o.avatar_url, ''),
		       COALESCE(SUM(d.current_amount_in_cents), 0)::bigint AS amount_in_cents
		FROM donations d
		JOIN organizations o ON o.id = d.organization_id
		WHERE d.status = 'succeeded'
		  AND (cardinality($1::uuid[]) = 0 OR d.organization_id = ANY($1::uuid[]))
		GROUP BY o.id, o.name, o.avatar_url
		ORDER BY amount_in_cents DESC
		LIMIT NULLIF($2, 0)`

	rows, err := r.pool.Query(ctx, q, ids, limit)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list org contributions: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	result := make([]models.OrgContribution, 0, len(ids))
	for rows.Next() {
		var c models.OrgContribution
		if err := rows.Scan(&c.OrgID, &c.Name, &c.AvatarURL, &c.AmountCents); err != nil {
			return nil, fmt.Errorf("scan org contribution: %w", err)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate org contributions: %w", err)
	}
	return result, nil
}

// GetInitiativeNamesByIDs returns a map of initiative UUID → name for the given IDs.
// Missing IDs are absent from the map.
func (r *StatisticsRepository) GetInitiativeNamesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	ctx, span := statisticsTracer.Start(ctx, "db.statistics.GetInitiativeNamesByIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("db.id_count", len(ids)))

	result := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	const q = `SELECT id, name FROM initiatives WHERE id = ANY($1::uuid[])`

	rows, err := r.pool.Query(ctx, q, ids)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get initiative names by IDs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan initiative name: %w", err)
		}
		result[id] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate initiative names: %w", err)
	}
	return result, nil
}

// GetUsersByIDs returns a map of user UUID → User for the given IDs.
// Missing IDs are absent from the map.
func (r *StatisticsRepository) GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]models.User, error) {
	ctx, span := statisticsTracer.Start(ctx, "db.statistics.GetUsersByIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("db.id_count", len(userIDs)))

	result := make(map[string]models.User, len(userIDs))
	userIDs = filterValidUUIDs(userIDs)
	if len(userIDs) == 0 {
		return result, nil
	}

	const q = `
		SELECT id, name, avatar_url, created_on, updated_on
		FROM users
		WHERE id = ANY($1::uuid[])`

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
		if err := rows.Scan(&u.ID, &name, &avatarURL, &createdOn, &updatedOn); err != nil {
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
		result[u.ID] = u
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return result, nil
}

// GetUsersByLegacyIDs returns a map of legacy_user_id → User for all Auth0 subjects provided.
// The Ledger service identifies users by their Auth0 subject (legacy_user_id), so this method
// queries by that column rather than the internal UUID primary key.
// Missing IDs are absent from the map.
func (r *StatisticsRepository) GetUsersByLegacyIDs(ctx context.Context, legacyIDs []string) (map[string]models.User, error) {
	ctx, span := statisticsTracer.Start(ctx, "db.statistics.GetUsersByLegacyIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("db.id_count", len(legacyIDs)))

	result := make(map[string]models.User, len(legacyIDs))
	if len(legacyIDs) == 0 {
		return result, nil
	}

	const q = `SELECT legacy_user_id, name, avatar_url FROM users WHERE legacy_user_id = ANY($1)`
	rows, err := r.pool.Query(ctx, q, legacyIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get users by legacy IDs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var (
			u         models.User
			legacyID  *string
			name      *string
			avatarURL *string
		)
		if err := rows.Scan(&legacyID, &name, &avatarURL); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if legacyID != nil {
			u.LegacyUserID = *legacyID
		}
		if name != nil {
			u.Name = *name
		}
		if avatarURL != nil {
			u.AvatarURL = *avatarURL
		}
		if u.LegacyUserID != "" {
			result[u.LegacyUserID] = u
		}
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return result, fmt.Errorf("iterate users by legacy IDs: %w", err)
	}
	return result, nil
}
