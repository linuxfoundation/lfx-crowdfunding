// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

func TestDonorPrivacyRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.subscriptions", "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "privacy-owner")
	donor := seedUser(t, ctx, "privacy-donor")
	other := seedUser(t, ctx, "privacy-other")
	for _, u := range []struct{ id, legacy string }{{donor.ID, "auth0|donor"}, {other.ID, "auth0|other"}} {
		if _, err := testPool.Exec(ctx, "UPDATE crowdfunding.users SET legacy_user_id = $2 WHERE id = $1", u.id, u.legacy); err != nil {
			t.Fatalf("set legacy_user_id: %v", err)
		}
	}
	first := seedInitiative(t, ctx, owner.ID, "Privacy One", "privacy-one")
	second := seedInitiative(t, ctx, owner.ID, "Privacy Two", "privacy-two")

	donationRepo := NewDonationRepository(testPool)
	hidden, err := donationRepo.Create(ctx, &models.Donation{
		UserID: donor.ID, InitiativeID: first.ID, CurrentAmountCents: 1000, Status: models.DonationStatusSucceeded,
		StripePaymentIntentID: "pi_hidden", StripeChargeID: "ch_hidden", Anonymous: true,
	})
	if err != nil {
		t.Fatalf("create hidden donation: %v", err)
	}
	if !hidden.Anonymous {
		t.Fatalf("Create did not return anonymous = true")
	}
	shown, err := donationRepo.Create(ctx, &models.Donation{
		UserID: donor.ID, InitiativeID: second.ID, CurrentAmountCents: 1000, Status: models.DonationStatusSucceeded,
		StripePaymentIntentID: "pi_shown",
	})
	if err != nil {
		t.Fatalf("create shown donation: %v", err)
	}
	sub, err := NewSubscriptionRepository(testPool).Create(ctx, &models.Subscription{
		UserID: other.ID, InitiativeID: second.ID, CurrentAmountCents: 500, Frequency: "monthly",
		Status: models.SubscriptionStatusActive, StripeSubscriptionID: "sub_hidden", Anonymous: true,
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	if !sub.Anonymous {
		t.Fatalf("subscription Create did not return anonymous = true")
	}

	repo := NewDonorPrivacyRepository(testPool)

	t.Run("MatchLedgerRefs matches charge, PaymentIntent and subscription IDs", func(t *testing.T) {
		m, err := repo.MatchLedgerRefs(ctx, []string{"ch_hidden", "pi_shown", "ch_unknown"}, []string{"sub_hidden", "sub_unknown"})
		if err != nil {
			t.Fatalf("MatchLedgerRefs: %v", err)
		}
		if got := m.Donations["ch_hidden"]; got.ID != hidden.ID || !got.Anonymous {
			t.Errorf("ch_hidden = %+v", got)
		}
		if got := m.Donations["pi_shown"]; got.ID != shown.ID || got.Anonymous {
			t.Errorf("pi_shown = %+v", got)
		}
		if _, ok := m.Donations["ch_unknown"]; ok {
			t.Error("ch_unknown matched")
		}
		if got := m.Subscriptions["sub_hidden"]; got.ID != sub.ID || !got.Anonymous {
			t.Errorf("sub_hidden = %+v", got)
		}
		if _, ok := m.Subscriptions["sub_unknown"]; ok {
			t.Error("sub_unknown matched")
		}
	})

	t.Run("AnonymousDonors groups by initiative", func(t *testing.T) {
		got, err := repo.AnonymousDonors(ctx, []string{first.ID, second.ID})
		if err != nil {
			t.Fatalf("AnonymousDonors: %v", err)
		}
		if !got[first.ID]["auth0|donor"] || got[second.ID]["auth0|donor"] {
			t.Errorf("donor: %v", got)
		}
		if !got[second.ID]["auth0|other"] || got[first.ID]["auth0|other"] {
			t.Errorf("other: %v", got)
		}
	})

	t.Run("AnonymousLedgerUsers spans initiatives", func(t *testing.T) {
		got, err := repo.AnonymousLedgerUsers(ctx, []string{"auth0|donor", "auth0|other", "auth0|nobody"})
		if err != nil {
			t.Fatalf("AnonymousLedgerUsers: %v", err)
		}
		if !got["auth0|donor"] || !got["auth0|other"] || got["auth0|nobody"] {
			t.Errorf("got %v", got)
		}
	})

	t.Run("Is*Anonymous", func(t *testing.T) {
		for _, c := range []struct {
			name string
			fn   func() (bool, error)
			want bool
		}{
			{"pi_hidden", func() (bool, error) { return repo.IsPaymentIntentAnonymous(ctx, "pi_hidden") }, true},
			{"pi_shown", func() (bool, error) { return repo.IsPaymentIntentAnonymous(ctx, "pi_shown") }, false},
			{"pi_missing", func() (bool, error) { return repo.IsPaymentIntentAnonymous(ctx, "pi_missing") }, false},
			{"sub_hidden", func() (bool, error) { return repo.IsStripeSubscriptionAnonymous(ctx, "sub_hidden") }, true},
		} {
			got, err := c.fn()
			if err != nil || got != c.want {
				t.Errorf("%s = %v, %v; want %v", c.name, got, err, c.want)
			}
		}
	})

	t.Run("Set*Anonymous only changes the caller's own rows", func(t *testing.T) {
		if err := repo.SetDonationAnonymous(ctx, shown.ID, other.ID, true); !errors.Is(err, domain.ErrDonationNotFound) {
			t.Errorf("other user's donation: err = %v, want ErrDonationNotFound", err)
		}
		if err := repo.SetDonationAnonymous(ctx, uuid.NewString(), donor.ID, true); !errors.Is(err, domain.ErrDonationNotFound) {
			t.Errorf("missing donation: err = %v, want ErrDonationNotFound", err)
		}
		if err := repo.SetDonationAnonymous(ctx, shown.ID, donor.ID, true); err != nil {
			t.Fatalf("SetDonationAnonymous: %v", err)
		}
		got, err := donationRepo.GetByID(ctx, shown.ID)
		if err != nil || !got.Anonymous {
			t.Errorf("after set: %+v, %v", got, err)
		}

		if err := repo.SetSubscriptionAnonymous(ctx, sub.ID, donor.ID, false); !errors.Is(err, domain.ErrSubscriptionNotFound) {
			t.Errorf("other user's subscription: err = %v, want ErrSubscriptionNotFound", err)
		}
		if err := repo.SetSubscriptionAnonymous(ctx, sub.ID, other.ID, false); err != nil {
			t.Fatalf("SetSubscriptionAnonymous: %v", err)
		}
		gotSub, err := NewSubscriptionRepository(testPool).GetByIDForUser(ctx, sub.ID, other.ID)
		if err != nil || gotSub.Anonymous {
			t.Errorf("after unset: %+v, %v", gotSub, err)
		}
	})
}

func TestDonationRepository_ListByB2BOrg_ReturnsAnonymous(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.subscriptions", "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "anon-lens-owner")
	donor := seedUser(t, ctx, "anon-lens-donor")
	initiative := seedInitiative(t, ctx, owner.ID, "Anon Lens", "anon-lens")
	repo := NewDonationRepository(testPool)
	const sfid = "001000000000009AAA"

	d, err := repo.Create(ctx, &models.Donation{
		UserID: donor.ID, InitiativeID: initiative.ID, CurrentAmountCents: 1000,
		Status: models.DonationStatusSucceeded, Anonymous: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.SetB2BOrgSFID(ctx, d.ID, sfid); err != nil {
		t.Fatalf("SetB2BOrgSFID: %v", err)
	}
	rows, _, err := repo.ListByB2BOrg(ctx, sfid, 20, 0)
	if err != nil {
		t.Fatalf("ListByB2BOrg: %v", err)
	}
	if len(rows) != 1 || !rows[0].Anonymous {
		t.Fatalf("rows = %+v, want one anonymous row", rows)
	}
}
