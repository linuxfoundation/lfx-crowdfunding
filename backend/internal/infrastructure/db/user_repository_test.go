// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// isNotFound reports whether err wraps target using errors.Is.
func isNotFound(err, target error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, target)
}

func TestUserRepository_Upsert_and_GetByUsername(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.users")

	repo := NewUserRepository(testPool)

	in := &models.User{
		Username:  "testuser-upsert",
		Email:     "testuser@example.com",
		GivenName: "Test",
		Name:      "Test User",
	}

	created, err := repo.Upsert(ctx, in)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Upsert() returned empty ID")
	}
	if created.Username != in.Username {
		t.Errorf("Username = %q, want %q", created.Username, in.Username)
	}

	got, err := repo.GetByUsername(ctx, in.Username)
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("GetByUsername ID = %q, want %q", got.ID, created.ID)
	}

	// Upsert again — should update, not duplicate
	in.Email = "updated@example.com"
	updated, err := repo.Upsert(ctx, in)
	if err != nil {
		t.Fatalf("Upsert (update) error = %v", err)
	}
	if updated.ID != created.ID {
		t.Errorf("Upsert (update) changed ID: got %q, want %q", updated.ID, created.ID)
	}
	if updated.Email != "updated@example.com" {
		t.Errorf("Upsert (update) Email = %q, want %q", updated.Email, "updated@example.com")
	}
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isNotFound(err, domain.ErrUserNotFound) {
		t.Errorf("error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepository_UpdateStripeInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.users")

	repo := NewUserRepository(testPool)
	user, err := repo.Upsert(ctx, &models.User{Username: "stripe-test-user"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	if err := repo.UpdateStripeInfo(ctx, user.ID, "cus_test123", "pm_test456"); err != nil {
		t.Fatalf("UpdateStripeInfo() error = %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.StripeCustomerID != "cus_test123" {
		t.Errorf("StripeCustomerID = %q, want %q", got.StripeCustomerID, "cus_test123")
	}
	if got.StripeDefaultPaymentMethod != "pm_test456" {
		t.Errorf("StripeDefaultPaymentMethod = %q, want %q", got.StripeDefaultPaymentMethod, "pm_test456")
	}
}

func TestUserRepository_ClearStripePaymentMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.users")

	repo := NewUserRepository(testPool)
	user, err := repo.Upsert(ctx, &models.User{Username: "clear-pm-user"})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := repo.UpdateStripeInfo(ctx, user.ID, "cus_x", "pm_x"); err != nil {
		t.Fatalf("seed stripe info: %v", err)
	}

	if err := repo.ClearStripePaymentMethod(ctx, user.ID); err != nil {
		t.Fatalf("ClearStripePaymentMethod() error = %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() after clear: %v", err)
	}
	if got.StripeDefaultPaymentMethod != "" {
		t.Errorf("StripeDefaultPaymentMethod = %q, want empty string", got.StripeDefaultPaymentMethod)
	}
}

func TestUserRepository_UpdateStripeInfo_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	repo := NewUserRepository(testPool)

	err := repo.UpdateStripeInfo(ctx, "00000000-0000-0000-0000-000000000000", "cus_x", "pm_x")
	if !isNotFound(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}
