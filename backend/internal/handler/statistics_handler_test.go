// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

// statsRepo is a no-op StatisticsRepository for statistics handler tests.
type statsRepo struct{}

func (r *statsRepo) GetPlatformStatistics(_ context.Context) (*models.PlatformStatistics, error) {
	return &models.PlatformStatistics{}, nil
}
func (r *statsRepo) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	return map[string]models.Organization{}, nil
}
func (r *statsRepo) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return map[string]models.User{}, nil
}
func (r *statsRepo) GetUsersByLegacyIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return map[string]models.User{}, nil
}
func (r *statsRepo) GetInitiativeNamesByIDs(_ context.Context, _ []string) (map[string]string, error) {
	return map[string]string{}, nil
}

// statsLedgerClient is a configurable LedgerClient stub for statistics handler tests.
type statsLedgerClient struct {
	onGetPlatformBalance func(ctx context.Context, topLimit int) (*clients.LedgerPlatformBalance, error)
}

func (c *statsLedgerClient) GetBalance(_ context.Context, _ string) (*clients.LedgerBalance, error) {
	return nil, nil
}
func (c *statsLedgerClient) GetAllBalances(_ context.Context) ([]models.LedgerRawBalance, error) {
	return nil, nil
}
func (c *statsLedgerClient) GetTransactions(_ context.Context, _ clients.TransactionFilter) (*models.TransactionList, error) {
	return nil, nil
}
func (c *statsLedgerClient) GetPlatformBalance(ctx context.Context, topLimit int) (*clients.LedgerPlatformBalance, error) {
	if c.onGetPlatformBalance != nil {
		return c.onGetPlatformBalance(ctx, topLimit)
	}
	return &clients.LedgerPlatformBalance{}, nil
}
func (c *statsLedgerClient) GetPlatformMonthly(_ context.Context, _ int) (*clients.LedgerPlatformMonthly, error) {
	return &clients.LedgerPlatformMonthly{}, nil
}
func (c *statsLedgerClient) GetPlatformRecentDonations(_ context.Context) ([]clients.LedgerRecentDonation, error) {
	return nil, nil
}
func (c *statsLedgerClient) PostTransaction(_ context.Context, _ clients.LedgerTransaction) error {
	return nil
}

// newTestStatisticsHandler wires up a StatisticsHandler with the given ledger client stub.
func newTestStatisticsHandler(lc *statsLedgerClient) *StatisticsHandler {
	svc := service.NewStatisticsService(&statsRepo{}, lc)
	return NewStatisticsHandler(svc)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestGetPlatformDetails_DefaultTopLimit(t *testing.T) {
	var capturedLimit int
	lc := &statsLedgerClient{
		onGetPlatformBalance: func(_ context.Context, topLimit int) (*clients.LedgerPlatformBalance, error) {
			capturedLimit = topLimit
			return &clients.LedgerPlatformBalance{}, nil
		},
	}
	h := newTestStatisticsHandler(lc)

	r := httptest.NewRequest(http.MethodGet, "/v1/statistics/platform", nil)
	w := httptest.NewRecorder()
	h.GetPlatformDetails(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedLimit != 10 {
		t.Errorf("expected default topLimit=10, got %d", capturedLimit)
	}
}

func TestGetPlatformDetails_ValidTopLimit(t *testing.T) {
	var capturedLimit int
	lc := &statsLedgerClient{
		onGetPlatformBalance: func(_ context.Context, topLimit int) (*clients.LedgerPlatformBalance, error) {
			capturedLimit = topLimit
			return &clients.LedgerPlatformBalance{}, nil
		},
	}
	h := newTestStatisticsHandler(lc)

	r := httptest.NewRequest(http.MethodGet, "/v1/statistics/platform?top_limit=20", nil)
	w := httptest.NewRecorder()
	h.GetPlatformDetails(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedLimit != 20 {
		t.Errorf("expected topLimit=20, got %d", capturedLimit)
	}
}

func TestGetPlatformDetails_CapsAt100(t *testing.T) {
	var capturedLimit int
	lc := &statsLedgerClient{
		onGetPlatformBalance: func(_ context.Context, topLimit int) (*clients.LedgerPlatformBalance, error) {
			capturedLimit = topLimit
			return &clients.LedgerPlatformBalance{}, nil
		},
	}
	h := newTestStatisticsHandler(lc)

	r := httptest.NewRequest(http.MethodGet, "/v1/statistics/platform?top_limit=150", nil)
	w := httptest.NewRecorder()
	h.GetPlatformDetails(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedLimit != 100 {
		t.Errorf("expected topLimit capped at 100, got %d", capturedLimit)
	}
}

func TestGetPlatformDetails_InvalidTopLimit(t *testing.T) {
	cases := []struct {
		name  string
		query string
	}{
		{"non-integer", "?top_limit=abc"},
		{"negative", "?top_limit=-5"},
		{"zero", "?top_limit=0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestStatisticsHandler(&statsLedgerClient{})

			r := httptest.NewRequest(http.MethodGet, "/v1/statistics/platform"+tc.query, nil)
			w := httptest.NewRecorder()
			h.GetPlatformDetails(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
			var body map[string]any
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("response body is not valid JSON: %v", err)
			}
		})
	}
}
