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

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// userBalanceLedger records the userID passed to GetUserBalance.
type userBalanceLedger struct {
	apprLedgerClient
	lastUserID string
	balance    *clients.LedgerUserBalance
	err        error
}

func (c *userBalanceLedger) GetUserBalance(_ context.Context, userID string) (*clients.LedgerUserBalance, error) {
	c.lastUserID = userID
	return c.balance, c.err
}

func newDonationStatsHandler(ledger *userBalanceLedger) *InitiativeHandler {
	svc := service.NewInitiativeService(&initiativeRepo{}, &initiativeUserRepo{user: &models.User{LegacyUserID: "auth0|michal"}},
		ledger, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	return NewInitiativeHandler(svc, nil, slog.Default())
}

func TestGetMyDonationStats_NoPrincipal_Returns401(t *testing.T) {
	h := newDonationStatsHandler(&userBalanceLedger{})
	w := httptest.NewRecorder()
	h.GetMyDonationStats(w, httptest.NewRequest(http.MethodGet, "/crowdfunding/me/donations/stats", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// The Ledger keys rows by legacy user ID, so the aggregate must be requested under
// it rather than the Heimdall username subject.
func TestGetMyDonationStats_UsesLegacyUserID(t *testing.T) {
	ledger := &userBalanceLedger{balance: &clients.LedgerUserBalance{TotalCreditCents: 125000, ProjectsSupported: 3}}
	h := newDonationStatsHandler(ledger)

	req := withPrincipal(httptest.NewRequest(http.MethodGet, "/crowdfunding/me/donations/stats", nil), &models.Principal{UserID: "michal", Username: "michal"})
	w := httptest.NewRecorder()
	h.GetMyDonationStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ledger.lastUserID != "auth0|michal" {
		t.Errorf("ledger userID = %q, want auth0|michal", ledger.lastUserID)
	}
	var body models.DonationStats
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TotalDonatedCents != 125000 || body.InitiativesSupported != 3 {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
	if w.Header().Get("Vary") != "Authorization" || w.Header().Get("Cache-Control") != "private, max-age=60" {
		t.Errorf("missing private cache headers: %v", w.Header())
	}
}

func TestGetMyDonationStats_LedgerError_Returns503(t *testing.T) {
	h := newDonationStatsHandler(&userBalanceLedger{err: domain.ErrUpstreamUnavailable})

	req := withPrincipal(httptest.NewRequest(http.MethodGet, "/crowdfunding/me/donations/stats", nil), &models.Principal{UserID: "michal", Username: "michal"})
	w := httptest.NewRecorder()
	h.GetMyDonationStats(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
