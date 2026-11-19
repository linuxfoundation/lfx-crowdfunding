// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var adjustmentTracer = otel.Tracer("ledger-adjustments-service")

// ErrAdjustmentRowUnverified means a retry found the reference in the Ledger
// but couldn't confirm the row is this adjustment, so it stays pending.
var ErrAdjustmentRowUnverified = fmt.Errorf("%w: the Ledger already has a transaction with this source_txn_id "+
	"that doesn't match this adjustment; check the Ledger before submitting again", domain.ErrConflict)

// AdjustmentLedgerPoster posts a return or payout to the Ledger and finds an
// initiative's manual row by sourceTxnID (nil when there is none).
type AdjustmentLedgerPoster interface {
	PostOfflineTransaction(ctx context.Context, txn clients.LedgerTransaction) (duplicate bool, err error)
	FindOfflineAdjustment(ctx context.Context, projectID, sourceTxnID string) (*models.Transaction, error)
}

// LedgerAdjustmentService records money approvers take out of an initiative
// (#489): returns of unused funds to the donor and payouts from a fund. Each
// is stored in CF for the audit trail and posted once to the Ledger, the way
// OfflineDonationService records offline donations.
type LedgerAdjustmentService struct {
	repo           domain.LedgerAdjustmentRepository
	initiativeRepo domain.InitiativeRepository
	userRepo       domain.UserRepository
	ledger         AdjustmentLedgerPoster
	b2bOrgResolver B2BOrgResolver
	now            func() time.Time
}

// NewLedgerAdjustmentService returns a LedgerAdjustmentService.
func NewLedgerAdjustmentService(
	repo domain.LedgerAdjustmentRepository,
	initiativeRepo domain.InitiativeRepository,
	userRepo domain.UserRepository,
	ledger AdjustmentLedgerPoster,
) *LedgerAdjustmentService {
	return &LedgerAdjustmentService{repo: repo, initiativeRepo: initiativeRepo, userRepo: userRepo, ledger: ledger, now: time.Now}
}

// SetB2BOrgResolver wires the b2b_org match for the donor org's SFID.
func (s *LedgerAdjustmentService) SetB2BOrgResolver(r B2BOrgResolver) {
	s.b2bOrgResolver = r
}

// Record stores an adjustment and posts it to the Ledger. Submitting the same
// adjustment again returns the stored one without a second Ledger row; created
// is false in that case. A reference already used (by any adjustment, offline
// donation or Ledger row) is ErrSourceTxnIDTaken.
func (s *LedgerAdjustmentService) Record(ctx context.Context, approverUsername string, input models.LedgerAdjustmentInput) (*models.LedgerAdjustment, bool, error) {
	ctx, span := adjustmentTracer.Start(ctx, "LedgerAdjustmentService.Record")
	defer span.End()

	want, err := s.validate(input)
	if err != nil {
		return nil, false, err
	}
	span.SetAttributes(attribute.String("initiative.id", want.InitiativeID), attribute.String("adjustment.kind", want.Kind))

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

	match := matchB2BOrgNow(ctx, s.b2bOrgResolver, "ledger adjustment", want.OrganizationDomain)
	want.B2BOrgSFID = match.SFID

	// One request at a time per reference, shared with offline donations.
	unlock, err := s.repo.LockReference(ctx, want.SourceTxnID)
	if err != nil {
		return nil, false, err
	}
	defer unlock()

	stored, created, err := s.repo.CreateAdjustment(ctx, want)
	if err != nil {
		return nil, false, err
	}
	if !created {
		if !sameAdjustment(stored, want) {
			return nil, false, domain.ErrSourceTxnIDTaken
		}
		if stored.Status == models.DonationStatusSucceeded {
			return stored, false, nil
		}
		// Pending: an earlier attempt may or may not have reached the Ledger.
		if stored.B2BOrgSFID == "" && match.SFID != "" {
			if err := s.repo.SetAdjustmentB2BOrg(ctx, stored.ID, match.SFID); err != nil {
				return nil, false, err
			}
			stored.B2BOrgSFID = match.SFID
		}
	}

	duplicate, err := s.ledger.PostOfflineTransaction(ctx, adjustmentLedgerTxn(stored))
	if err != nil {
		span.RecordError(err)
		return nil, false, fmt.Errorf("post ledger adjustment: %w", err)
	}
	if duplicate && created {
		// The Ledger had the reference before CF recorded anything (e.g. a
		// hand-posted row), so it dropped this one. Free the reference.
		if err := s.repo.DeleteAdjustmentPending(ctx, stored.ID); err != nil {
			slog.ErrorContext(ctx, "ledger adjustment: delete after ledger duplicate failed", "id", stored.ID, "error", err)
		}
		return nil, false, domain.ErrSourceTxnIDTaken
	}
	if duplicate {
		if err := s.verifyLedgerRow(ctx, stored); err != nil {
			span.RecordError(err)
			return nil, false, err
		}
	}
	if err := s.repo.MarkAdjustmentSucceeded(ctx, stored.ID); err != nil {
		return nil, false, err
	}
	stored.Status = models.DonationStatusSucceeded
	slog.InfoContext(ctx, "ledger adjustment recorded",
		"id", stored.ID, "kind", stored.Kind, "initiative_id", stored.InitiativeID, "source_txn_id", stored.SourceTxnID,
		"amount_cents", stored.AmountCents, "recorded_by", approverUsername, "b2b_org_sfid", stored.B2BOrgSFID)
	return stored, created, nil
}

// List returns recorded adjustments; an empty initiativeID lists all.
func (s *LedgerAdjustmentService) List(ctx context.Context, initiativeID string, limit, offset int) ([]models.LedgerAdjustment, *models.PaginationMeta, error) {
	ctx, span := adjustmentTracer.Start(ctx, "LedgerAdjustmentService.List")
	defer span.End()

	if initiativeID != "" {
		initiative, err := s.initiativeRepo.GetByID(ctx, initiativeID)
		if err != nil {
			return nil, nil, err
		}
		initiativeID = initiative.ID
	}
	return s.repo.ListAdjustments(ctx, initiativeID, limit, offset)
}

// verifyLedgerRow checks that the Ledger's row for a's reference is a: same
// initiative, kind, negated amount and org domain.
func (s *LedgerAdjustmentService) verifyLedgerRow(ctx context.Context, a *models.LedgerAdjustment) error {
	txn, err := s.ledger.FindOfflineAdjustment(ctx, a.InitiativeID, a.SourceTxnID)
	if err != nil {
		return fmt.Errorf("verify ledger row for adjustment: %w", err)
	}
	wantType := models.TransactionTypeReturn
	if a.Kind == models.AdjustmentKindPayout {
		wantType = models.TransactionTypeReimbursement
	}
	if txn == nil || txn.LedgerProjectID != a.InitiativeID || txn.Type != wantType ||
		txn.AmountCents != -a.AmountCents || !strings.EqualFold(txn.LedgerOrgDomain, a.OrganizationDomain) {
		slog.WarnContext(ctx, "ledger adjustment: ledger row for reference doesn't match, left pending",
			"id", a.ID, "source_txn_id", a.SourceTxnID, "found", txn != nil)
		return ErrAdjustmentRowUnverified
	}
	return nil
}

// validate checks the input and returns the adjustment to store.
func (s *LedgerAdjustmentService) validate(in models.LedgerAdjustmentInput) (*models.LedgerAdjustment, error) {
	kind := strings.ToLower(strings.TrimSpace(in.Kind))
	if kind != models.AdjustmentKindReturn && kind != models.AdjustmentKindPayout {
		return nil, fmt.Errorf("%w: kind must be %q or %q", domain.ErrInvalidInput, models.AdjustmentKindReturn, models.AdjustmentKindPayout)
	}
	initiativeID := strings.TrimSpace(in.InitiativeID)
	if initiativeID == "" {
		return nil, fmt.Errorf("%w: initiative_id is required", domain.ErrInvalidInput)
	}
	var org models.OrgClaim
	if in.Organization != nil {
		var err error
		if org, err = orgClaim(in.Organization); err != nil {
			return nil, err
		}
	}
	var paidTo string
	switch kind {
	case models.AdjustmentKindReturn:
		if in.Organization == nil {
			return nil, fmt.Errorf("%w: organization is required for a return", domain.ErrInvalidInput)
		}
		if strings.TrimSpace(in.PaidTo) != "" {
			return nil, fmt.Errorf("%w: paid_to is only for a payout; a return goes back to the organization", domain.ErrInvalidInput)
		}
	case models.AdjustmentKindPayout:
		var err error
		if paidTo, err = boundedText("paid_to", in.PaidTo, 255); err != nil {
			return nil, err
		}
	}
	cents, err := dollarsToCents(in.Amount)
	if err != nil {
		return nil, err
	}
	adjustedOn, err := parseLedgerDate("adjusted_on", in.AdjustedOn, s.now())
	if err != nil {
		return nil, err
	}
	category, err := boundedText("category", in.Category, 100)
	if err != nil {
		return nil, err
	}
	reason, err := boundedText("reason", in.Reason, 500)
	if err != nil {
		return nil, err
	}
	sourceTxnID, err := boundedText("source_txn_id", in.SourceTxnID, 255)
	if err != nil {
		return nil, err
	}
	return &models.LedgerAdjustment{
		Kind:               kind,
		InitiativeID:       initiativeID,
		OrganizationName:   org.Name,
		OrganizationDomain: org.Domain,
		PaidTo:             paidTo,
		AmountCents:        cents,
		Category:           category,
		Reason:             reason,
		AdjustedOn:         adjustedOn.Format(time.DateOnly),
		SourceTxnID:        sourceTxnID,
	}, nil
}

// sameAdjustment reports whether a stored adjustment is the one being
// submitted again, rather than a different one reusing its reference.
func sameAdjustment(stored, want *models.LedgerAdjustment) bool {
	return stored.Kind == want.Kind &&
		stored.InitiativeID == want.InitiativeID &&
		stored.OrganizationName == want.OrganizationName &&
		stored.OrganizationDomain == want.OrganizationDomain &&
		stored.PaidTo == want.PaidTo &&
		stored.AmountCents == want.AmountCents &&
		stored.Category == want.Category &&
		stored.Reason == want.Reason &&
		stored.AdjustedOn == want.AdjustedOn
}

// adjustmentLedgerTxn builds the Ledger row approvers used to post by hand:
// a return is a negative credit with the donor org, a payout a negative debit
// with the payee as submitterName. The reason goes in txnComment.
func adjustmentLedgerTxn(a *models.LedgerAdjustment) clients.LedgerTransaction {
	adjusted, _ := time.Parse(time.DateOnly, a.AdjustedOn)
	txnType := domain.LedgerTxnTypeCredit
	if a.Kind == models.AdjustmentKindPayout {
		txnType = domain.LedgerTxnTypeDebit
	}
	return clients.LedgerTransaction{
		ProjectID:          a.InitiativeID,
		UserID:             domain.LedgerOfflineUserID,
		AccountEmail:       domain.LedgerOfflineAccountEmail,
		OrganizationName:   a.OrganizationName,
		OrganizationDomain: a.OrganizationDomain,
		OrganizationSFID:   a.B2BOrgSFID,
		SubmitterName:      a.PaidTo,
		TxnComment:         a.Reason,
		SourceType:         domain.LedgerOfflineSourceType,
		SourceTxnID:        a.SourceTxnID,
		SourceAccountID:    domain.LedgerOfflineSourceAccountID,
		TxnType:            txnType,
		TxnCategory:        a.Category,
		Fee:                0,
		Amount:             -int(a.AmountCents),
		TxnDate:            adjusted.Unix(),
	}
}
