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

var donationSvcTracer = otel.Tracer("donations-service")

// DonationService orchestrates donation creation and retrieval.
type DonationService struct {
	repo           domain.DonationRepository
	initiativeRepo domain.InitiativeRepository
	stripe         clients.StripeClient
}

// NewDonationService returns a DonationService.
func NewDonationService(
	repo domain.DonationRepository,
	initiativeRepo domain.InitiativeRepository,
	stripe clients.StripeClient,
) *DonationService {
	return &DonationService{repo: repo, initiativeRepo: initiativeRepo, stripe: stripe}
}

// ListByInitiative returns paginated donations for an initiative.
func (s *DonationService) ListByInitiative(ctx context.Context, initiativeID string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	ctx, span := donationSvcTracer.Start(ctx, "DonationService.ListByInitiative")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", initiativeID))

	donations, meta, err := s.repo.ListByInitiative(ctx, initiativeID, filter)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list donations: %w", err)
	}
	return donations, meta, nil
}

// Create processes a one-time donation: validates the initiative, charges via Stripe, records in DB.
func (s *DonationService) Create(ctx context.Context, initiativeID, userID string, input models.DonationCreateInput) (*models.Donation, error) {
	ctx, span := donationSvcTracer.Start(ctx, "DonationService.Create")
	defer span.End()
	span.SetAttributes(
		attribute.String("initiative.id", initiativeID),
		attribute.String("user.id", userID),
	)

	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: amount_in_cents must be positive", domain.ErrInvalidInput)
	}

	// Verify the initiative exists and accepts funding.
	initiative, err := s.initiativeRepo.GetByID(ctx, initiativeID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if !initiative.AcceptFunding {
		return nil, fmt.Errorf("%w: initiative does not accept funding", domain.ErrInvalidInput)
	}

	// Charge via Stripe.
	pi, err := s.stripe.CreatePaymentIntent(ctx, models.PaymentIntentRequest{
		InitiativeID:    initiativeID,
		UserID:          userID,
		AmountCents:     input.AmountCents,
		PaymentMethodID: input.StripePaymentMethodID,
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe charge: %w", err)
	}

	donation := &models.Donation{
		UserID:             userID,
		InitiativeID:       initiativeID,
		OrganizationID:     input.OrganizationID,
		Category:           input.Category,
		CurrentAmountCents: input.AmountCents,
		PONumber:           input.PONumber,
		PaymentMethod:      input.PaymentMethod,
		Status:             pi.Status,
		StripeChargeID:     pi.ID,
	}

	created, err := s.repo.Create(ctx, donation)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("record donation: %w", err)
	}
	return created, nil
}
