// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// offlineDonationCols reads an offline donation from donations d, with the
// initiative (i) and the approver who recorded it (rb) LEFT JOINed.
const offlineDonationCols = `
	d.id, COALESCE(CAST(d.initiative_id AS text), ''), COALESCE(i.name, ''), COALESCE(i.slug, ''),
	COALESCE(d.org_name, ''), COALESCE(d.org_domain, ''), d.current_amount_in_cents,
	COALESCE(d.category, ''), d.received_on, COALESCE(d.source_txn_id, ''), COALESCE(d.status, ''),
	COALESCE(rb.username, ''), d.created_on, d.user_id, COALESCE(CAST(d.recorded_by AS text), ''),
	COALESCE(d.b2b_org_sfid, '')`

const offlineDonationFrom = `
	FROM donations d
	LEFT JOIN initiatives i ON i.id = d.initiative_id
	LEFT JOIN users rb ON rb.id = d.recorded_by`

// CreateOffline inserts an offline donation as pending (#464). A donation with
// the same source_txn_id is returned as is, with created=false.
func (r *DonationRepository) CreateOffline(ctx context.Context, d *models.OfflineDonation) (*models.OfflineDonation, bool, error) {
	ctx, span := donationTracer.Start(ctx, "db.donations.CreateOffline")
	defer span.End()

	receivedOn, err := time.Parse(time.DateOnly, d.ReceivedOn)
	if err != nil {
		return nil, false, fmt.Errorf("%w: received_on must be YYYY-MM-DD", domain.ErrInvalidInput)
	}
	var id string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO donations
		       (user_id, initiative_id, org_name, org_domain, b2b_org_sfid, category,
		        current_amount_in_cents, payment_method, status,
		        source_txn_id, received_on, recorded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (source_txn_id) WHERE source_txn_id IS NOT NULL DO NOTHING
		RETURNING id`,
		d.UserID, d.InitiativeID, d.OrganizationName, d.OrganizationDomain,
		nullableString(d.B2BOrgSFID), nullableString(d.Category),
		d.AmountCents, models.PaymentMethodOffline, models.DonationStatusPending,
		d.SourceTxnID, receivedOn, d.RecordedByID,
	).Scan(&id)
	created := true
	if errors.Is(err, pgx.ErrNoRows) {
		created = false
		err = r.pool.QueryRow(ctx, `SELECT id FROM donations WHERE source_txn_id = $1`, d.SourceTxnID).Scan(&id)
	}
	if err != nil {
		span.RecordError(err)
		return nil, false, fmt.Errorf("create offline donation: %w", err)
	}

	stored, err := scanOfflineDonation(r.pool.QueryRow(ctx,
		`SELECT `+offlineDonationCols+offlineDonationFrom+` WHERE d.id = $1`, id))
	if err != nil {
		span.RecordError(err)
		return nil, false, fmt.Errorf("read offline donation: %w", err)
	}
	return stored, created, nil
}

// MarkOfflineSucceeded flips a pending offline donation to succeeded.
func (r *DonationRepository) MarkOfflineSucceeded(ctx context.Context, id string) error {
	ctx, span := donationTracer.Start(ctx, "db.donations.MarkOfflineSucceeded")
	defer span.End()

	if _, err := r.pool.Exec(ctx, `
		UPDATE donations SET status = $2, updated_on = NOW()
		WHERE id = $1 AND payment_method = $3`,
		id, models.DonationStatusSucceeded, models.PaymentMethodOffline); err != nil {
		span.RecordError(err)
		return fmt.Errorf("mark offline donation succeeded: %w", err)
	}
	return nil
}

// DeleteOfflinePending removes a pending offline donation.
func (r *DonationRepository) DeleteOfflinePending(ctx context.Context, id string) error {
	ctx, span := donationTracer.Start(ctx, "db.donations.DeleteOfflinePending")
	defer span.End()

	if _, err := r.pool.Exec(ctx, `
		DELETE FROM donations
		WHERE id = $1 AND payment_method = $2 AND status = $3`,
		id, models.PaymentMethodOffline, models.DonationStatusPending); err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete pending offline donation: %w", err)
	}
	return nil
}

// ListOffline returns offline donations, newest received first. An empty
// initiativeID lists every initiative's.
func (r *DonationRepository) ListOffline(ctx context.Context, initiativeID string, limit, offset int) ([]models.OfflineDonation, *models.PaginationMeta, error) {
	ctx, span := donationTracer.Start(ctx, "db.donations.ListOffline")
	defer span.End()

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	const where = ` WHERE d.payment_method = $1 AND ($2 = '' OR CAST(d.initiative_id AS text) = $2)`

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM donations d`+where,
		models.PaymentMethodOffline, initiativeID).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("count offline donations: %w", err)
	}

	rows, err := r.pool.Query(ctx, `SELECT `+offlineDonationCols+offlineDonationFrom+where+`
		ORDER BY d.received_on DESC, d.created_on DESC, d.id
		LIMIT $3 OFFSET $4`, models.PaymentMethodOffline, initiativeID, limit, offset)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list offline donations: %w", err)
	}
	defer rows.Close()

	result := []models.OfflineDonation{}
	for rows.Next() {
		d, err := scanOfflineDonation(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan offline donation: %w", err)
		}
		result = append(result, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate offline donations: %w", err)
	}
	return result, &models.PaginationMeta{Total: total, Limit: limit, Offset: offset}, nil
}

func scanOfflineDonation(row scanner) (*models.OfflineDonation, error) {
	d := &models.OfflineDonation{}
	var receivedOn *time.Time
	var createdOn *time.Time
	err := row.Scan(&d.ID, &d.InitiativeID, &d.InitiativeName, &d.InitiativeSlug,
		&d.OrganizationName, &d.OrganizationDomain, &d.AmountCents,
		&d.Category, &receivedOn, &d.SourceTxnID, &d.Status,
		&d.RecordedBy, &createdOn, &d.UserID, &d.RecordedByID, &d.B2BOrgSFID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDonationNotFound
		}
		return nil, err
	}
	if receivedOn != nil {
		d.ReceivedOn = receivedOn.Format(time.DateOnly)
	}
	if createdOn != nil {
		d.CreatedOn = *createdOn
	}
	return d, nil
}
