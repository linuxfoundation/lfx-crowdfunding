// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// seedOrganization creates an organization owned by ownerID and returns it.
func seedOrganization(t *testing.T, ctx context.Context, ownerID, name string) *models.Organization { //nolint:revive // t first is Go test convention
	t.Helper()
	org, err := NewOrganizationRepository(testPool).Create(ctx, ownerID, models.OrganizationCreateInput{Name: name})
	if err != nil {
		t.Fatalf("seedOrganization: %v", err)
	}
	return org
}

// seedDonationWithOrg creates a donation with the given status and organization, bypassing seedDonation's fixed pending status.
func seedDonationWithOrg(t *testing.T, ctx context.Context, userID, initiativeID, orgID, status string, amountCents int64) { //nolint:revive // t first is Go test convention
	t.Helper()
	donation := &models.Donation{
		UserID:             userID,
		InitiativeID:       initiativeID,
		OrganizationID:     orgID,
		CurrentAmountCents: amountCents,
		Status:             status,
	}
	if _, err := NewDonationRepository(testPool).Create(ctx, donation); err != nil {
		t.Fatalf("seedDonationWithOrg: %v", err)
	}
}

func TestStatisticsRepository_GetPlatformStatistics_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiative_ledger_stats", "crowdfunding.initiatives", "crowdfunding.users")

	repo := NewStatisticsRepository(testPool)
	stats, err := repo.GetPlatformStatistics(ctx)
	if err != nil {
		t.Fatalf("GetPlatformStatistics() error = %v", err)
	}

	if stats == nil {
		t.Fatal("GetPlatformStatistics() returned nil")
	}
	if stats.TotalRaisedCents != 0 {
		t.Errorf("TotalRaisedCents = %d, want 0", stats.TotalRaisedCents)
	}
	if stats.TotalSupporters != 0 {
		t.Errorf("TotalSupporters = %d, want 0", stats.TotalSupporters)
	}
	if stats.TotalInitiatives != 0 {
		t.Errorf("TotalInitiatives = %d, want 0", stats.TotalInitiatives)
	}
}

func TestStatisticsRepository_GetPlatformStatistics_WithPublishedInitiative(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.initiative_ledger_stats", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative", "test-initiative")

	// Insert a row into initiative_ledger_stats for the published initiative
	const insertSQL = `
		INSERT INTO crowdfunding.initiative_ledger_stats (initiative_id, total_raised_cents, supporters)
		VALUES ($1, 50000, 5)
		ON CONFLICT (initiative_id) DO UPDATE
		  SET total_raised_cents = EXCLUDED.total_raised_cents,
		      supporters = EXCLUDED.supporters
	`
	if _, err := testPool.Exec(ctx, insertSQL, initiative.ID); err != nil {
		t.Fatalf("insert ledger stats: %v", err)
	}

	repo := NewStatisticsRepository(testPool)
	stats, err := repo.GetPlatformStatistics(ctx)
	if err != nil {
		t.Fatalf("GetPlatformStatistics() error = %v", err)
	}

	if stats == nil {
		t.Fatal("GetPlatformStatistics() returned nil")
	}
	if stats.TotalRaisedCents != 50000 {
		t.Errorf("TotalRaisedCents = %d, want 50000", stats.TotalRaisedCents)
	}
	if stats.TotalSupporters != 5 {
		t.Errorf("TotalSupporters = %d, want 5", stats.TotalSupporters)
	}
	if stats.TotalInitiatives != 1 {
		t.Errorf("TotalInitiatives = %d, want 1", stats.TotalInitiatives)
	}
}

func TestStatisticsRepository_ListOrgContributions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.donations", "crowdfunding.organizations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-org-contrib")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative", "test-initiative-org-contrib")

	orgA := seedOrganization(t, ctx, owner.ID, "Org A")
	orgB := seedOrganization(t, ctx, owner.ID, "Org B")
	orgC := seedOrganization(t, ctx, owner.ID, "Org C (pending only)")

	// Org A: two succeeded donations, should aggregate to 300.
	seedDonationWithOrg(t, ctx, owner.ID, initiative.ID, orgA.ID, models.DonationStatusSucceeded, 100)
	seedDonationWithOrg(t, ctx, owner.ID, initiative.ID, orgA.ID, models.DonationStatusSucceeded, 200)
	// Org B: one succeeded donation, larger than Org A's total.
	seedDonationWithOrg(t, ctx, owner.ID, initiative.ID, orgB.ID, models.DonationStatusSucceeded, 500)
	// Org C: pending donation only — must not be counted or returned.
	seedDonationWithOrg(t, ctx, owner.ID, initiative.ID, orgC.ID, models.DonationStatusPending, 999)

	repo := NewStatisticsRepository(testPool)

	t.Run("filters by ids and aggregates succeeded donations", func(t *testing.T) {
		got, err := repo.ListOrgContributions(ctx, []string{orgA.ID}, 0)
		if err != nil {
			t.Fatalf("ListOrgContributions() error = %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 result, got %d: %+v", len(got), got)
		}
		if got[0].OrgID != orgA.ID || got[0].AmountCents != 300 {
			t.Errorf("got %+v, want OrgID=%s AmountCents=300", got[0], orgA.ID)
		}
	})

	t.Run("empty ids returns all orgs with succeeded donations, amount descending", func(t *testing.T) {
		got, err := repo.ListOrgContributions(ctx, nil, 0)
		if err != nil {
			t.Fatalf("ListOrgContributions() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 results (pending-only org excluded), got %d: %+v", len(got), got)
		}
		if got[0].OrgID != orgB.ID || got[1].OrgID != orgA.ID {
			t.Errorf("expected order [%s, %s], got [%s, %s]", orgB.ID, orgA.ID, got[0].OrgID, got[1].OrgID)
		}
	})

	t.Run("limit caps results", func(t *testing.T) {
		got, err := repo.ListOrgContributions(ctx, nil, 1)
		if err != nil {
			t.Fatalf("ListOrgContributions() error = %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 result, got %d", len(got))
		}
		if got[0].OrgID != orgB.ID {
			t.Errorf("OrgID = %s, want %s (highest amount)", got[0].OrgID, orgB.ID)
		}
	})
}
