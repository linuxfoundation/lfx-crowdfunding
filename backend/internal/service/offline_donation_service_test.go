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
// locked tracks LockOfflineReference, so tests can check what ran under it.
type fakeOfflineRepo struct {
	bySource  map[string]*models.OfflineDonation
	succeeded []string
	deleted   []string
	locked    map[string]bool
	unlocks   int
}

func newFakeOfflineRepo() *fakeOfflineRepo {
	return &fakeOfflineRepo{bySource: map[string]*models.OfflineDonation{}, locked: map[string]bool{}}
}

func (r *fakeOfflineRepo) LockOfflineReference(_ context.Context, ref string) (func(), error) {
	if r.locked[ref] {
		return nil, errors.New("fake: reference already locked")
	}
	r.locked[ref] = true
	return func() { r.locked[ref] = false; r.unlocks++ }, nil
}

func (r *fakeOfflineRepo) SetOfflineB2BOrg(_ context.Context, id, sfid string) error {
	for _, d := range r.bySource {
		if d.ID == id && d.Status == models.DonationStatusPending && d.B2BOrgSFID == "" {
			d.B2BOrgSFID = sfid
		}
	}
	return nil
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
	for _, d := range r.bySource {
		if d.ID == id && d.Status == models.DonationStatusPending {
			d.Status = models.DonationStatusSucceeded
			r.succeeded = append(r.succeeded, id)
			return nil
		}
	}
	return errors.New("fake: no pending offline donation " + id)
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
// it already holds, like the Ledger does, and keeps the first row per
// sourceTxnID for FindOfflineTransaction.
type fakeOfflineLedger struct {
	posts  []clients.LedgerTransaction
	have   map[string]bool
	rows   map[string]clients.LedgerTransaction
	err    error
	onPost func(clients.LedgerTransaction)
}

func (l *fakeOfflineLedger) PostOfflineTransaction(_ context.Context, txn clients.LedgerTransaction) (bool, error) {
	if l.onPost != nil {
		l.onPost(txn)
	}
	if l.err != nil {
		return false, l.err
	}
	l.posts = append(l.posts, txn)
	if l.have == nil {
		l.have = map[string]bool{}
	}
	dup := l.have[txn.SourceTxnID]
	l.have[txn.SourceTxnID] = true
	if !dup {
		l.addRow(txn)
	}
	return dup, nil
}

// addRow stores txn as the Ledger's row for its sourceTxnID (e.g. a Postman entry).
func (l *fakeOfflineLedger) addRow(txn clients.LedgerTransaction) {
	if l.rows == nil {
		l.rows = map[string]clients.LedgerTransaction{}
	}
	if l.have == nil {
		l.have = map[string]bool{}
	}
	l.rows[txn.SourceTxnID] = txn
	l.have[txn.SourceTxnID] = true
}

func (l *fakeOfflineLedger) FindOfflineTransaction(_ context.Context, projectID, ref string) (*models.Transaction, error) {
	txn, ok := l.rows[ref]
	if !ok || txn.ProjectID != projectID {
		return nil, nil
	}
	return &models.Transaction{
		AmountCents: int64(txn.Amount), LedgerProjectID: txn.ProjectID,
		LedgerOrgDomain: txn.OrganizationDomain, LedgerSourceTxnID: txn.SourceTxnID,
	}, nil
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
		if d == "example.com" {
			return models.B2BOrgMatch{SFID: "001EXAMPLE", LogoURL: "https://logo.example.com/example.png"}, nil
		}
		return models.B2BOrgMatch{}, nil
	}))
	return f
}

func validOfflineInput() models.OfflineDonationInput {
	return models.OfflineDonationInput{
		InitiativeID: "init-1",
		Organization: &models.OrgClaim{Name: "Example Corp", Domain: "https://www.Example.com/"},
		Amount:       "1000",
		ReceivedOn:   "2026-10-20",
		Category:     "Donation",
		SourceTxnID:  "TEST-REF-001",
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
	stored := f.repo.bySource["TEST-REF-001"]
	if stored.UserID != "offline-user" || stored.RecordedByID != "approver-1" || stored.B2BOrgSFID != "001EXAMPLE" {
		t.Errorf("stored user=%q recorded_by=%q sfid=%q", stored.UserID, stored.RecordedByID, stored.B2BOrgSFID)
	}
	if len(f.ledger.posts) != 1 {
		t.Fatalf("ledger posts = %d, want 1", len(f.ledger.posts))
	}
	want := clients.LedgerTransaction{
		ProjectID:          "init-1",
		UserID:             "auth0|communitybridge",
		AccountEmail:       "admin@communitybridge.org",
		OrganizationName:   "Example Corp",
		OrganizationDomain: "example.com",
		OrganizationSFID:   "001EXAMPLE",
		SourceType:         "expensify",
		SourceTxnID:        "TEST-REF-001",
		SourceAccountID:    "manual_from_lf",
		TxnType:            "credit",
		TxnCategory:        "Donation",
		Amount:             100000,
		TxnDate:            time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC).Unix(),
	}
	if f.ledger.posts[0] != want {
		t.Errorf("ledger txn =\n%+v\nwant\n%+v", f.ledger.posts[0], want)
	}
	if f.logos.set["001EXAMPLE"] != "https://logo.example.com/example.png" {
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
	f.ledger.addRow(clients.LedgerTransaction{ProjectID: "init-1", SourceTxnID: "TEST-REF-001", Amount: 100000})

	_, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
	if !errors.Is(err, ErrSourceTxnIDTaken) {
		t.Fatalf("err = %v, want ErrSourceTxnIDTaken", err)
	}
	if len(f.repo.deleted) != 1 || len(f.repo.succeeded) != 0 {
		t.Errorf("deleted=%v succeeded=%v, want the pending row deleted", f.repo.deleted, f.repo.succeeded)
	}
	if _, ok := f.repo.bySource["TEST-REF-001"]; ok {
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
	if got := f.repo.bySource["TEST-REF-001"].Status; got != models.DonationStatusPending {
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
		InitiativeID: "init-1", OrganizationName: "Example Corp", OrganizationDomain: "example.com",
		AmountCents: 100000, Category: "Donation", ReceivedOn: "2026-10-20", SourceTxnID: "TEST-REF-001",
	}); err != nil {
		t.Fatal(err)
	}
	f.ledger.addRow(clients.LedgerTransaction{
		ProjectID: "init-1", OrganizationDomain: "example.com", Amount: 100000, SourceTxnID: "TEST-REF-001",
	})

	got, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if got.Status != models.DonationStatusSucceeded || len(f.repo.deleted) != 0 {
		t.Errorf("status=%q deleted=%v, want succeeded and nothing deleted", got.Status, f.repo.deleted)
	}
}

func TestOfflineDonationRecord_ResubmitWithChangedDetailsIsAConflict(t *testing.T) {
	for name, mutate := range map[string]func(*models.OfflineDonationInput){
		"category":          func(in *models.OfflineDonationInput) { in.Category = "general fund" },
		"organization name": func(in *models.OfflineDonationInput) { in.Organization.Name = "Example Corp Foundation" },
	} {
		t.Run(name, func(t *testing.T) {
			f := newOfflineFixture(t)
			if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); err != nil {
				t.Fatalf("first Record: %v", err)
			}
			changed := validOfflineInput()
			mutate(&changed)
			if _, _, err := f.svc.Record(context.Background(), "approver", changed); !errors.Is(err, ErrSourceTxnIDTaken) {
				t.Fatalf("err = %v, want ErrSourceTxnIDTaken", err)
			}
			if len(f.ledger.posts) != 1 {
				t.Errorf("ledger posts = %d, want 1", len(f.ledger.posts))
			}
		})
	}
}

func TestOfflineDonationRecord_PendingRetryWithChangedCategoryIsAConflict(t *testing.T) {
	f := newOfflineFixture(t)
	f.ledger.err = domain.ErrUpstreamUnavailable
	if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("first Record err = %v", err)
	}
	f.ledger.err = nil
	changed := validOfflineInput()
	changed.Category = "general fund"

	if _, _, err := f.svc.Record(context.Background(), "approver", changed); !errors.Is(err, ErrSourceTxnIDTaken) {
		t.Fatalf("err = %v, want ErrSourceTxnIDTaken", err)
	}
	if len(f.ledger.posts) != 0 {
		t.Errorf("posted %d rows with the stored category, want none", len(f.ledger.posts))
	}
}

func TestOfflineDonationRecord_RetryUsesARecoveredOrgMatch(t *testing.T) {
	f := newOfflineFixture(t)
	resolver := f.svc.b2bOrgResolver
	f.svc.SetB2BOrgResolver(b2bOrgResolverFunc(func(context.Context, string) (models.B2BOrgMatch, error) {
		return models.B2BOrgMatch{}, errors.New("nats timeout")
	}))
	f.ledger.err = domain.ErrUpstreamUnavailable
	if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("first Record err = %v", err)
	}

	f.svc.SetB2BOrgResolver(resolver)
	f.ledger.err = nil
	if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if got := f.ledger.posts[0].OrganizationSFID; got != "001EXAMPLE" {
		t.Errorf("ledger SFID = %q, want the retry's match 001EXAMPLE", got)
	}
	if got := f.repo.bySource["TEST-REF-001"].B2BOrgSFID; got != "001EXAMPLE" {
		t.Errorf("stored SFID = %q, want 001EXAMPLE", got)
	}
}

func TestOfflineDonationRecord_RetryDoesNotAdoptADifferentLedgerRow(t *testing.T) {
	for name, row := range map[string]*clients.LedgerTransaction{
		"different amount": {ProjectID: "init-1", OrganizationDomain: "example.com", Amount: 50000, SourceTxnID: "TEST-REF-001"},
		"different org":    {ProjectID: "init-1", OrganizationDomain: "example.org", Amount: 100000, SourceTxnID: "TEST-REF-001"},
		"other initiative": {ProjectID: "init-2", OrganizationDomain: "example.com", Amount: 100000, SourceTxnID: "TEST-REF-001"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newOfflineFixture(t)
			// The first attempt failed before reaching the Ledger...
			f.ledger.err = domain.ErrUpstreamUnavailable
			if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); !errors.Is(err, domain.ErrUpstreamUnavailable) {
				t.Fatalf("first Record err = %v", err)
			}
			// ...and someone posted the same reference by hand meanwhile.
			f.ledger.err = nil
			f.ledger.addRow(*row)

			_, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput())
			if !errors.Is(err, ErrLedgerRowUnverified) || !errors.Is(err, domain.ErrConflict) {
				t.Fatalf("err = %v, want ErrLedgerRowUnverified (a conflict)", err)
			}
			if got := f.repo.bySource["TEST-REF-001"].Status; got != models.DonationStatusPending {
				t.Errorf("status = %q, want pending", got)
			}
			if len(f.repo.deleted) != 0 || len(f.repo.succeeded) != 0 {
				t.Errorf("deleted=%v succeeded=%v, want the row left alone", f.repo.deleted, f.repo.succeeded)
			}
		})
	}
}

func TestOfflineDonationRecord_PostsUnderTheReferenceLock(t *testing.T) {
	f := newOfflineFixture(t)
	lockedAtPost := false
	f.ledger.onPost = func(txn clients.LedgerTransaction) { lockedAtPost = f.repo.locked[txn.SourceTxnID] }

	if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if !lockedAtPost {
		t.Error("ledger post ran without the reference lock")
	}
	if f.repo.locked["TEST-REF-001"] || f.repo.unlocks != 1 {
		t.Errorf("locked=%v unlocks=%d, want released once", f.repo.locked["TEST-REF-001"], f.repo.unlocks)
	}
}

func TestOfflineDonationRecord_FinalizeOfAMissingRowFails(t *testing.T) {
	f := newOfflineFixture(t)
	// Another request removed the pending row between this one's post and finalize.
	f.ledger.onPost = func(txn clients.LedgerTransaction) { delete(f.repo.bySource, txn.SourceTxnID) }

	if _, _, err := f.svc.Record(context.Background(), "approver", validOfflineInput()); err == nil {
		t.Fatal("Record succeeded with no CF row behind it")
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
