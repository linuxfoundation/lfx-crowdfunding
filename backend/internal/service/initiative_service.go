// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service contains the orchestration layer for the initiatives domain.
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

var initiativeSvcTracer = otel.Tracer("initiatives-service")

// InitiativeService orchestrates initiative reads and writes, enriching with Ledger balance data.
type InitiativeService struct {
	repo   domain.InitiativeRepository
	ledger clients.LedgerClient
	stripe clients.StripeClient
}

// NewInitiativeService returns an InitiativeService.
func NewInitiativeService(
	repo domain.InitiativeRepository,
	ledger clients.LedgerClient,
	stripe clients.StripeClient,
) *InitiativeService {
	return &InitiativeService{repo: repo, ledger: ledger, stripe: stripe}
}

// GetByID retrieves an initiative enriched with goals and live Ledger balance.
// Balance fetch failure is non-fatal — returns zero balance and logs via span.
func (s *InitiativeService) GetByID(ctx context.Context, id string) (*models.InitiativeDetail, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetByID")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", id))

	initiative, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get initiative: %w", err)
	}

	goals, err := s.repo.ListGoals(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list goals: %w", err)
	}

	// Balance is computed at read time — never stored in PostgreSQL.
	balance, err := s.ledger.GetBalance(ctx, id)
	if err != nil {
		span.RecordError(err) // non-fatal: degraded response with zero balance
		balance = &models.Balance{InitiativeID: id}
	}

	return &models.InitiativeDetail{
		Initiative: *initiative,
		Goals:      goals,
		Balance:    balance,
	}, nil
}

// GetBySlug retrieves an initiative by its URL slug.
func (s *InitiativeService) GetBySlug(ctx context.Context, slug string) (*models.InitiativeDetail, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetBySlug")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.slug", slug))

	initiative, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get initiative by slug: %w", err)
	}
	return s.GetByID(ctx, initiative.ID)
}

// List retrieves a filtered, paginated list of initiatives.
func (s *InitiativeService) List(ctx context.Context, filter models.InitiativeFilter) ([]models.Initiative, *models.PaginationMeta, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.List")
	defer span.End()

	initiatives, meta, err := s.repo.List(ctx, filter)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list initiatives: %w", err)
	}
	return initiatives, meta, nil
}

// Create creates a new initiative owned by the given principal.
func (s *InitiativeService) Create(ctx context.Context, ownerID string, input models.InitiativeCreateInput) (*models.Initiative, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.Create")
	defer span.End()

	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	if input.InitiativeType == "" {
		return nil, fmt.Errorf("%w: initiative_type is required", domain.ErrInvalidInput)
	}

	initiative := &models.Initiative{
		InitiativeType: input.InitiativeType,
		OwnerID:        ownerID,
		Name:           input.Name,
		Slug:           input.Slug,
		Description:    input.Description,
		Industry:       input.Industry,
		Color:          input.Color,
		LogoURL:        input.LogoURL,
		WebsiteURL:     input.WebsiteURL,
		CocURL:         input.CocURL,
		AcceptFunding:  input.AcceptFunding,
		Status:         "submitted",
	}

	created, err := s.repo.Create(ctx, initiative)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create initiative: %w", err)
	}
	return created, nil
}

// Update applies changes to an existing initiative, enforcing owner authorisation.
func (s *InitiativeService) Update(ctx context.Context, id, callerID string, input models.InitiativeUpdateInput) (*models.Initiative, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.Update")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", id))

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	if existing.OwnerID != callerID {
		return nil, domain.ErrForbidden
	}

	// Apply partial updates
	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Slug != nil {
		existing.Slug = *input.Slug
	}
	if input.Status != nil {
		existing.Status = *input.Status
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.Industry != nil {
		existing.Industry = *input.Industry
	}
	if input.Color != nil {
		existing.Color = *input.Color
	}
	if input.LogoURL != nil {
		existing.LogoURL = *input.LogoURL
	}
	if input.WebsiteURL != nil {
		existing.WebsiteURL = *input.WebsiteURL
	}
	if input.CocURL != nil {
		existing.CocURL = *input.CocURL
	}
	if input.AcceptFunding != nil {
		existing.AcceptFunding = *input.AcceptFunding
	}

	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update initiative: %w", err)
	}
	return updated, nil
}

// Delete removes an initiative, enforcing owner authorisation.
func (s *InitiativeService) Delete(ctx context.Context, id, callerID string) error {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.Delete")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", id))

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if existing.OwnerID != callerID {
		return domain.ErrForbidden
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete initiative: %w", err)
	}
	return nil
}
