// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

const (
	initiativeTypeMentorship = "mentorship"
	menteeGoalName           = "mentee"
	menteeGoalAmountCents    = 600000
	menteeGoalSortOrder      = 7
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
// jobspring_project_id, ensures the fixed mentee funding goal exists, and runs
// all writes in a single transaction. Returns the initiative UUID.
func (r *MentorshipRepositoryImpl) UpsertProgram(ctx context.Context, p models.MentorshipProgram) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx for program %q: %w", p.JobspringProjectID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Resolve owner: upsert a stub user row by LF username, then fetch the UUID.
	// DO NOTHING avoids a no-op UPDATE that would fire the set_updated_on trigger.
	const insertOwner = `
INSERT INTO users (username, created_on, updated_on)
VALUES ($1, NOW(), NOW())
ON CONFLICT (username) DO NOTHING`
	const selectOwner = `SELECT id FROM users WHERE username = $1`

	if _, err := tx.Exec(ctx, insertOwner, p.OwnerLFUsername); err != nil {
		return "", fmt.Errorf("insert owner user %q: %w", p.OwnerLFUsername, err)
	}
	var ownerID string
	if err := tx.QueryRow(ctx, selectOwner, p.OwnerLFUsername).Scan(&ownerID); err != nil {
		return "", fmt.Errorf("get owner user %q: %w", p.OwnerLFUsername, err)
	}

	// 2. Upsert initiative row with all Snowflake-sourced fields.
	// PROGRAM_ID is used directly as initiatives.id (PK) and jobspring_project_id,
	// so ON CONFLICT (id) is always valid and needs no separate unique index.
	const upsertInitiative = `
INSERT INTO initiatives (
	id,
	initiative_type,
	jobspring_project_id,
	owner_id,
	name,
	status,
	description,
	slug,
	industry,
	created_on,
	updated_on
) VALUES (
	$1::uuid,
	$2,
	$1::text,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8,
	NOW(),
	NOW()
)
ON CONFLICT (id) DO UPDATE SET
	jobspring_project_id = EXCLUDED.jobspring_project_id,
	owner_id    = EXCLUDED.owner_id,
	name        = EXCLUDED.name,
	status      = EXCLUDED.status,
	description = EXCLUDED.description,
	slug        = EXCLUDED.slug,
	industry    = EXCLUDED.industry,
	updated_on  = NOW()
RETURNING id`

	var id string
	if err := tx.QueryRow(ctx, upsertInitiative,
		p.JobspringProjectID,
		initiativeTypeMentorship,
		ownerID,
		p.Name,
		p.Status,
		nullableMentorshipString(p.Description),
		nullableMentorshipString(p.Slug),
		nullableMentorshipString(p.Industry),
	).Scan(&id); err != nil {
		return "", fmt.Errorf("upsert mentorship program %q: %w", p.JobspringProjectID, err)
	}

	// 3. Ensure the fixed "mentee" funding goal row exists (idempotent).
	const insertMenteeGoal = `
INSERT INTO initiative_goals (id, initiative_id, name, amount_in_cents, sort_order)
VALUES (gen_random_uuid(), $1, $2, $3, $4)
ON CONFLICT (initiative_id, name) DO NOTHING`
	if _, err := tx.Exec(ctx, insertMenteeGoal, id, menteeGoalName, menteeGoalAmountCents, menteeGoalSortOrder); err != nil {
		return "", fmt.Errorf("insert mentee goal for %q: %w", p.JobspringProjectID, err)
	}

	// 4. Replace beneficiaries (nil = skip; non-nil = replace all).
	if p.Beneficiaries != nil {
		if _, err := tx.Exec(ctx,
			`DELETE FROM initiative_beneficiaries WHERE initiative_id = $1`, id,
		); err != nil {
			return "", fmt.Errorf("delete beneficiaries for %q: %w", p.JobspringProjectID, err)
		}
		for _, b := range p.Beneficiaries {
			if _, err := tx.Exec(ctx,
				`INSERT INTO initiative_beneficiaries (id, initiative_id, name, email, created_on, updated_on)
				 VALUES ($1, $2, $3, $4, NOW(), NOW())`,
				uuid.New().String(), id, b.Name, b.Email,
			); err != nil {
				return "", fmt.Errorf("insert beneficiary %q for %q: %w", b.Email, p.JobspringProjectID, err)
			}
		}
	}

	// 5. Replace skills (nil = skip; non-nil = replace all).
	if p.Skills != nil {
		if _, err := tx.Exec(ctx,
			`DELETE FROM initiative_program_info_skills WHERE initiative_id = $1`, id,
		); err != nil {
			return "", fmt.Errorf("delete skills for %q: %w", p.JobspringProjectID, err)
		}
		for _, skill := range p.Skills {
			skill = strings.TrimSpace(skill)
			if skill == "" {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO initiative_program_info_skills (id, initiative_id, skill)
				 VALUES (gen_random_uuid(), $1, $2)
				 ON CONFLICT (initiative_id, skill) DO NOTHING`,
				id, skill,
			); err != nil {
				return "", fmt.Errorf("insert skill %q for %q: %w", skill, p.JobspringProjectID, err)
			}
		}
	}

	// 6. Replace mentors (nil = skip; non-nil = replace all).
	if p.Mentors != nil {
		if _, err := tx.Exec(ctx,
			`DELETE FROM initiative_mentors WHERE initiative_id = $1`, id,
		); err != nil {
			return "", fmt.Errorf("delete mentors for %q: %w", p.JobspringProjectID, err)
		}
		for _, m := range p.Mentors {
			if _, err := tx.Exec(ctx,
				`INSERT INTO initiative_mentors (id, initiative_id, name, email, avatar_url)
				 VALUES (gen_random_uuid(), $1, $2, $3, $4)`,
				id,
				nullableMentorshipString(m.Name),
				nullableMentorshipString(m.Email),
				nullableMentorshipString(m.AvatarURL),
			); err != nil {
				return "", fmt.Errorf("insert mentor %q for %q: %w", m.Email, p.JobspringProjectID, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit program tx for %q: %w", p.JobspringProjectID, err)
	}
	return id, nil
}

// ListJobspringIDs returns the jobspring_project_id for all existing mentorship initiatives.
func (r *MentorshipRepositoryImpl) ListJobspringIDs(ctx context.Context) ([]string, error) {
	const q = `
SELECT jobspring_project_id
FROM initiatives
WHERE initiative_type = $1
  AND jobspring_project_id IS NOT NULL
`
	rows, err := r.pool.Query(ctx, q, initiativeTypeMentorship)
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

// nullableMentorshipString converts an empty string to nil (stored as NULL in Postgres).
func nullableMentorshipString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
