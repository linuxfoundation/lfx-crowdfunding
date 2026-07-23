// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service contains the orchestration layer for the initiatives domain.
package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var initiativeSvcTracer = otel.Tracer("initiatives-service")

// allowedContactTypes is the exhaustive set of valid contact_type values.
var allowedContactTypes = map[string]struct{}{
	"primary":        {},
	"secondary":      {},
	"technical_lead": {},
}

// InitiativeService orchestrates initiative reads and writes.
// Cached financials come from initiative_ledger_stats (CronJob); per-goal
// donated/spent is enriched live from Ledger GetBalance on each detail request.
type InitiativeService struct {
	repo          domain.InitiativeRepository
	userRepo      domain.UserRepository
	ledger        clients.LedgerClient
	stripe        clients.StripeClient
	emailService  domain.EmailService
	reimbursement clients.ReimbursementClient // nil when RS integration is disabled
	logger        *slog.Logger
}

// NewInitiativeService returns an InitiativeService.
func NewInitiativeService(
	repo domain.InitiativeRepository,
	userRepo domain.UserRepository,
	ledger clients.LedgerClient,
	stripe clients.StripeClient,
	emailService domain.EmailService,
	reimbursement clients.ReimbursementClient,
	logger *slog.Logger,
) *InitiativeService {
	return &InitiativeService{
		repo:          repo,
		userRepo:      userRepo,
		ledger:        ledger,
		stripe:        stripe,
		emailService:  emailService,
		reimbursement: reimbursement,
		logger:        logger,
	}
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
// single flat slice sorted by total descending.  Entries with a non-positive
// total are expense-payout recipients, not donors, and are excluded.
func flattenSponsors(list models.LedgerSponsorList) []models.Sponsor {
	sponsors := make([]models.Sponsor, 0, len(list.Orgs)+len(list.Individuals))
	for _, o := range list.Orgs {
		if o.Total <= 0 {
			continue
		}
		avatarURL := o.AvatarURL
		if avatarURL == "" {
			avatarURL = generatedAvatarURL(o.ID, o.Name)
		}
		sponsors = append(sponsors, models.Sponsor{ID: o.ID, Name: o.Name, AvatarURL: avatarURL, TotalCents: o.Total})
	}
	for _, u := range list.Individuals {
		if u.Total <= 0 {
			continue
		}
		avatarURL := u.AvatarURL
		if avatarURL == "" {
			avatarURL = generatedAvatarURL(u.ID, u.Name)
		}
		sponsors = append(sponsors, models.Sponsor{ID: u.ID, Name: u.Name, AvatarURL: avatarURL, TotalCents: u.Total})
	}
	slices.SortFunc(sponsors, func(a, b models.Sponsor) int {
		return cmp.Compare(b.TotalCents, a.TotalCents) // descending
	})
	return sponsors
}

// CheckPublishedByID verifies that a UUID identifies a published initiative.
// It does not trigger Ledger enrichment — use instead of GetByID when only
// status validation is needed (e.g. the transactions handler).
func (s *InitiativeService) CheckPublishedByID(ctx context.Context, id string) error {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.CheckPublishedByID")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", id))

	initiative, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("get initiative: %w", err)
	}
	if !initiative.Status.EqualFold(models.StatusPublished) {
		return domain.ErrInitiativeNotFound
	}
	return nil
}

// GetIDBySlug returns only the UUID for the given slug (published initiatives only).
// Used by handlers that need to resolve a public slug without triggering Ledger enrichment.
func (s *InitiativeService) GetIDBySlug(ctx context.Context, slug string) (string, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetIDBySlug")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.slug", slug))

	id, err := s.repo.GetIDBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("get id by slug: %w", err)
	}
	return id, nil
}

// ResolveSlug returns the UUID for the given slug regardless of status.
// Used by admin flows (e.g. approval) where the initiative may not yet be published.
func (s *InitiativeService) ResolveSlug(ctx context.Context, slug string) (string, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.ResolveSlug")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.slug", slug))

	id, err := s.repo.ResolveSlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("resolve slug: %w", err)
	}
	return id, nil
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

// GetOwnerInfoBySlug returns the email and display name of the owner of the initiative
// identified by slug. It uses a single JOIN query — no status filter is applied,
// so it works for initiatives in any status. Intended for M2M callers only.
func (s *InitiativeService) GetOwnerInfoBySlug(ctx context.Context, slug string) (models.OwnerInfo, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetOwnerInfoBySlug")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.slug", slug))

	info, err := s.repo.GetOwnerInfoBySlug(ctx, slug)
	if err != nil {
		if !errors.Is(err, domain.ErrInitiativeNotFound) {
			span.RecordError(err)
		}
		if errors.Is(err, domain.ErrProfileNotSynced) {
			return models.OwnerInfo{}, fmt.Errorf("%w: initiative owner has no email address set", domain.ErrProfileNotSynced)
		}
		return models.OwnerInfo{}, fmt.Errorf("get owner info by slug: %w", err)
	}
	return info, nil
}

// ListPublished returns the ID and Name of every published initiative.
// Intended for M2M callers (e.g. Reimbursement Service initiative picker).
func (s *InitiativeService) ListPublished(ctx context.Context) ([]models.InitiativeSummary, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.ListPublished")
	defer span.End()

	results, err := s.repo.ListPublished(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("list published initiatives: %w", err)
	}
	return results, nil
}

// GetForUser retrieves an initiative owned by the authenticated caller, by slug or
// UUID, regardless of its status. The public GetByID/GetBySlug path only exposes
// published initiatives to non-approvers, so owners need this identity-scoped read
// to open their own drafts/submitted initiatives from the "My Initiatives" list.
func (s *InitiativeService) GetForUser(ctx context.Context, idOrSlug, callerUsername string) (*models.Initiative, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetForUser")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id_or_slug", idOrSlug))

	caller, err := s.userRepo.GetByUsername(ctx, callerUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Unknown caller cannot own any initiative.
			return nil, domain.ErrInitiativeNotFound
		}
		span.RecordError(err)
		return nil, err
	}

	var initiative *models.Initiative
	if _, parseErr := uuid.Parse(idOrSlug); parseErr == nil {
		initiative, err = s.repo.GetByID(ctx, idOrSlug)
	} else {
		initiative, err = s.repo.GetBySlug(ctx, idOrSlug)
	}
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if initiative.OwnerID != caller.ID {
		// Do not leak existence of initiatives the caller does not own.
		return nil, domain.ErrInitiativeNotFound
	}

	initiative.Sponsors = flattenSponsors(initiative.RawSponsors)
	enrichGoalsFromLedger(ctx, s.ledger, initiative)
	return initiative, nil
}

// ResolveOwnedInitiativeID resolves a slug or UUID to the initiative's UUID, but
// only if the initiative is owned by the authenticated caller — regardless of its
// status. Mirrors GetForUser's ownership semantics for the transactions endpoint,
// where the public path resolves published-only. Returns ErrInitiativeNotFound for
// unknown callers, missing initiatives, or initiatives the caller does not own.
func (s *InitiativeService) ResolveOwnedInitiativeID(ctx context.Context, idOrSlug, callerUsername string) (string, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.ResolveOwnedInitiativeID")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id_or_slug", idOrSlug))

	caller, err := s.userRepo.GetByUsername(ctx, callerUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInitiativeNotFound
		}
		span.RecordError(err)
		return "", err
	}

	var initiativeID, ownerID string
	if _, parseErr := uuid.Parse(idOrSlug); parseErr == nil {
		initiative, gErr := s.repo.GetByID(ctx, idOrSlug)
		if gErr != nil {
			span.RecordError(gErr)
			return "", gErr
		}
		initiativeID, ownerID = initiative.ID, initiative.OwnerID
	} else {
		// Resolve the slug to a UUID regardless of status, then read the owner cheaply.
		id, rErr := s.repo.ResolveSlug(ctx, idOrSlug)
		if rErr != nil {
			span.RecordError(rErr)
			return "", rErr
		}
		initiative, gErr := s.repo.GetByID(ctx, id)
		if gErr != nil {
			span.RecordError(gErr)
			return "", gErr
		}
		initiativeID, ownerID = initiative.ID, initiative.OwnerID
	}

	if ownerID != caller.ID {
		return "", domain.ErrInitiativeNotFound
	}
	return initiativeID, nil
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
	// Short-circuit: balance is only dereferenced after the nil err check passes.
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
			spent := -sub.Debit // Debit is negative in Ledger; normalize to positive
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
	for _, i := range initiatives {
		i.Sponsors = flattenSponsors(i.RawSponsors)
	}
	return initiatives, meta, nil
}

// ListForUser retrieves initiatives owned by the authenticated caller.
func (s *InitiativeService) ListForUser(ctx context.Context, ownerUsername string, filter models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.ListForUser")
	defer span.End()

	owner, err := s.userRepo.GetByUsername(ctx, ownerUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			limit := filter.Limit
			if limit <= 0 || limit > 100 {
				limit = 20
			}
			offset := filter.Offset
			if offset < 0 {
				offset = 0
			}
			return []*models.Initiative{}, &models.PaginationMeta{Limit: limit, Offset: offset}, nil
		}
		span.RecordError(err)
		return nil, nil, fmt.Errorf("resolve owner: %w", err)
	}

	filter.OwnerID = owner.ID
	initiatives, meta, err := s.repo.List(ctx, filter)
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("list initiatives for user: %w", err)
	}
	for _, i := range initiatives {
		i.Sponsors = flattenSponsors(i.RawSponsors)
	}
	return initiatives, meta, nil
}

// Create creates a new initiative owned by the given principal.
func (s *InitiativeService) Create(ctx context.Context, ownerUsername string, input models.InitiativeCreateInput) (*models.Initiative, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.Create")
	defer span.End()

	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	if utf8.RuneCountInString(input.Description) > 1500 {
		return nil, fmt.Errorf("%w: description must be 1500 characters or fewer", domain.ErrInvalidInput)
	}
	if input.Slug == "" {
		input.Slug = slug.Make(input.Name)
	}
	if input.InitiativeType == "" {
		return nil, fmt.Errorf("%w: initiative_type is required", domain.ErrInvalidInput)
	}
	if !models.ValidInitiativeTypes[input.InitiativeType] {
		return nil, fmt.Errorf("%w: unknown initiative_type %q", domain.ErrInvalidInput, input.InitiativeType)
	}

	// Validate required child-record fields early to produce clear errors before
	// any Stripe or DB calls are made.
	seenGoalNames := make(map[string]struct{}, len(input.Goals))
	for idx, g := range input.Goals {
		if g.Name == "" {
			return nil, fmt.Errorf("%w: goals[%d]: name is required", domain.ErrInvalidInput, idx)
		}
		if _, dup := seenGoalNames[g.Name]; dup {
			return nil, fmt.Errorf("%w: goals[%d]: duplicate goal name %q", domain.ErrInvalidInput, idx, g.Name)
		}
		seenGoalNames[g.Name] = struct{}{}
	}
	for idx, w := range input.CustomWebsites {
		if w.URL == "" {
			return nil, fmt.Errorf("%w: custom_websites[%d]: url is required", domain.ErrInvalidInput, idx)
		}
	}
	seenContactTypes := make(map[string]struct{}, len(input.Contacts))
	for idx, c := range input.Contacts {
		if _, ok := allowedContactTypes[c.ContactType]; !ok {
			return nil, fmt.Errorf("%w: contacts[%d]: contact_type %q must be one of primary, secondary, technical_lead", domain.ErrInvalidInput, idx, c.ContactType)
		}
		if _, dup := seenContactTypes[c.ContactType]; dup {
			return nil, fmt.Errorf("%w: contacts[%d]: duplicate contact_type %q (at most one per type)", domain.ErrInvalidInput, idx, c.ContactType)
		}
		seenContactTypes[c.ContactType] = struct{}{}
	}

	// Validate and default donation_mode.
	if input.DonationMode == "" {
		input.DonationMode = models.DonationModeOpen
	}
	input.DonationMode = models.DonationMode(strings.ToLower(string(input.DonationMode)))
	if !input.DonationMode.IsValid() {
		return nil, fmt.Errorf("%w: invalid donation_mode %q", domain.ErrInvalidInput, input.DonationMode)
	}
	if input.DonationMode == models.DonationModeTiers {
		for idx, t := range input.SponsorshipTiers {
			if t.Name != "" && !models.ValidTierNames[t.Name] {
				return nil, fmt.Errorf("%w: sponsorship_tiers[%d]: invalid tier name %q", domain.ErrInvalidInput, idx, t.Name)
			}
			// Default nil Enabled to true — omitting the field means "enabled".
			if input.SponsorshipTiers[idx].Enabled == nil {
				enabled := true
				input.SponsorshipTiers[idx].Enabled = &enabled
			}
			cleaned := []string{}
			for _, b := range t.Benefits {
				if strings.TrimSpace(b) != "" {
					cleaned = append(cleaned, b)
				}
			}
			input.SponsorshipTiers[idx].Benefits = cleaned
		}
	} else {
		// Open mode: ignore any tiers the caller may have sent.
		input.SponsorshipTiers = nil
	}

	// Pre-generate the UUID so the same ID is embedded in both the Stripe
	// Product metadata and the DB INSERT — no follow-up UPDATE needed.
	initiativeID := uuid.New().String()
	span.SetAttributes(attribute.String("initiative.id", initiativeID))

	// Resolve owner before creating any external resources so we fail fast
	// with a clean error when the user does not exist.
	owner, err := s.userRepo.GetByUsername(ctx, ownerUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrForbidden
		}
		span.RecordError(err)
		return nil, fmt.Errorf("get owner: %w", err)
	}

	// Create the Stripe Product first. If Stripe is unavailable, the whole
	// creation fails cleanly and no DB row is created.
	productID, err := s.stripe.CreateProduct(ctx, initiativeID, input.Name)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("stripe product: %w", err)
	}

	initiative := &models.Initiative{
		ID:              initiativeID,
		InitiativeType:  input.InitiativeType,
		OwnerID:         owner.ID,
		Name:            input.Name,
		Slug:            input.Slug,
		Description:     input.Description,
		Industry:        input.Industry,
		Color:           input.Color,
		LogoURL:         input.LogoURL,
		WebsiteURL:      input.WebsiteURL,
		CocURL:          input.CocURL,
		AcceptFunding:   input.AcceptFunding,
		Status:          models.StatusSubmitted,
		StripeProductID: productID,
		CiiProjectID:    input.CiiProjectID,
		DonationMode:    input.DonationMode,

		// Entity-only display fields
		EventbriteURL:  input.EventbriteURL,
		ApplicationURL: input.ApplicationURL,
		EventStartDate: input.EventStartDate,
		EventEndDate:   input.EventEndDate,
		Country:        input.Country,
		City:           input.City,
		IsOnline:       input.IsOnline,
	}

	created, err := s.repo.Create(ctx, initiative, input)
	if err != nil {
		span.RecordError(err)
		// Compensating transaction: remove the Stripe Product so Stripe stays in sync.
		// Use a detached context so the cleanup runs even if the request context is cancelled.
		if delErr := s.stripe.DeleteProduct(context.WithoutCancel(ctx), productID); delErr != nil {
			s.logger.Warn("failed to roll back Stripe product after DB insert failure",
				"product_id", productID, "error", delErr)
		}
		return nil, fmt.Errorf("create initiative: %w", err)
	}

	// Notify reviewers that a new initiative has been submitted. Non-fatal.
	// owner is already resolved above — use it directly.
	displayName := owner.Name
	if displayName == "" {
		displayName = owner.Email
	}
	initiativeURL := s.emailService.InitiativeURL(created.Slug)
	approveURL := s.emailService.InitiativeURL(created.Slug) + "/process-approval/approve"
	declineURL := s.emailService.InitiativeURL(created.Slug) + "/process-approval/decline"
	if emailErr := s.emailService.SendProjectForReviewEmail(ctx, displayName, owner.Email, created.Name, initiativeURL, approveURL, declineURL); emailErr != nil {
		s.logger.WarnContext(ctx, "initiative create: failed to send for-review notification",
			"initiative_id", created.ID, "error", emailErr)
	}
	return created, nil
}

// Update applies changes to an existing initiative, enforcing owner authorisation.
func (s *InitiativeService) Update(ctx context.Context, id, callerUsername string, input models.InitiativeUpdateInput) (*models.Initiative, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.Update")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", id))

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	caller, err := s.userRepo.GetByUsername(ctx, callerUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Unknown caller cannot own any initiative.
			return nil, domain.ErrForbidden
		}
		span.RecordError(err)
		return nil, err
	}
	if existing.OwnerID != caller.ID {
		return nil, domain.ErrForbidden
	}

	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Slug != nil {
		existing.Slug = *input.Slug
	}
	if input.Status != nil {
		if err := validateOwnerStatusTransition(existing.Status, *input.Status); err != nil {
			return nil, err
		}
		existing.Status = *input.Status
	}
	if input.Description != nil {
		if utf8.RuneCountInString(*input.Description) > 1500 {
			return nil, fmt.Errorf("%w: description must be 1500 characters or fewer", domain.ErrInvalidInput)
		}
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
	if input.CiiProjectID != nil {
		existing.CiiProjectID = *input.CiiProjectID
	}

	if input.EventbriteURL != nil {
		existing.EventbriteURL = *input.EventbriteURL
	}
	if input.ApplicationURL != nil {
		existing.ApplicationURL = *input.ApplicationURL
	}
	if input.EventStartDate != nil {
		existing.EventStartDate = input.EventStartDate
	}
	if input.EventEndDate != nil {
		existing.EventEndDate = input.EventEndDate
	}
	if input.Country != nil {
		existing.Country = *input.Country
	}
	if input.City != nil {
		existing.City = *input.City
	}
	if input.IsOnline != nil {
		existing.IsOnline = *input.IsOnline
	}
	if input.DonationMode != nil {
		normalizedMode := models.DonationMode(strings.ToLower(string(*input.DonationMode)))
		if !normalizedMode.IsValid() {
			return nil, fmt.Errorf("%w: invalid donation_mode %q", domain.ErrInvalidInput, *input.DonationMode)
		}
		existing.DonationMode = normalizedMode
	}
	// Open-mode invariant: tiers must never be written when mode is open.
	// Always force-clear when mode is open, regardless of whether it just changed
	// or was already open — this also discards tiers sent by accident.
	if existing.DonationMode == models.DonationModeOpen {
		input.SponsorshipTiers = []models.SponsorshipTierInput{}
	}

	// Validate required child-record fields before any DB calls.
	seenGoalNames := make(map[string]struct{}, len(input.Goals))
	for idx, g := range input.Goals {
		if g.Name == "" {
			return nil, fmt.Errorf("%w: goals[%d]: name is required", domain.ErrInvalidInput, idx)
		}
		if _, dup := seenGoalNames[g.Name]; dup {
			return nil, fmt.Errorf("%w: goals[%d]: duplicate goal name %q", domain.ErrInvalidInput, idx, g.Name)
		}
		seenGoalNames[g.Name] = struct{}{}
	}
	for idx, w := range input.CustomWebsites {
		if w.URL == "" {
			return nil, fmt.Errorf("%w: custom_websites[%d]: url is required", domain.ErrInvalidInput, idx)
		}
	}
	seenContactTypes := make(map[string]struct{}, len(input.Contacts))
	for idx, c := range input.Contacts {
		if _, ok := allowedContactTypes[c.ContactType]; !ok {
			return nil, fmt.Errorf("%w: contacts[%d]: contact_type %q must be one of primary, secondary, technical_lead", domain.ErrInvalidInput, idx, c.ContactType)
		}
		if _, dup := seenContactTypes[c.ContactType]; dup {
			return nil, fmt.Errorf("%w: contacts[%d]: duplicate contact_type %q (at most one per type)", domain.ErrInvalidInput, idx, c.ContactType)
		}
		seenContactTypes[c.ContactType] = struct{}{}
	}
	if existing.DonationMode == models.DonationModeTiers {
		for idx, t := range input.SponsorshipTiers {
			if t.Name != "" && !models.ValidTierNames[t.Name] {
				return nil, fmt.Errorf("%w: sponsorship_tiers[%d]: invalid tier name %q", domain.ErrInvalidInput, idx, t.Name)
			}
			// Default nil Enabled to true — omitting the field means "enabled".
			if input.SponsorshipTiers[idx].Enabled == nil {
				enabled := true
				input.SponsorshipTiers[idx].Enabled = &enabled
			}
			cleaned := []string{}
			for _, b := range t.Benefits {
				if strings.TrimSpace(b) != "" {
					cleaned = append(cleaned, b)
				}
			}
			input.SponsorshipTiers[idx].Benefits = cleaned
		}
	}

	updated, err := s.repo.Update(ctx, existing, input)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update initiative: %w", err)
	}
	// Sync beneficiaries and policy with the Reimbursement Service.
	// Non-fatal; only takes effect when the initiative is published.
	s.syncReimbursementPolicy(ctx, updated)
	return updated, nil
}

// ProcessApproval updates an initiative's status based on the given approval action.
// ApprovalActionApprove transitions the initiative to StatusPublished;
// ApprovalActionDecline transitions it to StatusDeclined.
func (s *InitiativeService) ProcessApproval(ctx context.Context, initiativeID string, action models.InitiativeApprovalAction) (*models.Initiative, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.ProcessApproval")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", initiativeID))

	// Validate action at the service boundary so the method is self-contained.
	if _, err := models.ParseApprovalAction(string(action)); err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err)
	}

	initiative, err := s.repo.GetByID(ctx, initiativeID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Guard: only initiatives in a reviewable state can be approved or declined.
	// Use EqualFold to handle any casing stored by earlier writes.
	if !initiative.Status.EqualFold(models.StatusSubmitted) && !initiative.Status.EqualFold(models.StatusPending) {
		return nil, fmt.Errorf("%w: initiative with status %q cannot be approved or declined",
			domain.ErrInvalidInput, initiative.Status)
	}

	switch action {
	case models.ApprovalActionApprove:
		initiative.Status = models.StatusPublished
	case models.ApprovalActionDecline:
		initiative.Status = models.StatusDeclined
	}

	processed, err := s.repo.Update(ctx, initiative, models.InitiativeUpdateInput{})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("update initiative: %w", err)
	}

	// Notify the initiative owner by email. Errors are non-fatal — log at WARN and continue.
	owner, ownerErr := s.userRepo.GetByID(ctx, processed.OwnerID)
	if ownerErr != nil {
		s.logger.WarnContext(ctx, "initiative approval: could not fetch owner for email notification",
			"initiative_id", initiativeID, "owner_id", processed.OwnerID, "error", ownerErr)
	} else if owner != nil {
		displayName := owner.Name
		if displayName == "" {
			displayName = owner.Email
		}
		initiativeURL := s.emailService.InitiativeURL(processed.Slug)
		switch action {
		case models.ApprovalActionApprove:
			if emailErr := s.emailService.SendProjectApprovedEmail(ctx, owner.Email, displayName, processed.Name, initiativeURL); emailErr != nil {
				s.logger.WarnContext(ctx, "initiative approval: failed to send approved email",
					"initiative_id", initiativeID, "error", emailErr)
			}
		case models.ApprovalActionDecline:
			if emailErr := s.emailService.SendProjectDeclinedEmail(ctx, owner.Email, displayName, processed.Name, initiativeURL); emailErr != nil {
				s.logger.WarnContext(ctx, "initiative approval: failed to send declined email",
					"initiative_id", initiativeID, "error", emailErr)
			}
		}
	}

	// On approval the initiative is now published — sync to the Reimbursement
	// Service so beneficiaries are added to the Expensify policy immediately.
	// Not called on decline: syncReimbursementPolicy's published guard would be a
	// no-op, but calling it is misleading at the call site.
	if action == models.ApprovalActionApprove {
		s.syncReimbursementPolicy(ctx, processed)
	}
	return processed, nil
}

// validateOwnerStatusTransition validates status changes requested through owner
// update flows. The permitted transitions are:
//
//	submitted  → pending | declined
//	pending    → declined
//	published  → hidden
//	hidden     → published
func validateOwnerStatusTransition(from, to models.InitiativeStatus) error {
	if !to.IsValid() {
		return fmt.Errorf("%w: unknown status %q", domain.ErrInvalidInput, to)
	}

	allowed := map[models.InitiativeStatus][]models.InitiativeStatus{
		models.StatusSubmitted: {models.StatusPending, models.StatusDeclined},
		models.StatusPending:   {models.StatusDeclined},
		models.StatusPublished: {models.StatusHidden},
		models.StatusHidden:    {models.StatusPublished},
	}
	for _, permitted := range allowed[from] {
		if to.EqualFold(permitted) {
			return nil
		}
	}

	return fmt.Errorf("%w: status transition %q -> %q is not permitted for owners",
		domain.ErrForbidden, from, to)
}

// GetMyTransactions fetches transactions for the given initiative that belong to the
// specified user (identified by their Auth0 subject / legacy_user_id). The userID
// filter is forwarded to the Ledger API. If the Ledger returns rows that belong to
// other users the param was ignored server-side — the request fails rather than
// returning incorrect pagination metadata. The same negative-donation post-processing
// applied by GetTransactions is also applied here.
func (s *InitiativeService) GetMyTransactions(ctx context.Context, initiativeID, userID, txnType string, subscriptionOnly bool, limit, offset int) (*models.TransactionList, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetMyTransactions")
	defer span.End()

	list, err := s.ledger.GetTransactions(ctx, clients.TransactionFilter{
		ProjectID:        initiativeID,
		TxnType:          txnType,
		UserID:           userID,
		SubscriptionOnly: subscriptionOnly,
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		return nil, err
	}

	// Detect whether the Ledger API applied the userID filter server-side.
	// If any returned row belongs to a different user the param was ignored and
	// we cannot produce a valid TotalCount — fail rather than expose another
	// user's contributions or return misleading pagination metadata.
	for _, t := range list.Data {
		if t.LedgerUserID != userID {
			span.RecordError(fmt.Errorf("ledger returned foreign rows for userID filter"))
			return nil, fmt.Errorf(
				"ledger returned rows for other users (server-side userID filtering unavailable): %w",
				domain.ErrUpstreamUnavailable,
			)
		}
	}

	// The Ledger stores some grant disbursements as credit-type rows with
	// negative amounts. Apply the same post-processing as GetTransactions so
	// both endpoints have identical transaction-type semantics.
	if txnType == "donation" {
		fullPageLen := len(list.Data)
		hasMorePages := list.TotalCount > offset+fullPageLen

		kept := list.Data[:0]
		for _, t := range list.Data {
			if t.AmountCents > 0 {
				kept = append(kept, t)
			}
		}
		dropped := len(list.Data) - len(kept)
		list.Data = kept

		adjusted := list.TotalCount - dropped
		minTotal := offset + len(kept)
		if hasMorePages && len(kept) == 0 {
			minTotal = offset + limit + 1
		}
		if adjusted < minTotal {
			adjusted = minTotal
		}
		list.TotalCount = adjusted
	}

	enrichTransactionsFromDB(ctx, s.repo, list.Data)
	if initiative, err := s.repo.GetByID(ctx, initiativeID); err == nil && initiative != nil {
		list.InitiativeName = initiative.Name
	}
	return list, nil
}

// GetTransactions fetches transactions from Ledger and enriches each with donor
// name and avatar from the CF DB (users / organizations tables).
// When no CF DB record matches, a generated avatar URL is returned as fallback.
func (s *InitiativeService) GetTransactions(ctx context.Context, initiativeID, txnType string, subscriptionOnly bool, limit, offset int) (*models.TransactionList, error) {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.GetTransactions")
	defer span.End()

	list, err := s.ledger.GetTransactions(ctx, clients.TransactionFilter{
		ProjectID:        initiativeID,
		TxnType:          txnType,
		SubscriptionOnly: subscriptionOnly,
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		return nil, err
	}

	// The Ledger stores some grant disbursements as credit-type rows with
	// negative amounts (e.g. SOS pays grants out of its fund). These are not
	// donations and must not appear in the "Donations received" table.
	if txnType == "donation" {
		fullPageLen := len(list.Data) // capture before filtering
		// The ledger client encodes HasNext by adding list.Limit to TotalCount;
		// if TotalCount > offset+fullPageLen the Ledger signalled more pages.
		hasMorePages := list.TotalCount > offset+fullPageLen

		kept := list.Data[:0]
		for _, t := range list.Data {
			if t.AmountCents > 0 {
				kept = append(kept, t)
			}
		}
		dropped := len(list.Data) - len(kept)
		list.Data = kept
		// Adjust the Ledger's total estimate by the number of rows dropped
		// from this page.  Two clamp rules keep the frontend's
		// "nextOffset < totalCount" guard from stopping pagination early:
		//
		//  1. Normal case (some items kept): TotalCount ≥ offset+len(kept)
		//     so the items already delivered are accounted for.
		//
		//  2. Entire page filtered out but HasNext=true: TotalCount must be
		//     > nextOffset (offset+limit) or the frontend halts before
		//     reaching later pages that may still contain positive entries
		//     (e.g. the manual_from_lf $1M credit on SOS page 6, which sits
		//     behind five all-negative disbursement pages).
		adjusted := list.TotalCount - dropped
		minTotal := offset + len(kept)
		if hasMorePages && len(kept) == 0 {
			minTotal = offset + limit + 1
		}
		if adjusted < minTotal {
			adjusted = minTotal
		}
		list.TotalCount = adjusted
	}

	enrichTransactionsFromDB(ctx, s.repo, list.Data)
	if initiative, err := s.repo.GetByID(ctx, initiativeID); err == nil && initiative != nil {
		list.InitiativeName = initiative.Name
	}
	return list, nil
}

// syncReimbursementPolicy upserts the initiative's policy in the Reimbursement
// Service. It is a no-op when the RS client is disabled or the initiative is not
// published. The sync runs in a background goroutine so RS latency (or
// unavailability) never blocks the user-facing update/approval response. The RS
// client's own timeout (10 s) bounds how long the goroutine can run.
func (s *InitiativeService) syncReimbursementPolicy(ctx context.Context, initiative *models.Initiative) {
	if s.reimbursement == nil {
		return
	}
	// Guard before the DB lookup — unpublished initiatives are never synced.
	if !initiative.Status.EqualFold(models.StatusPublished) {
		return
	}
	// Snapshot the initiative before launching the goroutine to avoid data races
	// if the caller or another goroutine mutates the struct after this returns.
	snap := *initiative
	// Use a detached context so the goroutine is not cancelled when the HTTP
	// request that triggered this call completes.
	detached := context.WithoutCancel(ctx)
	go func() {
		owner, err := s.userRepo.GetByID(detached, snap.OwnerID)
		if err != nil {
			s.logger.WarnContext(detached, "reimbursement sync: could not fetch owner",
				"initiative_id", snap.ID, "owner_id", snap.OwnerID, "error", err)
			return
		}
		if syncErr := s.reimbursement.SyncPolicy(detached, &snap, owner); syncErr != nil {
			s.logger.WarnContext(detached, "reimbursement sync: failed to sync policy",
				"initiative_id", snap.ID, "error", syncErr)
		}
	}()
}

// enrichTransactionsFromDB batch-looks up users and organizations from the CF DB
// and merges name + avatar_url onto each transaction.
// Falls back to a deterministic generated avatar when no DB record is found.
func enrichTransactionsFromDB(ctx context.Context, repo domain.InitiativeRepository, txns []models.Transaction) {
	if len(txns) == 0 {
		return
	}
	// Collect unique IDs to look up.
	userIDs := make([]string, 0, len(txns))
	orgIDs := make([]string, 0, len(txns))
	seenUsers := map[string]bool{}
	seenOrgs := map[string]bool{}
	for _, t := range txns {
		if t.LedgerUserID != "" && !seenUsers[t.LedgerUserID] {
			userIDs = append(userIDs, t.LedgerUserID)
			seenUsers[t.LedgerUserID] = true
		}
		if t.LedgerOrgID != "" && !seenOrgs[t.LedgerOrgID] {
			orgIDs = append(orgIDs, t.LedgerOrgID)
			seenOrgs[t.LedgerOrgID] = true
		}
	}

	users, err := repo.GetUsersByLegacyIDs(ctx, userIDs)
	if err != nil {
		slog.WarnContext(ctx, "failed to look up donor users", "error", err)
		users = map[string]models.User{}
	}
	orgs, err := repo.GetOrganizationsByIDs(ctx, orgIDs)
	if err != nil {
		slog.WarnContext(ctx, "failed to look up donor organizations", "error", err)
		orgs = map[string]models.Organization{}
	}

	for i := range txns {
		t := &txns[i]
		if t.LedgerOrgID != "" {
			if org, ok := orgs[t.LedgerOrgID]; ok {
				t.DonorName = org.Name
				t.DonorLogoURL = org.AvatarURL
			}
			if t.DonorName == "" {
				t.DonorName = "Anonymous"
			}
			if t.DonorLogoURL == "" {
				t.DonorLogoURL = generatedAvatarURL(t.LedgerOrgID, t.DonorName)
			}
		} else if t.LedgerUserID != "" {
			if user, ok := users[t.LedgerUserID]; ok {
				if user.Name != "" {
					t.DonorName = user.Name
				}
				t.DonorLogoURL = user.AvatarURL
			}
			if t.DonorName == "" {
				t.DonorName = "Anonymous"
			}
			if t.DonorLogoURL == "" {
				t.DonorLogoURL = generatedAvatarURL(t.LedgerUserID, t.DonorName)
			}
		}
	}
}

// avatarPalette is the set of background colors used for generated avatars.
// Chosen to be visually distinct and accessible against white text.
var avatarPalette = []string{"326CE5", "E6522C", "425CC7", "2E7D32", "6A1B9A", "00838F", "C62828", "558B2F"}

// generatedAvatarURL returns a deterministic UI Avatars URL for the given id and name.
// The background color is derived from a hash of id so the same entity always
// gets the same color across requests.
func generatedAvatarURL(id, name string) string {
	h := fnv.New32a()
	h.Write([]byte(id))
	color := avatarPalette[h.Sum32()%uint32(len(avatarPalette))]
	return fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=%s&color=fff&size=128&bold=true",
		url.QueryEscape(name), color)
}

// Delete removes an initiative, enforcing owner authorisation.
func (s *InitiativeService) Delete(ctx context.Context, id, callerUsername string) error {
	ctx, span := initiativeSvcTracer.Start(ctx, "InitiativeService.Delete")
	defer span.End()
	span.SetAttributes(attribute.String("initiative.id", id))

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return err
	}

	caller, err := s.userRepo.GetByUsername(ctx, callerUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Unknown caller cannot own any initiative.
			return domain.ErrForbidden
		}
		span.RecordError(err)
		return err
	}
	if existing.OwnerID != caller.ID {
		return domain.ErrForbidden
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		span.RecordError(err)
		return fmt.Errorf("delete initiative: %w", err)
	}
	return nil
}
