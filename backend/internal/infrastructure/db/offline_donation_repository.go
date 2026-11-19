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
// the same source_txn_id, ignoring case, is returned as is, with created=false.
// Finance treats "Invoice 12" and "invoice 12" as one reference, so CF does too
// (migration 020's unique index is on lower(source_txn_id)).
func (r *DonationRepository) CreateOffline(ctx context.Context, d *models.OfflineDonation) (*models.OfflineDonation, bool, error) {
	ctx, span := donationTracer.Start(ctx, "db.donations.CreateOffline")
	defer span.End()

	receivedOn, err := time.Parse(time.DateOnly, d.ReceivedOn)
	if err != nil {
		return nil, false, fmt.Errorf("%w: received_on must be YYYY-MM-DD", domain.ErrInvalidInput)
	}
	// A ledger adjustment with this reference (#489) means the Ledger already
	// has a row with it. Callers hold the reference lock, so this can't race.
	var taken bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM ledger_adjustments WHERE lower(source_txn_id) = lower($1))`,
		d.SourceTxnID).Scan(&taken); err != nil {
		span.RecordError(err)
		return nil, false, fmt.Errorf("create offline donation: check adjustments: %w", err)
	}
	if taken {
		return nil, false, domain.ErrSourceTxnIDTaken
	}
	var id string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO donations
		       (user_id, initiative_id, org_name, org_domain, b2b_org_sfid, category,
		        current_amount_in_cents, payment_method, status,
		        source_txn_id, received_on, recorded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT ((lower(source_txn_id))) WHERE source_txn_id IS NOT NULL DO NOTHING
		RETURNING id`,
		d.UserID, d.InitiativeID, d.OrganizationName, d.OrganizationDomain,
		nullableString(d.B2BOrgSFID), nullableString(d.Category),
		d.AmountCents, models.PaymentMethodOffline, models.DonationStatusPending,
		d.SourceTxnID, receivedOn, d.RecordedByID,
	).Scan(&id)
	created := true
	if errors.Is(err, pgx.ErrNoRows) {
		created = false
		err = r.pool.QueryRow(ctx, `SELECT id FROM donations WHERE lower(source_txn_id) = lower($1)`, d.SourceTxnID).Scan(&id)
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

// offlineUnlockTimeout bounds releasing the advisory lock after the request's
// context is done.
const offlineUnlockTimeout = 5 * time.Second

// LockOfflineReference takes the reference lock (see lockLedgerReference), so
// the create, Ledger post and finalize of one reference run one at a time
// across replicas. unlock releases it; call it exactly once.
func (r *DonationRepository) LockOfflineReference(ctx context.Context, sourceTxnID string) (func(), error) {
	ctx, span := donationTracer.Start(ctx, "db.donations.LockOfflineReference")
	defer span.End()

	unlock, err := lockLedgerReference(ctx, r.pool, sourceTxnID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("lock offline reference: %w", err)
	}
	return unlock, nil
}

// lockLedgerReference takes a Postgres advisory lock on a Ledger sourceTxnID,
// ignoring case. Offline donations and ledger adjustments (#489) share the key,
// so the cross-table uniqueness checks in CreateOffline and CreateAdjustment
// can't race. unlock releases it; call it exactly once.
func lockLedgerReference(ctx context.Context, pool *pgxpool.Pool, sourceTxnID string) (func(), error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire: %w", err)
	}
	const key = `hashtextextended('offline-donation:' || lower($1), 0)`
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(`+key+`)`, sourceTxnID); err != nil {
		// The lock may or may not be held: close the session so it's freed.
		_ = conn.Conn().Close(context.WithoutCancel(ctx))
		conn.Release()
		return nil, err
	}
	unlock := func() {
		uctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), offlineUnlockTimeout)
		defer cancel()
		if _, err := conn.Exec(uctx, `SELECT pg_advisory_unlock(`+key+`)`, sourceTxnID); err != nil {
			// Closing the session releases every lock it holds.
			_ = conn.Conn().Close(uctx)
		}
		conn.Release()
	}
	return unlock, nil
}

// SetOfflineB2BOrg fills the b2b_org SFID of a pending offline donation that
// has none, for a retry whose org lookup succeeded where the first one failed.
func (r *DonationRepository) SetOfflineB2BOrg(ctx context.Context, id, sfid string) error {
	ctx, span := donationTracer.Start(ctx, "db.donations.SetOfflineB2BOrg")
	defer span.End()

	if _, err := r.pool.Exec(ctx, `
		UPDATE donations SET b2b_org_sfid = $2, updated_on = NOW()
		WHERE id = $1 AND payment_method = $3 AND status = $4
		  AND (b2b_org_sfid IS NULL OR b2b_org_sfid = '')`,
		id, sfid, models.PaymentMethodOffline, models.DonationStatusPending); err != nil {
		span.RecordError(err)
		return fmt.Errorf("set offline donation b2b_org: %w", err)
	}
	return nil
}

// MarkOfflineSucceeded flips a pending offline donation to succeeded. It fails
// when no pending row with that id exists, so a caller never reports success
// for a row that isn't there.
func (r *DonationRepository) MarkOfflineSucceeded(ctx context.Context, id string) error {
	ctx, span := donationTracer.Start(ctx, "db.donations.MarkOfflineSucceeded")
	defer span.End()

	tag, err := r.pool.Exec(ctx, `
		UPDATE donations SET status = $2, updated_on = NOW()
		WHERE id = $1 AND payment_method = $3 AND status = $4`,
		id, models.DonationStatusSucceeded, models.PaymentMethodOffline, models.DonationStatusPending)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("mark offline donation succeeded: %w", err)
	}
	if tag.RowsAffected() == 0 {
		err := fmt.Errorf("mark offline donation succeeded: no pending offline donation %s", id)
		span.RecordError(err)
		return err
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
