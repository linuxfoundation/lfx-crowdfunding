// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// DonationHandler holds Chi handlers for the /v1/initiatives/{id}/donations resource.
type DonationHandler struct {
	svc *service.DonationService
}

// NewDonationHandler creates a DonationHandler.
func NewDonationHandler(svc *service.DonationService) *DonationHandler {
	return &DonationHandler{svc: svc}
}

// List handles GET /v1/initiatives/{id}/donations
func (h *DonationHandler) List(w http.ResponseWriter, r *http.Request) {
	initiativeID := chi.URLParam(r, "id")
	limit, offset, ok := parsePaginationParams(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()

	summaries, meta, err := h.svc.ListByInitiative(r.Context(), initiativeID, models.DonationFilter{
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": summaries,
		"meta": meta,
	})
}

// ListForUser handles GET /v1/me/donations — requires JWT.
// Returns the authenticated user's own donations across all initiatives, paginated.
func (h *DonationHandler) ListForUser(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	limit, offset, ok := parsePaginationParams(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()

	donations, meta, err := h.svc.ListByUser(r.Context(), principal.Username, models.DonationFilter{
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		Error(w, err)
		return
	}
	if donations == nil {
		donations = []models.Donation{}
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": donations,
		"meta": meta,
	})
}

// Create handles POST /v1/initiatives/{id}/donations — requires JWT.
// Clients MUST supply an Idempotency-Key header (a UUID they generate per
// logical donation attempt). The backend passes it verbatim to Stripe so
// that retries of the same timed-out request are de-duped rather than
// creating duplicate charges.
func (h *DonationHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		Error(w, fmt.Errorf("%w: Idempotency-Key header is required", domain.ErrInvalidInput))
		return
	}

	initiativeID := chi.URLParam(r, "id")
	var input models.DonationCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}
	input.IdempotencyKey = idempotencyKey

	created, err := h.svc.Create(r.Context(), initiativeID, principal.Username, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusCreated, created)
}
