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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var announcementSvcTracer = otel.Tracer("announcements-service")

// AnnouncementService orchestrates announcement reads and writes.
type AnnouncementService struct {
	repo           domain.AnnouncementRepository
	initiativeRepo domain.InitiativeRepository
	userRepo       domain.UserRepository
}

// NewAnnouncementService returns an AnnouncementService.
func NewAnnouncementService(
	repo domain.AnnouncementRepository,
	initiativeRepo domain.InitiativeRepository,
	userRepo domain.UserRepository,
) *AnnouncementService {
	return &AnnouncementService{repo: repo, initiativeRepo: initiativeRepo, userRepo: userRepo}
}

// List returns paginated announcements for the given initiative.
// The initiative must exist; no ownership check is applied (public endpoint).
func (s *AnnouncementService) List(ctx context.Context, initiativeID string, filter models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error) {
	ctx, span := announcementSvcTracer.Start(ctx, "AnnouncementService.List")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", initiativeID))

	// Verify the initiative exists and is published. Non-published initiatives
	// must not expose their announcements on the public endpoint; treat them
	// as not-found to avoid leaking unpublished content.
	initiative, err := s.initiativeRepo.GetByID(ctx, initiativeID)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("get initiative: %w", err)
	}
	if !initiative.Status.EqualFold(models.StatusPublished) {
		return nil, nil, domain.ErrInitiativeNotFound
	}

	announcements, meta, err := s.repo.List(ctx, initiativeID, filter)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list announcements: %w", err)
	}
	return announcements, meta, nil
}

// Create adds a new announcement to an initiative owned by callerUsername.
// Returns ErrForbidden when the caller does not own the initiative.
func (s *AnnouncementService) Create(ctx context.Context, initiativeID, callerUsername string, input models.AnnouncementCreateInput) (*models.Announcement, error) {
	ctx, span := announcementSvcTracer.Start(ctx, "AnnouncementService.Create")
	defer span.End()
	span.SetAttributes(
		attribute.String("initiative.id", initiativeID),
		attribute.String("caller.username", callerUsername),
	)

	if input.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}
	if len(input.Title) > 255 {
		return nil, fmt.Errorf("%w: title must be 255 characters or fewer", domain.ErrInvalidInput)
	}
	if input.Description == "" {
		return nil, fmt.Errorf("%w: description is required", domain.ErrInvalidInput)
	}

	if err := s.requireOwnership(ctx, initiativeID, callerUsername); err != nil {
		span.RecordError(err)
		return nil, err
	}

	a := &models.Announcement{
		InitiativeID: initiativeID,
		CreatedBy:    callerUsername,
		Title:        input.Title,
		Description:  input.Description,
	}
	result, err := s.repo.Create(ctx, a)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("create announcement: %w", err)
	}
	return result, nil
}

// Update patches an existing announcement owned by the caller's initiative.
// Returns ErrForbidden when the caller does not own the initiative.
// Returns ErrAnnouncementNotFound when no matching announcement exists.
func (s *AnnouncementService) Update(ctx context.Context, initiativeID, announcementID, callerUsername string, input models.AnnouncementUpdateInput) (*models.Announcement, error) {
	ctx, span := announcementSvcTracer.Start(ctx, "AnnouncementService.Update")
	defer span.End()
	span.SetAttributes(
		attribute.String("initiative.id", initiativeID),
		attribute.String("announcement.id", announcementID),
		attribute.String("caller.username", callerUsername),
	)

	if input.Title == "" {
		return nil, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
	}
	if len(input.Title) > 255 {
		return nil, fmt.Errorf("%w: title must be 255 characters or fewer", domain.ErrInvalidInput)
	}
	if input.Description == "" {
		return nil, fmt.Errorf("%w: description is required", domain.ErrInvalidInput)
	}

	if err := s.requireOwnership(ctx, initiativeID, callerUsername); err != nil {
		span.RecordError(err)
		return nil, err
	}

	result, err := s.repo.Update(ctx, announcementID, initiativeID, input)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	return result, nil
}

// Delete removes an announcement from an initiative owned by the caller.
// Returns ErrForbidden when the caller does not own the initiative.
// Returns ErrAnnouncementNotFound when no matching announcement exists.
func (s *AnnouncementService) Delete(ctx context.Context, initiativeID, announcementID, callerUsername string) error {
	ctx, span := announcementSvcTracer.Start(ctx, "AnnouncementService.Delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("initiative.id", initiativeID),
		attribute.String("announcement.id", announcementID),
		attribute.String("caller.username", callerUsername),
	)

	if err := s.requireOwnership(ctx, initiativeID, callerUsername); err != nil {
		span.RecordError(err)
		return err
	}

	if err := s.repo.Delete(ctx, announcementID, initiativeID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

// requireOwnership verifies that callerUsername owns the given initiative.
// Returns ErrInitiativeNotFound when the initiative does not exist,
// ErrForbidden when the caller is not the owner.
func (s *AnnouncementService) requireOwnership(ctx context.Context, initiativeID, callerUsername string) error {
	caller, err := s.userRepo.GetByUsername(ctx, callerUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrForbidden
		}
		return fmt.Errorf("get user: %w", err)
	}

	initiative, err := s.initiativeRepo.GetByID(ctx, initiativeID)
	if err != nil {
		return fmt.Errorf("get initiative: %w", err)
	}

	if initiative.OwnerID != caller.ID {
		return domain.ErrForbidden
	}
	return nil
}
