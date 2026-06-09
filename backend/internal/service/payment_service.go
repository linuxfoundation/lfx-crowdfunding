// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service contains the application service layer.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var paymentSvcTracer = otel.Tracer("payment-service")

// PaymentService manages Stripe customer and card lifecycle for authenticated users.
type PaymentService struct {
	userRepo domain.UserRepository
	stripe   clients.StripeClient
}

// NewPaymentService returns a PaymentService.
func NewPaymentService(userRepo domain.UserRepository, stripe clients.StripeClient) *PaymentService {
	return &PaymentService{userRepo: userRepo, stripe: stripe}
}

// ensureCustomer returns the user's UUID and existing Stripe customer ID, or creates one.
// Requires the user row to already exist (created by PATCH /me on login — REQ-P4).
// Email is read from the DB row so JWT claims are never used as the source of
// truth for profile data (REQ-P5).
func (s *PaymentService) ensureCustomer(ctx context.Context, username string) (userUUID string, customerID string, err error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", "", fmt.Errorf("%w: no profile found — call PATCH /v1/me to sync your profile before using payment features", domain.ErrProfileNotSynced)
		}
		return "", "", err
	}
	if user.StripeCustomerID != "" {
		return user.ID, user.StripeCustomerID, nil
	}
	// Guard against legacy/migrated rows that have no email yet.
	// Stripe requires a non-empty email; direct the user to sync their profile.
	if user.Email == "" {
		return "", "", fmt.Errorf("%w: email not set — call PATCH /v1/me to sync your profile before using payment features", domain.ErrProfileNotSynced)
	}
	newCustomerID, err := s.stripe.CreateCustomer(ctx, user.LegacyUserID, user.Email)
	if err != nil {
		return "", "", fmt.Errorf("create stripe customer: %w", err)
	}
	if err := s.userRepo.UpdateStripeInfo(ctx, user.ID, newCustomerID, ""); err != nil {
		return "", "", fmt.Errorf("persist stripe customer: %w", err)
	}
	return user.ID, newCustomerID, nil
}

// CreateSetupIntent creates a Stripe SetupIntent for the authenticated user.
// The returned client_secret is passed to the frontend Stripe.js Payment Element
// to collect and 3DS-challenge the card before it is attached to the customer.
func (s *PaymentService) CreateSetupIntent(ctx context.Context, username string) (*models.SetupIntentResult, error) {
	ctx, span := paymentSvcTracer.Start(ctx, "PaymentService.CreateSetupIntent")
	defer span.End()
	span.SetAttributes(attribute.String("user.username", username))

	_, customerID, err := s.ensureCustomer(ctx, username)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	clientSecret, err := s.stripe.CreateSetupIntent(ctx, customerID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("setup intent: %w", err)
	}
	return &models.SetupIntentResult{ClientSecret: clientSecret}, nil
}

// AttachPaymentMethod attaches pm_xxx to the user's Stripe customer after the
// frontend has confirmed the SetupIntent. The card is set as the customer's
// default invoice payment method and persisted to the users table.
func (s *PaymentService) AttachPaymentMethod(ctx context.Context, username, paymentMethodID string) (*models.CardDetails, error) {
	ctx, span := paymentSvcTracer.Start(ctx, "PaymentService.AttachPaymentMethod")
	defer span.End()
	span.SetAttributes(attribute.String("user.username", username))

	if paymentMethodID == "" {
		return nil, fmt.Errorf("%w: payment_method_id is required", domain.ErrInvalidInput)
	}

	userUUID, customerID, err := s.ensureCustomer(ctx, username)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	card, err := s.stripe.AttachPaymentMethod(ctx, customerID, paymentMethodID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("attach payment method: %w", err)
	}

	// Note: if Stripe's attach succeeded but this DB write fails, the card exists
	// in Stripe but is not recorded locally. This is a known distributed-transaction
	// limitation; a subsequent AttachPaymentMethod call or admin re-sync will recover.
	if err := s.userRepo.UpdateStripeInfo(ctx, userUUID, customerID, paymentMethodID); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("persist payment method: %w", err)
	}
	return card, nil
}

// GetPaymentAccount returns the saved card details for the authenticated user.
// Returns ErrPaymentMethodNotFound (HTTP 404) when no card is on file.
func (s *PaymentService) GetPaymentAccount(ctx context.Context, username string) (*models.CardDetails, error) {
	ctx, span := paymentSvcTracer.Start(ctx, "PaymentService.GetPaymentAccount")
	defer span.End()
	span.SetAttributes(attribute.String("user.username", username))

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// User has no DB record — they have no payment method on file.
			return nil, fmt.Errorf("%w: no payment method on file", domain.ErrPaymentMethodNotFound)
		}
		span.RecordError(err)
		return nil, err
	}
	if user.StripeDefaultPaymentMethod == "" {
		return nil, fmt.Errorf("%w: no payment method on file", domain.ErrPaymentMethodNotFound)
	}

	card, err := s.stripe.GetPaymentMethod(ctx, user.StripeDefaultPaymentMethod)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get payment method: %w", err)
	}
	return card, nil
}

// DeletePaymentMethod detaches the user's saved card from Stripe and clears
// the reference in the database.
// Returns ErrPaymentMethodNotFound (HTTP 404) when no card is on file.
func (s *PaymentService) DeletePaymentMethod(ctx context.Context, username string) error {
	ctx, span := paymentSvcTracer.Start(ctx, "PaymentService.DeletePaymentMethod")
	defer span.End()
	span.SetAttributes(attribute.String("user.username", username))

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// User has no DB record — they have no payment method on file.
			return fmt.Errorf("%w: no payment method on file", domain.ErrPaymentMethodNotFound)
		}
		span.RecordError(err)
		return err
	}
	if user.StripeDefaultPaymentMethod == "" {
		return fmt.Errorf("%w: no payment method on file", domain.ErrPaymentMethodNotFound)
	}

	if err := s.stripe.DetachPaymentMethod(ctx, user.StripeDefaultPaymentMethod); err != nil {
		span.RecordError(err)
		return fmt.Errorf("detach payment method: %w", err)
	}

	if err := s.userRepo.ClearStripePaymentMethod(ctx, user.ID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("clear payment method: %w", err)
	}
	return nil
}
