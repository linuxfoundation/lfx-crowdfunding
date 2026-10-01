// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

func newListMyDonationsHandler(capture *filterCapturingLedger) *InitiativeHandler {
	svc := service.NewInitiativeService(&initiativeRepo{}, &initiativeUserRepo{user: &models.User{LegacyUserID: "auth0|michal"}},
		capture, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	return NewInitiativeHandler(svc, nil, slog.Default())
}

func TestListMyDonations_NoPrincipal_Returns401(t *testing.T) {
	h := newListMyDonationsHandler(&filterCapturingLedger{list: &models.TransactionList{}})
	w := httptest.NewRecorder()
	h.ListMyDonations(w, httptest.NewRequest(http.MethodGet, "/crowdfunding/me/donations", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// Total Donated in Self Serve sums this endpoint, so it must read the same Ledger
// donations as the history: one-time and recurring, under the legacy user ID, with
// the initiative ID on each row.
func TestListMyDonations_ReadsLedgerDonations_IncludingRecurring(t *testing.T) {
	capture := &filterCapturingLedger{
		list: &models.TransactionList{
			Data: []models.Transaction{
				{ID: "t1", AmountCents: 500, LedgerUserID: "auth0|michal", LedgerProjectID: "p1"},
				{ID: "t2", AmountCents: 300, Recurring: true, LedgerUserID: "auth0|michal", LedgerProjectID: "p2"},
			},
			TotalCount: 2,
			Limit:      100,
		},
	}
	h := newListMyDonationsHandler(capture)

	req := withPrincipal(httptest.NewRequest(http.MethodGet, "/crowdfunding/me/donations", nil), &models.Principal{UserID: "michal", Username: "michal"})
	w := httptest.NewRecorder()
	h.ListMyDonations(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if f := capture.lastFilter; f.UserID != "auth0|michal" || f.TxnType != models.TransactionTypeDonation || f.SubscriptionOnly {
		t.Errorf("unexpected Ledger filter: %+v", f)
	}
	var body struct {
		Data []struct {
			InitiativeID string `json:"initiative_id"`
			AmountCents  int64  `json:"amount_cents"`
		} `json:"data"`
		Meta models.PaginationMeta `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 2 || body.Data[0].InitiativeID != "p1" || body.Data[1].InitiativeID != "p2" || body.Data[1].AmountCents != 300 {
		t.Errorf("unexpected data: %+v", body.Data)
	}
	if body.Meta.Total != 2 {
		t.Errorf("meta.total = %d, want 2", body.Meta.Total)
	}
	if w.Header().Get("Vary") != "Authorization" || w.Header().Get("Cache-Control") != "private, max-age=60" {
		t.Errorf("missing private cache headers: %v", w.Header())
	}
}

// Self Serve pages with limit=500; the handler must clamp to the Ledger page cap.
func TestListMyDonations_ClampsLimit(t *testing.T) {
	capture := &filterCapturingLedger{list: &models.TransactionList{}}
	h := newListMyDonationsHandler(capture)

	req := withPrincipal(httptest.NewRequest(http.MethodGet, "/crowdfunding/me/donations?limit=500&offset=100", nil), &models.Principal{UserID: "michal", Username: "michal"})
	w := httptest.NewRecorder()
	h.ListMyDonations(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if capture.lastFilter.Limit != maxTransactionPageSize || capture.lastFilter.Offset != 100 {
		t.Errorf("filter limit/offset = %d/%d, want %d/100", capture.lastFilter.Limit, capture.lastFilter.Offset, maxTransactionPageSize)
	}
}
