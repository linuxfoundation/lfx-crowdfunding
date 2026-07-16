// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// seedUser upserts a user with the given username and returns it.
func seedUser(t *testing.T, ctx context.Context, username string) *models.User { //nolint:revive // t first is Go test convention
	t.Helper()
	userRepo := NewUserRepository(testPool)
	user, err := userRepo.Upsert(ctx, &models.User{
		Username:  username,
		Email:     username + "@example.com",
		Name:      username,
		GivenName: username,
	})
	if err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	return user
}

// seedInitiative creates a published "project" initiative and returns it.
func seedInitiative(t *testing.T, ctx context.Context, ownerID, name, slug string) *models.Initiative { //nolint:revive // t first is Go test convention
	t.Helper()
	initRepo := NewInitiativeRepository(testPool)

	input := models.InitiativeCreateInput{
		InitiativeType: "project",
		Name:           name,
		Slug:           slug,
	}

	initiative := &models.Initiative{
		ID:             uuid.New().String(),
		InitiativeType: "project",
		OwnerID:        ownerID,
		Name:           name,
		Slug:           slug,
		Status:         models.StatusPublished,
		DonationMode:   models.DonationModeOpen,
	}

	created, err := initRepo.Create(ctx, initiative, input)
	if err != nil {
		t.Fatalf("seedInitiative: %v", err)
	}
	return created
}

func TestInitiativeRepository_CreateAndGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner")
	repo := NewInitiativeRepository(testPool)

	input := models.InitiativeCreateInput{
		InitiativeType: "project",
		Name:           "Test Project",
		Slug:           "test-project",
	}

	initiative := &models.Initiative{
		ID:             uuid.New().String(),
		InitiativeType: "project",
		OwnerID:        owner.ID,
		Name:           "Test Project",
		Slug:           "test-project",
		Status:         models.StatusPublished,
		DonationMode:   models.DonationModeOpen,
	}

	created, err := repo.Create(ctx, initiative, input)
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
	if got.Name != "Test Project" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Project")
	}
	if got.Slug != "test-project" {
		t.Errorf("Slug = %q, want %q", got.Slug, "test-project")
	}
	if got.Status != models.StatusPublished {
		t.Errorf("Status = %v, want %v", got.Status, models.StatusPublished)
	}
}

func TestInitiativeRepository_GetByID_Hidden_ReturnsExpectedStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-hidden")
	repo := NewInitiativeRepository(testPool)

	input := models.InitiativeCreateInput{
		InitiativeType: "project",
		Name:           "Hidden Project",
		Slug:           "hidden-project",
	}

	initiative := &models.Initiative{
		ID:             uuid.New().String(),
		InitiativeType: "project",
		OwnerID:        owner.ID,
		Name:           "Hidden Project",
		Slug:           "hidden-project",
		Status:         models.StatusHidden,
		DonationMode:   models.DonationModeOpen,
	}

	created, err := repo.Create(ctx, initiative, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// GetByID still works regardless of status
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Status != models.StatusHidden {
		t.Errorf("Status = %v, want %v", got.Status, models.StatusHidden)
	}
}

func TestInitiativeRepository_List_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-pagination")
	repo := NewInitiativeRepository(testPool)

	// Create 3 published initiatives for the same owner
	seedInitiative(t, ctx, owner.ID, "Initiative 1", "init-1")
	seedInitiative(t, ctx, owner.ID, "Initiative 2", "init-2")
	seedInitiative(t, ctx, owner.ID, "Initiative 3", "init-3")

	// First page with Limit=2
	filter := models.InitiativeFilter{
		OwnerID: owner.ID,
		Status:  models.StatusPublished,
		Limit:   2,
		Offset:  0,
	}

	initiatives, meta, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(initiatives) != 2 {
		t.Errorf("first page: got %d items, want 2", len(initiatives))
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

	// Second page with Offset=2
	filter.Offset = 2
	initiatives, meta, err = repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() (page 2) error = %v", err)
	}

	if len(initiatives) != 1 {
		t.Errorf("second page: got %d items, want 1", len(initiatives))
	}
	if meta.Total != 3 {
		t.Errorf("page 2 meta.Total = %d, want 3", meta.Total)
	}
	if meta.Offset != 2 {
		t.Errorf("page 2 meta.Offset = %d, want 2", meta.Offset)
	}
}

func TestInitiativeRepository_List_SortByTrending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.donations", "crowdfunding.subscriptions", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-trending")
	donor := seedUser(t, ctx, "test-donor-trending")
	repo := NewInitiativeRepository(testPool)
	donationRepo := NewDonationRepository(testPool)
	subRepo := NewSubscriptionRepository(testPool)

	// Busy: 2 recent supports. Quiet: 1 recent + 1 stale support (stale shouldn't
	// count). Idle: no supports at all — must still be returned by List(),
	// just ranked last.
	busy := seedInitiative(t, ctx, owner.ID, "Busy Initiative", "busy-init")
	quiet := seedInitiative(t, ctx, owner.ID, "Quiet Initiative", "quiet-init")
	idle := seedInitiative(t, ctx, owner.ID, "Idle Initiative", "idle-init")

	if _, err := donationRepo.Create(ctx, &models.Donation{
		UserID: donor.ID, InitiativeID: busy.ID, CurrentAmountCents: 5000, Status: models.DonationStatusSucceeded,
	}); err != nil {
		t.Fatalf("seed busy donation: %v", err)
	}
	if _, err := subRepo.Create(ctx, &models.Subscription{
		UserID: donor.ID, InitiativeID: busy.ID, CurrentAmountCents: 5000, Frequency: "monthly",
		Status: models.SubscriptionStatusActive, StripeSubscriptionID: "sub_busy",
	}); err != nil {
		t.Fatalf("seed busy subscription: %v", err)
	}

	if _, err := donationRepo.Create(ctx, &models.Donation{
		UserID: donor.ID, InitiativeID: quiet.ID, CurrentAmountCents: 5000, Status: models.DonationStatusSucceeded,
	}); err != nil {
		t.Fatalf("seed quiet donation: %v", err)
	}
	staleDonation, err := donationRepo.Create(ctx, &models.Donation{
		UserID: donor.ID, InitiativeID: quiet.ID, CurrentAmountCents: 5000, Status: models.DonationStatusSucceeded,
	})
	if err != nil {
		t.Fatalf("seed stale donation: %v", err)
	}

	// Push one of quiet's donations outside the 30-day window; the other
	// donation/subscription rows are left at NOW() so they still count.
	if _, err := testPool.Exec(ctx,
		"UPDATE crowdfunding.donations SET created_on = NOW() - INTERVAL '40 days' WHERE id = $1",
		staleDonation.ID); err != nil {
		t.Fatalf("age stale donation: %v", err)
	}

	filter := models.InitiativeFilter{OwnerID: owner.ID, Status: models.StatusPublished, SortBy: "trending", SortDir: "desc", Limit: 10}
	initiatives, meta, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if meta.Total != 3 {
		t.Fatalf("meta.Total = %d, want 3 (idle initiative must still be returned)", meta.Total)
	}
	if len(initiatives) != 3 {
		t.Fatalf("got %d items, want 3", len(initiatives))
	}

	got := []string{initiatives[0].ID, initiatives[1].ID, initiatives[2].ID}
	want := []string{busy.ID, quiet.ID, idle.ID}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got initiative %v, want %v (order should be busy(2) > quiet(1) > idle(0))", i, got[i], want[i])
		}
	}
}

func TestInitiativeRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	repo := NewInitiativeRepository(testPool)

	_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isNotFound(err, domain.ErrInitiativeNotFound) {
		t.Errorf("error = %v, want ErrInitiativeNotFound", err)
	}
}

func TestInitiativeRepository_ListPublished(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "list-published-owner")
	repo := NewInitiativeRepository(testPool)

	// Create a published initiative.
	seedInitiative(t, ctx, owner.ID, "Zebra Fund", "zebra-fund")

	// Create a non-published initiative — must be excluded from results.
	hiddenInit := &models.Initiative{
		ID:             uuid.New().String(),
		InitiativeType: "project",
		OwnerID:        owner.ID,
		Name:           "Hidden Initiative",
		Slug:           "hidden-initiative",
		Status:         models.StatusHidden,
		DonationMode:   models.DonationModeOpen,
	}
	_, err := repo.Create(ctx, hiddenInit, models.InitiativeCreateInput{
		InitiativeType: "project",
		Name:           "Hidden Initiative",
		Slug:           "hidden-initiative",
	})
	if err != nil {
		t.Fatalf("Create() hidden initiative: %v", err)
	}

	// Create a second published initiative with a name that sorts before the first.
	seedInitiative(t, ctx, owner.ID, "Alpha Project", "alpha-project")

	results, err := repo.ListPublished(ctx)
	if err != nil {
		t.Fatalf("ListPublished() error = %v", err)
	}

	// Only the two published initiatives should be returned.
	if len(results) != 2 {
		t.Fatalf("ListPublished() returned %d items, want 2", len(results))
	}

	// Results must be ordered by name ascending.
	if results[0].Name != "Alpha Project" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "Alpha Project")
	}
	if results[1].Name != "Zebra Fund" {
		t.Errorf("results[1].Name = %q, want %q", results[1].Name, "Zebra Fund")
	}

	// IDs must be non-empty.
	for i, r := range results {
		if r.ID == "" {
			t.Errorf("results[%d].ID is empty", i)
		}
	}
}
