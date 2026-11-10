// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// A user with no Auth0 subject (legacy_user_id) would be charged but never
// reach the Ledger, so payments stop before Stripe is called (#468).
func TestPaymentsRequireLedgerUserID(t *testing.T) {
	noSubject := &testUserRepo{
		onGetByUsername: func(_ context.Context, u string) (*models.User, error) {
			return &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: u, Email: u + "@test.example"}, nil
		},
	}
	stripeCalled := false
	stripe := &configStripeClient{
		onCreateCustomer: func(context.Context, string, string) (string, error) {
			stripeCalled = true
			return "cus_new", nil
		},
		onCreatePaymentIntent: func(context.Context, models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			stripeCalled = true
			return &models.PaymentIntent{ID: "pi_x", Status: "succeeded"}, nil
		},
	}
	ctx := context.Background()

	cases := map[string]func() error{
		"donation": func() error {
			_, err := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), noSubject, stripe).Create(ctx, "init-1", "u1",
				models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_x", IdempotencyKey: "k"})
			return err
		},
		"subscription": func() error {
			_, err := NewSubscriptionService(&testSubscriptionRepo{}, acceptingInitiative(), noSubject, stripe).Create(ctx, "init-1", "u1",
				models.SubscriptionCreateInput{AmountCents: 1000, Frequency: "monthly", StripePaymentMethodID: "pm_x", IdempotencyKey: "k"})
			return err
		},
		"saved card": func() error {
			_, err := NewPaymentService(noSubject, stripe).CreateSetupIntent(ctx, "u1")
			return err
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			stripeCalled = false
			if err := run(); !errors.Is(err, domain.ErrProfileNotSynced) {
				t.Errorf("err = %v, want ErrProfileNotSynced", err)
			}
			if stripeCalled {
				t.Error("Stripe was called for a user with no legacy_user_id")
			}
		})
	}
}
