// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

const acmeLogo = "https://cdn.example/b2b_org_logos/001A?v=1"

// fakeOrgLogos serves LogosByDomains from a map and records what was asked.
type fakeOrgLogos struct {
	domain.OrgLogoRepository
	logos map[string]string
	err   error
	asked [][]string
}

func (f *fakeOrgLogos) LogosByDomains(_ context.Context, domains []string) (map[string]string, error) {
	f.asked = append(f.asked, domains)
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]string{}
	for _, d := range domains {
		if l, ok := f.logos[d]; ok {
			out[d] = l
		}
	}
	return out, nil
}

func TestOrgLogos(t *testing.T) {
	r := &fakeOrgLogos{logos: map[string]string{"acme.com": acmeLogo}}
	got := orgLogos(context.Background(), r, []string{"acme.com", "", "acme.com", "other.org"})
	if got["acme.com"] != acmeLogo || len(got) != 1 {
		t.Errorf("logos = %v", got)
	}
	// One query, with empty and repeated domains dropped.
	if len(r.asked) != 1 || !slices.Equal(r.asked[0], []string{"acme.com", "other.org"}) {
		t.Errorf("asked = %v", r.asked)
	}

	if got := orgLogos(context.Background(), &fakeOrgLogos{err: errors.New("db down")}, []string{"acme.com"}); got != nil {
		t.Errorf("failed lookup = %v, want nil", got)
	}
	if got := orgLogos(context.Background(), nil, []string{"acme.com"}); got != nil {
		t.Errorf("nil reader = %v, want nil", got)
	}
	none := &fakeOrgLogos{}
	if orgLogos(context.Background(), none, []string{""}); len(none.asked) != 0 {
		t.Errorf("no domains still queried: %v", none.asked)
	}
}

func TestEnrichTransactionsFromDB_OrgLogo(t *testing.T) {
	txns := []models.Transaction{
		{ID: "t1", LedgerOrgName: "Acme Inc.", LedgerOrgDomain: "acme.com"},
		{ID: "t2", LedgerOrgName: "Other", LedgerOrgDomain: "other.org"},
		// The org stays visible on an anonymous row (#466), so does its logo.
		{ID: "t3", LedgerOrgName: "Acme Inc.", LedgerOrgDomain: "acme.com", Anonymous: true},
	}
	logos := &fakeOrgLogos{logos: map[string]string{"acme.com": acmeLogo}}

	enrichTransactionsFromDB(context.Background(), &mockRepoForEnrich{}, logos, txns, true)

	if txns[0].DonorLogoURL != acmeLogo || txns[2].DonorLogoURL != acmeLogo {
		t.Errorf("acme rows = %q, %q; want the b2b_org logo", txns[0].DonorLogoURL, txns[2].DonorLogoURL)
	}
	if want := generatedAvatarURL("other.org", "Other"); txns[1].DonorLogoURL != want {
		t.Errorf("unmatched org = %q, want generated %q", txns[1].DonorLogoURL, want)
	}
}

func TestGetRecentDonations_OrgLogo(t *testing.T) {
	ledger := &testLedgerClient{
		recentDonations: []clients.LedgerRecentDonation{
			{TxnID: "txn-1", ProjectID: "proj-1", OrganizationName: "Acme Inc.", OrganizationDomain: "acme.com", Amount: 5_000, SourceType: domain.LedgerSourceTypeStripe},
			{TxnID: "txn-2", ProjectID: "proj-1", OrganizationName: "Other", OrganizationDomain: "other.org", Amount: 5_000, SourceType: domain.LedgerSourceTypeStripe},
		},
	}
	svc := newStatsSvc(&testStatisticsRepo{users: map[string]models.User{}}, ledger)
	svc.SetOrgLogoRepo(&fakeOrgLogos{logos: map[string]string{"acme.com": acmeLogo}})

	resp, err := svc.GetRecentDonations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := resp.Data[0].DonorAvatarURL; got != acmeLogo {
		t.Errorf("matched org avatar = %q, want %q", got, acmeLogo)
	}
	if want := generatedAvatarURL("other.org", "Other"); resp.Data[1].DonorAvatarURL != want {
		t.Errorf("unmatched org avatar = %q, want %q", resp.Data[1].DonorAvatarURL, want)
	}
}

func TestGetPlatformDetails_OrgLogo(t *testing.T) {
	ledger := &testLedgerClient{
		platformBalance: &clients.LedgerPlatformBalance{
			TopOrganizations: []clients.LedgerSponsorRaw{
				{ID: "acme.com", Name: "Acme Inc.", Total: 300},
				{ID: "c64448e2-0000-0000-0000-000000000000", Name: "Legacy Org", Total: 200},
			},
		},
	}
	svc := newStatsSvc(&testStatisticsRepo{}, ledger)
	svc.SetOrgLogoRepo(&fakeOrgLogos{logos: map[string]string{"acme.com": acmeLogo}})

	details, err := svc.GetPlatformDetails(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := details.TopOrganizations[0].AvatarURL; got != acmeLogo {
		t.Errorf("acme avatar = %q, want %q", got, acmeLogo)
	}
	// No logo leaves the avatar empty, and the frontend shows initials.
	if got := details.TopOrganizations[1].AvatarURL; got != "" {
		t.Errorf("legacy bucket avatar = %q, want empty", got)
	}
}

func TestProjectDonationSummaries_OrgLogo(t *testing.T) {
	donations := []models.Donation{
		{ID: "d1", UserID: "u1", OrgName: "Acme Inc.", OrgDomain: "acme.com"},
		{ID: "d2", UserID: "u1", OrgName: "Other", OrgDomain: "other.org"},
	}
	logos := &fakeOrgLogos{logos: map[string]string{"acme.com": acmeLogo}}

	got := projectDonationSummaries(context.Background(), &summaryInitiativeRepo{}, logos, donations)

	if got[0].DonorAvatarURL != acmeLogo {
		t.Errorf("matched org avatar = %q, want %q", got[0].DonorAvatarURL, acmeLogo)
	}
	if got[1].DonorAvatarURL != "" {
		t.Errorf("unmatched org avatar = %q, want empty", got[1].DonorAvatarURL)
	}
}
