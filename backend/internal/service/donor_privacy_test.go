// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// fakeDonorPrivacy is an in-memory domain.DonorPrivacyRepository.
type fakeDonorPrivacy struct {
	matches      models.LedgerRefMatches
	byInitiative map[string]map[string]bool
	anywhere     map[string]bool
	err          error

	setDonation     func(id, userID string, anonymous bool) error
	setSubscription func(id, userID string, anonymous bool) error
}

func (f *fakeDonorPrivacy) MatchLedgerRefs(_ context.Context, _, _ []string) (*models.LedgerRefMatches, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &f.matches, nil
}
func (f *fakeDonorPrivacy) AnonymousDonors(_ context.Context, _ []string) (map[string]map[string]bool, error) {
	return f.byInitiative, f.err
}
func (f *fakeDonorPrivacy) AnonymousLedgerUsers(_ context.Context, _ []string) (map[string]bool, error) {
	return f.anywhere, f.err
}
func (f *fakeDonorPrivacy) IsPaymentIntentAnonymous(_ context.Context, _ string) (bool, error) {
	return false, f.err
}
func (f *fakeDonorPrivacy) IsStripeSubscriptionAnonymous(_ context.Context, _ string) (bool, error) {
	return false, f.err
}
func (f *fakeDonorPrivacy) SetDonationAnonymous(_ context.Context, id, userID string, anonymous bool) error {
	return f.setDonation(id, userID, anonymous)
}
func (f *fakeDonorPrivacy) SetSubscriptionAnonymous(_ context.Context, id, userID string, anonymous bool) error {
	return f.setSubscription(id, userID, anonymous)
}

var _ domain.DonorPrivacyRepository = (*fakeDonorPrivacy)(nil)

func privacyTxns() []models.Transaction {
	return []models.Transaction{
		{ID: "one-time", LedgerUserID: "u1", LedgerSourceTxnID: "ch_1"},
		{ID: "recurring", LedgerUserID: "u2", LedgerSourceTxnID: "ch_2", LedgerSubscriptionID: "sub_1"},
		{ID: "public", LedgerUserID: "u3", LedgerSourceTxnID: "ch_3"},
		{ID: "offline", LedgerUserID: "u4"},
	}
}

func privacyMatches() models.LedgerRefMatches {
	return models.LedgerRefMatches{
		Donations: map[string]models.LedgerRef{
			"ch_1": {ID: "don-1", Anonymous: true},
			"ch_3": {ID: "don-3"},
		},
		Subscriptions: map[string]models.LedgerRef{"sub_1": {ID: "sub-cf-1", Anonymous: true}},
	}
}

func TestMarkPublicLedgerRows_FlagsMatchedRowsWithoutIDs(t *testing.T) {
	txns := privacyTxns()
	markPublicLedgerRows(context.Background(), &fakeDonorPrivacy{matches: privacyMatches()}, txns)

	want := map[string]bool{"one-time": true, "recurring": true, "public": false, "offline": false}
	for _, tx := range txns {
		if tx.Anonymous != want[tx.ID] {
			t.Errorf("%s: Anonymous = %v, want %v", tx.ID, tx.Anonymous, want[tx.ID])
		}
		if tx.DonationID != "" || tx.SubscriptionID != "" {
			t.Errorf("%s: public row carries CF IDs %q/%q", tx.ID, tx.DonationID, tx.SubscriptionID)
		}
	}
}

func TestMarkPublicLedgerRows_LookupFailureHidesEveryone(t *testing.T) {
	txns := privacyTxns()
	markPublicLedgerRows(context.Background(), &fakeDonorPrivacy{err: errors.New("db down")}, txns)
	for _, tx := range txns {
		if !tx.Anonymous {
			t.Errorf("%s: not hidden after a failed lookup", tx.ID)
		}
	}
}

func TestMarkOwnLedgerRows_AttachesToggleIDs(t *testing.T) {
	txns := privacyTxns()
	markOwnLedgerRows(context.Background(), &fakeDonorPrivacy{matches: privacyMatches()}, txns)

	if txns[0].DonationID != "don-1" || !txns[0].Anonymous {
		t.Errorf("one-time row = %+v", txns[0])
	}
	// A recurring row is matched by its subscription, not its charge.
	if txns[1].SubscriptionID != "sub-cf-1" || txns[1].DonationID != "" || !txns[1].Anonymous {
		t.Errorf("recurring row = %+v", txns[1])
	}
	if txns[2].DonationID != "don-3" || txns[2].Anonymous {
		t.Errorf("public row = %+v", txns[2])
	}
	if txns[3].DonationID != "" || txns[3].SubscriptionID != "" {
		t.Errorf("offline row = %+v", txns[3])
	}
}

func TestEnrichTransactionsFromDB_MasksAnonymousDonors(t *testing.T) {
	repo := &mockRepoForEnrich{
		users: map[string]models.User{"u1": {Name: "Alice", AvatarURL: "https://example.com/alice.png"}},
		orgs:  map[string]models.Organization{},
	}
	txns := []models.Transaction{
		{ID: "person", LedgerUserID: "u1", DonorName: "Alice (submitter)", Anonymous: true},
		{ID: "stated-org", LedgerUserID: "u1", LedgerOrgName: "Acme", LedgerOrgDomain: "acme.com", Anonymous: true},
		{ID: "unknown-org", LedgerUserID: "u1", LedgerOrgID: "org-gone", DonorName: "Alice (submitter)", Anonymous: true},
		{ID: "no-ids", DonorName: "Alice (submitter)", Anonymous: true},
	}
	enrichTransactionsFromDB(context.Background(), repo, txns, true)

	for _, tx := range txns {
		if tx.DonorName == "Alice" || tx.DonorName == "Alice (submitter)" || tx.DonorLogoURL == "https://example.com/alice.png" {
			t.Errorf("%s: anonymous donor shown: %q %q", tx.ID, tx.DonorName, tx.DonorLogoURL)
		}
	}
	if txns[0].DonorName != anonymousName || txns[0].DonorLogoURL != anonymousAvatarURL() {
		t.Errorf("person = %q %q", txns[0].DonorName, txns[0].DonorLogoURL)
	}
	if txns[1].DonorName != "Acme" {
		t.Errorf("stated org lost its name: %q", txns[1].DonorName)
	}
	if txns[2].DonorName != anonymousName || txns[3].DonorName != anonymousName {
		t.Errorf("fallbacks = %q, %q", txns[2].DonorName, txns[3].DonorName)
	}
}

func TestEnrichTransactionsFromDB_OwnRowsKeepName(t *testing.T) {
	repo := &mockRepoForEnrich{users: map[string]models.User{"u1": {Name: "Alice"}}, orgs: map[string]models.Organization{}}
	txns := []models.Transaction{{ID: "t1", LedgerUserID: "u1", Anonymous: true}}
	enrichTransactionsFromDB(context.Background(), repo, txns, false)
	if txns[0].DonorName != "Alice" || !txns[0].Anonymous {
		t.Errorf("own row = %+v", txns[0])
	}
}

func TestSetSponsors_HidesAnonymousIndividualsPerInitiative(t *testing.T) {
	sponsors := models.LedgerSponsorList{
		Orgs:        []models.LedgerSponsorOrg{{ID: "acme.com", Name: "Acme", Total: 900}},
		Individuals: []models.LedgerSponsorUser{{ID: "u1", Name: "Alice", Total: 500}, {ID: "u2", Name: "Bob", Total: 100}},
	}
	first := &models.Initiative{ID: "i1", RawSponsors: sponsors}
	second := &models.Initiative{ID: "i2", RawSponsors: sponsors}
	svc := &InitiativeService{privacy: &fakeDonorPrivacy{byInitiative: map[string]map[string]bool{"i1": {"u1": true}}}}

	svc.setSponsors(context.Background(), first, second)

	names := func(i *models.Initiative) []string {
		out := []string{}
		for _, s := range i.Sponsors {
			out = append(out, s.Name)
		}
		return out
	}
	if got := names(first); len(got) != 3 || got[0] != "Acme" || got[1] != anonymousName || got[2] != "Bob" {
		t.Errorf("i1 sponsors = %v", got)
	}
	if first.Sponsors[1].TotalCents != 500 || first.Sponsors[1].AvatarURL != anonymousAvatarURL() {
		t.Errorf("hidden sponsor = %+v", first.Sponsors[1])
	}
	if got := names(second); got[1] != "Alice" {
		t.Errorf("i2 sponsors = %v (Alice is only anonymous on i1)", got)
	}
}

func TestSetSponsors_LookupFailureHidesIndividuals(t *testing.T) {
	i := &models.Initiative{ID: "i1", RawSponsors: models.LedgerSponsorList{
		Orgs:        []models.LedgerSponsorOrg{{ID: "acme.com", Name: "Acme", Total: 900}},
		Individuals: []models.LedgerSponsorUser{{ID: "u1", Name: "Alice", Total: 500}},
	}}
	svc := &InitiativeService{privacy: &fakeDonorPrivacy{err: errors.New("db down")}}
	svc.setSponsors(context.Background(), i)
	if i.Sponsors[0].Name != "Acme" || i.Sponsors[1].Name != anonymousName {
		t.Errorf("sponsors = %+v", i.Sponsors)
	}
}

func TestGetRecentDonations_HidesAnonymousDonors(t *testing.T) {
	repo := &testStatisticsRepo{
		orgs:         map[string]models.Organization{},
		users:        map[string]models.User{"u1": {Name: "Alice", AvatarURL: "https://example.com/alice.png"}},
		projectNames: map[string]string{"p1": "One", "p2": "Two"},
	}
	ledger := &testLedgerClient{recentDonations: []clients.LedgerRecentDonation{
		{TxnID: "individual", ProjectID: "p1", UserID: "u1", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
		{TxnID: "other-initiative", ProjectID: "p2", UserID: "u1", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
		{TxnID: "stated-org", ProjectID: "p1", UserID: "u1", OrganizationName: "Acme", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
		{TxnID: "legacy-org", ProjectID: "p1", UserID: "u1", OrganizationID: "org-gone", SubmitterName: "Alice", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
	}}
	svc := newStatsSvc(repo, ledger)
	svc.SetDonorPrivacyRepo(&fakeDonorPrivacy{byInitiative: map[string]map[string]bool{"p1": {"u1": true}}})

	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("GetRecentDonations: %v", err)
	}
	got := map[string]models.RecentDonation{}
	for _, d := range resp.Data {
		got[d.TxnID] = d
	}
	if d := got["individual"]; d.DonorName != anonymousName || d.DonorAvatarURL != anonymousAvatarURL() || d.AmountCents != 100 {
		t.Errorf("individual = %+v", d)
	}
	if d := got["other-initiative"]; d.DonorName != "Alice" {
		t.Errorf("other initiative = %q, want Alice (not anonymous there)", d.DonorName)
	}
	if d := got["stated-org"]; d.DonorName != "Acme" {
		t.Errorf("stated org = %q", d.DonorName)
	}
	if d := got["legacy-org"]; d.DonorName != anonymousName {
		t.Errorf("legacy org fell back to %q", d.DonorName)
	}
}

func TestGetRecentDonations_LookupFailureHidesRowsWithoutUserID(t *testing.T) {
	repo := &testStatisticsRepo{
		orgs:         map[string]models.Organization{},
		users:        map[string]models.User{},
		projectNames: map[string]string{"p1": "One"},
	}
	ledger := &testLedgerClient{recentDonations: []clients.LedgerRecentDonation{
		{TxnID: "no-user", ProjectID: "p1", SubmitterName: "Carol", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
		{TxnID: "no-user-org", ProjectID: "p1", OrganizationID: "org-gone", SubmitterName: "Dave", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
	}}
	svc := newStatsSvc(repo, ledger)
	svc.SetDonorPrivacyRepo(&fakeDonorPrivacy{err: errors.New("db down")})

	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("GetRecentDonations: %v", err)
	}
	for _, d := range resp.Data {
		if d.DonorName != anonymousName {
			t.Errorf("%s = %q, want %q on a failed lookup", d.TxnID, d.DonorName, anonymousName)
		}
	}
}

func TestGetRecentDonations_HidesDonorsOfDeletedInitiatives(t *testing.T) {
	repo := &testStatisticsRepo{
		orgs:         map[string]models.Organization{},
		users:        map[string]models.User{"u1": {Name: "Alice"}},
		projectNames: map[string]string{"live": "Live"},
	}
	ledger := &testLedgerClient{recentDonations: []clients.LedgerRecentDonation{
		{TxnID: "deleted", ProjectID: "gone", UserID: "u1", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
		{TxnID: "deleted-org", ProjectID: "gone", OrganizationID: "org-gone", SubmitterName: "Alice", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
		{TxnID: "deleted-stated-org", ProjectID: "gone", UserID: "u1", OrganizationName: "Acme", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
		{TxnID: "live", ProjectID: "live", UserID: "u1", Amount: 100, SourceType: domain.LedgerSourceTypeStripe},
	}}
	svc := newStatsSvc(repo, ledger)
	// The deleted initiative's anonymous rows no longer match, so the lookup finds nothing.
	svc.SetDonorPrivacyRepo(&fakeDonorPrivacy{byInitiative: map[string]map[string]bool{}})

	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("GetRecentDonations: %v", err)
	}
	got := map[string]string{}
	for _, d := range resp.Data {
		got[d.TxnID] = d.DonorName
	}
	want := map[string]string{"deleted": anonymousName, "deleted-org": anonymousName, "deleted-stated-org": "Acme", "live": "Alice"}
	for id, name := range want {
		if got[id] != name {
			t.Errorf("%s = %q, want %q", id, got[id], name)
		}
	}
}

func TestGetPlatformDetails_HidesAnonymousTopIndividuals(t *testing.T) {
	repo := &testStatisticsRepo{users: map[string]models.User{
		"u1": {Name: "Alice"}, "u2": {Name: "Bob"},
	}}
	ledger := &testLedgerClient{platformBalance: &clients.LedgerPlatformBalance{
		TopIndividuals: []clients.LedgerSponsorRaw{{ID: "u1", Total: 500}, {ID: "u2", Total: 100}},
	}}
	svc := newStatsSvc(repo, ledger)
	svc.SetDonorPrivacyRepo(&fakeDonorPrivacy{anywhere: map[string]bool{"u1": true}})

	details, err := svc.GetPlatformDetails(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetPlatformDetails: %v", err)
	}
	if details.TopIndividuals[0].Name != anonymousName || details.TopIndividuals[0].TotalCents != 500 {
		t.Errorf("top[0] = %+v", details.TopIndividuals[0])
	}
	if details.TopIndividuals[1].Name != "Bob" {
		t.Errorf("top[1] = %+v", details.TopIndividuals[1])
	}
}

func TestDonationService_ListByB2BOrg_HidesAnonymousDonor(t *testing.T) {
	donRepo := &testDonationRepo{
		onListByB2BOrg: func(context.Context, string, int, int) ([]models.OrgLensDonation, *models.PaginationMeta, error) {
			return []models.OrgLensDonation{
				{ID: "hidden", DonorName: "Alice", DonorEmail: "alice@example.com", Anonymous: true, AmountCents: 100},
				{ID: "shown", DonorName: "Bob", DonorEmail: "bob@example.com"},
			}, &models.PaginationMeta{Total: 2}, nil
		},
	}
	svc := newDonationSvc(donRepo, &mockInitiativeRepo{}, &testUserRepo{}, &configStripeClient{})
	rows, _, err := svc.ListByB2BOrg(context.Background(), "001000000000001AAA", 20, 0)
	if err != nil {
		t.Fatalf("ListByB2BOrg: %v", err)
	}
	if rows[0].DonorName != anonymousName || rows[0].DonorEmail != "" || rows[0].AmountCents != 100 {
		t.Errorf("hidden row = %+v", rows[0])
	}
	if rows[1].DonorName != "Bob" || rows[1].DonorEmail != "bob@example.com" {
		t.Errorf("shown row = %+v", rows[1])
	}
}

func TestDonationService_ListOrgDonations_HidesAnonymousDonor(t *testing.T) {
	donRepo := &testDonationRepo{
		onListOrgDonations: func(context.Context) ([]models.OrgDonationRow, error) {
			return []models.OrgDonationRow{{OrganizationName: "Acme", DonorUserID: "u1", DonorName: "Alice", Anonymous: true}}, nil
		},
	}
	svc := newDonationSvc(donRepo, &mockInitiativeRepo{}, &testUserRepo{}, &configStripeClient{})
	rows, err := svc.ListOrgDonations(context.Background())
	if err != nil {
		t.Fatalf("ListOrgDonations: %v", err)
	}
	if rows[0].DonorName != anonymousName || rows[0].DonorUserID != "" || rows[0].OrganizationName != "Acme" {
		t.Errorf("row = %+v", rows[0])
	}
}

func TestProjectDonationSummaries_HidesAnonymousIndividual(t *testing.T) {
	repo := &summaryInitiativeRepo{
		onGetUsersByIDs: func(context.Context, []string) (map[string]models.User, error) {
			return map[string]models.User{"u1": {Name: "Alice"}}, nil
		},
	}
	got := projectDonationSummaries(context.Background(), repo, []models.Donation{
		{ID: "hidden", UserID: "u1", Anonymous: true},
		{ID: "org", UserID: "u1", OrgName: "Acme", Anonymous: true},
	})
	if got[0].DonorName != anonymousName || !got[0].Anonymous {
		t.Errorf("hidden = %+v", got[0])
	}
	if got[1].DonorName != "Acme" {
		t.Errorf("org = %+v", got[1])
	}
}

func TestDonationService_SetAnonymous(t *testing.T) {
	const id = "6f1b0b8e-3a52-4c2e-9a49-1f3f5d6b7c80"
	var gotID, gotUser string
	var gotAnon bool
	svc := newDonationSvc(&testDonationRepo{}, &mockInitiativeRepo{}, &testUserRepo{}, &configStripeClient{})
	svc.SetDonorPrivacyRepo(&fakeDonorPrivacy{setDonation: func(id, userID string, anonymous bool) error {
		gotID, gotUser, gotAnon = id, userID, anonymous
		return nil
	}})

	state, err := svc.SetAnonymous(context.Background(), id, "alice", true)
	if err != nil {
		t.Fatalf("SetAnonymous: %v", err)
	}
	if gotID != id || gotUser != "00000000-0000-0000-0000-000000000001" || !gotAnon || !state.Anonymous || state.ID != id {
		t.Errorf("stored (%s, %s, %v), returned %+v", gotID, gotUser, gotAnon, state)
	}

	if _, err := svc.SetAnonymous(context.Background(), "not-a-uuid", "alice", true); !errors.Is(err, domain.ErrDonationNotFound) {
		t.Errorf("bad id: err = %v, want ErrDonationNotFound", err)
	}
}

func TestSubscriptionService_SetAnonymous_NotOwned(t *testing.T) {
	svc := NewSubscriptionService(&testSubscriptionRepo{}, &mockInitiativeRepo{}, &testUserRepo{}, &configStripeClient{})
	svc.SetDonorPrivacyRepo(&fakeDonorPrivacy{setSubscription: func(string, string, bool) error {
		return domain.ErrSubscriptionNotFound
	}})
	_, err := svc.SetAnonymous(context.Background(), "6f1b0b8e-3a52-4c2e-9a49-1f3f5d6b7c80", "alice", false)
	if !errors.Is(err, domain.ErrSubscriptionNotFound) {
		t.Errorf("err = %v, want ErrSubscriptionNotFound", err)
	}
}
