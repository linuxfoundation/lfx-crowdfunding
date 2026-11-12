// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

// maxOrgLogoDomains bounds one org logo lookup; the donate drawer's typeahead
// asks for one page of suggestions at a time.
const maxOrgLogoDomains = 20

// orgLogoReader is the read side of domain.OrgLogoRepository.
type orgLogoReader interface {
	LogosByDomains(ctx context.Context, domains []string) (map[string]string, error)
}

// OrgLogoHandler serves the b2b_org logos CF has stored for org domains (#472).
type OrgLogoHandler struct {
	logos orgLogoReader
}

// NewOrgLogoHandler creates an OrgLogoHandler. A nil reader answers with no logos.
func NewOrgLogoHandler(logos orgLogoReader) *OrgLogoHandler {
	return &OrgLogoHandler{logos: logos}
}

// orgLogosResponse is the body of GET /crowdfunding/me/org-logos.
type orgLogosResponse struct {
	// Logos maps each requested domain, as sent, to its logo URL. Domains
	// without a stored logo are absent.
	Logos map[string]string `json:"logos"`
}

// List handles GET /crowdfunding/me/org-logos?domains=acme.com,example.org.
// It reads only CF's own table, never member-service, so the donate drawer's
// org typeahead can show logos for orgs that have donated before.
func (h *OrgLogoHandler) List(w http.ResponseWriter, r *http.Request) {
	requested := strings.Split(r.URL.Query().Get("domains"), ",")
	if len(requested) > maxOrgLogoDomains {
		Error(w, fmt.Errorf("%w: at most %d domains", domain.ErrInvalidInput, maxOrgLogoDomains))
		return
	}

	// Stored donations hold the normalized domain; answer under the caller's spelling.
	byNormalized := make(map[string][]string, len(requested))
	normalized := make([]string, 0, len(requested))
	for _, raw := range requested {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		d, err := domain.NormalizeDomain(raw)
		if err != nil {
			// A suggestion with an odd domain just gets no logo.
			continue
		}
		if _, seen := byNormalized[d]; !seen {
			normalized = append(normalized, d)
		}
		byNormalized[d] = append(byNormalized[d], raw)
	}

	resp := orgLogosResponse{Logos: map[string]string{}}
	if h.logos == nil || len(normalized) == 0 {
		JSON(w, http.StatusOK, resp)
		return
	}
	logos, err := h.logos.LogosByDomains(r.Context(), normalized)
	if err != nil {
		Error(w, err)
		return
	}
	for d, logo := range logos {
		for _, raw := range byNormalized[d] {
			resp.Logos[raw] = logo
		}
	}
	JSON(w, http.StatusOK, resp)
}
