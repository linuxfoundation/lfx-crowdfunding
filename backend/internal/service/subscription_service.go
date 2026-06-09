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

var subscriptionSvcTracer = otel.Tracer("subscriptions-service")

// SubscriptionService orchestrates recurring donation (subscription) lifecycle.
type SubscriptionService struct {
	repo           domain.SubscriptionRepository
	initiativeRepo domain.InitiativeRepository
	userRepo       domain.UserRepository
	stripe         clients.StripeClient
}

// NewSubscriptionService returns a SubscriptionService.
func NewSubscriptionService(
	repo domain.SubscriptionRepository,
	initiativeRepo domain.InitiativeRepository,
	userRepo domain.UserRepository,
	stripe clients.StripeClient,
) *SubscriptionService {
	return &SubscriptionService{repo: repo, initiativeRepo: initiativeRepo, userRepo: userRepo, stripe: stripe}
}

// ListByInitiative returns paginated subscriptions for an initiative.
func (s *SubscriptionService) ListByInitiative(ctx context.Context, initiativeID string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	ctx, span := subscriptionSvcTracer.Start(ctx, "SubscriptionService.ListByInitiative")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", initiativeID))

	subs, meta, err := s.repo.ListByInitiative(ctx, initiativeID, filter)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list subscriptions: %w", err)
	}
	return subs, meta, nil
}

// ListByUser returns paginated subscriptions owned by the authenticated user.
func (s *SubscriptionService) ListByUser(ctx context.Context, username string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	ctx, span := subscriptionSvcTracer.Start(ctx, "SubscriptionService.ListByUser")
	defer span.End()
	span.SetAttributes(attribute.String("user.username", username))

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// User has no DB record yet — they have never subscribed. Return empty list.
			return []models.Subscription{}, &models.PaginationMeta{Limit: filter.Limit, Offset: filter.Offset}, nil
		}
		span.RecordError(err)
		return nil, nil, fmt.Errorf("resolve user: %w", err)
	}

	subs, meta, err := s.repo.ListByUser(ctx, user.ID, filter)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list subscriptions: %w", err)
	}
	return subs, meta, nil
}

// Create creates a Stripe subscription with 3DS support and records it in the database.
// When the first invoice requires an authentication challenge, the returned Subscription
// has Status == "incomplete" and ClientSecret set — the frontend must call
// stripe.confirmPayment(ClientSecret) to complete 3DS and activate the subscription.
// The webhook (invoice.payment_succeeded) advances the status to "active".
func (s *SubscriptionService) Create(ctx context.Context, initiativeID, username string, input models.SubscriptionCreateInput) (*models.Subscription, error) {
	ctx, span := subscriptionSvcTracer.Start(ctx, "SubscriptionService.Create")
	defer span.End()
	span.SetAttributes(
		attribute.String("initiative.id", initiativeID),
		attribute.String("user.username", username),
	)

	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: amount_cents must be positive", domain.ErrInvalidInput)
	}
	if input.Frequency == "" {
		return nil, fmt.Errorf("%w: frequency is required", domain.ErrInvalidInput)
	}
	// Validate the frequency value before hitting Stripe so the client gets a
	// 400 (ErrInvalidInput) rather than a 500 for an unsupported billing interval.
	switch input.Frequency {
	case "monthly", "month", "yearly", "year", "annual", "weekly", "week", "daily", "day":
		// valid
	default:
		return nil, fmt.Errorf("%w: unsupported frequency %q; supported: monthly, yearly, weekly, daily", domain.ErrInvalidInput, input.Frequency)
	}
	if input.StripePaymentMethodID == "" {
		return nil, fmt.Errorf("%w: stripe_payment_method_id is required", domain.ErrInvalidInput)
	}

	initiative, err := s.initiativeRepo.GetByID(ctx, initiativeID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if !initiative.AcceptFunding {
		return nil, fmt.Errorf("%w: initiative does not accept funding", domain.ErrInvalidInput)
	}
	// Validate Stripe product before doing any customer work so we fail fast
	// and cheaply when an initiative hasn't been fully configured yet.
	if initiative.StripeProductID == "" {
		return nil, fmt.Errorf("%w: initiative has no Stripe product configured", domain.ErrInvalidInput)
	}
	// Idempotency key is required so that retries of the same logical request
	// are de-duped by Stripe for both the Price and the Subscription call.
	if input.IdempotencyKey == "" {
		return nil, fmt.Errorf("%w: idempotency_key is required", domain.ErrInvalidInput)
	}

	// Resolve the Stripe customer for this user. Requires the user row to
	// already exist — callers must have completed PATCH /me on login (REQ-P4).
	// Email is sourced from the DB row, not from caller-supplied claims (REQ-P5).
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, fmt.Errorf("%w: no profile found — call PATCH /v1/me to sync your profile before subscribing", domain.ErrProfileNotSynced)
		}
		return nil, fmt.Errorf("resolve user: %w", err)
	}
	// Guard against legacy/migrated rows that have no email yet.
	// Stripe requires a non-empty email; direct the user to sync their profile.
	if user.Email == "" {
		return nil, fmt.Errorf("%w: email not set — call PATCH /v1/me to sync your profile before subscribing", domain.ErrProfileNotSynced)
	}
	customerID := user.StripeCustomerID
	if customerID == "" {
		customerID, err = s.stripe.CreateCustomer(ctx, user.LegacyUserID, user.Email)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("create stripe customer: %w", err)
		}
		if persistErr := s.userRepo.UpdateStripeInfo(ctx, user.ID, customerID, ""); persistErr != nil {
			span.RecordError(persistErr)
			return nil, fmt.Errorf("persist stripe customer: %w", persistErr)
		}
	}

	// Attach the Price to the initiative's existing Stripe Product rather than
	// creating a new Product per Price — keeps the Stripe catalog manageable.
	priceID, err := s.stripe.GetOrCreatePrice(ctx, initiative.StripeProductID, initiativeID, input.AmountCents, input.Frequency, input.IdempotencyKey)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe price: %w", err)
	}

	// Best-effort owner email lookup for admin notification email.
	// Failure here is non-fatal — the subscription proceeds without the email.
	ownerEmail := ""
	if owner, ownerErr := s.userRepo.GetByID(ctx, initiative.OwnerID); ownerErr == nil {
		ownerEmail = owner.Email
	}

	result, err := s.stripe.CreateSubscription(ctx, models.StripeSubscriptionRequest{
		InitiativeID:     initiativeID,
		InitiativeSlug:   initiative.Slug,
		InitiativeName:   initiative.Name,
		UserID:           user.LegacyUserID,
		DonorName:        user.Name,
		OwnerEmail:       ownerEmail,
		StripeCustomerID: customerID,
		StripePriceID:    priceID,
		PaymentMethodID:  input.StripePaymentMethodID,
		Category:         input.Category,
		OrganizationID:   input.OrganizationID,
		IdempotencyKey:   input.IdempotencyKey,
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe subscription: %w", err)
	}

	sub := &models.Subscription{
		UserID:                   user.ID,
		InitiativeID:             initiativeID,
		OrganizationID:           input.OrganizationID,
		Category:                 input.Category,
		CurrentAmountCents:       input.AmountCents,
		Frequency:                input.Frequency,
		Status:                   result.Status,
		StripeSubscriptionID:     result.SubscriptionID,
		StripeSubscriptionItemID: result.SubscriptionItemID,
		StripePriceID:            result.PriceID,
	}

	created, err := s.repo.Create(ctx, sub)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("record subscription: %w", err)
	}

	// Surface client_secret when 3DS is needed on the first invoice — transient, not stored.
	created.ClientSecret = result.ClientSecret
	return created, nil
}

// Cancel cancels a Stripe subscription and marks it cancelled in the database.
func (s *SubscriptionService) Cancel(ctx context.Context, id, callerUsername string) error {
	ctx, span := subscriptionSvcTracer.Start(ctx, "SubscriptionService.Cancel")
	defer span.End()
	span.SetAttributes(attribute.String("subscription.id", id))

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("get subscription: %w", err)
	}

	caller, err := s.userRepo.GetByUsername(ctx, callerUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Unknown caller cannot own any subscription.
			return domain.ErrForbidden
		}
		span.RecordError(err)
		return fmt.Errorf("resolve caller: %w", err)
	}
	if sub.UserID != caller.ID {
		return domain.ErrForbidden
	}

	if err := s.stripe.CancelSubscription(ctx, sub.StripeSubscriptionID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("cancel stripe subscription: %w", err)
	}

	sub.Status = models.SubscriptionStatusCanceled
	if _, err := s.repo.Update(ctx, sub); err != nil {
		span.RecordError(err)
		return fmt.Errorf("update subscription status: %w", err)
	}
	return nil
}
