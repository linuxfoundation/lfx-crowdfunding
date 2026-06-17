// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var orgTracer = otel.Tracer("organizations-db")

// OrganizationRepository implements domain.OrganizationRepository.
type OrganizationRepository struct {
	pool *pgxpool.Pool
}

// NewOrganizationRepository creates a new OrganizationRepository.
func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

// GetByID retrieves an organization by UUID.
func (r *OrganizationRepository) GetByID(ctx context.Context, id string) (*models.Organization, error) {
	ctx, span := orgTracer.Start(ctx, "db.organizations.GetByID")
	defer span.End()
	span.SetAttributes(attribute.String("db.org_id", id))

	const q = `
		SELECT id, owner_id, name, avatar_url, status, created_on, updated_on
		FROM organizations WHERE id = $1`

	o, err := scanOrganization(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if !errors.Is(err, domain.ErrOrganizationNotFound) {
			span.RecordError(err)
			err = fmt.Errorf("get organization: %w", err)
		}
		return nil, err
	}
	return o, nil
}

// ListByOwner returns all organizations owned by the given user_id.
func (r *OrganizationRepository) ListByOwner(ctx context.Context, ownerID string) ([]models.Organization, error) {
	ctx, span := orgTracer.Start(ctx, "db.organizations.ListByOwner")
	defer span.End()

	const q = `
		SELECT id, owner_id, name, avatar_url, status, created_on, updated_on
		FROM organizations WHERE owner_id = $1 ORDER BY name`

	rows, err := r.pool.Query(ctx, q, ownerID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []models.Organization
	for rows.Next() {
		o, err := scanOrganization(rows)
		if err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		orgs = append(orgs, *o)
	}
	return orgs, rows.Err()
}

// Create inserts a new organization owned by ownerID and returns the persisted row.
func (r *OrganizationRepository) Create(ctx context.Context, ownerID string, input models.OrganizationCreateInput) (*models.Organization, error) {
	ctx, span := orgTracer.Start(ctx, "db.organizations.Create")
	defer span.End()

	const q = `
		INSERT INTO organizations (id, owner_id, name, avatar_url)
		VALUES (gen_random_uuid(), $1, $2, NULLIF($3, ''))
		RETURNING id, owner_id, name, avatar_url, status, created_on, updated_on`

	o, err := scanOrganization(r.pool.QueryRow(ctx, q, ownerID, input.Name, input.AvatarURL))
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create organization: %w", err)
	}
	return o, nil
}

// Update patches name and avatar_url for the organization identified by id,
// but only when the caller is the owner (ownerID guard prevents cross-user edits).
func (r *OrganizationRepository) Update(ctx context.Context, id string, ownerID string, input models.OrganizationUpdateInput) (*models.Organization, error) {
	ctx, span := orgTracer.Start(ctx, "db.organizations.Update")
	defer span.End()
	span.SetAttributes(attribute.String("db.org_id", id))

	const q = `
		UPDATE organizations
		SET name = $3, avatar_url = NULLIF($4, ''), updated_on = NOW()
		WHERE id = $1 AND owner_id = $2
		RETURNING id, owner_id, name, avatar_url, status, created_on, updated_on`

	o, err := scanOrganization(r.pool.QueryRow(ctx, q, id, ownerID, input.Name, input.AvatarURL))
	if err != nil {
		if !errors.Is(err, domain.ErrOrganizationNotFound) {
			span.RecordError(err)
			err = fmt.Errorf("update organization: %w", err)
		}
		return nil, err
	}
	return o, nil
}

// Delete removes the organization identified by id, but only when the caller is the owner.
// Returns ErrOrganizationNotFound when no row matches (missing or wrong owner).
func (r *OrganizationRepository) Delete(ctx context.Context, id string, ownerID string) error {
	ctx, span := orgTracer.Start(ctx, "db.organizations.Delete")
	defer span.End()
	span.SetAttributes(attribute.String("db.org_id", id))

	const q = `DELETE FROM organizations WHERE id = $1 AND owner_id = $2`
	tag, err := r.pool.Exec(ctx, q, id, ownerID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete organization: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOrganizationNotFound
	}
	return nil
}

func scanOrganization(row scanner) (*models.Organization, error) {
	o := &models.Organization{}
	var avatarURL, status *string
	var createdOn, updatedOn *time.Time
	err := row.Scan(&o.ID, &o.OwnerID, &o.Name, &avatarURL, &status, &createdOn, &updatedOn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, err
	}
	o.AvatarURL = derefString(avatarURL)
	o.Status = derefString(status)
	if createdOn != nil {
		o.CreatedOn = *createdOn
	}
	if updatedOn != nil {
		o.UpdatedOn = *updatedOn
	}
	return o, nil
}
