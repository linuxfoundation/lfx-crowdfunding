// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handler helpers.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

type errorBody struct {
	Error string `json:"error"`
}

// parsePaginationParams parses ?limit= and ?offset= from r. Returns 400 if
// either value is present but not a valid integer.
func parsePaginationParams(w http.ResponseWriter, r *http.Request) (limit, offset int, ok bool) {
	q := r.URL.Query()
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			Error(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidInput))
			return 0, 0, false
		}
		limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			Error(w, fmt.Errorf("%w: offset must be an integer", domain.ErrInvalidInput))
			return 0, 0, false
		}
		offset = n
	}
	return limit, offset, true
}

// JSON writes a JSON-encoded body with the given status code.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// mapError translates domain sentinel errors to HTTP status codes and messages.
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrInitiativeNotFound),
		errors.Is(err, domain.ErrDonationNotFound),
		errors.Is(err, domain.ErrSubscriptionNotFound),
		errors.Is(err, domain.ErrOrganizationNotFound),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrPaymentMethodNotFound),
		errors.Is(err, domain.ErrExpenseReportNotFound):
		return http.StatusNotFound, "not found"
	case errors.Is(err, domain.ErrProfileNotSynced):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrRateLimitExceeded):
		return http.StatusTooManyRequests, "rate limit exceeded"
	case errors.Is(err, domain.ErrUpstreamUnavailable):
		return http.StatusServiceUnavailable, "upstream unavailable"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

// Error writes a JSON error response derived from a domain error.
func Error(w http.ResponseWriter, err error) {
	status, msg := mapError(err)
	if status == http.StatusInternalServerError {
		slog.Error("internal error", "error", err)
	}
	JSON(w, status, errorBody{Error: msg})
}

// parseStatusFilter parses repeated ?status= query params (e.g. ?status=hidden&status=declined).
// Each value is validated against the known status set; an unknown value writes a 400 and returns ok=false.
// Returns nil (and ok=true) when no status params are present, leaving default handling to the caller.
func parseStatusFilter(w http.ResponseWriter, r *http.Request) ([]models.InitiativeStatus, bool) {
	raw := r.URL.Query()["status"]
	if len(raw) == 0 {
		return nil, true
	}
	statuses := make([]models.InitiativeStatus, 0, len(raw))
	for _, v := range raw {
		s := models.InitiativeStatus(v)
		if !s.IsValid() {
			Error(w, fmt.Errorf("%w: unknown status %q", domain.ErrInvalidInput, v))
			return nil, false
		}
		statuses = append(statuses, models.InitiativeStatus(strings.ToLower(v)))
	}
	return statuses, true
}
