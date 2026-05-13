// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

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
	stripe         clients.StripeClient
}

// NewSubscriptionService returns a SubscriptionService.
func NewSubscriptionService(
	repo domain.SubscriptionRepository,
	initiativeRepo domain.InitiativeRepository,
	stripe clients.StripeClient,
) *SubscriptionService {
	return &SubscriptionService{repo: repo, initiativeRepo: initiativeRepo, stripe: stripe}
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

// Create creates a Stripe subscription and records it in the database.
func (s *SubscriptionService) Create(ctx context.Context, initiativeID, userID string, input models.SubscriptionCreateInput) (*models.Subscription, error) {
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
	if initiative.StripePlanID == "" {
		return nil, fmt.Errorf("%w: initiative has no Stripe plan configured", domain.ErrInvalidInput)
	}

	result, err := s.stripe.CreateSubscription(ctx, models.StripeSubscriptionRequest{
		InitiativeID:    initiativeID,
		UserID:          userID,
		StripePriceID:   initiative.StripePlanID,
		PaymentMethodID: input.StripePaymentMethodID,
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
	}

	created, err := s.repo.Create(ctx, sub)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("record subscription: %w", err)
	}
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
