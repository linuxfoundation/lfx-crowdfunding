// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// OfflineDonationHandler serves the approver-only offline donation routes (#464).
// Heimdall checks the approvers team (when OpenFGA is on); the handler also
// checks ALLOWED_APPROVERS, as process-approval does.
type OfflineDonationHandler struct {
	svc       *service.OfflineDonationService
	approvers []string
}

// NewOfflineDonationHandler returns an OfflineDonationHandler.
func NewOfflineDonationHandler(svc *service.OfflineDonationService, approvers []string) *OfflineDonationHandler {
	return &OfflineDonationHandler{svc: svc, approvers: approvers}
}

// Create handles POST /crowdfunding/offline-donations. 201 for a new donation;
// 200 with the stored one when the same donation is submitted again.
func (h *OfflineDonationHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, ok := h.requireApprover(w, r)
	if !ok {
		return
	}
	var input models.OfflineDonationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}
	donation, created, err := h.svc.Record(r.Context(), principal.Username, input)
	if err != nil {
		Error(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	JSON(w, status, donation)
}

// List handles GET /crowdfunding/offline-donations?initiative_id=.
func (h *OfflineDonationHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireApprover(w, r); !ok {
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

// GetApproverStatus handles GET /crowdfunding/me/approver, which the UI uses to
// show approver-only actions.
func (h *OfflineDonationHandler) GetApproverStatus(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	JSON(w, http.StatusOK, map[string]bool{"approver": isApproverIn(h.approvers, principal)})
}

func (h *OfflineDonationHandler) requireApprover(w http.ResponseWriter, r *http.Request) (*models.Principal, bool) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return nil, false
	}
	if !isApproverIn(h.approvers, principal) {
		Error(w, domain.ErrForbidden)
		return nil, false
	}
	return principal, true
}
