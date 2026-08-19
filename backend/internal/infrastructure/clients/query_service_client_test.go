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

func TestNewQueryServiceClient_EmptyBaseURL_returnsNil(t *testing.T) {
	if c := clients.NewQueryServiceClient(clients.QueryServiceConfig{}); c != nil {
		t.Fatal("expected nil client when BaseURL is empty")
	}
}

func newQSClient(t *testing.T, baseURL string) clients.QueryServiceClient {
	t.Helper()
	c := clients.NewQueryServiceClient(clients.QueryServiceConfig{BaseURL: baseURL})
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	return c
}

func TestSearchOrganizationsForUser_200_stripsIDPrefixAndSetsBearer(t *testing.T) {
	var gotAuth, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resources":[{"type":"b2b_org","id":"b2b_org:0012M00002qnukOQAQ","data":{"name":"Acme","logo_url":"https://x/acme.png"}}]}`))
	}))
	defer srv.Close()

	orgs, err := newQSClient(t, srv.URL).SearchOrganizationsForUser(context.Background(), "v2-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer v2-token" {
		t.Errorf("expected Authorization: Bearer v2-token, got %q", gotAuth)
	}
	if gotQuery != "v=1&type=b2b_org&filter_grants=direct" {
		t.Errorf("unexpected query: %q", gotQuery)
	}
	if len(orgs) != 1 {
		t.Fatalf("expected 1 org, got %d", len(orgs))
	}
	want := clients.OrgCandidate{UID: "0012M00002qnukOQAQ", Name: "Acme", LogoURL: "https://x/acme.png"}
	if orgs[0] != want {
		t.Errorf("expected %+v, got %+v", want, orgs[0])
	}
}

func TestSearchOrganizationsForUser_Empty_returnsEmptySlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resources":[]}`))
	}))
	defer srv.Close()

	orgs, err := newQSClient(t, srv.URL).SearchOrganizationsForUser(context.Background(), "v2-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgs == nil || len(orgs) != 0 {
		t.Errorf("expected empty (non-nil) slice, got %+v", orgs)
	}
}

func TestSearchOrganizationsForUser_NonOK_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if _, err := newQSClient(t, srv.URL).SearchOrganizationsForUser(context.Background(), "v2-token"); err == nil {
		t.Fatal("expected error on non-200 response")
	}
}
