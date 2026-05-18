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
	ErrInvalidInput          = errors.New("invalid input")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrConflict              = errors.New("resource conflict")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
	ErrUpstreamUnavailable   = errors.New("upstream service unavailable")
)
