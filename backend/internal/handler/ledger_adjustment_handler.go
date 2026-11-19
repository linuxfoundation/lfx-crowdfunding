// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// LedgerAdjustmentHandler serves the approver-only return and payout routes
// (#489), gated like the offline donation routes.
type LedgerAdjustmentHandler struct {
	svc       *service.LedgerAdjustmentService
	approvers []string
}

// NewLedgerAdjustmentHandler returns a LedgerAdjustmentHandler.
func NewLedgerAdjustmentHandler(svc *service.LedgerAdjustmentService, approvers []string) *LedgerAdjustmentHandler {
	return &LedgerAdjustmentHandler{svc: svc, approvers: approvers}
}

// Create handles POST /crowdfunding/ledger-adjustments. 201 for a new
// adjustment; 200 with the stored one when the same one is submitted again.
func (h *LedgerAdjustmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := requireApprover(w, r, h.approvers)
	if !ok {
		return
	}
	var input models.LedgerAdjustmentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}
	adjustment, created, err := h.svc.Record(r.Context(), principal.Username, input)
	if err != nil {
		Error(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	JSON(w, status, adjustment)
}

// List handles GET /crowdfunding/ledger-adjustments?initiative_id=.
func (h *LedgerAdjustmentHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireApprover(w, r, h.approvers); !ok {
		return
	}
	limit, offset, ok := parsePaginationParams(w, r)
	if !ok {
		return
	}
	rows, meta, err := h.svc.List(r.Context(), strings.TrimSpace(r.URL.Query().Get("initiative_id")), limit, offset)
	if err != nil {
		Error(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	JSON(w, http.StatusOK, map[string]any{"data": rows, "meta": meta})
}
