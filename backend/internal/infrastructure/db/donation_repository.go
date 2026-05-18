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

var donationTracer = otel.Tracer("donations-db")

// DonationRepository implements domain.DonationRepository against PostgreSQL.
type DonationRepository struct {
	pool *pgxpool.Pool
}

// NewDonationRepository creates a new DonationRepository.
func NewDonationRepository(pool *pgxpool.Pool) *DonationRepository {
	return &DonationRepository{pool: pool}
}

// GetByID retrieves a single donation by UUID.
func (r *DonationRepository) GetByID(ctx context.Context, id string) (*models.Donation, error) {
	ctx, span := donationTracer.Start(ctx, "db.donations.GetByID")
	defer span.End()
	span.SetAttributes(attribute.String("db.donation_id", id))

	const q = `
		SELECT id, user_id, initiative_id, organization_id, category,
		       current_amount_in_cents, po_number, payment_method,
		       status, stripe_payment_intent_id, stripe_charge_id, created_on, updated_on
		FROM donations WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	d, err := scanDonation(row)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, domain.ErrDonationNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get donation: %w", err)
	}
	return d, nil
}

// ListByInitiative returns paginated donations for an initiative.
func (r *DonationRepository) ListByInitiative(ctx context.Context, initiativeID string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	ctx, span := donationTracer.Start(ctx, "db.donations.ListByInitiative")
	defer span.End()
	span.SetAttributes(attribute.String("db.initiative_id", initiativeID))

	return r.listDonations(ctx, "initiative_id", initiativeID, filter)
}

// ListByUser returns paginated donations for a user.
func (r *DonationRepository) ListByUser(ctx context.Context, userID string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	ctx, span := donationTracer.Start(ctx, "db.donations.ListByUser")
	defer span.End()
	return r.listDonations(ctx, "user_id", userID, filter)
}

func (r *DonationRepository) listDonations(ctx context.Context, col, val string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM donations WHERE %s = $1", col)
	if err := r.pool.QueryRow(ctx, countQ, val).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count donations: %w", err)
	}

	dataQ := fmt.Sprintf(`
		SELECT id, user_id, initiative_id, organization_id, category,
		       current_amount_in_cents, po_number, payment_method,
		       status, stripe_payment_intent_id, stripe_charge_id, created_on, updated_on
		FROM donations WHERE %s = $1
		ORDER BY created_on DESC LIMIT $2 OFFSET $3`, col)

	rows, err := r.pool.Query(ctx, dataQ, val, limit, offset)
	if err != nil {
		return nil, nil, fmt.Errorf("list donations: %w", err)
	}
	defer rows.Close()

	var donations []models.Donation
	for rows.Next() {
		d, err := scanDonation(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan donation: %w", err)
		}
		donations = append(donations, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate donations: %w", err)
	}
	return donations, &models.PaginationMeta{Total: total, Limit: limit, Offset: offset}, nil
}

// Create inserts a new donation row.
func (r *DonationRepository) Create(ctx context.Context, d *models.Donation) (*models.Donation, error) {
	ctx, span := donationTracer.Start(ctx, "db.donations.Create")
	defer span.End()

	const q = `
		INSERT INTO donations
		       (user_id, initiative_id, organization_id, category,
		        current_amount_in_cents, po_number, payment_method,
		        status, stripe_payment_intent_id, stripe_charge_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, user_id, initiative_id, organization_id, category,
		          current_amount_in_cents, po_number, payment_method,
		          status, stripe_payment_intent_id, stripe_charge_id, created_on, updated_on`

	row := r.pool.QueryRow(ctx, q,
		d.UserID, d.InitiativeID, nullableString(d.OrganizationID),
		nullableString(d.Category), d.CurrentAmountCents,
		nullableString(d.PONumber), nullableString(d.PaymentMethod),
		nullableString(d.Status), nullableString(d.StripePaymentIntentID),
		nullableString(d.StripeChargeID),
	)
	created, err := scanDonation(row)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create donation: %w", err)
	}
	return created, nil
}

// UpdateByPaymentIntentID is called by the Stripe webhook to reconcile the
// result of an async 3DS challenge. chargeID may be empty on failure events.
func (r *DonationRepository) UpdateByPaymentIntentID(ctx context.Context, piID, status, chargeID string) error {
	ctx, span := donationTracer.Start(ctx, "db.donations.UpdateByPaymentIntentID")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.payment_intent_id", piID),
		attribute.String("db.status", status),
	)

	const q = `
		UPDATE donations SET
			status          = $2,
			stripe_charge_id = COALESCE(NULLIF($3, ''), stripe_charge_id),
			updated_on      = NOW()
		WHERE stripe_payment_intent_id = $1`

	tag, err := r.pool.Exec(ctx, q, piID, status, chargeID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("update donation by payment intent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDonationNotFound
	}
	return nil
}

func scanDonation(row scanner) (*models.Donation, error) {
	d := &models.Donation{}
	var (
		initiativeID, organizationID, category *string
		poNumber, paymentMethod, status        *string
		stripePaymentIntentID, stripeChargeID  *string
		createdOn, updatedOn                   *time.Time
	)
	err := row.Scan(
		&d.ID, &d.UserID, &initiativeID, &organizationID, &category,
		&d.CurrentAmountCents, &poNumber, &paymentMethod,
		&status, &stripePaymentIntentID, &stripeChargeID, &createdOn, &updatedOn,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDonationNotFound
		}
		return nil, err
	}
	d.InitiativeID = derefString(initiativeID)
	d.OrganizationID = derefString(organizationID)
	d.Category = derefString(category)
	d.PONumber = derefString(poNumber)
	d.PaymentMethod = derefString(paymentMethod)
	d.Status = derefString(status)
	d.StripePaymentIntentID = derefString(stripePaymentIntentID)
	d.StripeChargeID = derefString(stripeChargeID)
	if createdOn != nil {
		d.CreatedOn = *createdOn
	}
	if updatedOn != nil {
		d.UpdatedOn = *updatedOn
	}
	return d, nil
}
