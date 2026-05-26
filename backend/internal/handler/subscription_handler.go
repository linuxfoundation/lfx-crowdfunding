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

// SubscriptionHandler holds Chi handlers for the subscriptions resource.
type SubscriptionHandler struct {
	svc *service.SubscriptionService
}

// NewSubscriptionHandler creates a SubscriptionHandler.
func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

// List handles GET /v1/initiatives/{id}/subscriptions
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	initiativeID := chi.URLParam(r, "id")
	limit, offset, ok := parsePaginationParams(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()

	subs, meta, err := h.svc.ListByInitiative(r.Context(), initiativeID, models.SubscriptionFilter{
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": subs,
		"meta": meta,
	})
}

// Create handles POST /v1/initiatives/{id}/subscriptions — requires JWT.
// Clients MUST supply an Idempotency-Key header (a UUID they generate per
// logical subscription attempt). The backend uses it for both Stripe Price
// and Subscription creation so that retries are idempotent end-to-end.
func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		Error(w, fmt.Errorf("%w: Idempotency-Key header is required", domain.ErrInvalidInput))
		return
	}

	initiativeID := chi.URLParam(r, "id")
	var input models.SubscriptionCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}
	input.IdempotencyKey = idempotencyKey

	created, err := h.svc.Create(r.Context(), initiativeID, principal.UserID, principal.Email, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusCreated, created)
}

// ListForUser handles GET /v1/me/subscriptions — requires JWT.
// Returns the authenticated user's own subscriptions, paginated.
func (h *SubscriptionHandler) ListForUser(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	limit, offset, ok := parsePaginationParams(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()

	subs, meta, err := h.svc.ListByUser(r.Context(), principal.UserID, models.SubscriptionFilter{
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		Error(w, err)
		return
	}
	if subs == nil {
		subs = []models.Subscription{}
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": subs,
		"meta": meta,
	})
}

// Cancel handles DELETE /v1/subscriptions/{id} — requires JWT.
func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if err := h.svc.Cancel(r.Context(), id, principal.UserID); err != nil {
		Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
