// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// generateTestPrivateKeyPEM generates a 2048-bit RSA private key and returns
// it PEM-encoded in PKCS8 format. Suitable for use as Auth0ClientPrivateKey in tests.
func generateTestPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test RSA key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal test RSA key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

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

	err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001", nil)
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

	err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001", nil)
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

	err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001", nil)
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

	err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001", nil)
	if err != nil {
		t.Errorf("expected nil error on 200, got: %v", err)
	}
}

// newRSClientM2M creates a client with M2M config pointing at tokenURL for the
// Auth0 token endpoint and rsURL for the RS API.
func newRSClientM2M(t *testing.T, rsURL, tokenURL string) clients.ReimbursementClient {
	t.Helper()
	cfg := clients.ReimbursementConfig{
		APIURL:                rsURL,
		APIKey:                "test-key",
		Timeout:               0,
		Auth0TokenURL:         tokenURL,
		Auth0ClientID:         "cid",
		Auth0ClientPrivateKey: generateTestPrivateKeyPEM(t),
		Auth0Audience:         "https://rs.example.com",
	}
	c := clients.NewReimbursementClient(cfg)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	return c
}

func TestProcessExpenseAction_M2M_AddsBearer(t *testing.T) {
	var gotAuth string
	rsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer rsSrv.Close()

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"my-m2m-token","expires_in":86400}`))
	}))
	defer tokenSrv.Close()

	err := newRSClientM2M(t, rsSrv.URL, tokenSrv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer my-m2m-token" {
		t.Errorf("expected Authorization: Bearer my-m2m-token, got %q", gotAuth)
	}
}

func TestProcessExpenseAction_M2M_CachesToken(t *testing.T) {
	tokenFetches := 0
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tokenFetches++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":86400}`))
	}))
	defer tokenSrv.Close()

	rsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer rsSrv.Close()

	c := newRSClientM2M(t, rsSrv.URL, tokenSrv.URL)
	for i := range 3 {
		if err := c.ProcessExpenseAction(context.Background(), "approve", "R-001", nil); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	if tokenFetches != 1 {
		t.Errorf("expected 1 token fetch for 3 calls, got %d", tokenFetches)
	}
}

func TestProcessExpenseAction_M2M_TokenFetchFails_mapsToUpstreamUnavailable(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer tokenSrv.Close()

	rsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer rsSrv.Close()

	err := newRSClientM2M(t, rsSrv.URL, tokenSrv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001", nil)
	if err == nil {
		t.Fatal("expected error when token fetch fails")
	}
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable, got: %v", err)
	}
}

func TestProcessExpenseAction_M2M_EmptyAccessToken_errors(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"","expires_in":86400}`))
	}))
	defer tokenSrv.Close()

	rsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer rsSrv.Close()

	err := newRSClientM2M(t, rsSrv.URL, tokenSrv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001", nil)
	if err == nil {
		t.Fatal("expected error when access_token is empty")
	}
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable, got: %v", err)
	}
}

func TestProcessExpenseAction_M2M_ShortTTL_doesNotCacheExpired(t *testing.T) {
	// expires_in=1 — buffer must be clamped (ttl/2 = 500ms), expiry stays in the future.
	fetchCount := 0
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetchCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"short-tok","expires_in":1}`))
	}))
	defer tokenSrv.Close()

	rsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer rsSrv.Close()

	c := newRSClientM2M(t, rsSrv.URL, tokenSrv.URL)
	// Two back-to-back calls — token should be cached (expiry is in the future).
	for i := range 2 {
		if err := c.ProcessExpenseAction(context.Background(), "approve", "R-001", nil); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	if fetchCount != 1 {
		t.Errorf("expected 1 token fetch for short TTL, got %d (expiry was immediately in the past)", fetchCount)
	}
}

func TestSyncPolicy_EscapesSlugInProjectURL(t *testing.T) {
	var body struct {
		ProjectURL string `json:"ProjectURL"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := clients.NewReimbursementClient(clients.ReimbursementConfig{
		APIURL:       srv.URL,
		APIKey:       "test-key",
		FrontendBase: "https://example.com",
	})
	initiative := &models.Initiative{
		ID:     "init-1",
		Slug:   "x/process-approval/decline#",
		Status: models.StatusPublished,
	}

	if err := c.SyncPolicy(context.Background(), initiative, &models.User{}); err != nil {
		t.Fatalf("SyncPolicy: %v", err)
	}
	want := "https://example.com/initiatives/x%2Fprocess-approval%2Fdecline%23"
	if body.ProjectURL != want {
		t.Errorf("ProjectURL = %q, want %q", body.ProjectURL, want)
	}
}

func TestProcessExpenseAction_SendsActorInBody(t *testing.T) {
	var got map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	actor := &models.Principal{Username: "jane", Email: "jane@example.org"}
	if err := newRSClient(t, srv.URL).ProcessExpenseAction(context.Background(), "approve", "R-001", actor); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["actorEmail"] != "jane@example.org" || got["actorUsername"] != "jane" {
		t.Errorf("expected actor in body, got %v", got)
	}
}

func TestGetExpenseApprovers_200_returnsEmails(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		_, _ = w.Write([]byte(`{"approverEmails":["owner@example.org","tf-admin@example.org"]}`))
	}))
	defer srv.Close()

	emails, err := newRSClient(t, srv.URL).GetExpenseApprovers(context.Background(), "R-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/expense/R-001/approvers" {
		t.Errorf("unexpected request %s %s", gotMethod, gotPath)
	}
	if len(emails) != 2 || emails[0] != "owner@example.org" || emails[1] != "tf-admin@example.org" {
		t.Errorf("unexpected emails: %v", emails)
	}
}

func TestGetExpenseApprovers_404_mapsToNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := newRSClient(t, srv.URL).GetExpenseApprovers(context.Background(), "R-001")
	if !errors.Is(err, domain.ErrExpenseReportNotFound) {
		t.Errorf("expected ErrExpenseReportNotFound, got: %v", err)
	}
}

func TestGetExpenseApprovers_HTTPError_mapsToUpstreamUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newRSClient(t, srv.URL).GetExpenseApprovers(context.Background(), "R-001")
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable, got: %v", err)
	}
}
