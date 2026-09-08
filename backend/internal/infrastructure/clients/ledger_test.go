// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
