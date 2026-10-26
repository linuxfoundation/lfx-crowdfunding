// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package domain

import (
	"errors"
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"acme.com", "acme.com"},
		{"https://Acme.com/path", "acme.com"},
		{"acme.com.", "acme.com"},
		{"www.acme.com", "acme.com"},
		{"  HTTP://WWW.Acme.COM/?q=1  ", "acme.com"},
		{"sub.acme.co.uk", "sub.acme.co.uk"},
		{"my-org.io", "my-org.io"},
	}
	for _, tt := range tests {
		got, err := NormalizeDomain(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("NormalizeDomain(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}

	for _, in := range []string{"", "acme", "acme .com", "-acme.com", "acme..com", "acme.com:8080", "user@acme.com", "acme_corp.com", "https://", "ácme.com"} {
		if got, err := NormalizeDomain(in); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("NormalizeDomain(%q) = %q, %v; want ErrInvalidInput", in, got, err)
		}
	}
}
