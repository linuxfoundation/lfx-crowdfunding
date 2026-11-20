// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// fakeAdjustmentRepo keeps adjustments in memory, keyed by lowercased
// source_txn_id; donationRefs are references offline donations already use.
type fakeAdjustmentRepo struct {
	bySource     map[string]*models.LedgerAdjustment
	donationRefs map[string]bool
	deleted      []string
	locked       map[string]bool
}

func newFakeAdjustmentRepo() *fakeAdjustmentRepo {
	return &fakeAdjustmentRepo{bySource: map[string]*models.LedgerAdjustment{}, donationRefs: map[string]bool{}, locked: map[string]bool{}}
}

func (r *fakeAdjustmentRepo) LockReference(_ context.Context, ref string) (func(), error) {
	if r.locked[ref] {
		return nil, errors.New("fake: reference already locked")
	}
	r.locked[ref] = true
	return func() { r.locked[ref] = false }, nil
}

func (r *fakeAdjustmentRepo) CreateAdjustment(_ context.Context, a *models.LedgerAdjustment) (*models.LedgerAdjustment, bool, error) {
	key := strings.ToLower(a.SourceTxnID)
	if r.donationRefs[key] {
		return nil, false, domain.ErrSourceTxnIDTaken
	}
	if existing, ok := r.bySource[key]; ok {
		cp := *existing
		return &cp, false, nil
	}
	stored := *a
	stored.ID = "adj-" + a.SourceTxnID
	stored.Status = models.DonationStatusPending
	r.bySource[key] = &stored
	cp := stored
	return &cp, true, nil
}

func (r *fakeAdjustmentRepo) SetAdjustmentB2BOrg(_ context.Context, id, sfid string) error {
	for _, a := range r.bySource {
		if a.ID == id && a.Status == models.DonationStatusPending && a.B2BOrgSFID == "" {
			a.B2BOrgSFID = sfid
		}
	}
	return nil
}

func (r *fakeAdjustmentRepo) MarkAdjustmentSucceeded(_ context.Context, id string) error {
	for _, a := range r.bySource {
		if a.ID == id && a.Status == models.DonationStatusPending {
			a.Status = models.DonationStatusSucceeded
			return nil
		}
	}
	return errors.New("fake: no pending adjustment " + id)
}

func (r *fakeAdjustmentRepo) DeleteAdjustmentPending(_ context.Context, id string) error {
	r.deleted = append(r.deleted, id)
	for k, a := range r.bySource {
		if a.ID == id {
			delete(r.bySource, k)
		}
	}
	return nil
}

func (r *fakeAdjustmentRepo) ListAdjustments(_ context.Context, _ string, _, _ int) ([]models.LedgerAdjustment, *models.PaginationMeta, error) {
	return nil, &models.PaginationMeta{}, nil
}

// FindOfflineAdjustment maps the fake Ledger's row the way the real client
// does: a debit is a reimbursement, a negative credit a return.
func (l *fakeOfflineLedger) FindOfflineAdjustment(_ context.Context, projectID, ref string) (*models.Transaction, error) {
	txn, ok := l.rows[ref]
	if !ok || txn.ProjectID != projectID {
		return nil, nil
	}
	typ := models.TransactionTypeDonation
	switch {
	case txn.TxnType == "debit":
		typ = models.TransactionTypeReimbursement
	case txn.Amount < 0:
		typ = models.TransactionTypeReturn
	}
	return &models.Transaction{
		Type: typ, AmountCents: int64(txn.Amount), LedgerProjectID: txn.ProjectID,
		LedgerOrgDomain: txn.OrganizationDomain, LedgerSourceTxnID: txn.SourceTxnID,
	}, nil
}

type adjustmentFixture struct {
	svc    *LedgerAdjustmentService
	repo   *fakeAdjustmentRepo
	ledger *fakeOfflineLedger
}

func newAdjustmentFixture(t *testing.T) *adjustmentFixture {
	t.Helper()
	initiatives := &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", Status: models.StatusPublished}}
	users := &offlineUserRepo{users: map[string]*models.User{"approver": {ID: "approver-1", Username: "approver"}}}
	f := &adjustmentFixture{repo: newFakeAdjustmentRepo(), ledger: &fakeOfflineLedger{}}
	f.svc = NewLedgerAdjustmentService(f.repo, initiatives, users, f.ledger)
	f.svc.now = func() time.Time { return time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC) }
	f.svc.SetB2BOrgResolver(b2bOrgResolverFunc(func(_ context.Context, d string) (models.B2BOrgMatch, error) {
		if d == "cncf.io" {
			return models.B2BOrgMatch{SFID: "001CNCF"}, nil
		}
		return models.B2BOrgMatch{}, nil
	}))
	return f
}

func validReturnInput() models.LedgerAdjustmentInput {
	return models.LedgerAdjustmentInput{
		Kind:         "return",
		InitiativeID: "init-1",
		Organization: &models.OrgClaim{Name: "Cloud Native Computing Foundation", Domain: "https://www.cncf.io/"},
		Amount:       "1500",
		AdjustedOn:   "2026-11-02",
		Category:     "mentorship",
		Reason:       "  Mentee failed the midterm evaluation  ",
		SourceTxnID:  "  TEST Spring 2026 Row 15 return ,",
	}
}

func validPayoutInput() models.LedgerAdjustmentInput {
	return models.LedgerAdjustmentInput{
		Kind:         "payout",
		InitiativeID: "init-1",
		PaidTo:       "Jane Payee",
		Amount:       "10000.50",
		AdjustedOn:   "2026-11-03",
		Category:     "general fund",
		Reason:       "Rewards payout row 31",
		SourceTxnID:  "TEST Rewards Payout Row 31",
	}
}

func TestLedgerAdjustmentRecord_ReturnPostsANegativeCreditWithTheOrg(t *testing.T) {
	f := newAdjustmentFixture(t)

	got, created, err := f.svc.Record(context.Background(), "approver", validReturnInput())
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if !created || got.Status != models.DonationStatusSucceeded || got.AmountCents != 150000 {
		t.Errorf("created=%v status=%q amount=%d, want true/succeeded/150000", created, got.Status, got.AmountCents)
	}
	if got.RecordedByID != "approver-1" || got.B2BOrgSFID != "001CNCF" {
		t.Errorf("recorded_by=%q sfid=%q", got.RecordedByID, got.B2BOrgSFID)
	}
	if len(f.ledger.posts) != 1 {
		t.Fatalf("ledger posts = %d, want 1", len(f.ledger.posts))
	}
	want := clients.LedgerTransaction{
		ProjectID:          "init-1",
		UserID:             "auth0|communitybridge",
		AccountEmail:       "admin@communitybridge.org",
		OrganizationName:   "Cloud Native Computing Foundation",
		OrganizationDomain: "cncf.io",
		OrganizationSFID:   "001CNCF",
		TxnComment:         "Mentee failed the midterm evaluation",
		SourceType:         "expensify",
		SourceTxnID:        "TEST Spring 2026 Row 15 return ,",
		SourceAccountID:    "manual_from_lf",
		TxnType:            "credit",
		TxnCategory:        "mentorship",
		Amount:             -150000,
		TxnDate:            time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC).Unix(),
	}
	if f.ledger.posts[0] != want {
		t.Errorf("ledger txn =\n%+v\nwant\n%+v", f.ledger.posts[0], want)
	}
}

func TestLedgerAdjustmentRecord_PayoutPostsANegativeDebitWithThePayee(t *testing.T) {
	f := newAdjustmentFixture(t)

	if _, _, err := f.svc.Record(context.Background(), "approver", validPayoutInput()); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if len(f.ledger.posts) != 1 {
		t.Fatalf("ledger posts = %d, want 1", len(f.ledger.posts))
	}
	got := f.ledger.posts[0]
	if got.TxnType != "debit" || got.Amount != -1000050 || got.SubmitterName != "Jane Payee" ||
		got.OrganizationName != "" || got.TxnComment != "Rewards payout row 31" {
		t.Errorf("payout txn = %+v", got)
	}
}

func TestLedgerAdjustmentRecord_PayoutCanNameTheFundingOrg(t *testing.T) {
	f := newAdjustmentFixture(t)
	in := validPayoutInput()
	in.Organization = &models.OrgClaim{Name: "Cloud Native Computing Foundation", Domain: "cncf.io"}

	if _, _, err := f.svc.Record(context.Background(), "approver", in); err != nil {
		t.Fatalf("Record: %v", err)
	}
	got := f.ledger.posts[0]
	if got.OrganizationDomain != "cncf.io" || got.OrganizationSFID != "001CNCF" || got.SubmitterName != "Jane Payee" {
		t.Errorf("payout txn = %+v", got)
	}
}

func TestLedgerAdjustmentRecord_ResubmitReturnsStoredWithoutSecondRow(t *testing.T) {
	f := newAdjustmentFixture(t)
	if _, _, err := f.svc.Record(context.Background(), "approver", validReturnInput()); err != nil {
		t.Fatalf("first Record: %v", err)
	}
	got, created, err := f.svc.Record(context.Background(), "approver", validReturnInput())
	if err != nil {
		t.Fatalf("second Record: %v", err)
	}
	if created || got.Status != models.DonationStatusSucceeded {
		t.Errorf("created=%v status=%q, want false/succeeded", created, got.Status)
	}
	if len(f.ledger.posts) != 1 {
		t.Errorf("ledger posts = %d, want 1", len(f.ledger.posts))
	}
}

func TestLedgerAdjustmentRecord_ReusedReferenceIsAConflict(t *testing.T) {
	f := newAdjustmentFixture(t)
	if _, _, err := f.svc.Record(context.Background(), "approver", validReturnInput()); err != nil {
		t.Fatalf("first Record: %v", err)
	}
	in := validReturnInput()
	in.Amount = "750"
	if _, _, err := f.svc.Record(context.Background(), "approver", in); !errors.Is(err, domain.ErrSourceTxnIDTaken) {
		t.Fatalf("err = %v, want ErrSourceTxnIDTaken", err)
	}
}

func TestLedgerAdjustmentRecord_ReferenceOfAnOfflineDonationIsAConflict(t *testing.T) {
	f := newAdjustmentFixture(t)
	f.repo.donationRefs["test rewards payout row 31"] = true

	if _, _, err := f.svc.Record(context.Background(), "approver", validPayoutInput()); !errors.Is(err, domain.ErrSourceTxnIDTaken) {
		t.Fatalf("err = %v, want ErrSourceTxnIDTaken", err)
	}
	if len(f.ledger.posts) != 0 {
		t.Errorf("ledger posts = %d, want 0", len(f.ledger.posts))
	}
}

func TestLedgerAdjustmentRecord_ReferenceAlreadyInLedgerFreesTheRow(t *testing.T) {
	f := newAdjustmentFixture(t)
	f.ledger.addRow(clients.LedgerTransaction{ProjectID: "init-1", SourceTxnID: "TEST Rewards Payout Row 31", Amount: -5})

	if _, _, err := f.svc.Record(context.Background(), "approver", validPayoutInput()); !errors.Is(err, domain.ErrSourceTxnIDTaken) {
		t.Fatalf("err = %v, want ErrSourceTxnIDTaken", err)
	}
	if len(f.repo.deleted) != 1 || len(f.repo.bySource) != 0 {
		t.Errorf("deleted=%v remaining=%d, want the pending row deleted", f.repo.deleted, len(f.repo.bySource))
	}
}

func TestLedgerAdjustmentRecord_RetryAfterLostLedgerReplyMarksSucceeded(t *testing.T) {
	f := newAdjustmentFixture(t)
	// The first post lands in the Ledger, but CF never sees the reply.
	f.ledger.onPost = func(txn clients.LedgerTransaction) {
		f.ledger.addRow(txn)
		f.ledger.onPost = nil
	}
	f.ledger.err = errors.New("timeout")
	if _, _, err := f.svc.Record(context.Background(), "approver", validReturnInput()); err == nil {
		t.Fatal("first Record: want the Ledger error")
	}
	f.ledger.err = nil

	got, created, err := f.svc.Record(context.Background(), "approver", validReturnInput())
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if created || got.Status != models.DonationStatusSucceeded || len(f.repo.deleted) != 0 {
		t.Errorf("created=%v status=%q deleted=%v, want false/succeeded/none", created, got.Status, f.repo.deleted)
	}
}

func TestLedgerAdjustmentRecord_RetryDoesNotAdoptADifferentLedgerRow(t *testing.T) {
	f := newAdjustmentFixture(t)
	f.ledger.err = errors.New("ledger down")
	if _, _, err := f.svc.Record(context.Background(), "approver", validReturnInput()); err == nil {
		t.Fatal("first Record: want the Ledger error")
	}
	// Meanwhile someone hand-posts a different row with the same reference.
	f.ledger.err = nil
	f.ledger.addRow(clients.LedgerTransaction{
		ProjectID: "init-1", SourceTxnID: "TEST Spring 2026 Row 15 return ,", TxnType: "credit",
		Amount: 150000, OrganizationDomain: "cncf.io",
	})

	if _, _, err := f.svc.Record(context.Background(), "approver", validReturnInput()); !errors.Is(err, ErrAdjustmentRowUnverified) {
		t.Fatalf("err = %v, want ErrAdjustmentRowUnverified", err)
	}
	if got := f.repo.bySource["test spring 2026 row 15 return ,"]; got.Status != models.DonationStatusPending {
		t.Errorf("status = %q, want pending", got.Status)
	}
}

func TestLedgerAdjustmentRecord_UnpublishedInitiativeIsNotFound(t *testing.T) {
	f := newAdjustmentFixture(t)
	f.svc.initiativeRepo = &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", Status: models.StatusPending}}

	if _, _, err := f.svc.Record(context.Background(), "approver", validReturnInput()); !errors.Is(err, domain.ErrInitiativeNotFound) {
		t.Fatalf("err = %v, want ErrInitiativeNotFound", err)
	}
}

func TestLedgerAdjustmentRecord_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input func() models.LedgerAdjustmentInput
	}{
		{"unknown kind", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.Kind = "refund"; return in }},
		{"return without org", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.Organization = nil; return in }},
		{"return with payee", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.PaidTo = "Someone"; return in }},
		{"payout without payee", func() models.LedgerAdjustmentInput { in := validPayoutInput(); in.PaidTo = " "; return in }},
		{"negative amount", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.Amount = "-1500"; return in }},
		{"zero amount", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.Amount = "0"; return in }},
		{"cents typed as dollars", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.Amount = "1500.005"; return in }},
		{"missing reason", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.Reason = ""; return in }},
		{"long reason", func() models.LedgerAdjustmentInput {
			in := validReturnInput()
			in.Reason = strings.Repeat("x", 501)
			return in
		}},
		{"missing reference", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.SourceTxnID = " "; return in }},
		{"missing category", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.Category = ""; return in }},
		{"future date", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.AdjustedOn = "2026-12-01"; return in }},
		{"not a date", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.AdjustedOn = "11/02/2026"; return in }},
		{"missing initiative", func() models.LedgerAdjustmentInput { in := validReturnInput(); in.InitiativeID = ""; return in }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdjustmentFixture(t)
			if _, _, err := f.svc.Record(context.Background(), "approver", tt.input()); !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput", err)
			}
			if len(f.ledger.posts) != 0 {
				t.Errorf("ledger posts = %d, want 0", len(f.ledger.posts))
			}
		})
	}
}
