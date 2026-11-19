// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

func TestLedgerAdjustmentRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB integration test")
	}
	ctx := context.Background()
	truncate(t, ctx, "crowdfunding.ledger_adjustments", "crowdfunding.donations", "crowdfunding.initiatives", "crowdfunding.users")

	owner := seedUser(t, ctx, "test-owner-adjust")
	offline := seedUser(t, ctx, "lf:offline-donations")
	approver := seedUser(t, ctx, "test-approver-adjust")
	initiative := seedInitiative(t, ctx, owner.ID, "Test Initiative Adjust", "test-initiative-adjust")
	other := seedInitiative(t, ctx, owner.ID, "Other Initiative Adjust", "other-initiative-adjust")
	repo := NewLedgerAdjustmentRepository(testPool)
	donations := NewDonationRepository(testPool)

	ret := func(initiativeID, ref, on string) *models.LedgerAdjustment {
		return &models.LedgerAdjustment{
			Kind: models.AdjustmentKindReturn, InitiativeID: initiativeID,
			OrganizationName: "CNCF", OrganizationDomain: "cncf.io", B2BOrgSFID: "001CNCF",
			AmountCents: 150000, Category: "mentorship", Reason: "Mentee failed", AdjustedOn: on,
			SourceTxnID: ref, RecordedByID: approver.ID,
		}
	}

	stored, created, err := repo.CreateAdjustment(ctx, ret(initiative.ID, "TEST-RETURN-1", "2026-11-02"))
	if err != nil || !created {
		t.Fatalf("CreateAdjustment() created=%v err=%v", created, err)
	}
	if stored.Status != models.DonationStatusPending || stored.AdjustedOn != "2026-11-02" || stored.Kind != "return" ||
		stored.RecordedBy != "test-approver-adjust" || stored.InitiativeSlug != "test-initiative-adjust" ||
		stored.B2BOrgSFID != "001CNCF" || stored.PaidTo != "" || stored.Reason != "Mentee failed" {
		t.Errorf("stored = %+v", stored)
	}

	// Same reference, any case: the first row comes back.
	again, created, err := repo.CreateAdjustment(ctx, ret(other.ID, "test-return-1", "2026-11-03"))
	if err != nil || created || again.ID != stored.ID || again.InitiativeID != initiative.ID {
		t.Fatalf("same reference: created=%v err=%v got %+v, want the first row back", created, err, again)
	}

	// A payout with a payee and no org.
	payout := &models.LedgerAdjustment{
		Kind: models.AdjustmentKindPayout, InitiativeID: other.ID, PaidTo: "Jane Payee",
		AmountCents: 1000000, Category: "general fund", Reason: "Rewards payout", AdjustedOn: "2026-11-05",
		SourceTxnID: "TEST-PAYOUT-1", RecordedByID: approver.ID,
	}
	p, created, err := repo.CreateAdjustment(ctx, payout)
	if err != nil || !created || p.PaidTo != "Jane Payee" || p.OrganizationName != "" || p.B2BOrgSFID != "" {
		t.Fatalf("payout: created=%v err=%v got %+v", created, err, p)
	}

	// A reference is unique across offline donations and adjustments.
	if _, _, err := donations.CreateOffline(ctx, &models.OfflineDonation{
		InitiativeID: initiative.ID, OrganizationName: "CNCF", OrganizationDomain: "cncf.io",
		AmountCents: 100, Category: "Donation", ReceivedOn: "2026-11-01", SourceTxnID: "Test-Payout-1",
		UserID: offline.ID, RecordedByID: approver.ID,
	}); !errors.Is(err, domain.ErrSourceTxnIDTaken) {
		t.Fatalf("offline donation reusing an adjustment's reference: err=%v, want ErrSourceTxnIDTaken", err)
	}
	if _, _, err := donations.CreateOffline(ctx, &models.OfflineDonation{
		InitiativeID: initiative.ID, OrganizationName: "CNCF", OrganizationDomain: "cncf.io",
		AmountCents: 100, Category: "Donation", ReceivedOn: "2026-11-01", SourceTxnID: "TEST-DONATION-1",
		UserID: offline.ID, RecordedByID: approver.ID,
	}); err != nil {
		t.Fatalf("CreateOffline: %v", err)
	}
	if _, _, err := repo.CreateAdjustment(ctx, ret(initiative.ID, "test-donation-1", "2026-11-02")); !errors.Is(err, domain.ErrSourceTxnIDTaken) {
		t.Fatalf("adjustment reusing an offline donation's reference: err=%v, want ErrSourceTxnIDTaken", err)
	}

	// SFID fill only on a pending row with none.
	if err := repo.SetAdjustmentB2BOrg(ctx, p.ID, "001OTHER"); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkAdjustmentSucceeded(ctx, stored.ID); err != nil {
		t.Fatalf("MarkAdjustmentSucceeded: %v", err)
	}
	if err := repo.MarkAdjustmentSucceeded(ctx, stored.ID); err == nil {
		t.Error("MarkAdjustmentSucceeded twice: want an error for a row that's not pending")
	}
	// A succeeded row isn't deleted.
	if err := repo.DeleteAdjustmentPending(ctx, stored.ID); err != nil {
		t.Fatal(err)
	}

	all, meta, err := repo.ListAdjustments(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("ListAdjustments: %v", err)
	}
	if meta.Total != 2 || len(all) != 2 || all[0].ID != p.ID || all[1].ID != stored.ID ||
		all[1].Status != models.DonationStatusSucceeded || all[0].B2BOrgSFID != "001OTHER" {
		t.Errorf("list = %+v meta=%+v, want payout (newest, SFID filled) then the succeeded return", all, meta)
	}
	one, meta, err := repo.ListAdjustments(ctx, initiative.ID, 10, 0)
	if err != nil || meta.Total != 1 || len(one) != 1 || one[0].ID != stored.ID {
		t.Errorf("list by initiative = %+v meta=%+v err=%v", one, meta, err)
	}

	// A pending row is deleted, freeing its reference.
	if err := repo.DeleteAdjustmentPending(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, created, err := repo.CreateAdjustment(ctx, payout); err != nil || !created {
		t.Fatalf("reference after delete: created=%v err=%v", created, err)
	}

	// The reference lock is shared with offline donations.
	unlock, err := repo.LockReference(ctx, "TEST-LOCK")
	if err != nil {
		t.Fatal(err)
	}
	lockCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if _, err := donations.LockOfflineReference(lockCtx, "test-lock"); err == nil {
		t.Error("LockOfflineReference while an adjustment holds the reference: want a timeout")
	}
	unlock()
	unlock2, err := donations.LockOfflineReference(ctx, "test-lock")
	if err != nil {
		t.Fatalf("LockOfflineReference after unlock: %v", err)
	}
	unlock2()
}
