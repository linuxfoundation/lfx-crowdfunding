// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var offlineDonationTracer = otel.Tracer("offline-donations-service")

// b2bOrgLookupTimeout bounds the member-service match on the approver's request;
// past it the donation is recorded without an SFID and b2b-org-backfill fills it.
const b2bOrgLookupTimeout = 10 * time.Second

// earliestReceivedOn rejects typos like 0202 in the received date.
var earliestReceivedOn = time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

// dollarAmount is a positive dollar amount with at most two decimals and no
// separators, up to $9,999,999.99.
var dollarAmount = regexp.MustCompile(`^(\d{1,7})(?:\.(\d{1,2}))?$`)

// ErrSourceTxnIDTaken means the Ledger already has a transaction with the
// given source reference, so it would have dropped this one.
var ErrSourceTxnIDTaken = fmt.Errorf("%w: source_txn_id is already recorded", domain.ErrConflict)

// OfflineLedgerPoster posts an offline donation to the Ledger and reports
// whether the Ledger already had its sourceTxnID.
type OfflineLedgerPoster interface {
	PostOfflineTransaction(ctx context.Context, txn clients.LedgerTransaction) (duplicate bool, err error)
}

// OfflineDonationService records offline org donations for approvers (#464):
// stored in CF so the Org lens lists them, and posted once to the Ledger so
// initiative totals and donor lists include them.
type OfflineDonationService struct {
	repo           domain.OfflineDonationRepository
	initiativeRepo domain.InitiativeRepository
	userRepo       domain.UserRepository
	ledger         OfflineLedgerPoster
	b2bOrgResolver B2BOrgResolver
	orgLogos       domain.OrgLogoRepository
	now            func() time.Time
}

// NewOfflineDonationService returns an OfflineDonationService.
func NewOfflineDonationService(
	repo domain.OfflineDonationRepository,
	initiativeRepo domain.InitiativeRepository,
	userRepo domain.UserRepository,
	ledger OfflineLedgerPoster,
) *OfflineDonationService {
	return &OfflineDonationService{repo: repo, initiativeRepo: initiativeRepo, userRepo: userRepo, ledger: ledger, now: time.Now}
}

// SetB2BOrgResolver wires the b2b_org match by domain; unset leaves the SFID
// for b2b-org-backfill.
func (s *OfflineDonationService) SetB2BOrgResolver(r B2BOrgResolver) {
	s.b2bOrgResolver = r
}

// SetOrgLogoRepo wires the store for the matched b2b_org's logo (#472).
func (s *OfflineDonationService) SetOrgLogoRepo(r domain.OrgLogoRepository) {
	s.orgLogos = r
}

// Record stores an offline donation and posts it to the Ledger. Submitting the
// same donation again (same source_txn_id and details) returns the stored one
// without a second Ledger row. created is false in that case.
func (s *OfflineDonationService) Record(ctx context.Context, approverUsername string, input models.OfflineDonationInput) (*models.OfflineDonation, bool, error) {
	ctx, span := offlineDonationTracer.Start(ctx, "OfflineDonationService.Record")
	defer span.End()

	want, err := s.validate(input)
	if err != nil {
		return nil, false, err
	}
	span.SetAttributes(attribute.String("initiative.id", want.InitiativeID))

	initiative, err := s.initiativeRepo.GetByID(ctx, want.InitiativeID)
	if err != nil {
		return nil, false, err
	}
	if !initiative.Status.EqualFold(models.StatusPublished) {
		return nil, false, domain.ErrInitiativeNotFound
	}
	want.InitiativeID = initiative.ID

	approver, err := s.userRepo.GetByUsername(ctx, approverUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, false, fmt.Errorf("%w: no profile found — call PATCH /crowdfunding/me first", domain.ErrProfileNotSynced)
		}
		return nil, false, fmt.Errorf("resolve approver: %w", err)
	}
	want.RecordedByID = approver.ID

	account, err := s.userRepo.GetByUsername(ctx, domain.OfflineAccountUsername)
	if errors.Is(err, domain.ErrUserNotFound) {
		// Not the caller's fault: a 500, not a 404.
		return nil, false, fmt.Errorf("offline account %s has no user row (migration 019)", domain.OfflineAccountUsername)
	}
	if err != nil {
		return nil, false, fmt.Errorf("resolve offline account %s (migration 019): %w", domain.OfflineAccountUsername, err)
	}
	want.UserID = account.ID

	match := s.matchB2BOrg(ctx, want.OrganizationDomain)
	want.B2BOrgSFID = match.SFID

	stored, created, err := s.repo.CreateOffline(ctx, want)
	if err != nil {
		return nil, false, err
	}
	if !created {
		if !sameOfflineDonation(stored, want) {
			return nil, false, ErrSourceTxnIDTaken
		}
		if stored.Status == models.DonationStatusSucceeded {
			return stored, false, nil
		}
		// Pending: an earlier attempt may or may not have reached the Ledger.
		// Re-posting is safe, since the Ledger keeps one row per sourceTxnID.
	}

	duplicate, err := s.ledger.PostOfflineTransaction(ctx, offlineLedgerTxn(stored))
	if err != nil {
		span.RecordError(err)
		// The row stays pending; resubmitting the same donation retries the post.
		return nil, false, fmt.Errorf("post offline donation to ledger: %w", err)
	}
	if duplicate && created {
		// The reference was in the Ledger before CF recorded anything (e.g. a
		// Postman entry), so the Ledger dropped this one. Free the reference.
		if err := s.repo.DeleteOfflinePending(ctx, stored.ID); err != nil {
			slog.ErrorContext(ctx, "offline donation: delete after ledger duplicate failed", "id", stored.ID, "error", err)
		}
		return nil, false, ErrSourceTxnIDTaken
	}
	if err := s.repo.MarkOfflineSucceeded(ctx, stored.ID); err != nil {
		return nil, false, err
	}
	stored.Status = models.DonationStatusSucceeded
	if match.SFID != "" && s.orgLogos != nil {
		if err := s.orgLogos.SetOrgLogo(ctx, match.SFID, match.LogoURL); err != nil {
			slog.WarnContext(ctx, "offline donation: b2b_org logo store failed", "b2b_org_sfid", match.SFID, "error", err)
		}
	}
	slog.InfoContext(ctx, "offline donation recorded",
		"id", stored.ID, "initiative_id", stored.InitiativeID, "source_txn_id", stored.SourceTxnID,
		"amount_cents", stored.AmountCents, "recorded_by", approverUsername, "b2b_org_sfid", stored.B2BOrgSFID)
	return stored, created, nil
}

// List returns recorded offline donations; an empty initiativeID lists all.
func (s *OfflineDonationService) List(ctx context.Context, initiativeID string, limit, offset int) ([]models.OfflineDonation, *models.PaginationMeta, error) {
	ctx, span := offlineDonationTracer.Start(ctx, "OfflineDonationService.List")
	defer span.End()

	if initiativeID != "" {
		initiative, err := s.initiativeRepo.GetByID(ctx, initiativeID)
		if err != nil {
			return nil, nil, err
		}
		initiativeID = initiative.ID
	}
	return s.repo.ListOffline(ctx, initiativeID, limit, offset)
}

// matchB2BOrg looks the org up by domain on this request, since the SFID goes
// into the Ledger row. A failure only logs: b2b-org-backfill retries CF rows.
func (s *OfflineDonationService) matchB2BOrg(ctx context.Context, orgDomain string) models.B2BOrgMatch {
	if s.b2bOrgResolver == nil {
		return models.B2BOrgMatch{}
	}
	lookupCtx, cancel := context.WithTimeout(ctx, b2bOrgLookupTimeout)
	defer cancel()
	match, err := s.b2bOrgResolver.LookupB2BOrgByWebsite(lookupCtx, orgDomain)
	if err != nil {
		slog.WarnContext(ctx, "offline donation: b2b_org lookup failed", "org_domain", orgDomain, "error", err)
		return models.B2BOrgMatch{}
	}
	return match
}

// validate checks the input and returns the donation to store.
func (s *OfflineDonationService) validate(in models.OfflineDonationInput) (*models.OfflineDonation, error) {
	initiativeID := strings.TrimSpace(in.InitiativeID)
	if initiativeID == "" {
		return nil, fmt.Errorf("%w: initiative_id is required", domain.ErrInvalidInput)
	}
	if in.Organization == nil {
		return nil, fmt.Errorf("%w: organization is required", domain.ErrInvalidInput)
	}
	org, err := orgClaim(in.Organization)
	if err != nil {
		return nil, err
	}
	cents, err := dollarsToCents(in.Amount)
	if err != nil {
		return nil, err
	}
	receivedOn, err := time.Parse(time.DateOnly, strings.TrimSpace(in.ReceivedOn))
	if err != nil {
		return nil, fmt.Errorf("%w: received_on must be a date (YYYY-MM-DD)", domain.ErrInvalidInput)
	}
	// A day of slack so an approver ahead of UTC can enter today's date.
	if receivedOn.Before(earliestReceivedOn) || receivedOn.After(s.now().UTC().AddDate(0, 0, 1)) {
		return nil, fmt.Errorf("%w: received_on must be between %s and today", domain.ErrInvalidInput, earliestReceivedOn.Format(time.DateOnly))
	}
	category, err := boundedText("category", in.Category, 100)
	if err != nil {
		return nil, err
	}
	sourceTxnID, err := boundedText("source_txn_id", in.SourceTxnID, 255)
	if err != nil {
		return nil, err
	}
	return &models.OfflineDonation{
		InitiativeID:       initiativeID,
		OrganizationName:   org.Name,
		OrganizationDomain: org.Domain,
		AmountCents:        cents,
		Category:           category,
		ReceivedOn:         receivedOn.Format(time.DateOnly),
		SourceTxnID:        sourceTxnID,
	}, nil
}

// dollarsToCents converts "1300" or "1300.5" to cents without floating point.
func dollarsToCents(raw string) (int64, error) {
	m := dollarAmount.FindStringSubmatch(strings.TrimSpace(raw))
	if m == nil {
		return 0, fmt.Errorf("%w: amount must be a dollar amount like 1300 or 1300.50", domain.ErrInvalidInput)
	}
	dollars, _ := strconv.ParseInt(m[1], 10, 64)
	frac := (m[2] + "00")[:2]
	cents, _ := strconv.ParseInt(frac, 10, 64)
	total := dollars*100 + cents
	if total <= 0 {
		return 0, fmt.Errorf("%w: amount must be positive", domain.ErrInvalidInput)
	}
	return total, nil
}

// boundedText trims raw and requires 1-max characters with no control characters.
func boundedText(field, raw string, max int) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" || utf8.RuneCountInString(v) > max || strings.ContainsFunc(v, unicode.IsControl) {
		return "", fmt.Errorf("%w: %s must be 1-%d characters with no control characters", domain.ErrInvalidInput, field, max)
	}
	return v, nil
}

// sameOfflineDonation reports whether a stored donation is the one being
// submitted again, rather than a different donation reusing its reference.
func sameOfflineDonation(stored, want *models.OfflineDonation) bool {
	return stored.InitiativeID == want.InitiativeID &&
		stored.OrganizationDomain == want.OrganizationDomain &&
		stored.AmountCents == want.AmountCents &&
		stored.ReceivedOn == want.ReceivedOn
}

// offlineLedgerTxn builds the Ledger row approvers used to post by hand.
func offlineLedgerTxn(d *models.OfflineDonation) clients.LedgerTransaction {
	received, _ := time.Parse(time.DateOnly, d.ReceivedOn)
	return clients.LedgerTransaction{
		ProjectID:          d.InitiativeID,
		UserID:             domain.LedgerOfflineUserID,
		AccountEmail:       domain.LedgerOfflineAccountEmail,
		OrganizationName:   d.OrganizationName,
		OrganizationDomain: d.OrganizationDomain,
		OrganizationSFID:   d.B2BOrgSFID,
		SourceType:         domain.LedgerOfflineSourceType,
		SourceTxnID:        d.SourceTxnID,
		SourceAccountID:    domain.LedgerOfflineSourceAccountID,
		TxnType:            domain.LedgerTxnTypeCredit,
		TxnCategory:        d.Category,
		Fee:                0,
		Amount:             int(d.AmountCents),
		TxnDate:            received.Unix(),
	}
}
