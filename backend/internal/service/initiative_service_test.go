// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	stripe "github.com/stripe/stripe-go/v82"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// --- mocks ---

type mockInitiativeRepo struct {
	initiative *models.Initiative
	err        error
}

func (m *mockInitiativeRepo) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return m.initiative, m.err
}
func (m *mockInitiativeRepo) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return m.initiative, m.err
}
func (m *mockInitiativeRepo) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (m *mockInitiativeRepo) Create(_ context.Context, i *models.Initiative) (*models.Initiative, error) {
	return i, nil
}
func (m *mockInitiativeRepo) Update(_ context.Context, i *models.Initiative) (*models.Initiative, error) {
	return i, nil
}
func (m *mockInitiativeRepo) Delete(_ context.Context, _ string) error { return nil }

type mockLedgerClient struct {
	balance *clients.LedgerBalance
	err     error
}

func (m *mockLedgerClient) GetBalance(_ context.Context, _ string) (*clients.LedgerBalance, error) {
	return m.balance, m.err
}
func (m *mockLedgerClient) GetAllBalances(_ context.Context) ([]models.LedgerRawBalance, error) {
	return nil, nil
}

type mockStripeClient struct{}

func (m *mockStripeClient) GetProduct(_ context.Context, _ string) (*models.StripeProduct, error) {
	return nil, nil
}
func (m *mockStripeClient) CreatePaymentIntent(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	return nil, nil
}
func (m *mockStripeClient) CreateSubscription(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	return nil, nil
}
func (m *mockStripeClient) CancelSubscription(_ context.Context, _ string) error { return nil }
func (m *mockStripeClient) ConstructWebhookEvent(_ []byte, _, _ string) (stripe.Event, error) {
	return stripe.Event{}, nil
}

// --- T006: TestGetByID_LedgerSuccess ---

func TestGetByID_LedgerSuccess(t *testing.T) {
	initiative := &models.Initiative{
		ID: "test-id",
		Goals: []models.Goal{
			{Name: "Member Stipends"},
		},
	}
	balance := &clients.LedgerBalance{
		TotalRaisedCents:    7000000,
		TotalDisbursedCents: 2000000,
		AvailableCents:      5000000,
		SubTotals: map[string]*clients.LedgerSubTotal{
			"Member Stipends": {Credit: 1800000, Debit: -900000},
		},
	}

	svc := NewInitiativeService(
		&mockInitiativeRepo{initiative: initiative},
		&mockLedgerClient{balance: balance},
		&mockStripeClient{},
	)

	result, err := svc.GetByID(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Balance == nil {
		t.Fatal("expected Balance to be set, got nil")
	}
	if len(result.Goals) == 0 {
		t.Fatal("expected goals to be present")
	}
	goal := result.Goals[0]
	if goal.DonatedCents == nil || *goal.DonatedCents != 1800000 {
		t.Errorf("expected DonatedCents=1800000, got %v", goal.DonatedCents)
	}
	if goal.SpentCents == nil || *goal.SpentCents != -900000 {
		t.Errorf("expected SpentCents=-900000, got %v", goal.SpentCents)
	}
}

// --- T007: TestGetByID_LedgerFailure ---

func TestGetByID_LedgerFailure(t *testing.T) {
	initiative := &models.Initiative{
		ID: "test-id",
		Goals: []models.Goal{
			{Name: "Member Stipends"},
		},
	}

	svc := NewInitiativeService(
		&mockInitiativeRepo{initiative: initiative},
		&mockLedgerClient{err: errors.New("ledger unavailable")},
		&mockStripeClient{},
	)

	result, err := svc.GetByID(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("expected no error on Ledger failure, got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil initiative on Ledger failure")
	}
	if result.Balance != nil {
		t.Errorf("expected Balance=nil on Ledger failure, got %+v", result.Balance)
	}
	for _, g := range result.Goals {
		if g.DonatedCents != nil {
			t.Errorf("expected DonatedCents=nil on Ledger failure, got %v", *g.DonatedCents)
		}
		if g.SpentCents != nil {
			t.Errorf("expected SpentCents=nil on Ledger failure, got %v", *g.SpentCents)
		}
	}
}

// --- T008: TestGetByID_GoalNameMatchingCaseInsensitive ---

func TestGetByID_GoalNameMatchingCaseInsensitive(t *testing.T) {
	initiative := &models.Initiative{
		ID: "test-id",
		Goals: []models.Goal{
			{Name: "Development"},
		},
	}
	balance := &clients.LedgerBalance{
		SubTotals: map[string]*clients.LedgerSubTotal{
			"development": {Credit: 500000, Debit: 0},
		},
	}

	svc := NewInitiativeService(
		&mockInitiativeRepo{initiative: initiative},
		&mockLedgerClient{balance: balance},
		&mockStripeClient{},
	)

	result, err := svc.GetByID(context.Background(), "test-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Goals) == 0 {
		t.Fatal("expected goals")
	}
	if result.Goals[0].DonatedCents == nil || *result.Goals[0].DonatedCents != 500000 {
		t.Errorf("expected case-insensitive match: DonatedCents=500000, got %v", result.Goals[0].DonatedCents)
	}
}

// --- T012: TestFlattenSponsors ---

func TestFlattenSponsors(t *testing.T) {
	list := models.LedgerSponsorList{
		Orgs: []models.LedgerSponsorOrg{
			{ID: "org-1", Name: "Big Corp", Total: 3_000_000},
			{ID: "org-2", Name: "Small Corp", Total: 500_000},
		},
		Individuals: []models.LedgerSponsorUser{
			{ID: "user-1", Name: "Top Donor", Total: 15_000_000},
		},
	}

	result := flattenSponsors(list)

	if len(result) != 3 {
		t.Fatalf("expected 3 sponsors, got %d", len(result))
	}
	// sorted descending by total
	if result[0].ID != "user-1" {
		t.Errorf("expected user-1 first (highest total), got %s", result[0].ID)
	}
	if result[1].ID != "org-1" {
		t.Errorf("expected org-1 second, got %s", result[1].ID)
	}
	if result[2].ID != "org-2" {
		t.Errorf("expected org-2 third, got %s", result[2].ID)
	}
}

// --- T012b: TestFlattenSponsors_EmptyList ---

func TestFlattenSponsors_EmptyList(t *testing.T) {
	result := flattenSponsors(models.LedgerSponsorList{})

	if result == nil {
		t.Error("result must be non-nil (must serialise as [] not null)")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(result))
	}
}
