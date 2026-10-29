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
		UserID:                owner.ID,
		InitiativeID:          initiative.ID,
		CurrentAmountCents:    5000,
		Status:                models.DonationStatusPending,
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

	// A second SFID must not overwrite the first one.
	for _, sfid := range []string{"001000000000001AAA", "001000000000002AAA"} {
		if err := repo.SetB2BOrgSFID(ctx, created.ID, sfid); err != nil {
			t.Fatalf("SetB2BOrgSFID(%q) error = %v", sfid, err)
		}
	}

	// Update to succeeded with chargeID
	sfid, err := repo.UpdateByPaymentIntentID(ctx, "pi_update_test", models.DonationStatusSucceeded, "ch_test_456")
	if err != nil {
		t.Fatalf("UpdateByPaymentIntentID() error = %v", err)
	}
	if sfid != "001000000000001AAA" {
		t.Errorf("UpdateByPaymentIntentID() sfid = %q, want the first stored SFID", sfid)
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
	_, err = repo.UpdateByPaymentIntentID(ctx, "pi_update_test", models.DonationStatusSucceeded, "ch_test_456")
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
	_, err := repo.UpdateByPaymentIntentID(ctx, "pi_nonexistent", models.DonationStatusSucceeded, "ch_test")
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

func TestDonationRepository_Create_ReplayedPaymentIntentReturnsExistingRow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-replay")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative Replay", "test-initiative-replay")
	repo := NewDonationRepository(testPool)

	newDonation := func() *models.Donation {
		return &models.Donation{
			UserID:                owner.ID,
			InitiativeID:          initiative.ID,
			CurrentAmountCents:    5000,
			Status:                models.DonationStatusPending,
			StripePaymentIntentID: "pi_replay",
		}
	}
	first, err := repo.Create(ctx, newDonation())
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	replay, err := repo.Create(ctx, newDonation())
	if err != nil {
		t.Fatalf("replayed Create() error = %v", err)
	}
	if replay.ID != first.ID {
		t.Errorf("replayed Create() ID = %q, want existing %q", replay.ID, first.ID)
	}

	// Rows without a PaymentIntent are not deduplicated.
	seedDonation(t, ctx, owner.ID, initiative.ID)
	seedDonation(t, ctx, owner.ID, initiative.ID)

	var n int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM crowdfunding.donations`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 3 {
		t.Errorf("donation rows = %d, want 3", n)
	}
}
