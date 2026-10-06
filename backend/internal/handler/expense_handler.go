// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// ExpenseHandler proxies expense-report actions to the Reimbursement Service.
type ExpenseHandler struct {
	rsClient clients.ReimbursementClient
	userRepo domain.UserRepository
}

// NewExpenseHandler creates an ExpenseHandler backed by the given client.
// userRepo resolves the caller's email for Heimdall-issued tokens, which carry
// no email claim.
func NewExpenseHandler(rsClient clients.ReimbursementClient, userRepo domain.UserRepository) *ExpenseHandler {
	return &ExpenseHandler{rsClient: rsClient, userRepo: userRepo}
}

// ProcessAction handles POST /crowdfunding/expense/{action}/{reportId}.
// It forwards the action to the Reimbursement Service and returns 204 No
// Content on success.
//
// The caller must be one of the approvers the Reimbursement Service emailed for
// this report (the initiative owner, or the Travel Fund admin). Callers who are
// not — and reports that don't exist — both get 404, so report IDs can't be
// probed, and the action is never forwarded for them.
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

	// Approvers are identified by email. A token without one can never be
	// authorized. Both outcomes below depend only on the caller's own identity,
	// so they reveal nothing about any report.
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}
	actor := *principal
	if actor.Email == "" && actor.IsHeimdallIssued {
		// Heimdall-minted tokens carry no email; use the one on the caller's
		// user row, the same source the RS owner email comes from.
		email, err := h.emailForUsername(r.Context(), actor.Username)
		if err != nil {
			Error(w, err)
			return
		}
		actor.Email = email
	}
	if actor.Email == "" {
		Error(w, domain.ErrForbidden)
		return
	}

	approvers, err := h.rsClient.GetExpenseApprovers(r.Context(), reportID)
	if err != nil {
		Error(w, err)
		return
	}
	if !slices.ContainsFunc(approvers, func(email string) bool {
		return strings.EqualFold(strings.TrimSpace(email), actor.Email)
	}) {
		Error(w, domain.ErrExpenseReportNotFound)
		return
	}

	if err := h.rsClient.ProcessExpenseAction(r.Context(), action, reportID, &actor); err != nil {
		Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// emailForUsername returns the email on the user's row, or "" when the user has
// no row or no email (the caller is then refused, not told why).
func (h *ExpenseHandler) emailForUsername(ctx context.Context, username string) (string, error) {
	if h.userRepo == nil || username == "" {
		return "", nil
	}
	user, err := h.userRepo.GetByUsername(ctx, username)
	if errors.Is(err, domain.ErrUserNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return user.Email, nil
}
