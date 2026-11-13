// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

func offlineRequest(method, body string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(method, "/crowdfunding/offline-donations", strings.NewReader(body))
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// The service is nil: every case here must be answered before it is reached.
func TestOfflineDonationHandler_ApproverGate(t *testing.T) {
	h := NewOfflineDonationHandler(nil, []string{"Approver"})
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		method    string
		principal *models.Principal
		want      int
	}{
		{"create without principal", h.Create, http.MethodPost, nil, http.StatusUnauthorized},
		{"create by non-approver", h.Create, http.MethodPost, &models.Principal{Username: "someone"}, http.StatusForbidden},
		{"list without principal", h.List, http.MethodGet, nil, http.StatusUnauthorized},
		{"list by non-approver", h.List, http.MethodGet, &models.Principal{Username: "someone"}, http.StatusForbidden},
		{"create by approver with a bad body", h.Create, http.MethodPost, &models.Principal{Username: "approver"}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.handler(w, offlineRequest(tt.method, "{not json", tt.principal))
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d (body %s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestOfflineDonationHandler_GetApproverStatus(t *testing.T) {
	h := NewOfflineDonationHandler(nil, []string{"Approver"})
	tests := []struct {
		name      string
		principal *models.Principal
		wantCode  int
		want      bool
	}{
		{"approver, case-insensitive", &models.Principal{Username: "approver"}, http.StatusOK, true},
		{"not an approver", &models.Principal{Username: "someone"}, http.StatusOK, false},
		{"no principal", nil, http.StatusUnauthorized, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.GetApproverStatus(w, offlineRequest(http.MethodGet, "", tt.principal))
			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if tt.wantCode != http.StatusOK {
				return
			}
			var body map[string]bool
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["approver"] != tt.want {
				t.Errorf("approver = %v, want %v", body["approver"], tt.want)
			}
			if cc := w.Header().Get("Cache-Control"); cc != "private, no-store" {
				t.Errorf("Cache-Control = %q", cc)
			}
		})
	}
}
