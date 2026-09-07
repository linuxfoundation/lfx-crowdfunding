// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// ── GetAllMyTransactions ──────────────────────────────────────────────────────

func TestGetAllMyTransactions_NoPrincipal_Returns401(t *testing.T) {
	h := newMyTxnHandler(&initiativeRepo{}, &apprLedgerClient{})

	req := httptest.NewRequest(http.MethodGet, "/v1/me/transactions", nil)
	w := httptest.NewRecorder()
	h.GetAllMyTransactions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetAllMyTransactions_EmptyUserID_Returns401(t *testing.T) {
	h := newMyTxnHandler(&initiativeRepo{}, &apprLedgerClient{})

	req := httptest.NewRequest(http.MethodGet, "/v1/me/transactions", nil)
	req = withPrincipal(req, &models.Principal{Username: "user", UserID: ""})
	w := httptest.NewRecorder()
	h.GetAllMyTransactions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetAllMyTransactions_MixedUserRows_Returns503(t *testing.T) {
	callerUserID := "auth0|caller"
	foreignUserID := "auth0|other"

	mixedRows := &models.TransactionList{
		Data: []models.Transaction{
			{ID: "t1", AmountCents: 300, LedgerUserID: callerUserID, Date: time.Time{}},
			{ID: "t2", AmountCents: 200, LedgerUserID: foreignUserID, Date: time.Time{}},
		},
		TotalCount: 20,
		Limit:      10,
	}
	h := newMyTxnHandler(&initiativeRepo{}, &filterCapturingLedger{list: mixedRows})

	req := httptest.NewRequest(http.MethodGet, "/v1/me/transactions", nil)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetAllMyTransactions(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for mixed-user Ledger rows, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAllMyTransactions_ForwardsUserIDInLedgerFilter(t *testing.T) {
	// Verify that principal.UserID is forwarded to the Ledger and that no
	// ProjectID is set (this is the cross-initiative endpoint).
	callerUserID := "auth0|cross-user"

	capture := &filterCapturingLedger{
		list: &models.TransactionList{
			Data: []models.Transaction{
				{ID: "t1", AmountCents: 500, LedgerUserID: callerUserID, Date: time.Time{}},
			},
			TotalCount: 1,
			Limit:      10,
		},
	}
	h := newMyTxnHandler(&initiativeRepo{}, capture)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/transactions", nil)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetAllMyTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if capture.lastFilter.UserID != callerUserID {
		t.Errorf("Ledger filter.UserID = %q, want %q", capture.lastFilter.UserID, callerUserID)
	}
	if capture.lastFilter.ProjectID != "" {
		t.Errorf("Ledger filter.ProjectID = %q, want empty (cross-initiative endpoint omits project)", capture.lastFilter.ProjectID)
	}
}

func TestGetAllMyTransactions_PrivateCacheHeaders(t *testing.T) {
	callerUserID := "auth0|cache-user"

	ledger := &filterCapturingLedger{
		list: &models.TransactionList{
			Data: []models.Transaction{
				{ID: "t1", AmountCents: 100, LedgerUserID: callerUserID, Date: time.Time{}},
			},
			TotalCount: 1,
			Limit:      10,
		},
	}
	h := newMyTxnHandler(&initiativeRepo{}, ledger)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/transactions", nil)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetAllMyTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if cc := w.Header().Get("Cache-Control"); cc != "private, max-age=60" {
		t.Errorf("Cache-Control = %q, want %q", cc, "private, max-age=60")
	}
	if v := w.Header().Get("Vary"); v != "Authorization" {
		t.Errorf("Vary = %q, want %q", v, "Authorization")
	}
	if w.Header().Get("ETag") == "" {
		t.Error("ETag header must be set on 200 response")
	}
}

func TestGetAllMyTransactions_304IncludesCacheHeaders(t *testing.T) {
	callerUserID := "auth0|etag-user"

	ledger := &filterCapturingLedger{
		list: &models.TransactionList{
			Data: []models.Transaction{
				{ID: "t1", AmountCents: 200, LedgerUserID: callerUserID, Date: time.Time{}},
			},
			TotalCount: 1,
			Limit:      10,
		},
	}
	h := newMyTxnHandler(&initiativeRepo{}, ledger)

	// First request: obtain ETag.
	req1 := httptest.NewRequest(http.MethodGet, "/v1/me/transactions", nil)
	req1 = withPrincipal(req1, &models.Principal{UserID: callerUserID})
	w1 := httptest.NewRecorder()
	h.GetAllMyTransactions(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first response missing ETag")
	}

	// Second request: conditional GET must return 304 with cache headers.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/me/transactions", nil)
	req2 = withPrincipal(req2, &models.Principal{UserID: callerUserID})
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	h.GetAllMyTransactions(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Fatalf("second request: expected 304, got %d", w2.Code)
	}
	if w2.Header().Get("ETag") != etag {
		t.Errorf("304 ETag = %q, want %q", w2.Header().Get("ETag"), etag)
	}
	if cc := w2.Header().Get("Cache-Control"); cc != "private, max-age=60" {
		t.Errorf("304 Cache-Control = %q, want %q", cc, "private, max-age=60")
	}
	if v := w2.Header().Get("Vary"); v != "Authorization" {
		t.Errorf("304 Vary = %q, want %q", v, "Authorization")
	}
}

func TestGetAllMyTransactions_SubscriptionOnly_ForwardsFlag(t *testing.T) {
	callerUserID := "auth0|sub-user"

	capture := &filterCapturingLedger{
		list: &models.TransactionList{
			Data: []models.Transaction{
				{ID: "t1", AmountCents: 300, LedgerUserID: callerUserID},
			},
			TotalCount: 1,
			Limit:      10,
		},
	}
	h := newMyTxnHandler(&initiativeRepo{}, capture)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/transactions?subscriptionOnly=true", nil)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetAllMyTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !capture.lastFilter.SubscriptionOnly {
		t.Error("TransactionFilter.SubscriptionOnly should be true when ?subscriptionOnly=true is sent")
	}
}

func TestGetAllMyTransactions_Returns200WithData(t *testing.T) {
	callerUserID := "auth0|happy-user"

	ledger := &filterCapturingLedger{
		list: &models.TransactionList{
			Data: []models.Transaction{
				{ID: "t1", AmountCents: 300, LedgerUserID: callerUserID, Date: time.Time{}},
				{ID: "t2", AmountCents: 500, LedgerUserID: callerUserID, Date: time.Time{}},
			},
			TotalCount: 2,
			Limit:      10,
		},
	}
	h := newMyTxnHandler(&initiativeRepo{}, ledger)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/transactions", nil)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetAllMyTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got models.TransactionList
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Data) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(got.Data))
	}
}
