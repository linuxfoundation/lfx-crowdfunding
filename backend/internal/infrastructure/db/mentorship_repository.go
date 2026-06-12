// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// MentorshipRepositoryImpl implements domain.MentorshipRepository against CF Postgres.
type MentorshipRepositoryImpl struct {
	pool *pgxpool.Pool
}

// NewMentorshipRepository returns a MentorshipRepositoryImpl backed by pool.
func NewMentorshipRepository(pool *pgxpool.Pool) *MentorshipRepositoryImpl {
	return &MentorshipRepositoryImpl{pool: pool}
}

// UpsertProgram inserts or updates the mentorship initiative row identified by
// jobspring_project_id. Returns the initiative UUID.
func (r *MentorshipRepositoryImpl) UpsertProgram(ctx context.Context, p models.MentorshipProgram) (string, error) {
	const q = `
INSERT INTO initiatives (
	id,
	initiative_type,
	jobspring_project_id,
	name,
	status,
	created_on,
	updated_on
) VALUES (
	gen_random_uuid(),
	'mentorship',
	$1,
	$2,
	$3,
	NOW(),
	NOW()
)
ON CONFLICT (jobspring_project_id) DO UPDATE SET
	name       = EXCLUDED.name,
	status     = EXCLUDED.status,
	updated_on = NOW()
RETURNING id
`
	var id string
	err := r.pool.QueryRow(ctx, q, p.JobspringProjectID, p.Name, p.Status).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert mentorship program %q: %w", p.JobspringProjectID, err)
	}
	return id, nil
}

// UpsertBeneficiaries replaces all beneficiary rows for the given initiative.
// Runs in a transaction: delete existing → insert new.
func (r *MentorshipRepositoryImpl) UpsertBeneficiaries(ctx context.Context, initiativeID string, beneficiaries []models.MentorshipBeneficiary) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx for beneficiaries %q: %w", initiativeID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`DELETE FROM initiative_beneficiaries WHERE initiative_id = $1`, initiativeID,
	); err != nil {
		return fmt.Errorf("delete beneficiaries for %q: %w", initiativeID, err)
	}

	for _, b := range beneficiaries {
		if _, err := tx.Exec(ctx,
			`INSERT INTO initiative_beneficiaries (id, initiative_id, name, email, created_on, updated_on)
			 VALUES ($1, $2, $3, $4, $5, $5)`,
			uuid.New().String(), initiativeID, b.Name, b.Email, time.Now(),
		); err != nil {
			return fmt.Errorf("insert beneficiary %q for %q: %w", b.Email, initiativeID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit beneficiaries tx for %q: %w", initiativeID, err)
	}
	return nil
}

// ListJobspringIDs returns the jobspring_project_id for all existing mentorship initiatives.
func (r *MentorshipRepositoryImpl) ListJobspringIDs(ctx context.Context) ([]string, error) {
	const q = `
SELECT jobspring_project_id
FROM initiatives
WHERE initiative_type = 'mentorship'
  AND jobspring_project_id IS NOT NULL
`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list jobspring IDs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan jobspring ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
