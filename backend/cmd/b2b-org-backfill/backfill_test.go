// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stripe/stripe-go/v85"
)

type fakeStore struct {
	domains   []string
	subs      map[string][]unmatchedSub
	donations map[string]string // domain -> sfid
	subSFIDs  map[string]string // subscription id -> sfid
}

func (s *fakeStore) UnmatchedDomains(_ context.Context) ([]string, error) { return s.domains, nil }
func (s *fakeStore) SetDonationsSFID(_ context.Context, domain, sfid string) (int64, error) {
	s.donations[domain] = sfid
	return 2, nil
}
func (s *fakeStore) UnmatchedSubscriptions(_ context.Context, domain string) ([]unmatchedSub, error) {
	return s.subs[domain], nil
}
func (s *fakeStore) SetSubscriptionSFID(_ context.Context, id, sfid string) error {
	s.subSFIDs[id] = sfid
	return nil
}

type fakeResolver map[string]string // domain -> sfid; "ERR" means a failed lookup

func (r fakeResolver) LookupB2BOrgByWebsite(_ context.Context, domain string) (string, error) {
	if r[domain] == "ERR" {
		return "", errors.New("b2b org lookup failed")
	}
	return r[domain], nil
}

type fakeStripe struct {
	errs    map[string]error // stripe subscription id -> error
	patched []string
}

func (s *fakeStripe) UpdateSubscriptionMetadata(_ context.Context, subID, _ string, md map[string]string) error {
	if err := s.errs[subID]; err != nil {
		return err
	}
	s.patched = append(s.patched, subID+"="+md["b2b_org_sfid"])
	return nil
}

func TestBackfillerRun(t *testing.T) {
	st := &fakeStore{
		domains: []string{"acme.com", "failed.org", "nomatch.org"},
		subs: map[string][]unmatchedSub{"acme.com": {
			{ID: "live", Status: "active", StripeSubscriptionID: "sub_live", StripeSubscriptionItemID: "si_live"},
			{ID: "canceled", Status: "canceled", StripeSubscriptionID: "sub_canceled"},
			{ID: "stale", Status: "active", StripeSubscriptionID: "sub_stale"},
			{ID: "flaky", Status: "active", StripeSubscriptionID: "sub_flaky"},
		}},
		donations: map[string]string{},
		subSFIDs:  map[string]string{},
	}
	sc := &fakeStripe{errs: map[string]error{
		"sub_stale": fmt.Errorf("stripe update subscription metadata: %w", &stripe.Error{HTTPStatusCode: http.StatusBadRequest}),
		"sub_flaky": fmt.Errorf("stripe update subscription metadata: %w", &stripe.Error{HTTPStatusCode: http.StatusInternalServerError}),
	}}
	b := &backfiller{
		store:  st,
		lookup: fakeResolver{"acme.com": "001A", "failed.org": "ERR"},
		stripe: sc,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	res, err := b.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if res.domains != 3 || res.matched != 1 || res.noMatch != 1 || res.lookupFailed != 1 {
		t.Errorf("lookups = %+v", res)
	}
	if len(st.donations) != 1 || st.donations["acme.com"] != "001A" {
		t.Errorf("donations filled = %v, want only acme.com", st.donations)
	}
	if len(sc.patched) != 1 || sc.patched[0] != "sub_live=001A" {
		t.Errorf("stripe patched = %v, want only sub_live", sc.patched)
	}
	// The canceled and stale subscriptions still get the SFID; the flaky one is left for the next run.
	want := map[string]string{"live": "001A", "canceled": "001A", "stale": "001A"}
	if fmt.Sprint(st.subSFIDs) != fmt.Sprint(want) {
		t.Errorf("subscriptions filled = %v, want %v", st.subSFIDs, want)
	}
	if res.stripePatched != 1 || res.stripeSkipped != 2 || res.stripeFailed != 1 {
		t.Errorf("stripe counts = %+v", res)
	}
}

func TestBackfillerRun_DryRunWritesNothing(t *testing.T) {
	st := &fakeStore{domains: []string{"acme.com"}, donations: map[string]string{}, subSFIDs: map[string]string{}}
	b := &backfiller{
		store:  st,
		lookup: fakeResolver{"acme.com": "001A"},
		stripe: &fakeStripe{},
		dryRun: true,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	res, err := b.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.matched != 1 || len(st.donations) != 0 || len(st.subSFIDs) != 0 {
		t.Errorf("dry run: matched %d, wrote %v / %v", res.matched, st.donations, st.subSFIDs)
	}
}
