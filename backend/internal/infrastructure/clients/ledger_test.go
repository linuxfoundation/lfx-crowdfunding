// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

func TestLedgerGetTransactions_MapsProjectIDToLedgerProjectID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transactions" {
			t.Fatalf("path = %q, want /transactions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"totalTransactionCount": 1,
			"transactionsPerPage": 10,
			"currentPage": 1,
			"hasNext": false,
			"transactions": [{
				"txnID": "txn-1",
				"projectID": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				"userID": "auth0|user1",
				"organizationID": "",
				"accountEmail": "",
				"submitterName": "Alice",
				"txnType": "credit",
				"txnCategory": "Mentorship",
				"amount": 100,
				"txnDate": 1735689600,
				"subscriptionID": ""
			}]
		}`))
	}))
	defer srv.Close()

	client := NewLedgerClient(LedgerConfig{BaseURL: srv.URL, APIKey: "test", Timeout: 2 * time.Second})
	list, err := client.GetTransactions(context.Background(), TransactionFilter{ProjectID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("GetTransactions error: %v", err)
	}
	if len(list.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(list.Data))
	}
	if list.Data[0].LedgerProjectID != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Errorf("LedgerProjectID = %q, want mapped projectID", list.Data[0].LedgerProjectID)
	}
}

func TestLedgerGetTransactions_NameOnlyOrgIsOrganizationDonor(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transactions": [{
			"txnID": "txn-1",
			"userID": "auth0|user1",
			"organizationName": "Acme Inc.",
			"organizationDomain": "acme.com",
			"txnType": "credit",
			"amount": 100
		}]}`))
	}))
	defer srv.Close()

	client := NewLedgerClient(LedgerConfig{BaseURL: srv.URL, APIKey: "test", Timeout: 2 * time.Second})
	list, err := client.GetTransactions(context.Background(), TransactionFilter{ProjectID: "p", Limit: 10})
	if err != nil {
		t.Fatalf("GetTransactions error: %v", err)
	}
	got := list.Data[0]
	if got.DonorType != "organization" || got.LedgerOrgName != "Acme Inc." || got.LedgerOrgDomain != "acme.com" {
		t.Errorf("got DonorType=%q LedgerOrgName=%q LedgerOrgDomain=%q, want organization/Acme Inc./acme.com",
			got.DonorType, got.LedgerOrgName, got.LedgerOrgDomain)
	}
}

func TestLedgerGetOrgDonationTotals_SendsKeysAndDecodes(t *testing.T) {
	t.Parallel()

	var gotKeys string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKeys = r.URL.Query().Get("keys")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"orgId": "cncf.io", "amount_in_cents": 500}, {"orgId": "c64448e2", "amount_in_cents": 200}]`))
	}))
	defer srv.Close()

	client := NewLedgerClient(LedgerConfig{BaseURL: srv.URL, APIKey: "test", Timeout: 2 * time.Second})
	totals, err := client.GetOrgDonationTotals(context.Background(), []string{"cncf.io", "c64448e2"})
	if err != nil {
		t.Fatalf("GetOrgDonationTotals error: %v", err)
	}
	if gotKeys != "cncf.io,c64448e2" {
		t.Errorf("keys query: got %q", gotKeys)
	}
	if totals["cncf.io"] != 500 || totals["c64448e2"] != 200 {
		t.Errorf("totals: got %v", totals)
	}
}

func TestLedgerGetUserBalance_EscapesUserIDAndDecodes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/balance/user/auth0%7Cabc" {
			t.Errorf("path = %q, want /balance/user/auth0%%7Cabc", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userID":"auth0|abc","totalCredit":125000,"projectsSupported":3}`))
	}))
	defer srv.Close()

	client := NewLedgerClient(LedgerConfig{BaseURL: srv.URL, APIKey: "test", Timeout: 2 * time.Second})
	b, err := client.GetUserBalance(context.Background(), "auth0|abc")
	if err != nil {
		t.Fatalf("GetUserBalance error: %v", err)
	}
	if b.TotalCreditCents != 125000 || b.ProjectsSupported != 3 {
		t.Errorf("got %+v, want totalCredit 125000, projectsSupported 3", b)
	}
}

func TestLedgerGetUserBalance_UpstreamError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewLedgerClient(LedgerConfig{BaseURL: srv.URL, APIKey: "test", Timeout: 2 * time.Second})
	if _, err := client.GetUserBalance(context.Background(), "auth0|abc"); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}

func TestLedgerGetUserBalance_TransportError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	client := NewLedgerClient(LedgerConfig{BaseURL: srv.URL, APIKey: "test", Timeout: 2 * time.Second})
	if _, err := client.GetUserBalance(context.Background(), "auth0|abc"); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("err = %v, want ErrUpstreamUnavailable", err)
	}
}
