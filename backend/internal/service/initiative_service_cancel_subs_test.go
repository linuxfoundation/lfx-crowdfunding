// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	stripe "github.com/stripe/stripe-go/v85"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// cancelSubsFixture wires an InitiativeService with three subscriptions on the
// initiative (active, past_due, already canceled) and records Stripe cancels
// and DB status writes.
type cancelSubsFixture struct {
	svc       *InitiativeService
	repo      *mockInitiativeRepo
	cancelled []string
	marked    map[string]string
	subs      []models.Subscription
}

func newCancelSubsFixture(status models.InitiativeStatus, stripeErr error) *cancelSubsFixture {
	f := &cancelSubsFixture{
		repo:   &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", OwnerID: "owner-1", Status: status}},
		marked: map[string]string{},
	}
	f.subs = []models.Subscription{
		{ID: "a", StripeSubscriptionID: "sub_a", Status: models.SubscriptionStatusActive},
		{ID: "b", StripeSubscriptionID: "sub_b", Status: models.SubscriptionStatusPastDue},
		{ID: "c", StripeSubscriptionID: "sub_c", Status: models.SubscriptionStatusCanceled},
	}
	subRepo := &testSubscriptionRepo{
		onListByInitiative: func(_ context.Context, _ string, _ models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
			return f.subs, nil, nil
		},
		onUpdate: func(_ context.Context, s *models.Subscription) (*models.Subscription, error) {
			f.marked[s.ID] = s.Status
			return s, nil
		},
	}
	stripeClient := &configStripeClient{onCancelSubscription: func(_ context.Context, id string) error {
		f.cancelled = append(f.cancelled, id)
		return stripeErr
	}}
	f.svc = NewInitiativeService(f.repo, &mockUserRepository{}, &mockLedgerClient{}, stripeClient, &mockEmailService{}, nil, slog.Default())
	f.svc.SetSubscriptionRepo(subRepo)
	return f
}

func (f *cancelSubsFixture) wantCancelled(t *testing.T, want int) {
	t.Helper()
	if len(f.cancelled) != want {
		t.Fatalf("stripe cancels = %v, want %d", f.cancelled, want)
	}
	for _, id := range []string{"a", "b"}[:min(want, 2)] {
		if f.marked[id] != models.SubscriptionStatusCanceled {
			t.Errorf("subscription %s status = %q, want canceled", id, f.marked[id])
		}
	}
	if _, touched := f.marked["c"]; touched {
		t.Error("already-canceled subscription was rewritten")
	}
}

func TestProcessApproval_DeclineCancelsSubscriptions(t *testing.T) {
	f := newCancelSubsFixture(models.StatusSubmitted, nil)
	if _, err := f.svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionDecline); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.wantCancelled(t, 2)
}

func TestProcessApproval_ApproveKeepsSubscriptions(t *testing.T) {
	f := newCancelSubsFixture(models.StatusSubmitted, nil)
	if _, err := f.svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionApprove); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.wantCancelled(t, 0)
}

func TestUpdate_HideCancelsSubscriptions(t *testing.T) {
	f := newCancelSubsFixture(models.StatusPublished, nil)
	hidden := models.StatusHidden
	if _, err := f.svc.Update(context.Background(), "init-1", "owner-1", models.InitiativeUpdateInput{Status: &hidden}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.wantCancelled(t, 2)
}

func TestUpdate_OwnerDeclineCancelsSubscriptions(t *testing.T) {
	f := newCancelSubsFixture(models.StatusPending, nil)
	declined := models.StatusDeclined
	if _, err := f.svc.Update(context.Background(), "init-1", "owner-1", models.InitiativeUpdateInput{Status: &declined}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.wantCancelled(t, 2)
}

func TestUpdate_UnhideKeepsSubscriptions(t *testing.T) {
	f := newCancelSubsFixture(models.StatusHidden, nil)
	published := models.StatusPublished
	if _, err := f.svc.Update(context.Background(), "init-1", "owner-1", models.InitiativeUpdateInput{Status: &published}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.wantCancelled(t, 0)
}

func TestProcessApproval_DeclineAbortsWhenStripeFails(t *testing.T) {
	f := newCancelSubsFixture(models.StatusSubmitted, errors.New("stripe down"))
	if _, err := f.svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionDecline); err == nil {
		t.Fatal("expected error")
	}
	if f.repo.lastUpdated != nil {
		t.Error("status was written despite failed cancellation")
	}
}

func TestProcessApproval_DeclineToleratesMissingStripeSubscription(t *testing.T) {
	missing := &stripe.Error{HTTPStatusCode: 404, Code: stripe.ErrorCodeResourceMissing}
	f := newCancelSubsFixture(models.StatusSubmitted, missing)
	if _, err := f.svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionDecline); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.wantCancelled(t, 2)
}

func TestProcessApproval_DeclineSkipsStripeForSubscriptionWithoutStripeID(t *testing.T) {
	f := newCancelSubsFixture(models.StatusSubmitted, nil)
	f.subs = []models.Subscription{{ID: "a", Status: models.SubscriptionStatusActive}}
	if _, err := f.svc.ProcessApproval(context.Background(), "init-1", models.ApprovalActionDecline); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.cancelled) != 0 {
		t.Errorf("stripe cancels = %v, want none", f.cancelled)
	}
	if f.marked["a"] != models.SubscriptionStatusCanceled {
		t.Errorf("subscription status = %q, want canceled", f.marked["a"])
	}
}
