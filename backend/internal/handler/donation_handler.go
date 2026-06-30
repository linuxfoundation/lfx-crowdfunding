// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// csvDownloaders is the comma-separated allowlist of LF SSO usernames that may
// call the org-donors CSV export endpoint (GET /v1/me/donations/csv).
// To grant access to additional users, append their username to this constant.
const csvDownloaders = "lewisoj"

// csvDownloaderSet is the parsed, O(1)-lookup form of csvDownloaders.
var csvDownloaderSet = func() map[string]struct{} {
	m := make(map[string]struct{})
	for _, u := range strings.Split(csvDownloaders, ",") {
		if trimmed := strings.TrimSpace(u); trimmed != "" {
			m[trimmed] = struct{}{}
		}
	}
	return m
}()

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
	limit, offset, ok := parsePaginationParams(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()

	summaries, meta, err := h.svc.ListByInitiative(r.Context(), initiativeID, models.DonationFilter{
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": summaries,
		"meta": meta,
	})
}

// ListForUser handles GET /v1/me/donations — requires JWT.
// Returns the authenticated user's own donations across all initiatives, paginated.
func (h *DonationHandler) ListForUser(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	limit, offset, ok := parsePaginationParams(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()

	donations, meta, err := h.svc.ListByUser(r.Context(), principal.Username, models.DonationFilter{
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		Error(w, err)
		return
	}
	if donations == nil {
		donations = []models.Donation{}
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": donations,
		"meta": meta,
	})
}

// Create handles POST /v1/initiatives/{id}/donations — requires JWT.
// Clients MUST supply an Idempotency-Key header (a UUID they generate per
// logical donation attempt). The backend passes it verbatim to Stripe so
// that retries of the same timed-out request are de-duped rather than
// creating duplicate charges.
func (h *DonationHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		Error(w, fmt.Errorf("%w: Idempotency-Key header is required", domain.ErrInvalidInput))
		return
	}

	initiativeID := chi.URLParam(r, "id")
	var input models.DonationCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}
	input.IdempotencyKey = idempotencyKey

	created, err := h.svc.Create(r.Context(), initiativeID, principal.Username, input)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusCreated, created)
}

// ExportOrgCSV handles GET /v1/me/donations/csv — restricted to csvDownloaders.
// Query param: type=detail (default) | type=summary
//
//   - detail  — one row per donation with org, initiative, donor, amount, date
//   - summary — one row per organisation with total donation count and USD amount
func (h *DonationHandler) ExportOrgCSV(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}
	if _, allowed := csvDownloaderSet[principal.Username]; !allowed {
		Error(w, domain.ErrUnauthorized)
		return
	}

	csvType := r.URL.Query().Get("type")
	if csvType == "" {
		csvType = "detail"
	}
	if csvType != "detail" && csvType != "summary" {
		Error(w, fmt.Errorf("%w: type must be 'detail' or 'summary'", domain.ErrInvalidInput))
		return
	}

	rows, err := h.svc.ListOrgDonations(r.Context())
	if err != nil {
		Error(w, err)
		return
	}

	filename := fmt.Sprintf("org-donations-%s-%s.csv", csvType, time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	wr := csv.NewWriter(w)

	switch csvType {
	case "detail":
		_ = wr.Write([]string{
			"organization_id", "organization_name", "initiative_name", "initiative_id",
			"amount_usd", "donor_user_id", "donor_name", "donated_at", "status",
		})
		for _, row := range rows {
			_ = wr.Write([]string{
				row.OrganizationID,
				sanitizeCSVField(row.OrganizationName),
				sanitizeCSVField(row.InitiativeName),
				row.InitiativeID,
				fmt.Sprintf("%.2f", float64(row.AmountCents)/100.0),
				row.DonorUserID,
				sanitizeCSVField(row.DonorName),
				row.DonatedAt.UTC().Format(time.RFC3339),
				row.Status,
			})
		}

	case "summary":
		type orgEntry struct {
			id    string
			count int
			total int64
		}
		order := make([]string, 0)
		sumsByName := make(map[string]*orgEntry)
		for _, row := range rows {
			if _, seen := sumsByName[row.OrganizationName]; !seen {
				sumsByName[row.OrganizationName] = &orgEntry{id: row.OrganizationID}
				order = append(order, row.OrganizationName)
			}
			sumsByName[row.OrganizationName].count++
			sumsByName[row.OrganizationName].total += row.AmountCents
		}
		_ = wr.Write([]string{"organization_id", "organization_name", "donation_count", "total_donated_usd"})
		for _, name := range order {
			e := sumsByName[name]
			_ = wr.Write([]string{
				e.id,
				sanitizeCSVField(name),
				strconv.Itoa(e.count),
				fmt.Sprintf("%.2f", float64(e.total)/100.0),
			})
		}
	}

	wr.Flush()
	if err := wr.Error(); err != nil {
		// Headers already sent; log the error rather than returning an HTTP response.
		slog.ErrorContext(r.Context(), "csv flush failed", "error", err)
	}
}

// sanitizeCSVField prevents formula injection in spreadsheet applications by
// prefixing cells whose first character is a recognised formula trigger
// (=, +, -, @, tab, carriage-return) with a tab character. The data is
// otherwise unchanged and remains valid UTF-8 text.
func sanitizeCSVField(s string) string {
	if len(s) == 0 {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "\t" + s
	}
	return s
}
