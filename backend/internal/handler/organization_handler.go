// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

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

type organizationService interface {
	ListByOwner(ctx context.Context, username string) ([]models.Organization, error)
	Create(ctx context.Context, username string, input models.OrganizationCreateInput) (*models.Organization, error)
	Update(ctx context.Context, username string, id string, input models.OrganizationUpdateInput) (*models.Organization, error)
}

// OrganizationHandler holds Chi handlers for organization resources.
type OrganizationHandler struct {
	svc organizationService
}

// NewOrganizationHandler creates an OrganizationHandler.
func NewOrganizationHandler(svc organizationService) *OrganizationHandler {
	return &OrganizationHandler{svc: svc}
}

// List handles GET /v1/me/organizations.
func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	orgs, err := h.svc.ListByOwner(r.Context(), principal.Username)
	if err != nil {
		Error(w, err)
		return
	}
	if orgs == nil {
		orgs = []models.Organization{}
	}
	JSON(w, http.StatusOK, orgs)
}

// Create handles POST /v1/me/organizations.
func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	var input models.OrganizationCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}

	org, err := h.svc.Create(r.Context(), principal.Username, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusCreated, org)
}

// Update handles PATCH /v1/me/organizations/{id}.
func (h *OrganizationHandler) Update(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		Error(w, domain.ErrInvalidInput)
		return
	}

	var input models.OrganizationUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}

	org, err := h.svc.Update(r.Context(), principal.Username, id, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, org)
}
