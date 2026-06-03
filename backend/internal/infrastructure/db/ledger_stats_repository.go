// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var ledgerStatsTracer = otel.Tracer("ledger-stats-db")

// LedgerStatsRepository implements domain.LedgerStatsRepository against PostgreSQL.
// It is used exclusively by the ledger-stats-sync CronJob.
type LedgerStatsRepository struct {
	pool *pgxpool.Pool
}

// NewLedgerStatsRepository creates a new LedgerStatsRepository.
func NewLedgerStatsRepository(pool *pgxpool.Pool) *LedgerStatsRepository {
	return &LedgerStatsRepository{pool: pool}
}

// ListActiveSyncIDs returns the UUIDs of all initiatives whose status is not
// 'archived' or 'draft'.  These are the rows the CronJob must attempt to match
// against the Ledger bulk balance response on every run.
func (r *LedgerStatsRepository) ListActiveSyncIDs(ctx context.Context) ([]string, error) {
	ctx, span := ledgerStatsTracer.Start(ctx, "db.ledger_stats.ListActiveSyncIDs")
	defer span.End()

	const q = `
		SELECT id
		FROM initiatives
		WHERE status NOT IN ('archived', 'draft')
		ORDER BY id`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list active initiative IDs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan initiative ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate initiative IDs: %w", err)
	}
	return ids, nil
}

// BulkUpsertLedgerStats inserts or updates rows in initiative_ledger_stats using
// a pgx batch so all upserts are sent in a single round-trip.
// Returns the number of rows successfully upserted.
func (r *LedgerStatsRepository) BulkUpsertLedgerStats(ctx context.Context, stats []models.LedgerStats) (int, error) {
	ctx, span := ledgerStatsTracer.Start(ctx, "db.ledger_stats.BulkUpsertLedgerStats")
	defer span.End()
	span.SetAttributes(attribute.Int("db.batch_size", len(stats)))

	if len(stats) == 0 {
		return 0, nil
	}

	const q = `
		INSERT INTO initiative_ledger_stats
		       (initiative_id, total_raised_cents, total_debited_cents,
		        total_balance_cents, available_balance_cents, fee_balance_cents,
		        supporters, sponsors, updated_on)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,NOW())
		ON CONFLICT (initiative_id) DO UPDATE SET
		       total_raised_cents      = EXCLUDED.total_raised_cents,
		       total_debited_cents     = EXCLUDED.total_debited_cents,
		       total_balance_cents     = EXCLUDED.total_balance_cents,
		       available_balance_cents = EXCLUDED.available_balance_cents,
		       fee_balance_cents       = EXCLUDED.fee_balance_cents,
		       supporters              = EXCLUDED.supporters,
		       sponsors                = EXCLUDED.sponsors,
		       updated_on              = NOW()`

	batch := &pgx.Batch{}
	for i := range stats {
		s := &stats[i]
		sponsorsJSON, err := json.Marshal(s.Sponsors)
		if err != nil {
			return 0, fmt.Errorf("marshal sponsors for initiative %s: %w", s.InitiativeID, err)
		}
		batch.Queue(q,
			s.InitiativeID,
			s.TotalRaisedCents,
			s.TotalDebitedCents,
			s.TotalBalanceCents,
			s.AvailableBalanceCents,
			s.FeeBalanceCents,
			s.Supporters,
			string(sponsorsJSON),
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close() //nolint:errcheck

	for i := range stats {
		if _, err := br.Exec(); err != nil {
			return i, fmt.Errorf("upsert initiative_ledger_stats[%d] %s: %w",
				i, stats[i].InitiativeID, err)
		}
	}
	return len(stats), nil
}

// GetOrganizationsByIDs returns a map of org UUID → Organization for all IDs
// provided.  IDs not present in the database are simply absent from the map.
// An empty input slice returns an empty map without querying the database.
func (r *LedgerStatsRepository) GetOrganizationsByIDs(ctx context.Context, ids []string) (map[string]models.Organization, error) {
	ctx, span := ledgerStatsTracer.Start(ctx, "db.ledger_stats.GetOrganizationsByIDs")
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

// GetUsersByIDs returns a map of legacy_user_id → User for all IDs provided.
// The Ledger service identifies users by their Auth0 subject (legacy_user_id),
// so we look up by that column rather than the internal UUID primary key.
// IDs not present in the database are simply absent from the map.
// An empty input slice returns an empty map without querying the database.
func (r *LedgerStatsRepository) GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]models.User, error) {
	ctx, span := ledgerStatsTracer.Start(ctx, "db.ledger_stats.GetUsersByIDs")
	defer span.End()
	span.SetAttributes(attribute.Int("db.id_count", len(userIDs)))

	result := make(map[string]models.User, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	const q = `
		SELECT id, legacy_user_id, name, avatar_url, created_on, updated_on
		FROM users
		WHERE legacy_user_id = ANY($1)`

	rows, err := r.pool.Query(ctx, q, userIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get users by IDs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var (
			u            models.User
			legacyUserID *string
			name         *string
			avatarURL    *string
			createdOn    *time.Time
			updatedOn    *time.Time
		)
		if err := rows.Scan(&u.ID, &legacyUserID, &name, &avatarURL, &createdOn, &updatedOn); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if legacyUserID != nil {
			u.LegacyUserID = *legacyUserID
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
		result[u.LegacyUserID] = u
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return result, nil
}
