// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var announcementTracer = otel.Tracer("announcements-db")

// AnnouncementRepository implements domain.AnnouncementRepository against PostgreSQL.
type AnnouncementRepository struct {
	pool *pgxpool.Pool
}

// NewAnnouncementRepository creates a new AnnouncementRepository.
func NewAnnouncementRepository(pool *pgxpool.Pool) *AnnouncementRepository {
	return &AnnouncementRepository{pool: pool}
}

// List returns paginated announcements for the given initiative ordered newest first.
func (r *AnnouncementRepository) List(ctx context.Context, initiativeID string, filter models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error) {
	ctx, span := announcementTracer.Start(ctx, "db.announcements.List")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_id", initiativeID))

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	const countQ = `SELECT COUNT(*) FROM initiative_announcements WHERE initiative_id = $1`
	if err := r.pool.QueryRow(ctx, countQ, initiativeID).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("count announcements: %w", err)
	}

	const q = `
		SELECT id, initiative_id, created_by, title, description, created_on, updated_on
		FROM initiative_announcements
		WHERE initiative_id = $1
		ORDER BY created_on DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, q, initiativeID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list announcements: %w", err)
	}
	defer rows.Close()

	var announcements []models.Announcement
	for rows.Next() {
		var a models.Announcement
		if err := rows.Scan(&a.ID, &a.InitiativeID, &a.CreatedBy, &a.Title, &a.Description, &a.CreatedOn, &a.UpdatedOn); err != nil {
			span.RecordError(err)
			return nil, nil, fmt.Errorf("scan announcement: %w", err)
		}
		announcements = append(announcements, a)
	}
	if err := rows.Err(); err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("rows error: %w", err)
	}
	if announcements == nil {
		announcements = []models.Announcement{}
	}

	meta := &models.PaginationMeta{Total: total, Limit: limit, Offset: offset}
	return announcements, meta, nil
}

// Create inserts a new announcement and returns the persisted record.
func (r *AnnouncementRepository) Create(ctx context.Context, a *models.Announcement) (*models.Announcement, error) {
	ctx, span := announcementTracer.Start(ctx, "db.announcements.Create")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_id", a.InitiativeID))

	const q = `
		INSERT INTO initiative_announcements (initiative_id, created_by, title, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, initiative_id, created_by, title, description, created_on, updated_on`

	row := r.pool.QueryRow(ctx, q, a.InitiativeID, a.CreatedBy, a.Title, a.Description)
	var result models.Announcement
	if err := row.Scan(&result.ID, &result.InitiativeID, &result.CreatedBy, &result.Title, &result.Description, &result.CreatedOn, &result.UpdatedOn); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create announcement: %w", err)
	}
	return &result, nil
}

// Update patches the title and description of the identified announcement.
// Returns ErrAnnouncementNotFound when no matching row exists.
func (r *AnnouncementRepository) Update(ctx context.Context, id, initiativeID string, input models.AnnouncementUpdateInput) (*models.Announcement, error) {
	ctx, span := announcementTracer.Start(ctx, "db.announcements.Update")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.announcement_id", id),
		attribute.String("db.initiative_id", initiativeID),
	)

	const q = `
		UPDATE initiative_announcements
		SET title = $1, description = $2, updated_on = NOW()
		WHERE id = $3 AND initiative_id = $4
		RETURNING id, initiative_id, created_by, title, description, created_on, updated_on`

	row := r.pool.QueryRow(ctx, q, input.Title, input.Description, id, initiativeID)
	var result models.Announcement
	if err := row.Scan(&result.ID, &result.InitiativeID, &result.CreatedBy, &result.Title, &result.Description, &result.CreatedOn, &result.UpdatedOn); err != nil {
		span.RecordError(err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAnnouncementNotFound
		}
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	return &result, nil
}

// Delete removes an announcement scoped to the given initiative.
// Returns ErrAnnouncementNotFound when no matching row exists.
func (r *AnnouncementRepository) Delete(ctx context.Context, id, initiativeID string) error {
	ctx, span := announcementTracer.Start(ctx, "db.announcements.Delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.announcement_id", id),
		attribute.String("db.initiative_id", initiativeID),
	)

	const q = `DELETE FROM initiative_announcements WHERE id = $1 AND initiative_id = $2`
	tag, err := r.pool.Exec(ctx, q, id, initiativeID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete announcement: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAnnouncementNotFound
	}
	return nil
}
