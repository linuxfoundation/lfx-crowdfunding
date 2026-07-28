// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

type announcementService interface {
	List(ctx context.Context, initiativeID string, filter models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error)
	Create(ctx context.Context, initiativeID, callerUsername string, input models.AnnouncementCreateInput) (*models.Announcement, error)
	Update(ctx context.Context, initiativeID, announcementID, callerUsername string, input models.AnnouncementUpdateInput) (*models.Announcement, error)
	Delete(ctx context.Context, initiativeID, announcementID, callerUsername string) error
}

// AnnouncementHandler holds Chi handlers for the initiative announcements resource.
type AnnouncementHandler struct {
	svc announcementService
}

// NewAnnouncementHandler creates an AnnouncementHandler.
func NewAnnouncementHandler(svc announcementService) *AnnouncementHandler {
	return &AnnouncementHandler{svc: svc}
}

// List handles GET /v1/initiatives/{id}/announcements — public, paginated.
func (h *AnnouncementHandler) List(w http.ResponseWriter, r *http.Request) {
	initiativeID := chi.URLParam(r, "id")

	limit, offset, ok := parsePaginationParams(w, r)
	if !ok {
		return
	}

	announcements, meta, err := h.svc.List(r.Context(), initiativeID, models.AnnouncementFilter{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": announcements,
		"meta": meta,
	})
}

// Create handles POST /v1/me/initiatives/{id}/announcements — requires JWT + ownership.
func (h *AnnouncementHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	initiativeID := chi.URLParam(r, "id")

	var input models.AnnouncementCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}

	result, err := h.svc.Create(r.Context(), initiativeID, principal.Username, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusCreated, result)
}

// Update handles PUT /v1/me/initiatives/{id}/announcements/{announcementId} — requires JWT + ownership.
func (h *AnnouncementHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	initiativeID := chi.URLParam(r, "id")
	announcementID := chi.URLParam(r, "announcementId")

	var input models.AnnouncementUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}

	result, err := h.svc.Update(r.Context(), initiativeID, announcementID, principal.Username, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, result)
}

// Delete handles DELETE /v1/me/initiatives/{id}/announcements/{announcementId} — requires JWT + ownership.
func (h *AnnouncementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	initiativeID := chi.URLParam(r, "id")
	announcementID := chi.URLParam(r, "announcementId")

	if err := h.svc.Delete(r.Context(), initiativeID, announcementID, principal.Username); err != nil {
		Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
