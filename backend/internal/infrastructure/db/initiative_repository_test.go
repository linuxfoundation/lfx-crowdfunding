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

// TestInitiativeRepository_Attribution_RoundTrip covers LFXV2-2956 M1:
// omitting attribution on Create defaults to personal, and an explicit
// organization/project attribution persists and round-trips through Update.
func TestInitiativeRepository_Attribution_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "attribution-owner")
	repo := NewInitiativeRepository(testPool)

	created := seedInitiative(t, ctx, owner.ID, "Attribution Fund", "attribution-fund")
	if created.Attribution.Type != models.AttributionPersonal {
		t.Fatalf("Attribution.Type = %v, want personal (default)", created.Attribution.Type)
	}
	if created.Attribution.EntityUID != "" {
		t.Fatalf("Attribution.EntityUID = %q, want empty", created.Attribution.EntityUID)
	}

	orgUID := "0012M00002qnukOQAQ" // 18-char Salesforce SFID, not a UUID (lfx-crowdfunding#263)
	created.Attribution = models.Attribution{Type: models.AttributionOrganization, EntityUID: orgUID}
	updated, err := repo.Update(ctx, created, models.InitiativeUpdateInput{})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Attribution.Type != models.AttributionOrganization {
		t.Errorf("Attribution.Type = %v, want organization", updated.Attribution.Type)
	}
	if updated.Attribution.EntityUID != orgUID {
		t.Errorf("Attribution.EntityUID = %q, want %q", updated.Attribution.EntityUID, orgUID)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Attribution.Type != models.AttributionOrganization || got.Attribution.EntityUID != orgUID {
		t.Errorf("GetByID() attribution = %+v, want {organization %s}", got.Attribution, orgUID)
	}
}

// Display fields go through positional INSERT/UPDATE args and the scan, so a
// misordered column shows up here; a label-only change must bump updated_on.
func TestInitiativeRepository_AttributionDisplay_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "attribution-display-owner")
	repo := NewInitiativeRepository(testPool)

	attr := models.Attribution{
		Type: models.AttributionOrganization, EntityUID: "0012M00002qnukOQAQ",
		Name: "Acme", LogoURL: "https://example.com/acme.png",
	}
	created, err := repo.Create(ctx, &models.Initiative{
		ID: uuid.New().String(), InitiativeType: "project", OwnerID: owner.ID,
		Name: "Display Fund", Slug: "display-fund", Status: models.StatusPublished,
		DonationMode: models.DonationModeOpen, Attribution: attr,
	}, models.InitiativeCreateInput{InitiativeType: "project", Name: "Display Fund", Slug: "display-fund"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Attribution != attr {
		t.Errorf("Create() attribution = %+v, want %+v", created.Attribution, attr)
	}

	if _, err := testPool.Exec(ctx, `UPDATE crowdfunding.initiatives SET updated_on = NOW() - INTERVAL '1 day' WHERE id = $1`, created.ID); err != nil {
		t.Fatalf("backdate updated_on: %v", err)
	}

	created.Attribution.Name, created.Attribution.LogoURL = "Acme Corp", "https://example.com/acme-2.png"
	if _, err := repo.Update(ctx, created, models.InitiativeUpdateInput{}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Attribution != created.Attribution {
		t.Errorf("GetByID() attribution = %+v, want %+v", got.Attribution, created.Attribution)
	}

	var stale bool
	if err := testPool.QueryRow(ctx, `SELECT updated_on < NOW() - INTERVAL '1 hour' FROM crowdfunding.initiatives WHERE id = $1`, created.ID).Scan(&stale); err != nil {
		t.Fatalf("select updated_on: %v", err)
	}
	if stale {
		t.Error("updated_on not bumped when only the attribution label changed")
	}
}

func TestInitiativeRepository_Update_EmptyUpdatedByPreservesLastEditor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "editor-owner")
	repo := NewInitiativeRepository(testPool)
	created := seedInitiative(t, ctx, owner.ID, "Audit Fund", "audit-fund")

	created.UpdatedBy = "editor-owner"
	if _, err := repo.Update(ctx, created, models.InitiativeUpdateInput{}); err != nil {
		t.Fatalf("Update() with editor error = %v", err)
	}

	// Simulates ProcessApproval: GetByID doesn't hydrate UpdatedBy, so it arrives empty.
	created.UpdatedBy = ""
	created.Status = models.StatusPublished
	if _, err := repo.Update(ctx, created, models.InitiativeUpdateInput{}); err != nil {
		t.Fatalf("Update() without editor error = %v", err)
	}

	var got *string
	if err := testPool.QueryRow(ctx, `SELECT updated_by FROM crowdfunding.initiatives WHERE id = $1`, created.ID).Scan(&got); err != nil {
		t.Fatalf("select updated_by: %v", err)
	}
	if got == nil || *got != "editor-owner" {
		t.Errorf("updated_by = %v, want editor-owner", got)
	}
}

func TestInitiativeRepository_Update_UpdatedByOnlyBumpsUpdatedOn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "editor-owner")
	repo := NewInitiativeRepository(testPool)
	created := seedInitiative(t, ctx, owner.ID, "Trigger Fund", "trigger-fund")

	// updated_on is not in the trigger's WHEN list, so this backdating does not fire it.
	if _, err := testPool.Exec(ctx, `UPDATE crowdfunding.initiatives SET updated_on = NOW() - INTERVAL '1 day' WHERE id = $1`, created.ID); err != nil {
		t.Fatalf("backdate updated_on: %v", err)
	}

	created.UpdatedBy = "editor-owner"
	if _, err := repo.Update(ctx, created, models.InitiativeUpdateInput{}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	var stale bool
	if err := testPool.QueryRow(ctx, `SELECT updated_on < NOW() - INTERVAL '1 hour' FROM crowdfunding.initiatives WHERE id = $1`, created.ID).Scan(&stale); err != nil {
		t.Fatalf("select updated_on: %v", err)
	}
	if stale {
		t.Error("updated_on not bumped when only updated_by changed")
	}
}

func TestInitiativeRepository_List_ByAttribution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "attr-owner")
	repo := NewInitiativeRepository(testPool)
	const sfid = "0014100000Te0yvAAB"

	mk := func(slug string, attr models.Attribution, status models.InitiativeStatus) *models.Initiative {
		i, err := repo.Create(ctx, &models.Initiative{
			ID: uuid.New().String(), InitiativeType: "project", OwnerID: owner.ID, Name: slug, Slug: slug,
			Status: status, DonationMode: models.DonationModeOpen, Attribution: attr,
		}, models.InitiativeCreateInput{InitiativeType: "project", Name: slug, Slug: slug})
		if err != nil {
			t.Fatalf("create %s: %v", slug, err)
		}
		return i
	}
	org := models.Attribution{Type: models.AttributionOrganization, EntityUID: sfid}
	mk("org-published", org, models.StatusPublished)
	mk("org-draft", org, models.StatusSubmitted)
	mk("org-other", models.Attribution{Type: models.AttributionOrganization, EntityUID: "0014100000Other0AAA"}, models.StatusPublished)
	mk("personal", models.Attribution{}, models.StatusPublished)

	got, meta, err := repo.List(ctx, models.InitiativeFilter{AttributedToType: models.AttributionOrganization, AttributedToUID: sfid, Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if meta.Total != 2 || len(got) != 2 {
		t.Fatalf("got %d items (total %d), want 2 (published + submitted for that org only)", len(got), meta.Total)
	}
}
