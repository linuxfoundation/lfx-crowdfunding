// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handler helpers.
package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

type errorBody struct {
	Error string `json:"error"`
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
		errors.Is(err, domain.ErrPaymentMethodNotFound):
		return http.StatusNotFound, "not found"
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
