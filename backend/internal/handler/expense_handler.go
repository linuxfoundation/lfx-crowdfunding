// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// ExpenseHandler proxies expense-report actions to the Reimbursement Service.
type ExpenseHandler struct {
	rsClient clients.ReimbursementClient
}

// NewExpenseHandler creates an ExpenseHandler backed by the given client.
func NewExpenseHandler(rsClient clients.ReimbursementClient) *ExpenseHandler {
	return &ExpenseHandler{rsClient: rsClient}
}

// ProcessAction handles POST /v1/expense/{action}/{reportId}.
// It forwards the action and report ID to the Reimbursement Service and returns
// 204 No Content on success. The Reimbursement Service is authenticated via
// X-API-KEY; this endpoint only requires the caller to hold a valid bearer
// token — no specific scope is enforced (called by the CF frontend).
func (h *ExpenseHandler) ProcessAction(w http.ResponseWriter, r *http.Request) {
	if h.rsClient == nil {
		Error(w, domain.ErrUpstreamUnavailable)
		return
	}

	action := chi.URLParam(r, "action")
	reportID := chi.URLParam(r, "reportId")

	if action != "approve" && action != "reject" {
		Error(w, fmt.Errorf("%w: expense action %q is not supported; use \"approve\" or \"reject\"", domain.ErrInvalidInput, action))
		return
	}

	// Extract the raw token (without "Bearer " prefix) to forward to the RS.
	// The RS API gateway requires the user's token alongside X-API-KEY.
	actorToken := ""
	if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
		actorToken = auth[7:]
	}

	if err := h.rsClient.ProcessExpenseAction(r.Context(), action, reportID, actorToken); err != nil {
		Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
