// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service contains the orchestration layer for the initiatives domain.
package service

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var initiativeSvcTracer = otel.Tracer("initiatives-service")

// InitiativeService orchestrates initiative reads and writes.
// Cached financials come from initiative_ledger_stats (CronJob); per-goal
// donated/spent is enriched live from Ledger GetBalance on each detail request.
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

// GetByID retrieves an initiative with goals, financials, and sponsors.
// Per-goal donated/spent is enriched from a live Ledger balance call; Ledger
// unavailability is non-fatal — goals are returned with zero donated/spent.
func (s *InitiativeService) GetByID(ctx context.Context, id string) (*models.Initiative, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetByID")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", id))

	initiative, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get initiative: %w", err)
	}

	initiative.Sponsors = flattenSponsors(initiative.RawSponsors)
	enrichGoalsFromLedger(ctx, s.ledger, initiative)
	return initiative, nil
}

// flattenSponsors merges orgs and individuals from the cached sponsor list into a
// single flat slice sorted by total descending.
func flattenSponsors(list models.LedgerSponsorList) []models.Sponsor {
	sponsors := make([]models.Sponsor, 0, len(list.Orgs)+len(list.Individuals))
	for _, o := range list.Orgs {
		sponsors = append(sponsors, models.Sponsor{ID: o.ID, Name: o.Name, AvatarURL: o.AvatarURL, TotalCents: o.Total})
	}
	for _, u := range list.Individuals {
		sponsors = append(sponsors, models.Sponsor{ID: u.ID, Name: u.Name, AvatarURL: u.AvatarURL, TotalCents: u.Total})
	}
	slices.SortFunc(sponsors, func(a, b models.Sponsor) int {
		return cmp.Compare(b.TotalCents, a.TotalCents) // descending
	})
	return sponsors
}

// GetBySlug retrieves an initiative by its URL slug, with the same enrichment as GetByID.
func (s *InitiativeService) GetBySlug(ctx context.Context, slug string) (*models.Initiative, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetBySlug")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.slug", slug))

	initiative, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get initiative by slug: %w", err)
	}

	initiative.Sponsors = flattenSponsors(initiative.RawSponsors)
	enrichGoalsFromLedger(ctx, s.ledger, initiative)
	return initiative, nil
}

// enrichGoalsFromLedger populates donated_cents/spent_cents on each goal by
// matching the goal name (case-insensitive) against Ledger subTotal categories.
// Ledger uses PascalCase keys ("Mentorship", "BugBounty"); our goal names are
// lowercase ("mentorship", "bugbounty"). Errors are non-fatal — goals keep zero values.
func enrichGoalsFromLedger(ctx context.Context, ledger clients.LedgerClient, initiative *models.Initiative) {
	if len(initiative.Goals) == 0 {
		return
	}
	balance, err := ledger.GetBalance(ctx, initiative.ID)
	if err != nil || len(balance.SubTotals) == 0 {
		return
	}
	// Build a normalised lookup: lowercase(category) → subTotal
	lookup := make(map[string]*clients.LedgerSubTotal, len(balance.SubTotals))
	for k, v := range balance.SubTotals {
		lookup[strings.ToLower(k)] = v
	}
	for i := range initiative.Goals {
		key := strings.ToLower(strings.ReplaceAll(initiative.Goals[i].Name, "_", ""))
		if sub, ok := lookup[key]; ok {
			donated := sub.Credit
			spent := sub.Debit
			initiative.Goals[i].DonatedCents = &donated
			initiative.Goals[i].SpentCents = &spent
		}
	}
}

// List retrieves a filtered, paginated list of initiatives.
func (s *InitiativeService) List(ctx context.Context, filter models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
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
	if !models.ValidInitiativeTypes[input.InitiativeType] {
		return nil, fmt.Errorf("%w: unknown initiative_type %q", domain.ErrInvalidInput, input.InitiativeType)
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

	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Slug != nil {
		existing.Slug = *input.Slug
	}
	if input.Status != nil {
		if !models.ValidInitiativeStatuses[*input.Status] {
			return nil, fmt.Errorf("%w: unknown status %q", domain.ErrInvalidInput, *input.Status)
		}
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

// GetTransactions proxies a paginated transaction request to the Ledger service.
// The initiative must exist; its ID is looked up by slug or UUID before calling Ledger.
func (s *InitiativeService) GetTransactions(ctx context.Context, initiativeID, txnType string, size, from int) (*models.TransactionList, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetTransactions")
	defer span.End()

	return s.ledger.GetTransactions(ctx, clients.TransactionFilter{
		ProjectID: initiativeID,
		TxnType:   txnType,
		Size:      size,
		From:      from,
	})
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
