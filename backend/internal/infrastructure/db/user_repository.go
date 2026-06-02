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

var userTracer = otel.Tracer("users-db")

// UserRepository implements domain.UserRepository against PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

const userColumns = `
	id, username, legacy_user_id, email, given_name, family_name, name, avatar_url,
	stripe_customer_id, stripe_default_payment_method,
	created_on, updated_on`

// GetByUsername retrieves a user by their LF SSO username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, span := userTracer.Start(ctx, "db.users.GetByUsername")
	defer span.End()
	span.SetAttributes(attribute.String("db.username", username))

	q := "SELECT " + userColumns + " FROM users WHERE username = $1"
	u, err := scanUser(r.pool.QueryRow(ctx, q, username))
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) {
			span.RecordError(err)
			err = fmt.Errorf("get user by username: %w", err)
		}
		return nil, err
	}
	return u, nil
}

// GetByID retrieves a user by their UUID primary key.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	ctx, span := userTracer.Start(ctx, "db.users.GetByID")
	defer span.End()
	span.SetAttributes(attribute.String("db.user.id", id))

	q := "SELECT " + userColumns + " FROM users WHERE id = $1"
	u, err := scanUser(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if !errors.Is(err, domain.ErrUserNotFound) {
			span.RecordError(err)
			err = fmt.Errorf("get user by id: %w", err)
		}
		return nil, err
	}
	return u, nil
}

// Upsert inserts or updates a user row identified by username (LF SSO username).
// Used by the auth flow to synchronise profile data on every login.
func (r *UserRepository) Upsert(ctx context.Context, u *models.User) (*models.User, error) {
	ctx, span := userTracer.Start(ctx, "db.users.Upsert")
	defer span.End()
	span.SetAttributes(attribute.String("db.username", u.Username))

	const q = `
		INSERT INTO users (username, legacy_user_id, email, given_name, family_name, name, avatar_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (username) DO UPDATE SET
			legacy_user_id = EXCLUDED.legacy_user_id,
			email          = EXCLUDED.email,
			given_name     = EXCLUDED.given_name,
			family_name    = EXCLUDED.family_name,
			name           = EXCLUDED.name,
			avatar_url     = EXCLUDED.avatar_url,
			updated_on     = NOW()
		RETURNING ` + userColumns

	result, err := scanUser(r.pool.QueryRow(ctx, q,
		u.Username, nullableString(u.LegacyUserID), nullableString(u.Email),
		nullableString(u.GivenName), nullableString(u.FamilyName),
		nullableString(u.Name), nullableString(u.AvatarURL),
	))
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return result, nil
}

// UpdateStripeInfo persists the Stripe Customer ID and default PaymentMethod
// for a user. Called after CreateCustomer and AttachPaymentMethod flows.
// userUUID is the users.id UUID primary key.
// An empty paymentMethodID leaves the existing stripe_default_payment_method
// unchanged (NULLIF ensures we never overwrite a real pm_xxx with an empty string).
func (r *UserRepository) UpdateStripeInfo(ctx context.Context, userUUID, customerID, paymentMethodID string) error {
	ctx, span := userTracer.Start(ctx, "db.users.UpdateStripeInfo")
	defer span.End()
	span.SetAttributes(attribute.String("db.user.id", userUUID))

	const q = `
		UPDATE users SET
			stripe_customer_id            = COALESCE(NULLIF($2, ''), stripe_customer_id),
			stripe_default_payment_method = COALESCE(NULLIF($3, ''), stripe_default_payment_method),
			updated_on = NOW()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, userUUID, customerID, paymentMethodID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("update stripe info: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// ClearStripePaymentMethod sets stripe_default_payment_method to NULL.
// userUUID is the users.id UUID primary key.
// Called when the user explicitly removes their saved card.
func (r *UserRepository) ClearStripePaymentMethod(ctx context.Context, userUUID string) error {
	ctx, span := userTracer.Start(ctx, "db.users.ClearStripePaymentMethod")
	defer span.End()
	span.SetAttributes(attribute.String("db.user.id", userUUID))

	const q = `
		UPDATE users
		SET stripe_default_payment_method = NULL, updated_on = NOW()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, userUUID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("clear stripe payment method: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func scanUser(row scanner) (*models.User, error) {
	u := &models.User{}
	var (
		legacyUserID                                  *string
		email, givenName, familyName, name, avatarURL *string
		stripeCustomerID, stripeDefaultPaymentMethod  *string
		createdOn, updatedOn                          *time.Time
	)
	err := row.Scan(
		&u.ID, &u.Username, &legacyUserID, &email, &givenName, &familyName, &name, &avatarURL,
		&stripeCustomerID, &stripeDefaultPaymentMethod,
		&createdOn, &updatedOn,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	u.LegacyUserID = derefString(legacyUserID)
	u.Email = derefString(email)
	u.GivenName = derefString(givenName)
	u.FamilyName = derefString(familyName)
	u.Name = derefString(name)
	u.AvatarURL = derefString(avatarURL)
	u.StripeCustomerID = derefString(stripeCustomerID)
	u.StripeDefaultPaymentMethod = derefString(stripeDefaultPaymentMethod)
	if createdOn != nil {
		u.CreatedOn = *createdOn
	}
	if updatedOn != nil {
		u.UpdatedOn = *updatedOn
	}
	return u, nil
}
