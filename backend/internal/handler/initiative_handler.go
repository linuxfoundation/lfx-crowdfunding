// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// InitiativeHandler holds Chi handlers for the /v1/initiatives resource.
type InitiativeHandler struct {
	svc *service.InitiativeService
}

// NewInitiativeHandler creates an InitiativeHandler.
func NewInitiativeHandler(svc *service.InitiativeService) *InitiativeHandler {
	return &InitiativeHandler{svc: svc}
}

// List handles GET /v1/initiatives
func (h *InitiativeHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := models.InitiativeFilter{
		OwnerID:        q.Get("owner_id"),
		InitiativeType: q.Get("type"),
		Status:         q.Get("status"),
		Search:         q.Get("search"),
		SortBy:         strings.ToLower(q.Get("sort_by")),
		SortDir:        strings.ToLower(q.Get("sort_dir")),
		Limit:          limit,
		Offset:         offset,
	}
	initiatives, meta, err := h.svc.List(r.Context(), filter)
	if err != nil {
		Error(w, err)
		return
	}
	if initiatives == nil {
		initiatives = []*models.Initiative{}
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": initiatives,
		"meta": meta,
	})
}

// GetByID handles GET /v1/initiatives/{id} — accepts a slug or UUID.
// Slugs are the canonical public identifier; UUIDs are supported as a fallback.
func (h *InitiativeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var (
		initiative *models.Initiative
		err        error
	)
	if uuidPattern.MatchString(id) {
		initiative, err = h.svc.GetByID(r.Context(), id)
	} else {
		initiative, err = h.svc.GetBySlug(r.Context(), id)
	}
	if err != nil {
		Error(w, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	JSON(w, http.StatusOK, initiative)
}

// Create handles POST /v1/initiatives — requires JWT.
func (h *InitiativeHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	var input models.InitiativeCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}

	created, err := h.svc.Create(r.Context(), principal.UserID, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusCreated, created)
}

// Update handles PATCH /v1/initiatives/{id} — requires JWT.
func (h *InitiativeHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	var input models.InitiativeUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}

	updated, err := h.svc.Update(r.Context(), id, principal.UserID, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /v1/initiatives/{id} — requires JWT.
func (h *InitiativeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id, principal.UserID); err != nil {
		Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
