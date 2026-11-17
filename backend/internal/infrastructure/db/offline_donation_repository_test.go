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
			InitiativeID: initiativeID, OrganizationName: "Example Corp", OrganizationDomain: "example.com",
			AmountCents: 100000, Category: "Donation", ReceivedOn: receivedOn, SourceTxnID: ref,
			UserID: offline.ID, RecordedByID: approver.ID, B2BOrgSFID: sfid,
		}
	}

	stored, created, err := repo.CreateOffline(ctx, donation(initiative.ID, "TEST-REF-001", "2026-10-20"))
	if err != nil || !created {
		t.Fatalf("CreateOffline() created=%v err=%v", created, err)
	}
	if stored.Status != models.DonationStatusPending || stored.ReceivedOn != "2026-10-20" ||
		stored.RecordedBy != "test-approver-offline" || stored.InitiativeSlug != "test-initiative-offline" ||
		stored.UserID != offline.ID || stored.B2BOrgSFID != sfid {
		t.Errorf("stored = %+v", stored)
	}

	again, created, err := repo.CreateOffline(ctx, donation(other.ID, "TEST-REF-001", "2026-10-21"))
	if err != nil || created || again.ID != stored.ID || again.InitiativeID != initiative.ID {
		t.Fatalf("same reference: created=%v err=%v got %+v, want the first row back", created, err, again)
	}

	// References are unique ignoring case ("Testing 1234" vs "testing 1234").
	recased, created, err := repo.CreateOffline(ctx, donation(initiative.ID, "test-ref-001", "2026-10-20"))
	if err != nil || created || recased.ID != stored.ID || recased.SourceTxnID != "TEST-REF-001" {
		t.Fatalf("same reference, other case: created=%v err=%v got %+v, want the first row back", created, err, recased)
	}

	// A pending row with no SFID gets the one a retry matched; a set one is kept.
	noSFID := donation(other.ID, "TEST-REF-002", "2026-10-01")
	noSFID.B2BOrgSFID = ""
	bare, _, err := repo.CreateOffline(ctx, noSFID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetOfflineB2BOrg(ctx, bare.ID, sfid); err != nil {
		t.Fatalf("SetOfflineB2BOrg: %v", err)
	}
	if err := repo.SetOfflineB2BOrg(ctx, stored.ID, "001000000000010AAA"); err != nil {
		t.Fatalf("SetOfflineB2BOrg(set): %v", err)
	}
	if got, _, _ := repo.CreateOffline(ctx, noSFID); got.B2BOrgSFID != sfid {
		t.Errorf("recovered SFID = %q, want %q", got.B2BOrgSFID, sfid)
	}
	if got, _, _ := repo.CreateOffline(ctx, donation(initiative.ID, "TEST-REF-001", "2026-10-20")); got.B2BOrgSFID != sfid {
		t.Errorf("SFID overwritten: %q", got.B2BOrgSFID)
	}
	if err := repo.DeleteOfflinePending(ctx, bare.ID); err != nil {
		t.Fatal(err)
	}

	// Pending rows stay off the Org lens until the Ledger has them.
	if rows, _, err := repo.ListByB2BOrg(ctx, sfid, 20, 0); err != nil || len(rows) != 0 {
		t.Fatalf("Org lens before succeeded: rows=%d err=%v", len(rows), err)
	}
	if err := repo.MarkOfflineSucceeded(ctx, stored.ID); err != nil {
		t.Fatalf("MarkOfflineSucceeded: %v", err)
	}
	// Finalizing a row that isn't pending (or is gone) is an error, not a no-op.
	if err := repo.MarkOfflineSucceeded(ctx, stored.ID); err == nil {
		t.Error("MarkOfflineSucceeded(succeeded) = nil, want an error")
	}
	if err := repo.MarkOfflineSucceeded(ctx, "00000000-0000-0000-0000-000000000000"); err == nil {
		t.Error("MarkOfflineSucceeded(missing) = nil, want an error")
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
	if all[0].SourceTxnID != "TEST-REF-001" || all[0].Status != models.DonationStatusSucceeded {
		t.Errorf("newest received first: got %+v", all[0])
	}
	one, meta, err := repo.ListOffline(ctx, other.ID, 20, 0)
	if err != nil || meta.Total != 1 || one[0].SourceTxnID != "Other ref" {
		t.Errorf("ListOffline(other) = %+v total=%v err=%v", one, meta, err)
	}
}

func TestDonationRepository_LockOfflineReference(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	repo := NewDonationRepository(testPool)

	unlock, err := repo.LockOfflineReference(ctx, "TEST-REF-LOCK")
	if err != nil {
		t.Fatalf("LockOfflineReference: %v", err)
	}

	// Another session waits for the same reference, in any case...
	acquired := make(chan func(), 1)
	go func() {
		u, err := repo.LockOfflineReference(ctx, "test-ref-lock")
		if err != nil {
			t.Errorf("second LockOfflineReference: %v", err)
			close(acquired)
			return
		}
		acquired <- u
	}()
	select {
	case <-acquired:
		t.Fatal("second lock acquired while the first was held")
	case <-time.After(300 * time.Millisecond):
	}

	// ...but not for a different one.
	other, err := repo.LockOfflineReference(ctx, "TEST-REF-OTHER")
	if err != nil {
		t.Fatalf("LockOfflineReference(other): %v", err)
	}
	other()

	unlock()
	select {
	case u, ok := <-acquired:
		if ok {
			u()
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second lock not acquired after unlock")
	}
}
