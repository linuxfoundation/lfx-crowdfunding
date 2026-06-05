// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service contains the application service layer.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	userRepo       domain.UserRepository
	stripe         clients.StripeClient
}

// NewDonationService returns a DonationService.
func NewDonationService(
	repo domain.DonationRepository,
	initiativeRepo domain.InitiativeRepository,
	userRepo domain.UserRepository,
	stripe clients.StripeClient,
) *DonationService {
	return &DonationService{repo: repo, initiativeRepo: initiativeRepo, userRepo: userRepo, stripe: stripe}
}

// ListByInitiative returns a paginated public summary of donations for an
// initiative. Each entry is enriched with donor name and avatar from the CF DB;
// Stripe IDs and user_id are never included in the summary.
func (s *DonationService) ListByInitiative(ctx context.Context, initiativeID string, filter models.DonationFilter) ([]models.DonationSummary, *models.PaginationMeta, error) {
	ctx, span := donationSvcTracer.Start(ctx, "DonationService.ListByInitiative")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", initiativeID))

	donations, meta, err := s.repo.ListByInitiative(ctx, initiativeID, filter)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list donations: %w", err)
	}

	summaries := projectDonationSummaries(ctx, s.initiativeRepo, donations)
	return summaries, meta, nil
}

// projectDonationSummaries converts raw donation rows into public-facing
// DonationSummary values, enriching each with donor name and avatar from the
// CF DB. No Stripe IDs or user_id values are included in the output.
func projectDonationSummaries(ctx context.Context, repo domain.InitiativeRepository, donations []models.Donation) []models.DonationSummary {
	if len(donations) == 0 {
		return []models.DonationSummary{}
	}

	// Collect unique IDs for batch lookup.
	userIDs := make([]string, 0, len(donations))
	orgIDs := make([]string, 0, len(donations))
	seenUsers := map[string]bool{}
	seenOrgs := map[string]bool{}
	for _, d := range donations {
		if d.UserID != "" && !seenUsers[d.UserID] {
			userIDs = append(userIDs, d.UserID)
			seenUsers[d.UserID] = true
		}
		if d.OrganizationID != "" && !seenOrgs[d.OrganizationID] {
			orgIDs = append(orgIDs, d.OrganizationID)
			seenOrgs[d.OrganizationID] = true
		}
	}

	ctx, span := donationSvcTracer.Start(ctx, "projectDonationSummaries")
	defer span.End()

	users, err := repo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		span.RecordError(err)
		slog.WarnContext(ctx, "failed to look up donor users", "error", err)
		users = map[string]models.User{}
	}
	orgs, err := repo.GetOrganizationsByIDs(ctx, orgIDs)
	if err != nil {
		span.RecordError(err)
		slog.WarnContext(ctx, "failed to look up donor organizations", "error", err)
		orgs = map[string]models.Organization{}
	}

	summaries := make([]models.DonationSummary, 0, len(donations))
	for _, d := range donations {
		s := models.DonationSummary{
			ID:          d.ID,
			AmountCents: d.CurrentAmountCents,
			Status:      d.Status,
			Category:    d.Category,
			CreatedOn:   d.CreatedOn,
		}
		if d.OrganizationID != "" {
			s.DonorType = donorTypeOrganization
			if org, ok := orgs[d.OrganizationID]; ok {
				s.DonorName = org.Name
				s.DonorAvatarURL = org.AvatarURL
			}
		} else {
			s.DonorType = donorTypeIndividual
			if user, ok := users[d.UserID]; ok {
				s.DonorName = user.Name
				s.DonorAvatarURL = user.AvatarURL
			}
		}
		summaries = append(summaries, s)
	}
	return summaries
}

// ListByUser returns paginated donations for the authenticated user.
func (s *DonationService) ListByUser(ctx context.Context, username string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	ctx, span := donationSvcTracer.Start(ctx, "DonationService.ListByUser")
	defer span.End()
	span.SetAttributes(attribute.String("user.username", username))

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// User has no DB record yet — they have never donated. Return empty list.
			return []models.Donation{}, &models.PaginationMeta{Limit: filter.Limit, Offset: filter.Offset}, nil
		}
		span.RecordError(err)
		return nil, nil, fmt.Errorf("resolve user: %w", err)
	}

	donations, meta, err := s.repo.ListByUser(ctx, user.ID, filter)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list donations by user: %w", err)
	}
	return donations, meta, nil
}

// Create processes a one-time donation with 3DS support.
// When the bank requires an authentication challenge, the returned Donation
// has Status == "requires_action" and ClientSecret set — the frontend must
// call stripe.confirmCardPayment(ClientSecret) to complete the 3DS flow.
// The webhook (payment_intent.succeeded / .payment_failed) advances the status.
func (s *DonationService) Create(ctx context.Context, initiativeID, username string, input models.DonationCreateInput) (*models.Donation, error) {
	ctx, span := donationSvcTracer.Start(ctx, "DonationService.Create")
	defer span.End()
	span.SetAttributes(
		attribute.String("initiative.id", initiativeID),
		attribute.String("user.username", username),
	)

	if input.AmountCents <= 0 {
		return nil, fmt.Errorf("%w: amount_cents must be positive", domain.ErrInvalidInput)
	}
	if input.StripePaymentMethodID == "" {
		return nil, fmt.Errorf("%w: stripe_payment_method_id is required", domain.ErrInvalidInput)
	}
	if input.IdempotencyKey == "" {
		return nil, fmt.Errorf("%w: idempotency_key is required", domain.ErrInvalidInput)
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

	// Resolve the Stripe customer for this user. Requires the user row to
	// already exist — callers must have completed PATCH /me on login (REQ-P4).
	// Email is sourced from the DB row, not from caller-supplied claims (REQ-P5).
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, fmt.Errorf("%w: call PATCH /v1/me to sync your profile before donating", domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("resolve user: %w", err)
	}
	// Guard against legacy/migrated rows that have no email yet.
	// Stripe requires a non-empty email; direct the user to sync their profile.
	if user.Email == "" {
		return nil, fmt.Errorf("%w: email not set — call PATCH /v1/me to sync your profile before donating", domain.ErrProfileNotSynced)
	}
	customerID := user.StripeCustomerID
	if customerID == "" {
		customerID, err = s.stripe.CreateCustomer(ctx, user.ID, user.Email)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("create stripe customer: %w", err)
		}
		if persistErr := s.userRepo.UpdateStripeInfo(ctx, user.ID, customerID, ""); persistErr != nil {
			span.RecordError(persistErr)
			return nil, fmt.Errorf("persist stripe customer: %w", persistErr)
		}
	}

	// Create a PaymentIntent with automatic 3DS.
	// The client-supplied idempotency key is forwarded to Stripe verbatim:
	// if the client retries the same request it sends the same key, Stripe
	// returns the cached response instead of creating a duplicate charge.
	pi, err := s.stripe.CreatePaymentIntent(ctx, models.PaymentIntentRequest{
		InitiativeID:    initiativeID,
		UserID:          user.ID,
		CustomerID:      customerID,
		AmountCents:     input.AmountCents,
		PaymentMethodID: input.StripePaymentMethodID,
		Category:        input.Category,
		OrganizationID:  input.OrganizationID,
		IdempotencyKey:  input.IdempotencyKey,
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe charge: %w", err)
	}

	donation := &models.Donation{
		UserID:                user.ID,
		InitiativeID:          initiativeID,
		OrganizationID:        input.OrganizationID,
		Category:              input.Category,
		CurrentAmountCents:    input.AmountCents,
		PONumber:              input.PONumber,
		PaymentMethod:         input.PaymentMethod,
		Status:                pi.Status,
		StripePaymentIntentID: pi.ID,
		StripeChargeID:        pi.ChargeID,
	}

	created, err := s.repo.Create(ctx, donation)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("record donation: %w", err)
	}

	// Surface client_secret when 3DS challenge is needed — transient, not stored.
	created.ClientSecret = pi.ClientSecret
	return created, nil
}
