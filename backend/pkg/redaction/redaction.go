// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package redaction provides helpers to redact PII and secrets before logging.
package redaction

import "strings"

// RedactEmail replaces the local part of an email address with asterisks.
// e.g. "user@example.com" -> "u***@example.com"
func RedactEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***"
	}
	local := parts[0]
	if len(local) <= 1 {
		return "*@" + parts[1]
	}
	return string(local[0]) + strings.Repeat("*", len(local)-1) + "@" + parts[1]
}

// RedactToken redacts an API key or bearer token, showing only the last 4 characters.
func RedactToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(token)-4) + token[len(token)-4:]
}

// RedactStripeKey redacts a Stripe secret key for safe logging.
func RedactStripeKey(key string) string {
	return RedactToken(key)
}
