// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── UserInfo.EffectiveEmail ───────────────────────────────────────────────────

func TestEffectiveEmail_SSOClaimWins(t *testing.T) {
	u := &UserInfo{
		SSOEmail: "sso@example.com",
		Email:    "standard@example.com",
	}
	if got := u.EffectiveEmail(); got != "sso@example.com" {
		t.Errorf("EffectiveEmail = %q, want sso claim %q", got, "sso@example.com")
	}
}

func TestEffectiveEmail_FallsBackToStandardWhenSSOAbsent(t *testing.T) {
	u := &UserInfo{
		SSOEmail: "",
		Email:    "standard@example.com",
	}
	if got := u.EffectiveEmail(); got != "standard@example.com" {
		t.Errorf("EffectiveEmail = %q, want standard claim %q", got, "standard@example.com")
	}
}

func TestEffectiveEmail_SSOWhitespaceOnlyFallsBack(t *testing.T) {
	u := &UserInfo{
		SSOEmail: "   ",
		Email:    "standard@example.com",
	}
	if got := u.EffectiveEmail(); got != "standard@example.com" {
		t.Errorf("EffectiveEmail = %q, want standard claim when SSO is whitespace", got)
	}
}

// ── UserInfoClient.FetchUserInfo ──────────────────────────────────────────────

func TestFetchUserInfo_Success(t *testing.T) {
	want := &UserInfo{
		Sub:        "auth0|abc",
		Username:   "jdoe",
		SSOEmail:   "jdoe@lf.org",
		Email:      "jdoe@personal.com",
		Name:       "Jane Doe",
		GivenName:  "Jane",
		FamilyName: "Doe",
		Picture:    "https://cdn.example.com/jdoe.png",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		// Verify the Authorization header is forwarded correctly.
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("Authorization header = %q, want Bearer prefix", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewUserInfoClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("NewUserInfoClient: %v", err)
	}

	got, err := client.FetchUserInfo(context.Background(), "test-access-token")
	if err != nil {
		t.Fatalf("FetchUserInfo: %v", err)
	}

	if got.Sub != want.Sub {
		t.Errorf("Sub = %q, want %q", got.Sub, want.Sub)
	}
	if got.SSOEmail != want.SSOEmail {
		t.Errorf("SSOEmail = %q, want %q", got.SSOEmail, want.SSOEmail)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	// Verify SSO claim takes precedence in EffectiveEmail.
	if got.EffectiveEmail() != want.SSOEmail {
		t.Errorf("EffectiveEmail = %q, want SSO claim %q", got.EffectiveEmail(), want.SSOEmail)
	}
}

func TestFetchUserInfo_Non200Response_ReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewUserInfoClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("NewUserInfoClient: %v", err)
	}

	_, err = client.FetchUserInfo(context.Background(), "expired-token")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
	if !errors.Is(err, ErrUserInfoTokenRejected) {
		t.Errorf("expected ErrUserInfoTokenRejected, got: %v", err)
	}
}

func TestFetchUserInfo_5xxResponse_ReturnsUnavailableError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewUserInfoClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("NewUserInfoClient: %v", err)
	}

	_, err = client.FetchUserInfo(context.Background(), "token")
	if err == nil {
		t.Fatal("expected error for 5xx response, got nil")
	}
	if !errors.Is(err, ErrUserInfoUnavailable) {
		t.Errorf("expected ErrUserInfoUnavailable, got: %v", err)
	}
}

func TestFetchUserInfo_OversizedBody_ReturnsDecodeError(t *testing.T) {
	// Serve a body larger than 64 KB — LimitReader should truncate it and
	// json.Decode should return an error (unexpected EOF or invalid JSON).
	mux := http.NewServeMux()
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Write 65 KB of valid-looking JSON prefix (no closing brace) so the
		// truncated read produces malformed JSON.
		_, _ = w.Write([]byte(`{"sub":"` + strings.Repeat("x", 65*1024) + `"`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewUserInfoClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("NewUserInfoClient: %v", err)
	}

	_, err = client.FetchUserInfo(context.Background(), "token")
	if err == nil {
		t.Fatal("expected decode error for oversized body, got nil")
	}
}

func TestFetchUserInfo_InvalidJSON_ReturnsDecodeError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewUserInfoClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("NewUserInfoClient: %v", err)
	}

	_, err = client.FetchUserInfo(context.Background(), "token")
	if err == nil {
		t.Fatal("expected decode error for invalid JSON, got nil")
	}
}

func TestNewUserInfoClient_EmptyIssuer_ReturnsError(t *testing.T) {
	_, err := NewUserInfoClient("", nil)
	if err == nil {
		t.Fatal("expected error for empty issuer, got nil")
	}
}

func TestNewUserInfoClient_NilHTTPClient_UsesDefault(t *testing.T) {
	// Passing nil httpClient should not panic — a default client is used.
	client, err := NewUserInfoClient("https://example.auth0.com", nil)
	if err != nil {
		t.Fatalf("NewUserInfoClient with nil http.Client: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// ── MockUserInfoFetcher ───────────────────────────────────────────────────────

func TestMockUserInfoFetcher_ReturnsDerivedProfile(t *testing.T) {
	fetcher := NewMockUserInfoFetcher("jdoe")

	info, err := fetcher.FetchUserInfo(context.Background(), "ignored-token")
	if err != nil {
		t.Fatalf("FetchUserInfo: %v", err)
	}
	if info.Username != "jdoe" {
		t.Errorf("Username = %q, want jdoe", info.Username)
	}
	if info.Sub != "jdoe" {
		t.Errorf("Sub = %q, want jdoe", info.Sub)
	}
	if info.Email != "jdoe@local.dev" {
		t.Errorf("Email = %q, want jdoe@local.dev", info.Email)
	}
}
