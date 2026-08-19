// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

type organizationAffiliationService interface {
	ListForUser(ctx context.Context, rawToken string) ([]clients.OrgCandidate, error)
}

// OrganizationAffiliationHandler serves the fundraise-form organization
// affiliation picker (LFXV2-3322/3323). Nil svc means the integration is not
// configured (Auth0 token exchange / query-service base URL unset).
type OrganizationAffiliationHandler struct {
	svc organizationAffiliationService
}

// NewOrganizationAffiliationHandler creates an OrganizationAffiliationHandler.
// Pass a nil svc (a bare `nil`, not a nil-valued concrete pointer — callers
// must check their concrete service for nil before passing it in, to avoid
// the typed-nil-interface pitfall) when the integration is not configured;
// List then returns ErrUpstreamUnavailable instead of panicking.
func NewOrganizationAffiliationHandler(svc organizationAffiliationService) *OrganizationAffiliationHandler {
	return &OrganizationAffiliationHandler{svc: svc}
}

// List handles GET /v1/me/organization-affiliations. Returns the caller's
// directly-granted v2 platform organizations (uid, name, logo) for the
// fundraise-form affiliation picker.
func (h *OrganizationAffiliationHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		Error(w, domain.ErrUpstreamUnavailable)
		return
	}

	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	orgs, err := h.svc.ListForUser(r.Context(), principal.RawToken)
	if err != nil {
		Error(w, err)
		return
	}
	if orgs == nil {
		orgs = []clients.OrgCandidate{}
	}
	JSON(w, http.StatusOK, map[string]any{"data": orgs})
}
