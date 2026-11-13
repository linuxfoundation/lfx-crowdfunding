// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// offlineUserRepo answers GetByUsername from users; a missing username is
// ErrUserNotFound.
type offlineUserRepo struct {
	mockUserRepository
	users map[string]*models.User
}

func (r *offlineUserRepo) GetByUsername(_ context.Context, username string) (*models.User, error) {
	if u, ok := r.users[username]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

// fakeOfflineRepo keeps offline donations in memory, keyed by source_txn_id.
type fakeOfflineRepo struct {
	bySource  map[string]*models.OfflineDonation
	succeeded []string
	deleted   []string
}

func newFakeOfflineRepo() *fakeOfflineRepo {
	return &fakeOfflineRepo{bySource: map[string]*models.OfflineDonation{}}
}

func (r *fakeOfflineRepo) CreateOffline(_ context.Context, d *models.OfflineDonation) (*models.OfflineDonation, bool, error) {
	if existing, ok := r.bySource[d.SourceTxnID]; ok {
		cp := *existing
		return &cp, false, nil
	}
	stored := *d
	stored.ID = "don-" + d.SourceTxnID
	stored.Status = models.DonationStatusPending
	r.bySource[d.SourceTxnID] = &stored
	cp := stored
	return &cp, true, nil
}

func (r *fakeOfflineRepo) MarkOfflineSucceeded(_ context.Context, id string) error {
	r.succeeded = append(r.succeeded, id)
	for _, d := range r.bySource {
		if d.ID == id {
			d.Status = models.DonationStatusSucceeded
		}
	}
	return nil
}

func (r *fakeOfflineRepo) DeleteOfflinePending(_ context.Context, id string) error {
	r.deleted = append(r.deleted, id)
	for k, d := range r.bySource {
		if d.ID == id {
			delete(r.bySource, k)
		}
	}
	return nil
}

func (r *fakeOfflineRepo) ListOffline(_ context.Context, _ string, _, _ int) ([]models.OfflineDonation, *models.PaginationMeta, error) {
	return nil, &models.PaginationMeta{}, nil
}

// fakeOfflineLedger records posts; it reports a duplicate for any sourceTxnID
// it already holds, like the Ledger does.
type fakeOfflineLedger struct {
	posts []clients.LedgerTransaction
	have  map[string]bool
	err   error
}

func (l *fakeOfflineLedger) PostOfflineTransaction(_ context.Context, txn clients.LedgerTransaction) (bool, error) {
	if l.err != nil {
		return false, l.err
	}
	l.posts = append(l.posts, txn)
	if l.have == nil {
		l.have = map[string]bool{}
	}
	dup := l.have[txn.SourceTxnID]
	l.have[txn.SourceTxnID] = true
	return dup, nil
}

type offlineFixture struct {
	svc    *OfflineDonationService
	repo   *fakeOfflineRepo
	ledger *fakeOfflineLedger
	logos  *recordingLogos
}

func newOfflineFixture(t *testing.T) *offlineFixture {
	t.Helper()
	initiatives := &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", Status: models.StatusPublished}}
	users := &offlineUserRepo{users: map[string]*models.User{
		"approver":                    {ID: "approver-1", Username: "approver"},
		domain.OfflineAccountUsername: {ID: "offline-user", Username: domain.OfflineAccountUsername},
	}}
	f := &offlineFixture{repo: newFakeOfflineRepo(), ledger: &fakeOfflineLedger{}, logos: &recordingLogos{set: map[string]string{}}}
	f.svc = NewOfflineDonationService(f.repo, initiatives, users, f.ledger)
	f.svc.now = func() time.Time { return time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC) }
	f.svc.SetOrgLogoRepo(f.logos)
	f.svc.SetB2BOrgResolver(b2bOrgResolverFunc(func(_ context.Context, d string) (models.B2BOrgMatch, error) {
		if d == "dell.com" {
			return models.B2BOrgMatch{SFID: "001DELL", LogoURL: "https://logo/dell.png"}, nil
		}
		return models.B2BOrgMatch{}, nil
	}))
	return f
}

func validOfflineInput() models.OfflineDonationInput {
	return models.OfflineDonationInput{
		InitiativeID: "init-1",
		Organization: &models.OrgClaim{Name: "Dell Corporation", Domain: "https://www.Dell.com/"},
		Amount:       "8000",
		ReceivedOn:   "2026-10-20",
		Category:     "Donation",
		SourceTxnID:  "SPDK Dell 2026-10",
	}
}

func TestOfflineDonationRecord_PostsTheManualLedgerRow(t *testing.T) {
	f := newOfflineFixture(t)

	got, created, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if !created || got.Status != models.DonationStatusSucceeded {
		t.Errorf("created=%v status=%q, want true/succeeded", created, got.Status)
	}
	stored := f.repo.bySource["SPDK Dell 2026-10"]
	if stored.UserID != "offline-user" || stored.RecordedByID != "approver-1" || stored.B2BOrgSFID != "001DELL" {
		t.Errorf("stored user=%q recorded_by=%q sfid=%q", stored.UserID, stored.RecordedByID, stored.B2BOrgSFID)
	}
	if len(f.ledger.posts) != 1 {
		t.Fatalf("ledger posts = %d, want 1", len(f.ledger.posts))
	}
	want := clients.LedgerTransaction{
		ProjectID:          "init-1",
		UserID:             "auth0|communitybridge",
		AccountEmail:       "admin@communitybridge.org",
		OrganizationName:   "Dell Corporation",
		OrganizationDomain: "dell.com",
		OrganizationSFID:   "001DELL",
		SourceType:         "expensify",
		SourceTxnID:        "SPDK Dell 2026-10",
		SourceAccountID:    "manual_from_lf",
		TxnType:            "credit",
		TxnCategory:        "Donation",
		Amount:             800000,
		TxnDate:            time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC).Unix(),
	}
	if f.ledger.posts[0] != want {
		t.Errorf("ledger txn =\n%+v\nwant\n%+v", f.ledger.posts[0], want)
	}
	if f.logos.set["001DELL"] != "https://logo/dell.png" {
		t.Errorf("logo not stored: %v", f.logos.set)
	}
}

func TestOfflineDonationRecord_ResubmitReturnsStoredWithoutSecondRow(t *testing.T) {
	f := newOfflineFixture(t)
	if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); err != nil {
		t.Fatalf("first Record: %v", err)
	}

	got, created, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
	if err != nil {
		t.Fatalf("second Record: %v", err)
	}
	if created || got.Status != models.DonationStatusSucceeded {
		t.Errorf("created=%v status=%q, want false/succeeded", created, got.Status)
	}
	if len(f.ledger.posts) != 1 {
		t.Errorf("ledger posts = %d, want 1 (no re-post of a succeeded donation)", len(f.ledger.posts))
	}
}

func TestOfflineDonationRecord_ReusedReferenceIsAConflict(t *testing.T) {
	f := newOfflineFixture(t)
	if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); err != nil {
		t.Fatalf("first Record: %v", err)
	}
	other := validOfflineInput()
	other.Amount = "500"

	_, _, err := f.svc.Record(context.Background(), "approver", other)
	if !errors.Is(err, ErrSourceTxnIDTaken) || !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, want ErrSourceTxnIDTaken (a conflict)", err)
	}
	if len(f.ledger.posts) != 1 {
		t.Errorf("ledger posts = %d, want 1", len(f.ledger.posts))
	}
}

func TestOfflineDonationRecord_ReferenceAlreadyInLedgerFreesTheRow(t *testing.T) {
	f := newOfflineFixture(t)
	// Posted by hand before CF recorded offline donations.
	f.ledger.have = map[string]bool{"SPDK Dell 2026-10": true}

	_, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
	if !errors.Is(err, ErrSourceTxnIDTaken) {
		t.Fatalf("err = %v, want ErrSourceTxnIDTaken", err)
	}
	if len(f.repo.deleted) != 1 || len(f.repo.succeeded) != 0 {
		t.Errorf("deleted=%v succeeded=%v, want the pending row deleted", f.repo.deleted, f.repo.succeeded)
	}
	if _, ok := f.repo.bySource["SPDK Dell 2026-10"]; ok {
		t.Error("reference still taken in CF")
	}
}

func TestOfflineDonationRecord_LedgerFailureKeepsPendingAndRetrySucceeds(t *testing.T) {
	f := newOfflineFixture(t)
	f.ledger.err = domain.ErrUpstreamUnavailable

	_, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
	if got := f.repo.bySource["SPDK Dell 2026-10"].Status; got != models.DonationStatusPending {
		t.Fatalf("status after failure = %q, want pending", got)
	}

	f.ledger.err = nil
	got, created, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if created || got.Status != models.DonationStatusSucceeded || len(f.ledger.posts) != 1 {
		t.Errorf("retry created=%v status=%q posts=%d, want false/succeeded/1", created, got.Status, len(f.ledger.posts))
	}
}

func TestOfflineDonationRecord_RetryAfterLostLedgerReplyIsNotADuplicate(t *testing.T) {
	f := newOfflineFixture(t)
	// The first post reached the Ledger but CF never marked the row succeeded.
	if _, _, err := f.repo.CreateOffline(context.Background(), &models.OfflineDonation{
		InitiativeID: "init-1", OrganizationDomain: "dell.com", AmountCents: 800000,
		ReceivedOn: "2026-10-20", SourceTxnID: "SPDK Dell 2026-10",
	}); err != nil {
		t.Fatal(err)
	}
	f.ledger.have = map[string]bool{"SPDK Dell 2026-10": true}

	got, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if got.Status != models.DonationStatusSucceeded || len(f.repo.deleted) != 0 {
		t.Errorf("status=%q deleted=%v, want succeeded and nothing deleted", got.Status, f.repo.deleted)
	}
}

func TestOfflineDonationRecord_LookupFailureStillRecords(t *testing.T) {
	f := newOfflineFixture(t)
	f.svc.SetB2BOrgResolver(b2bOrgResolverFunc(func(context.Context, string) (models.B2BOrgMatch, error) {
		return models.B2BOrgMatch{}, errors.New("nats timeout")
	}))

	if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if f.ledger.posts[0].OrganizationSFID != "" {
		t.Errorf("SFID = %q, want empty", f.ledger.posts[0].OrganizationSFID)
	}
}

func TestOfflineDonationRecord_Preconditions(t *testing.T) {
	t.Run("unpublished initiative", func(t *testing.T) {
		f := newOfflineFixture(t)
		f.svc.initiativeRepo.(*mockInitiativeRepo).initiative.Status = models.StatusPending
		if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); !errors.Is(err, domain.ErrInitiativeNotFound) {
			t.Errorf("err = %v, want ErrInitiativeNotFound", err)
		}
	})
	t.Run("approver without a profile", func(t *testing.T) {
		f := newOfflineFixture(t)
		delete(f.svc.userRepo.(*offlineUserRepo).users, "approver")
		if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); !errors.Is(err, domain.ErrProfileNotSynced) {
			t.Errorf("err = %v, want ErrProfileNotSynced", err)
		}
	})
	t.Run("offline account missing", func(t *testing.T) {
		f := newOfflineFixture(t)
		delete(f.svc.userRepo.(*offlineUserRepo).users, domain.OfflineAccountUsername)
		_, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
		if err == nil || errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrUserNotFound) {
			t.Errorf("err = %v, want an internal error (not a 404)", err)
		}
		if len(f.ledger.posts) != 0 {
			t.Error("posted to the ledger without a CF row")
		}
	})
}

func TestOfflineDonationRecord_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.OfflineDonationInput)
	}{
		{"no initiative", func(in *models.OfflineDonationInput) { in.InitiativeID = " " }},
		{"no organization", func(in *models.OfflineDonationInput) { in.Organization = nil }},
		{"bad domain", func(in *models.OfflineDonationInput) { in.Organization.Domain = "not a domain" }},
		{"zero amount", func(in *models.OfflineDonationInput) { in.Amount = "0.00" }},
		{"three decimals", func(in *models.OfflineDonationInput) { in.Amount = "10.005" }},
		{"thousands separator", func(in *models.OfflineDonationInput) { in.Amount = "1,300" }},
		{"negative", func(in *models.OfflineDonationInput) { in.Amount = "-5" }},
		{"bad date", func(in *models.OfflineDonationInput) { in.ReceivedOn = "10/20/2026" }},
		{"future date", func(in *models.OfflineDonationInput) { in.ReceivedOn = "2026-11-20" }},
		{"ancient date", func(in *models.OfflineDonationInput) { in.ReceivedOn = "0202-10-20" }},
		{"no category", func(in *models.OfflineDonationInput) { in.Category = "" }},
		{"no reference", func(in *models.OfflineDonationInput) { in.SourceTxnID = "  " }},
		{"control char in reference", func(in *models.OfflineDonationInput) { in.SourceTxnID = "a\nb" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOfflineFixture(t)
			in := validOfflineInput()
			tt.mutate(&in)
			if _, _, err := f.svc.Record(context.Background(), "approver", in); !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
			if len(f.ledger.posts) != 0 {
				t.Error("posted to the ledger")
			}
		})
	}
}

func TestDollarsToCents(t *testing.T) {
	tests := map[string]int64{"1300": 130000, "1300.5": 130050, "1300.05": 130005, " 0.01 ": 1, "9999999.99": 999999999}
	for in, want := range tests {
		got, err := dollarsToCents(in)
		if err != nil || got != want {
			t.Errorf("dollarsToCents(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
}
