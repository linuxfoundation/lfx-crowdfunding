// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// seedSubscription creates a subscription for a user and initiative, returning it.
func seedSubscription(t *testing.T, ctx context.Context, userID, initiativeID, status, stripeSubID string) *models.Subscription { //nolint:revive // t first is Go test convention
	t.Helper()
	subRepo := NewSubscriptionRepository(testPool)

	sub := &models.Subscription{
		UserID:               userID,
		InitiativeID:         initiativeID,
		CurrentAmountCents:   5000,
		Frequency:            "monthly",
		Status:               status,
		StripeSubscriptionID: stripeSubID,
	}

	created, err := subRepo.Create(ctx, sub)
	if err != nil {
		t.Fatalf("seedSubscription: %v", err)
	}
	return created
}

func TestSubscriptionRepository_CreateAndGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.subscriptions", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "sub-owner-1")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative", "test-initiative")
	repo := NewSubscriptionRepository(testPool)

	sub := &models.Subscription{
		UserID:               owner.ID,
		InitiativeID:         initiative.ID,
		CurrentAmountCents:   5000,
		Frequency:            "monthly",
		Status:               models.SubscriptionStatusIncomplete,
		StripeSubscriptionID: "sub_test_001",
	}

	created, err := repo.Create(ctx, sub)
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
	if got.Frequency != "monthly" {
		t.Errorf("Frequency = %q, want %q", got.Frequency, "monthly")
	}
	if got.StripeSubscriptionID != "sub_test_001" {
		t.Errorf("StripeSubscriptionID = %q, want %q", got.StripeSubscriptionID, "sub_test_001")
	}
	if got.Status != models.SubscriptionStatusIncomplete {
		t.Errorf("Status = %q, want %q", got.Status, models.SubscriptionStatusIncomplete)
	}
}

func TestSubscriptionRepository_UpdateByStripeSubscriptionID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.subscriptions", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "sub-owner-2")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative 2", "test-initiative-2")
	repo := NewSubscriptionRepository(testPool)

	// Create subscription in "incomplete" status
	incomplete := &models.Subscription{
		UserID:               owner.ID,
		InitiativeID:         initiative.ID,
		CurrentAmountCents:   5000,
		Frequency:            "monthly",
		Status:               models.SubscriptionStatusIncomplete,
		StripeSubscriptionID: "sub_test_002",
	}
	created, err := repo.Create(ctx, incomplete)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Advance to "active"
	err = repo.UpdateByStripeSubscriptionID(ctx, "sub_test_002", models.SubscriptionStatusActive)
	if err != nil {
		t.Fatalf("UpdateByStripeSubscriptionID(incomplete→active) error = %v", err)
	}

	// Verify status changed
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != models.SubscriptionStatusActive {
		t.Errorf("Status after update = %q, want %q", got.Status, models.SubscriptionStatusActive)
	}

	// Call again with same status — should return ErrAlreadyProcessed (idempotency)
	err = repo.UpdateByStripeSubscriptionID(ctx, "sub_test_002", models.SubscriptionStatusActive)
	if err == nil {
		t.Fatal("expected error on second update with same status, got nil")
	}
	if !isNotFound(err, domain.ErrAlreadyProcessed) {
		t.Errorf("error = %v, want ErrAlreadyProcessed", err)
	}
}

func TestSubscriptionRepository_UpdateByStripeSubscriptionID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	repo := NewSubscriptionRepository(testPool)

	// Attempt to update nonexistent subscription
	err := repo.UpdateByStripeSubscriptionID(ctx, "sub_nonexistent_123", models.SubscriptionStatusActive)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isNotFound(err, domain.ErrSubscriptionNotFound) {
		t.Errorf("error = %v, want ErrSubscriptionNotFound", err)
	}
}

func TestSubscriptionRepository_GetActiveByUserAndInitiative(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.subscriptions", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "sub-owner-3")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative 3", "test-initiative-3")
	repo := NewSubscriptionRepository(testPool)

	// Before creation: expect ErrSubscriptionNotFound
	_, err := repo.GetActiveByUserAndInitiative(ctx, owner.ID, initiative.ID)
	if err == nil {
		t.Fatal("expected error before creating subscription, got nil")
	}
	if !isNotFound(err, domain.ErrSubscriptionNotFound) {
		t.Errorf("error = %v, want ErrSubscriptionNotFound", err)
	}

	// Create active subscription
	active := &models.Subscription{
		UserID:               owner.ID,
		InitiativeID:         initiative.ID,
		CurrentAmountCents:   5000,
		Frequency:            "monthly",
		Status:               models.SubscriptionStatusActive,
		StripeSubscriptionID: "sub_test_003",
	}
	_, err = repo.Create(ctx, active)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Now GetActiveByUserAndInitiative should return it
	got, err := repo.GetActiveByUserAndInitiative(ctx, owner.ID, initiative.ID)
	if err != nil {
		t.Fatalf("GetActiveByUserAndInitiative() error = %v", err)
	}
	if got.Status != models.SubscriptionStatusActive {
		t.Errorf("Status = %q, want %q", got.Status, models.SubscriptionStatusActive)
	}
}

func TestSubscriptionRepository_ListByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.subscriptions", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "sub-owner-4")
	repo := NewSubscriptionRepository(testPool)

	// Create 3 initiatives and 3 subscriptions for the same user
	init1 := seedInitiative(t, ctx, owner.ID, "Initiative 1", "initiative-1")
	init2 := seedInitiative(t, ctx, owner.ID, "Initiative 2", "initiative-2")
	init3 := seedInitiative(t, ctx, owner.ID, "Initiative 3", "initiative-3")

	seedSubscription(t, ctx, owner.ID, init1.ID, models.SubscriptionStatusActive, "sub_test_004a")
	seedSubscription(t, ctx, owner.ID, init2.ID, models.SubscriptionStatusActive, "sub_test_004b")
	seedSubscription(t, ctx, owner.ID, init3.ID, models.SubscriptionStatusIncomplete, "sub_test_004c")

	// ListByUser with Limit:2 should return 2 items with Total:3
	filter := models.SubscriptionFilter{
		Limit:  2,
		Offset: 0,
	}
	subs, meta, err := repo.ListByUser(ctx, owner.ID, filter)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if len(subs) != 2 {
		t.Errorf("ListByUser() returned %d items, want 2", len(subs))
	}
	if meta.Total != 3 {
		t.Errorf("ListByUser() Total = %d, want 3", meta.Total)
	}
	if meta.Limit != 2 {
		t.Errorf("ListByUser() Limit = %d, want 2", meta.Limit)
	}
	if meta.Offset != 0 {
		t.Errorf("ListByUser() Offset = %d, want 0", meta.Offset)
	}
}
