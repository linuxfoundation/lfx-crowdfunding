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
// userRepo resolves the caller's email from their user row; token email claims
// are never used for authorization.
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

	// Approvers are identified by email. The email always comes from the caller's
	// user row (looked up by LF username), never from a token claim: neither
	// Auth0 nor Heimdall access tokens reliably carry one, and a claim isn't
	// proof of mailbox ownership. The row is written only by login sync from
	// Auth0 /userinfo, and is the same source the RS owner email comes from.
	// Only the initiative owner is ever emailed an approve/reject link (RS sends
	// it to the owner email from that same row, and cannot send without one), so
	// a legitimate approver always has a synced email; no Heimdall email-sync
	// path is needed here.
	// A caller with no resolvable email can never be authorized. Both outcomes
	// below depend only on the caller's own identity, so they reveal nothing
	// about any report.
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}
	email, err := h.emailForUsername(r.Context(), principal.Username)
	if err != nil {
		Error(w, err)
		return
	}
	if email == "" {
		Error(w, domain.ErrForbidden)
		return
	}
	actor := *principal
	actor.Email = email // forwarded to RS and shown in the Slack message

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
