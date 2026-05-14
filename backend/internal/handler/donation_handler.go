// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	donations, meta, err := h.svc.ListByInitiative(r.Context(), initiativeID, models.DonationFilter{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": donations,
		"meta": meta,
	})
}

// Create handles POST /v1/initiatives/{id}/donations — requires JWT.
func (h *DonationHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	initiativeID := chi.URLParam(r, "id")
	var input models.DonationCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}

	created, err := h.svc.Create(r.Context(), initiativeID, principal.UserID, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusCreated, created)
}
