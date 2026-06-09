// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package domain defines shared domain errors and repository contracts.
package domain

import "errors"

// Sentinel errors — mapped to HTTP status codes in handler/respond.go.
var (
	ErrInitiativeNotFound    = errors.New("initiative not found")
	ErrDonationNotFound      = errors.New("donation not found")
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrOrganizationNotFound  = errors.New("organization not found")
	ErrUserNotFound          = errors.New("user not found")
	ErrPaymentMethodNotFound = errors.New("payment method not found")
	// ErrProfileNotSynced signals that the user row exists but is missing required
	// profile data (e.g. email). The caller must PATCH /v1/me to sync their profile.
	// Maps to 400 Bad Request so the hint message is surfaced to the API client.
	ErrProfileNotSynced    = errors.New("profile not synced")
	ErrInvalidInput        = errors.New("invalid input")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrConflict            = errors.New("resource conflict")
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrUpstreamUnavailable = errors.New("upstream service unavailable")
	// ErrAlreadyProcessed is returned by repository update methods (donations and
	// subscriptions) when a record is already in the requested terminal state.
	// Webhook handlers use this to skip idempotent re-processing (e.g. Ledger POST,
	// emails) on Stripe retry events without returning an error.
	ErrAlreadyProcessed = errors.New("already processed")
)
