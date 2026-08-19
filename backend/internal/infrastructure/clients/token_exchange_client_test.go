// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

func TestNewTokenExchangeClient_EmptyURL_returnsNil(t *testing.T) {
	if c := clients.NewTokenExchangeClient(clients.TokenExchangeConfig{}); c != nil {
		t.Fatal("expected nil client when Auth0TokenURL is empty")
	}
}

func newTEClient(t *testing.T, tokenURL string) clients.TokenExchangeClient {
	t.Helper()
	c := clients.NewTokenExchangeClient(clients.TokenExchangeConfig{
		Auth0TokenURL:    tokenURL,
		ClientID:         "cid",
		ClientSecret:     "csecret",
		SubjectTokenType: "https://lfx.dev/cf",
		Audience:         "https://lfx.dev/v2",
	})
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	return c
}

func TestExchange_200_returnsAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("grant_type"); got != "urn:ietf:params:oauth:grant-type:token-exchange" {
			t.Errorf("unexpected grant_type: %q", got)
		}
		if got := r.FormValue("subject_token"); got != "user-token" {
			t.Errorf("unexpected subject_token: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"v2-token","expires_in":86400}`))
	}))
	defer srv.Close()

	token, err := newTEClient(t, srv.URL).Exchange(context.Background(), "user-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "v2-token" {
		t.Errorf("expected v2-token, got %q", token)
	}
}

func TestExchange_CachesPerSubjectToken(t *testing.T) {
	fetches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetches++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"v2-token","expires_in":86400}`))
	}))
	defer srv.Close()

	c := newTEClient(t, srv.URL)
	for i := range 3 {
		if _, err := c.Exchange(context.Background(), "user-token"); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	if fetches != 1 {
		t.Errorf("expected 1 upstream fetch for 3 calls with same subject token, got %d", fetches)
	}

	if _, err := c.Exchange(context.Background(), "other-user-token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetches != 2 {
		t.Errorf("expected a fresh fetch for a different subject token, got %d fetches", fetches)
	}
}

func TestExchange_NonOK_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if _, err := newTEClient(t, srv.URL).Exchange(context.Background(), "user-token"); err == nil {
		t.Fatal("expected error on non-200 response")
	}
}

func TestExchange_EmptyAccessToken_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"","expires_in":86400}`))
	}))
	defer srv.Close()

	if _, err := newTEClient(t, srv.URL).Exchange(context.Background(), "user-token"); err == nil {
		t.Fatal("expected error when access_token is empty")
	}
}
