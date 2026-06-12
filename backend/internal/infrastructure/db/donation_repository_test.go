// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// seedDonation creates a minimal pending donation and returns it.
func seedDonation(t *testing.T, ctx context.Context, userID, initiativeID string) *models.Donation { //nolint:revive // t first is Go test convention
	t.Helper()
	donationRepo := NewDonationRepository(testPool)
	donation := &models.Donation{
		UserID:             userID,
		InitiativeID:       initiativeID,
		CurrentAmountCents: 5000,
		Status:             models.DonationStatusPending,
	}

	created, err := donationRepo.Create(ctx, donation)
	if err != nil {
		t.Fatalf("seedDonation: %v", err)
	}
	return created
}

func TestDonationRepository_CreateAndGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-donation")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative", "test-initiative")

	repo := NewDonationRepository(testPool)

	donation := &models.Donation{
		UserID:               owner.ID,
		InitiativeID:         initiative.ID,
		CurrentAmountCents:   5000,
		Status:               models.DonationStatusPending,
		StripePaymentIntentID: "pi_test_123",
	}

	created, err := repo.Create(ctx, donation)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create() returned empty ID")
	}

	// GetByID round-trip
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.CurrentAmountCents != 5000 {
		t.Errorf("CurrentAmountCents = %d, want %d", got.CurrentAmountCents, 5000)
	}
	if got.StripePaymentIntentID != "pi_test_123" {
		t.Errorf("StripePaymentIntentID = %q, want %q", got.StripePaymentIntentID, "pi_test_123")
	}
	if got.Status != models.DonationStatusPending {
		t.Errorf("Status = %q, want %q", got.Status, models.DonationStatusPending)
	}
}

func TestDonationRepository_UpdateByPaymentIntentID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-update")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative Update", "test-initiative-update")

	repo := NewDonationRepository(testPool)

	// Create a pending donation with known piID
	donation := &models.Donation{
		UserID:                owner.ID,
		InitiativeID:          initiative.ID,
		CurrentAmountCents:    5000,
		Status:                models.DonationStatusPending,
		StripePaymentIntentID: "pi_update_test",
	}

	created, err := repo.Create(ctx, donation)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update to succeeded with chargeID
	err = repo.UpdateByPaymentIntentID(ctx, "pi_update_test", models.DonationStatusSucceeded, "ch_test_456")
	if err != nil {
		t.Fatalf("UpdateByPaymentIntentID() error = %v", err)
	}

	// Verify the update
	updated, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if updated.Status != models.DonationStatusSucceeded {
		t.Errorf("Status = %q, want %q", updated.Status, models.DonationStatusSucceeded)
	}
	if updated.StripeChargeID != "ch_test_456" {
		t.Errorf("StripeChargeID = %q, want %q", updated.StripeChargeID, "ch_test_456")
	}

	// Call again with same args → expect ErrAlreadyProcessed (idempotency)
	err = repo.UpdateByPaymentIntentID(ctx, "pi_update_test", models.DonationStatusSucceeded, "ch_test_456")
	if err == nil {
		t.Fatal("UpdateByPaymentIntentID() expected ErrAlreadyProcessed, got nil")
	}
	if !isNotFound(err, domain.ErrAlreadyProcessed) {
		t.Errorf("error = %v, want ErrAlreadyProcessed", err)
	}
}

func TestDonationRepository_UpdateByPaymentIntentID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	repo := NewDonationRepository(testPool)

	// Call with nonexistent piID
	err := repo.UpdateByPaymentIntentID(ctx, "pi_nonexistent", models.DonationStatusSucceeded, "ch_test")
	if err == nil {
		t.Fatal("UpdateByPaymentIntentID() expected ErrDonationNotFound, got nil")
	}
	if !isNotFound(err, domain.ErrDonationNotFound) {
		t.Errorf("error = %v, want ErrDonationNotFound", err)
	}
}

func TestDonationRepository_ListByInitiative_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-pagination-donation")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative Pagination", "test-initiative-pagination")

	repo := NewDonationRepository(testPool)

	// Create 3 donations
	seedDonation(t, ctx, owner.ID, initiative.ID)
	seedDonation(t, ctx, owner.ID, initiative.ID)
	seedDonation(t, ctx, owner.ID, initiative.ID)

	// List with Limit=2
	filter := models.DonationFilter{
		Limit:  2,
		Offset: 0,
	}

	donations, meta, err := repo.ListByInitiative(ctx, initiative.ID, filter)
	if err != nil {
		t.Fatalf("ListByInitiative() error = %v", err)
	}

	if len(donations) != 2 {
		t.Errorf("got %d items, want 2", len(donations))
	}
	if meta.Total != 3 {
		t.Errorf("meta.Total = %d, want 3", meta.Total)
	}
	if meta.Limit != 2 {
		t.Errorf("meta.Limit = %d, want 2", meta.Limit)
	}
	if meta.Offset != 0 {
		t.Errorf("meta.Offset = %d, want 0", meta.Offset)
	}
}
