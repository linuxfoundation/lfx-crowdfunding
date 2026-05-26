// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"crypto/md5" //nolint:gosec // MD5 used for non-cryptographic ETag generation only
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
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

const (
	maxTransactionPageSize     = 100
	defaultTransactionPageSize = 10
)

// InitiativeHandler holds Chi handlers for the /v1/initiatives resource.
type InitiativeHandler struct {
	svc              *service.InitiativeService
	allowedApprovers []string
	logger           *slog.Logger
}

// NewInitiativeHandler creates an InitiativeHandler.
// allowedApprovers is the list of usernames permitted to approve or decline
// initiatives (sourced from the ALLOWED_APPROVERS env var).
func NewInitiativeHandler(svc *service.InitiativeService, allowedApprovers []string, logger *slog.Logger) *InitiativeHandler {
	return &InitiativeHandler{svc: svc, allowedApprovers: allowedApprovers, logger: logger}
}

// List handles GET /v1/initiatives
func (h *InitiativeHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := models.InitiativeFilter{
		OwnerID:        q.Get("owner_id"),
		InitiativeType: q.Get("type"),
		Status:         models.InitiativeStatus(q.Get("status")),
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
// Only published initiatives are returned; others produce a 404.
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
	if !initiative.Status.EqualFold(models.StatusPublished) {
		Error(w, domain.ErrInitiativeNotFound)
		return
	}

	body, err := json.Marshal(initiative)
	if err != nil {
		Error(w, err)
		return
	}
	etag := etagOf(body)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
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

// GetTransactions handles GET /v1/initiatives/{id}/transactions
// Accepts ?type=donations|expenses&size=N&page=N (1-based page, defaults to 1).
// Resolves the initiative by slug or UUID, verifies it is published, then calls Ledger.
func (h *InitiativeHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	value := chi.URLParam(r, "id")
	q := r.URL.Query()

	txnTypeParam := strings.ToLower(q.Get("type"))
	var ledgerTxnType string
	switch txnTypeParam {
	case "donations":
		ledgerTxnType = "donation"
	case "expenses":
		ledgerTxnType = "reimbursement"
	}

	size, _ := strconv.Atoi(q.Get("size"))
	if size <= 0 {
		size = defaultTransactionPageSize
	} else if size > maxTransactionPageSize {
		size = maxTransactionPageSize
	}

	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}

	// Resolve identifier to a UUID, verifying the initiative exists and is published.
	// Use lightweight lookups (no Ledger enrichment) since transactions come from Ledger directly.
	var initiativeID string
	if uuidPattern.MatchString(value) {
		if err := h.svc.CheckPublishedByID(r.Context(), value); err != nil {
			Error(w, err)
			return
		}
		initiativeID = value
	} else {
		id, err := h.svc.GetIDBySlug(r.Context(), value)
		if err != nil {
			Error(w, err)
			return
		}
		initiativeID = id
	}

	list, err := h.svc.GetTransactions(r.Context(), initiativeID, ledgerTxnType, size, page)
	if err != nil {
		Error(w, err)
		return
	}

	body, err := json.Marshal(list)
	if err != nil {
		Error(w, err)
		return
	}
	etag := etagOf(body)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// ProcessApproval handles POST /v1/initiatives/{id}/process-approval/{action} — requires JWT.
// The caller's Username must appear in the AllowedApprovers list configured via
// ALLOWED_APPROVERS. {action} must be "approve" or "decline".
func (h *InitiativeHandler) ProcessApproval(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	// Validate action first to avoid reflecting unvalidated input in error messages.
	rawAction := chi.URLParam(r, "action")
	action, err := models.ParseApprovalAction(rawAction)
	if err != nil {
		Error(w, fmt.Errorf("%w: %s", domain.ErrInvalidInput, err))
		return
	}

	// Authorise: caller's username must be in the approvers list.
	allowed := false
	for _, a := range h.allowedApprovers {
		if strings.EqualFold(a, principal.Username) {
			allowed = true
			break
		}
	}
	if !allowed {
		h.logger.WarnContext(r.Context(), "initiative approval rejected: caller not in allowed list",
			"username", principal.Username,
			"action", action,
			"initiative_id", chi.URLParam(r, "id"))
		Error(w, domain.ErrForbidden)
		return
	}

	id := chi.URLParam(r, "id")
	if !uuidPattern.MatchString(id) {
		resolved, resolveErr := h.svc.ResolveSlug(r.Context(), id)
		if resolveErr != nil {
			Error(w, resolveErr)
			return
		}
		id = resolved
	}
	updated, err := h.svc.ProcessApproval(r.Context(), id, action)
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

// etagOf returns a quoted ETag for the given response body using MD5.
func etagOf(body []byte) string {
	sum := md5.Sum(body) //nolint:gosec // MD5 is fine for ETags (not security-sensitive)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}
