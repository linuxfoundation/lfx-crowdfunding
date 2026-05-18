// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service contains the application service layer.
package service

import (
	"context"
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

// Create creates a Stripe subscription with 3DS support and records it in the database.
// When the first invoice requires an authentication challenge, the returned Subscription
// has Status == "incomplete" and ClientSecret set — the frontend must call
// stripe.confirmPayment(ClientSecret) to complete 3DS and activate the subscription.
// The webhook (invoice.payment_succeeded) advances the status to "active".
func (s *SubscriptionService) Create(ctx context.Context, initiativeID, userID, userEmail string, input models.SubscriptionCreateInput) (*models.Subscription, error) {
	ctx, span := subscriptionSvcTracer.Start(ctx, "SubscriptionService.Create")
	defer span.End()
	span.SetAttributes(
		attribute.String("initiative.id", initiativeID),
		attribute.String("user.id", userID),
	)

	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: amount_in_cents must be positive", domain.ErrInvalidInput)
	}
	if input.Frequency == "" {
		return nil, fmt.Errorf("%w: frequency is required", domain.ErrInvalidInput)
	}

	initiative, err := s.initiativeRepo.GetByID(ctx, initiativeID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if !initiative.AcceptFunding {
		return nil, fmt.Errorf("%w: initiative does not accept funding", domain.ErrInvalidInput)
	}

	// Resolve the Stripe customer for this user (create if first payment).
	customerID := ""
	user, err := s.userRepo.GetByUserID(ctx, userID)
	if err == nil {
		customerID = user.StripeCustomerID
	}
	if customerID == "" {
		customerID, err = s.stripe.CreateCustomer(ctx, userID, userEmail)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("create stripe customer: %w", err)
		}
		if persistErr := s.userRepo.UpdateStripeInfo(ctx, userID, customerID, ""); persistErr != nil {
			span.RecordError(persistErr)
			return nil, fmt.Errorf("persist stripe customer: %w", persistErr)
		}
	}

	// Get or create a recurring Price for this initiative / amount / interval.
	priceID, err := s.stripe.GetOrCreatePrice(ctx, initiativeID, input.AmountCents, input.Frequency)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe price: %w", err)
	}

	result, err := s.stripe.CreateSubscription(ctx, models.StripeSubscriptionRequest{
		InitiativeID:     initiativeID,
		UserID:           userID,
		StripeCustomerID: customerID,
		StripePriceID:    priceID,
		PaymentMethodID:  input.StripePaymentMethodID,
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe subscription: %w", err)
	}

	sub := &models.Subscription{
		UserID:                   userID,
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
func (s *SubscriptionService) Cancel(ctx context.Context, id, callerID string) error {
	ctx, span := subscriptionSvcTracer.Start(ctx, "SubscriptionService.Cancel")
	defer span.End()
	span.SetAttributes(attribute.String("subscription.id", id))

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if sub.UserID != callerID {
		return domain.ErrForbidden
	}

	if err := s.stripe.CancelSubscription(ctx, sub.StripeSubscriptionID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("cancel stripe subscription: %w", err)
	}

	sub.Status = "canceled"
	if _, err := s.repo.Update(ctx, sub); err != nil {
		span.RecordError(err)
		return fmt.Errorf("update subscription status: %w", err)
	}
	return nil
}
