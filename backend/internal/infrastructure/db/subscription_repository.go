// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package db provides PostgreSQL connection helpers and repositories.
package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var subscriptionTracer = otel.Tracer("subscriptions-db")

// SubscriptionRepository implements domain.SubscriptionRepository against PostgreSQL.
type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

const subscriptionColumns = `
	id, user_id, initiative_id, organization_id, category,
	current_amount_in_cents, frequency, status,
	stripe_subscription_id, stripe_subscription_item_id, stripe_price_id,
	created_on, updated_on`

func scanSubscription(row scanner) (*models.Subscription, error) {
	s := &models.Subscription{}
	var (
		initiativeID, organizationID, category                        *string
		frequency, status                                             *string
		stripeSubscriptionID, stripeSubscriptionItemID, stripePriceID *string
		createdOn, updatedOn                                          *time.Time
	)
	err := row.Scan(
		&s.ID, &s.UserID, &initiativeID, &organizationID, &category,
		&s.CurrentAmountCents, &frequency, &status,
		&stripeSubscriptionID, &stripeSubscriptionItemID, &stripePriceID,
		&createdOn, &updatedOn,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, err
	}
	s.InitiativeID = derefString(initiativeID)
	s.OrganizationID = derefString(organizationID)
	s.Category = derefString(category)
	s.Frequency = derefString(frequency)
	s.Status = derefString(status)
	s.StripeSubscriptionID = derefString(stripeSubscriptionID)
	s.StripeSubscriptionItemID = derefString(stripeSubscriptionItemID)
	s.StripePriceID = derefString(stripePriceID)
	if createdOn != nil {
		s.CreatedOn = *createdOn
	}
	if updatedOn != nil {
		s.UpdatedOn = *updatedOn
	}
	return s, nil
}

// GetByID retrieves a subscription by UUID.
func (r *SubscriptionRepository) GetByID(ctx context.Context, id string) (*models.Subscription, error) {
	ctx, span := subscriptionTracer.Start(ctx, "db.subscriptions.GetByID")
	defer span.End()
	span.SetAttributes(attribute.String("db.subscription_id", id))

	q := "SELECT " + subscriptionColumns + " FROM subscriptions WHERE id = $1"
	row := r.pool.QueryRow(ctx, q, id)
	sub, err := scanSubscription(row)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return sub, nil
}

// ListByInitiative returns paginated subscriptions for an initiative.
func (r *SubscriptionRepository) ListByInitiative(ctx context.Context, initiativeID string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	ctx, span := subscriptionTracer.Start(ctx, "db.subscriptions.ListByInitiative")
	defer span.End()
	return r.listSubs(ctx, "initiative_id", initiativeID, filter)
}

// ListByUser returns paginated subscriptions for a user.
func (r *SubscriptionRepository) ListByUser(ctx context.Context, userID string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	ctx, span := subscriptionTracer.Start(ctx, "db.subscriptions.ListByUser")
	defer span.End()
	return r.listSubs(ctx, "user_id", userID, filter)
}

func (r *SubscriptionRepository) listSubs(ctx context.Context, col, val string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Build WHERE clause. col is always a hardcoded internal value (never user input).
	args := []any{val}
	clauses := []string{fmt.Sprintf("%s = $1", col)}
	if filter.Status != "" {
		args = append(args, filter.Status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	where := strings.Join(clauses, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM subscriptions WHERE %s", where), args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count subscriptions: %w", err)
	}

	dataArgs := append(args, limit, offset) //nolint:gocritic // intentional re-slice
	q := fmt.Sprintf("SELECT %s FROM subscriptions WHERE %s ORDER BY created_on DESC LIMIT $%d OFFSET $%d",
		subscriptionColumns, where, len(args)+1, len(args)+2)
	rows, err := r.pool.Query(ctx, q, dataArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		s, err := scanSubscription(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, *s)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate subscriptions: %w", err)
	}
	return subs, &models.PaginationMeta{Total: total, Limit: limit, Offset: offset}, nil
}

// Create inserts a new subscription row.
func (r *SubscriptionRepository) Create(ctx context.Context, s *models.Subscription) (*models.Subscription, error) {
	ctx, span := subscriptionTracer.Start(ctx, "db.subscriptions.Create")
	defer span.End()

	const q = `
		INSERT INTO subscriptions
		       (user_id, initiative_id, organization_id, category,
		        current_amount_in_cents, frequency, status,
		        stripe_subscription_id, stripe_subscription_item_id, stripe_price_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING ` + subscriptionColumns

	row := r.pool.QueryRow(ctx, q,
		s.UserID, s.InitiativeID, nullableString(s.OrganizationID),
		nullableString(s.Category), s.CurrentAmountCents, s.Frequency,
		nullableString(s.Status), nullableString(s.StripeSubscriptionID),
		nullableString(s.StripeSubscriptionItemID), nullableString(s.StripePriceID),
	)
	created, err := scanSubscription(row)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return created, nil
}

// UpdateByStripeSubscriptionID is called by the Stripe webhook to advance
// the subscription status (e.g. incomplete → active, active → past_due).
// Returns ErrAlreadyProcessed if the subscription is already in the target
// status (enabling idempotent webhook retries), or ErrSubscriptionNotFound.
func (r *SubscriptionRepository) UpdateByStripeSubscriptionID(ctx context.Context, stripeSubID, status string) error {
	ctx, span := subscriptionTracer.Start(ctx, "db.subscriptions.UpdateByStripeSubscriptionID")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.stripe_subscription_id", stripeSubID),
		attribute.String("db.status", status),
	)

	const q = `
		UPDATE subscriptions SET status = $2, updated_on = NOW()
		WHERE stripe_subscription_id = $1
		  AND status IS DISTINCT FROM $2`

	tag, err := r.pool.Exec(ctx, q, stripeSubID, status)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("update subscription by stripe id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM subscriptions WHERE stripe_subscription_id = $1)`, stripeSubID,
		).Scan(&exists); err != nil {
			span.RecordError(err)
			return fmt.Errorf("check subscription existence: %w", err)
		}
		if !exists {
			return domain.ErrSubscriptionNotFound
		}
		return domain.ErrAlreadyProcessed
	}
	return nil
}

// Update saves changes to a subscription (status, amount, Stripe IDs).
func (r *SubscriptionRepository) Update(ctx context.Context, s *models.Subscription) (*models.Subscription, error) {
	ctx, span := subscriptionTracer.Start(ctx, "db.subscriptions.Update")
	defer span.End()
	span.SetAttributes(attribute.String("db.subscription_id", s.ID))

	const q = `
		UPDATE subscriptions SET
		    status                      = $2,
		    current_amount_in_cents     = $3,
		    stripe_subscription_id      = $4,
		    stripe_subscription_item_id = $5
		WHERE id = $1
		RETURNING ` + subscriptionColumns

	row := r.pool.QueryRow(ctx, q,
		s.ID, nullableString(s.Status), s.CurrentAmountCents,
		nullableString(s.StripeSubscriptionID), nullableString(s.StripeSubscriptionItemID),
	)
	updated, err := scanSubscription(row)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return updated, nil
}
