// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

func TestDonationRepository_OfflineDonations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.subscriptions", "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-offline")
	offline := seedUser(t, ctx, "lf:offline-donations")
	approver := seedUser(t, ctx, "test-approver-offline")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative Offline", "test-initiative-offline")
	other := seedInitiative(t, ctx, owner.ID, "Other Initiative Offline", "other-initiative-offline")
	repo := NewDonationRepository(testPool)
	const sfid = "001000000000009AAA"

	donation := func(initiativeID, ref, receivedOn string) *models.OfflineDonation {
		return &models.OfflineDonation{
			InitiativeID: initiativeID, OrganizationName: "Dell Corporation", OrganizationDomain: "dell.com",
			AmountCents: 800000, Category: "Donation", ReceivedOn: receivedOn, SourceTxnID: ref,
			UserID: offline.ID, RecordedByID: approver.ID, B2BOrgSFID: sfid,
		}
	}

	stored, created, err := repo.CreateOffline(ctx, donation(initiative.ID, "SPDK Dell 2026-10", "2026-10-20"))
	if err != nil || !created {
		t.Fatalf("CreateOffline() created=%v err=%v", created, err)
	}
	if stored.Status != models.DonationStatusPending || stored.ReceivedOn != "2026-10-20" ||
		stored.RecordedBy != "test-approver-offline" || stored.InitiativeSlug != "test-initiative-offline" ||
		stored.UserID != offline.ID || stored.B2BOrgSFID != sfid {
		t.Errorf("stored = %+v", stored)
	}

	again, created, err := repo.CreateOffline(ctx, donation(other.ID, "SPDK Dell 2026-10", "2026-10-21"))
	if err != nil || created || again.ID != stored.ID || again.InitiativeID != initiative.ID {
		t.Fatalf("same reference: created=%v err=%v got %+v, want the first row back", created, err, again)
	}

	// Pending rows stay off the Org lens until the Ledger has them.
	if rows, _, err := repo.ListByB2BOrg(ctx, sfid, 20, 0); err != nil || len(rows) != 0 {
		t.Fatalf("Org lens before succeeded: rows=%d err=%v", len(rows), err)
	}
	if err := repo.MarkOfflineSucceeded(ctx, stored.ID); err != nil {
		t.Fatalf("MarkOfflineSucceeded: %v", err)
	}
	rows, _, err := repo.ListByB2BOrg(ctx, sfid, 20, 0)
	if err != nil || len(rows) != 1 {
		t.Fatalf("Org lens: rows=%d err=%v", len(rows), err)
	}
	if want := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC); !rows[0].CreatedOn.Equal(want) {
		t.Errorf("Org lens date = %v, want the received date %v", rows[0].CreatedOn, want)
	}
	if rows[0].DonorName != "lf:offline-donations" || rows[0].Type != models.OrgLensDonationOneTime {
		t.Errorf("Org lens row = %+v", rows[0])
	}

	// A succeeded row is never deleted; a pending one is.
	if err := repo.DeleteOfflinePending(ctx, stored.ID); err != nil {
		t.Fatalf("DeleteOfflinePending(succeeded): %v", err)
	}
	pending, _, err := repo.CreateOffline(ctx, donation(other.ID, "Other ref", "2026-09-01"))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteOfflinePending(ctx, pending.ID); err != nil {
		t.Fatalf("DeleteOfflinePending(pending): %v", err)
	}
	if _, created, err := repo.CreateOffline(ctx, donation(other.ID, "Other ref", "2026-09-01")); err != nil || !created {
		t.Fatalf("reference not freed: created=%v err=%v", created, err)
	}

	// A Stripe donation isn't listed as offline.
	seedDonation(t, ctx, owner.ID, initiative.ID)

	all, meta, err := repo.ListOffline(ctx, "", 20, 0)
	if err != nil || meta.Total != 2 || len(all) != 2 {
		t.Fatalf("ListOffline(all) total=%v len=%d err=%v", meta, len(all), err)
	}
	if all[0].SourceTxnID != "SPDK Dell 2026-10" || all[0].Status != models.DonationStatusSucceeded {
		t.Errorf("newest received first: got %+v", all[0])
	}
	one, meta, err := repo.ListOffline(ctx, other.ID, 20, 0)
	if err != nil || meta.Total != 1 || one[0].SourceTxnID != "Other ref" {
		t.Errorf("ListOffline(other) = %+v total=%v err=%v", one, meta, err)
	}
}
