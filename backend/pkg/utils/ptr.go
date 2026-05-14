// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package utils provides shared helper functions.
package utils

// Ptr returns a pointer to the provided value.
// Useful for setting optional pointer fields from literals.
func Ptr[T any](v T) *T {
	return &v
}
