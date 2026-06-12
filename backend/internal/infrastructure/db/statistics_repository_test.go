// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"testing"
)

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
