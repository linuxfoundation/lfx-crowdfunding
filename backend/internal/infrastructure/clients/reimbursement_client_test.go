// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// newRSClient creates a reimbursementHTTPClient pointed at the given test server.
func newRSClient(t *testing.T, serverURL string) clients.ReimbursementClient {
	t.Helper()
	cfg := clients.ReimbursementConfig{
		APIURL:  serverURL,
		APIKey:  "test-key",
		Timeout: 0, // no timeout in tests
	}
	c := clients.NewReimbursementClient(cfg)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	return c
}

func TestProcessExpenseAction_404_mapsToNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domain.ErrExpenseReportNotFound) {
		t.Errorf("expected ErrExpenseReportNotFound, got: %v", err)
	}
}

func TestProcessExpenseAction_HTTPError_mapsToUpstreamUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable, got: %v", err)
	}
}

func TestProcessExpenseAction_NetworkError_mapsToUpstreamUnavailable(t *testing.T) {
	// Point at a server that is immediately closed so the request fails at the
	// transport layer (non-HTTP error — no *rsHTTPError wrapping possible).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // close before the request is made

	err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable for network failure, got: %v", err)
	}
}

func TestProcessExpenseAction_200_returnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001")
	if err != nil {
		t.Errorf("expected nil error on 200, got: %v", err)
	}
}
