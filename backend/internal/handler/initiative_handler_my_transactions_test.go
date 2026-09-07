// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// filterCapturingLedger records the last TransactionFilter passed to GetTransactions
// and returns the configured response. Other methods delegate to apprLedgerClient.
type filterCapturingLedger struct {
	apprLedgerClient
	lastFilter clients.TransactionFilter
	list       *models.TransactionList
	err        error
}

func (c *filterCapturingLedger) GetTransactions(_ context.Context, f clients.TransactionFilter) (*models.TransactionList, error) {
	c.lastFilter = f
	if c.err != nil {
		return nil, c.err
	}
	if c.list != nil {
		return c.list, nil
	}
	return &models.TransactionList{}, nil
}

// newMyTxnHandler wires up a handler backed by the given Ledger stub.
func newMyTxnHandler(repo *initiativeRepo, ledger clients.LedgerClient) *InitiativeHandler {
	svc := service.NewInitiativeService(repo, &initiativeUserRepo{}, ledger, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	return NewInitiativeHandler(svc, nil, slog.Default())
}

// ── GetMyTransactions ─────────────────────────────────────────────────────────

func TestGetMyTransactions_NoPrincipal_Returns401(t *testing.T) {
	h := newMyTxnHandler(&initiativeRepo{}, &apprLedgerClient{})

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/some-id/my-transactions", nil)
	req = withURLParam(req, "id", "some-id")
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetMyTransactions_EmptyUserID_Returns401(t *testing.T) {
	h := newMyTxnHandler(&initiativeRepo{}, &apprLedgerClient{})

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/some-id/my-transactions", nil)
	req = withURLParam(req, "id", "some-id")
	// Principal present but UserID is empty (Username-only token from an older flow).
	req = withPrincipal(req, &models.Principal{Username: "user", UserID: ""})
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetMyTransactions_UnpublishedByID_Returns404(t *testing.T) {
	initiativeID := "e1e1e1e1-e1e1-e1e1-e1e1-e1e1e1e1e1e1"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusSubmitted, // not published
		},
	}
	h := newMyTxnHandler(repo, &apprLedgerClient{})

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/my-transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{UserID: "auth0|u1"})
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unpublished initiative by ID, got %d", w.Code)
	}
}

func TestGetMyTransactions_UnpublishedBySlug_Returns404(t *testing.T) {
	// Slug lookup delegates to repo.GetIDBySlug, which returns ErrInitiativeNotFound
	// when the initiative is not published.
	repo := &initiativeRepo{getErr: domain.ErrInitiativeNotFound}
	h := newMyTxnHandler(repo, &apprLedgerClient{})

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/my-project/my-transactions", nil)
	req = withURLParam(req, "id", "my-project") // slug (no UUID pattern)
	req = withPrincipal(req, &models.Principal{UserID: "auth0|u1"})
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unpublished initiative by slug, got %d", w.Code)
	}
}

func TestGetMyTransactions_MixedUserRows_Returns503(t *testing.T) {
	// When the Ledger returns rows from another user, the service detects missing
	// server-side filtering and returns ErrUpstreamUnavailable → 503.
	initiativeID := "e2e2e2e2-e2e2-e2e2-e2e2-e2e2e2e2e2e2"
	callerUserID := "auth0|caller"
	foreignUserID := "auth0|other"

	mixedRows := &models.TransactionList{
		Data: []models.Transaction{
			{ID: "t1", AmountCents: 300, LedgerUserID: callerUserID, Date: time.Time{}},
			{ID: "t2", AmountCents: 200, LedgerUserID: foreignUserID, Date: time.Time{}}, // foreign
		},
		TotalCount: 20,
		Limit:      10,
	}
	ledger := &filterCapturingLedger{list: mixedRows}
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusPublished,
		},
	}
	h := newMyTxnHandler(repo, ledger)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/my-transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for mixed-user Ledger rows, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetMyTransactions_ForwardsUserIDInLedgerFilter(t *testing.T) {
	// Verify that principal.UserID is forwarded as TransactionFilter.UserID to
	// the Ledger client so server-side filtering can apply.
	initiativeID := "e3e3e3e3-e3e3-e3e3-e3e3-e3e3e3e3e3e3"
	callerUserID := "auth0|caller-fwd"

	ledger := &filterCapturingLedger{
		list: &models.TransactionList{
			Data: []models.Transaction{
				{ID: "t1", AmountCents: 500, LedgerUserID: callerUserID, Date: time.Time{}},
			},
			TotalCount: 1,
			Limit:      10,
		},
	}
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusPublished,
		},
	}
	h := newMyTxnHandler(repo, ledger)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/my-transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ledger.lastFilter.UserID != callerUserID {
		t.Errorf("Ledger filter.UserID = %q, want %q", ledger.lastFilter.UserID, callerUserID)
	}
	if ledger.lastFilter.ProjectID != initiativeID {
		t.Errorf("Ledger filter.ProjectID = %q, want %q", ledger.lastFilter.ProjectID, initiativeID)
	}
}

func TestGetMyTransactions_PrivateCacheHeaders(t *testing.T) {
	// 200 response must carry Cache-Control: private and Vary: Authorization
	// so no shared cache can serve one user's transactions to another.
	initiativeID := "e4e4e4e4-e4e4-e4e4-e4e4-e4e4e4e4e4e4"
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
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusPublished,
		},
	}
	h := newMyTxnHandler(repo, ledger)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/my-transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

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

func TestGetMyTransactions_304IncludesCacheHeaders(t *testing.T) {
	// A conditional GET that hits the ETag must still receive Vary, Cache-Control,
	// and ETag on the 304 response. RFC 9110 §15.4.5 requires that 304 carries
	// the same validator fields that would accompany a 200.
	initiativeID := "e5e5e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e5"
	callerUserID := "auth0|etag-user"

	txnList := &models.TransactionList{
		Data: []models.Transaction{
			{ID: "t1", AmountCents: 200, LedgerUserID: callerUserID, Date: time.Time{}},
		},
		TotalCount: 1,
		Limit:      10,
	}
	ledger := &filterCapturingLedger{list: txnList}
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusPublished,
		},
	}
	h := newMyTxnHandler(repo, ledger)

	// First request: get the ETag.
	req1 := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/my-transactions", nil)
	req1 = withURLParam(req1, "id", initiativeID)
	req1 = withPrincipal(req1, &models.Principal{UserID: callerUserID})
	w1 := httptest.NewRecorder()
	h.GetMyTransactions(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first response missing ETag")
	}

	// Second request: conditional GET with matching ETag.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/my-transactions", nil)
	req2 = withURLParam(req2, "id", initiativeID)
	req2 = withPrincipal(req2, &models.Principal{UserID: callerUserID})
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	h.GetMyTransactions(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Fatalf("second request: expected 304, got %d", w2.Code)
	}
	// RFC 9110: the 304 MUST carry the same cache-related headers as the 200.
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

func TestGetMyTransactions_SubscriptionOnly_ForwardsFlag(t *testing.T) {
	// Verify that ?subscriptionOnly=true is parsed and forwarded to the Ledger
	// client as TransactionFilter.SubscriptionOnly = true.
	initiativeID := "e7e7e7e7-e7e7-e7e7-e7e7-e7e7e7e7e7e7"
	callerUserID := "auth0|sub-filter-user"

	capture := &filterCapturingLedger{
		list: &models.TransactionList{
			Data: []models.Transaction{
				{ID: "t1", AmountCents: 300, LedgerUserID: callerUserID},
			},
			TotalCount: 1,
			Limit:      10,
		},
	}
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusPublished,
		},
	}
	h := newMyTxnHandler(repo, capture)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/my-transactions?subscriptionOnly=true", nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !capture.lastFilter.SubscriptionOnly {
		t.Error("TransactionFilter.SubscriptionOnly should be true when ?subscriptionOnly=true is sent")
	}
}

func TestGetMyTransactions_Returns200WithData(t *testing.T) {
	initiativeID := "e6e6e6e6-e6e6-e6e6-e6e6-e6e6e6e6e6e6"
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
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusPublished,
		},
	}
	h := newMyTxnHandler(repo, ledger)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/my-transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{UserID: callerUserID})
	w := httptest.NewRecorder()
	h.GetMyTransactions(w, req)

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
