// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// The service is nil: every case here must be answered before it is reached.
func TestLedgerAdjustmentHandler_ApproverGate(t *testing.T) {
	h := NewLedgerAdjustmentHandler(nil, []string{"Approver"})
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
			r := httptest.NewRequest(tt.method, "/crowdfunding/ledger-adjustments", strings.NewReader("{not json"))
			if tt.principal != nil {
				r = r.WithContext(auth.ContextWithPrincipal(r.Context(), tt.principal))
			}
			w := httptest.NewRecorder()
			tt.handler(w, r)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d (body %s)", w.Code, tt.want, w.Body.String())
			}
		})
	}
}
