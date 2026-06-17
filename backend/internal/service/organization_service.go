// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"fmt"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var orgSvcTracer = otel.Tracer("organizations-service")

// OrganizationService orchestrates organization retrieval and creation.
type OrganizationService struct {
	repo     domain.OrganizationRepository
	userRepo domain.UserRepository
}

// NewOrganizationService returns an OrganizationService.
func NewOrganizationService(repo domain.OrganizationRepository, userRepo domain.UserRepository) *OrganizationService {
	return &OrganizationService{repo: repo, userRepo: userRepo}
}

// ListByOwner returns all organizations owned by the given user (identified by LF SSO username).
func (s *OrganizationService) ListByOwner(ctx context.Context, username string) ([]models.Organization, error) {
	ctx, span := orgSvcTracer.Start(ctx, "OrganizationService.ListByOwner")
	defer span.End()
	span.SetAttributes(attribute.String("owner.username", username))

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	orgs, err := s.repo.ListByOwner(ctx, user.ID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	return orgs, nil
}

// Update patches name and avatar_url for the given organization, verifying the caller owns it.
func (s *OrganizationService) Update(ctx context.Context, username string, id string, input models.OrganizationUpdateInput) (*models.Organization, error) {
	ctx, span := orgSvcTracer.Start(ctx, "OrganizationService.Update")
	defer span.End()
	span.SetAttributes(attribute.String("owner.username", username), attribute.String("org.id", id))

	if input.Name == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	org, err := s.repo.Update(ctx, id, user.ID, input)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update organization: %w", err)
	}
	return org, nil
}

// Create inserts a new organization owned by the given user (identified by LF SSO username).
func (s *OrganizationService) Create(ctx context.Context, username string, input models.OrganizationCreateInput) (*models.Organization, error) {
	ctx, span := orgSvcTracer.Start(ctx, "OrganizationService.Create")
	defer span.End()
	span.SetAttributes(attribute.String("owner.username", username))

	if input.Name == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	org, err := s.repo.Create(ctx, user.ID, input)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create organization: %w", err)
	}
	return org, nil
}
