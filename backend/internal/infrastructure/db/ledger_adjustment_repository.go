// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

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
)

var adjustmentTracer = otel.Tracer("ledger-adjustments-db")

// LedgerAdjustmentRepository stores returns and payouts approvers record
// (#489) in ledger_adjustments.
type LedgerAdjustmentRepository struct {
	pool *pgxpool.Pool
}

// NewLedgerAdjustmentRepository returns a LedgerAdjustmentRepository.
func NewLedgerAdjustmentRepository(pool *pgxpool.Pool) *LedgerAdjustmentRepository {
	return &LedgerAdjustmentRepository{pool: pool}
}

// adjustmentCols reads an adjustment from ledger_adjustments a, with the
// initiative (i) and the approver who recorded it (rb) LEFT JOINed.
const adjustmentCols = `
	a.id, a.kind, CAST(a.initiative_id AS text), COALESCE(i.name, ''), COALESCE(i.slug, ''),
	COALESCE(a.org_name, ''), COALESCE(a.org_domain, ''), COALESCE(a.paid_to, ''),
	a.amount_cents, a.category, a.reason, a.adjusted_on, a.source_txn_id, a.status,
	COALESCE(rb.username, ''), a.created_on, CAST(a.recorded_by AS text), COALESCE(a.b2b_org_sfid, '')`

const adjustmentFrom = `
	FROM ledger_adjustments a
	LEFT JOIN initiatives i ON i.id = a.initiative_id
	LEFT JOIN users rb ON rb.id = a.recorded_by`

// LockReference takes the reference lock offline donations use too.
func (r *LedgerAdjustmentRepository) LockReference(ctx context.Context, sourceTxnID string) (func(), error) {
	ctx, span := adjustmentTracer.Start(ctx, "db.ledger_adjustments.LockReference")
	defer span.End()

	unlock, err := lockLedgerReference(ctx, r.pool, sourceTxnID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("lock adjustment reference: %w", err)
	}
	return unlock, nil
}

// CreateAdjustment inserts a as pending. An adjustment with the same
// source_txn_id, ignoring case, is returned as is with created=false; an
// offline donation with it is ErrSourceTxnIDTaken. Callers hold LockReference.
func (r *LedgerAdjustmentRepository) CreateAdjustment(ctx context.Context, a *models.LedgerAdjustment) (*models.LedgerAdjustment, bool, error) {
	ctx, span := adjustmentTracer.Start(ctx, "db.ledger_adjustments.CreateAdjustment")
	defer span.End()

	adjustedOn, err := time.Parse(time.DateOnly, a.AdjustedOn)
	if err != nil {
		return nil, false, fmt.Errorf("%w: adjusted_on must be YYYY-MM-DD", domain.ErrInvalidInput)
	}
	var taken bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM donations WHERE lower(source_txn_id) = lower($1))`,
		a.SourceTxnID).Scan(&taken); err != nil {
		span.RecordError(err)
		return nil, false, fmt.Errorf("create adjustment: check donations: %w", err)
	}
	if taken {
		return nil, false, domain.ErrSourceTxnIDTaken
	}

	var id string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO ledger_adjustments
		       (kind, initiative_id, amount_cents, org_name, org_domain, b2b_org_sfid,
		        paid_to, reason, category, adjusted_on, source_txn_id, status, recorded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT ((lower(source_txn_id))) DO NOTHING
		RETURNING id`,
		a.Kind, a.InitiativeID, a.AmountCents,
		nullableString(a.OrganizationName), nullableString(a.OrganizationDomain), nullableString(a.B2BOrgSFID),
		nullableString(a.PaidTo), a.Reason, a.Category, adjustedOn, a.SourceTxnID,
		models.DonationStatusPending, a.RecordedByID,
	).Scan(&id)
	created := true
	if errors.Is(err, pgx.ErrNoRows) {
		created = false
		err = r.pool.QueryRow(ctx, `SELECT id FROM ledger_adjustments WHERE lower(source_txn_id) = lower($1)`, a.SourceTxnID).Scan(&id)
	}
	if err != nil {
		span.RecordError(err)
		return nil, false, fmt.Errorf("create adjustment: %w", err)
	}

	stored, err := scanAdjustment(r.pool.QueryRow(ctx, `SELECT `+adjustmentCols+adjustmentFrom+` WHERE a.id = $1`, id))
	if err != nil {
		span.RecordError(err)
		return nil, false, fmt.Errorf("read adjustment: %w", err)
	}
	return stored, created, nil
}

// SetAdjustmentB2BOrg fills the b2b_org SFID of a pending adjustment that has
// none, for a retry whose org lookup succeeded where the first one failed.
func (r *LedgerAdjustmentRepository) SetAdjustmentB2BOrg(ctx context.Context, id, sfid string) error {
	ctx, span := adjustmentTracer.Start(ctx, "db.ledger_adjustments.SetAdjustmentB2BOrg")
	defer span.End()

	if _, err := r.pool.Exec(ctx, `
		UPDATE ledger_adjustments SET b2b_org_sfid = $2
		WHERE id = $1 AND status = $3 AND (b2b_org_sfid IS NULL OR b2b_org_sfid = '')`,
		id, sfid, models.DonationStatusPending); err != nil {
		span.RecordError(err)
		return fmt.Errorf("set adjustment b2b_org: %w", err)
	}
	return nil
}

// MarkAdjustmentSucceeded flips a pending adjustment to succeeded. It fails
// when no pending row with that id exists.
func (r *LedgerAdjustmentRepository) MarkAdjustmentSucceeded(ctx context.Context, id string) error {
	ctx, span := adjustmentTracer.Start(ctx, "db.ledger_adjustments.MarkAdjustmentSucceeded")
	defer span.End()

	tag, err := r.pool.Exec(ctx, `
		UPDATE ledger_adjustments SET status = $2 WHERE id = $1 AND status = $3`,
		id, models.DonationStatusSucceeded, models.DonationStatusPending)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("mark adjustment succeeded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		err := fmt.Errorf("mark adjustment succeeded: no pending adjustment %s", id)
		span.RecordError(err)
		return err
	}
	return nil
}

// DeleteAdjustmentPending removes a pending adjustment.
func (r *LedgerAdjustmentRepository) DeleteAdjustmentPending(ctx context.Context, id string) error {
	ctx, span := adjustmentTracer.Start(ctx, "db.ledger_adjustments.DeleteAdjustmentPending")
	defer span.End()

	if _, err := r.pool.Exec(ctx, `DELETE FROM ledger_adjustments WHERE id = $1 AND status = $2`,
		id, models.DonationStatusPending); err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete pending adjustment: %w", err)
	}
	return nil
}

// ListAdjustments returns adjustments, newest first. An empty initiativeID
// lists every initiative's.
func (r *LedgerAdjustmentRepository) ListAdjustments(ctx context.Context, initiativeID string, limit, offset int) ([]models.LedgerAdjustment, *models.PaginationMeta, error) {
	ctx, span := adjustmentTracer.Start(ctx, "db.ledger_adjustments.ListAdjustments")
	defer span.End()

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	const where = ` WHERE ($1 = '' OR CAST(a.initiative_id AS text) = $1)`

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM ledger_adjustments a`+where, initiativeID).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("count adjustments: %w", err)
	}

	rows, err := r.pool.Query(ctx, `SELECT `+adjustmentCols+adjustmentFrom+where+`
		ORDER BY a.adjusted_on DESC, a.created_on DESC, a.id
		LIMIT $2 OFFSET $3`, initiativeID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list adjustments: %w", err)
	}
	defer rows.Close()

	result := []models.LedgerAdjustment{}
	for rows.Next() {
		a, err := scanAdjustment(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan adjustment: %w", err)
		}
		result = append(result, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate adjustments: %w", err)
	}
	return result, &models.PaginationMeta{Total: total, Limit: limit, Offset: offset}, nil
}

func scanAdjustment(row scanner) (*models.LedgerAdjustment, error) {
	a := &models.LedgerAdjustment{}
	var adjustedOn time.Time
	err := row.Scan(&a.ID, &a.Kind, &a.InitiativeID, &a.InitiativeName, &a.InitiativeSlug,
		&a.OrganizationName, &a.OrganizationDomain, &a.PaidTo,
		&a.AmountCents, &a.Category, &a.Reason, &adjustedOn, &a.SourceTxnID, &a.Status,
		&a.RecordedBy, &a.CreatedOn, &a.RecordedByID, &a.B2BOrgSFID)
	if err != nil {
		return nil, err
	}
	a.AdjustedOn = adjustedOn.Format(time.DateOnly)
	return a, nil
}
