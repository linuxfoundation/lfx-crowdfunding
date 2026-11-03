// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

type b2bOrgResolverFunc func(ctx context.Context, domain string) (string, error)

func (f b2bOrgResolverFunc) LookupB2BOrgByWebsite(ctx context.Context, domain string) (string, error) {
	return f(ctx, domain)
}

func TestLookupAndStoreB2BOrg(t *testing.T) {
	tests := []struct {
		name      string
		sfid      string
		lookupErr error
		wantStore string
	}{
		{name: "match is stored", sfid: "0012M00002qnukOQAQ", wantStore: "0012M00002qnukOQAQ"},
		{name: "no match stores nothing"},
		{name: "lookup error stores nothing", lookupErr: errors.New("b2b org lookup failed")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := b2bOrgResolverFunc(func(_ context.Context, domain string) (string, error) {
				if domain != "acme.com" {
					t.Errorf("domain = %q, want acme.com", domain)
				}
				return tt.sfid, tt.lookupErr
			})
			var stored string
			lookupAndStoreB2BOrg(context.Background(), r, "donation", "d1", "acme.com", func(_ context.Context, sfid string) error {
				stored = sfid
				return nil
			})
			if stored != tt.wantStore {
				t.Errorf("stored = %q, want %q", stored, tt.wantStore)
			}
		})
	}
}

// waitSFID waits for the background lookup to store an SFID.
func waitSFID(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case sfid := <-ch:
		return sfid
	case <-time.After(2 * time.Second):
		t.Fatal("b2b_org SFID was never stored")
		return ""
	}
}

func TestDonationService_Create_ResolvesB2BOrg(t *testing.T) {
	stored := make(chan string, 1)
	donRepo := &testDonationRepo{
		onCreate: func(_ context.Context, d *models.Donation) (*models.Donation, error) {
			d.ID = "don-1"
			return d, nil
		},
		onSetB2BOrgSFID: func(_ context.Context, id, sfid string) error {
			if id != "don-1" {
				t.Errorf("SetB2BOrgSFID id = %q, want don-1", id)
			}
			stored <- sfid
			return nil
		},
	}
	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, _, _ string) (string, error) { return "cus_1", nil },
		onCreatePaymentIntent: func(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			return &models.PaymentIntent{ID: "pi_1", Status: "succeeded"}, nil
		},
	}
	svc := newDonationSvc(donRepo, acceptingInitiative(), &testUserRepo{}, stripe)
	svc.SetB2BOrgResolver(b2bOrgResolverFunc(func(_ context.Context, _ string) (string, error) {
		return "0012M00002qnukOQAQ", nil
	}))

	_, err := svc.Create(context.Background(), "init-1", "u1", models.DonationCreateInput{
		AmountCents: 1000, StripePaymentMethodID: "pm_test", IdempotencyKey: "idem",
		Organization: &models.OrgClaim{Name: "Acme", Domain: "acme.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := waitSFID(t, stored); got != "0012M00002qnukOQAQ" {
		t.Errorf("stored SFID = %q", got)
	}
}

func TestSubscriptionService_Create_ResolvesB2BOrg(t *testing.T) {
	stored := make(chan string, 1)
	var metaSub, metaItem, metaSFID string
	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, _, _ string) (string, error) { return "cus_1", nil },
		onGetOrCreatePrice: func(_ context.Context, _, _ string, _ int64, _ string, _ string) (string, error) {
			return "price_1", nil
		},
		onCreateSubscription: func(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
			return &models.StripeSubscriptionResult{SubscriptionID: "sub_1", SubscriptionItemID: "si_1", PriceID: "price_1", Status: "active"}, nil
		},
		onUpdateSubscriptionMetadata: func(_ context.Context, subID, itemID string, md map[string]string) error {
			metaSub, metaItem, metaSFID = subID, itemID, md["b2b_org_sfid"]
			return nil
		},
	}
	subRepo := &testSubscriptionRepo{
		onCreate: func(_ context.Context, s *models.Subscription) (*models.Subscription, error) {
			s.ID = "sub-row-1"
			return s, nil
		},
		onSetB2BOrgSFID: func(_ context.Context, _, sfid string) error {
			stored <- sfid
			return nil
		},
	}
	svc := newSubscriptionSvc(subRepo, acceptingInitiative(), &testUserRepo{}, stripe)
	svc.SetB2BOrgResolver(b2bOrgResolverFunc(func(_ context.Context, _ string) (string, error) {
		return "0012M00002qnukOQAQ", nil
	}))

	_, err := svc.Create(context.Background(), "init-1", "u1", models.SubscriptionCreateInput{
		AmountCents: 2000, Frequency: "monthly", StripePaymentMethodID: "pm_test", IdempotencyKey: "idem",
		Organization: &models.OrgClaim{Name: "Acme", Domain: "acme.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	waitSFID(t, stored)
	// The Stripe patch runs before the DB write, so it has finished by now.
	if metaSub != "sub_1" || metaItem != "si_1" || metaSFID != "0012M00002qnukOQAQ" {
		t.Errorf("Stripe metadata patch = %q / %q / %q", metaSub, metaItem, metaSFID)
	}
}
