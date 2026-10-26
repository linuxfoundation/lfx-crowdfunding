// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package domain

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeDomain reduces a donor-supplied website or domain to its canonical
// form (doc 13 §4.1): the hostname, lowercased, without a trailing dot or a
// leading "www.". "https://Acme.com/path", "acme.com" and "acme.com." all become
// "acme.com". Anything that isn't a dotted hostname returns ErrInvalidInput.
// ponytail: no public-suffix check, so "acme.invalid" passes; add x/net/publicsuffix if junk TLDs show up.
func NormalizeDomain(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Port() != "" || u.User != nil {
		return "", fmt.Errorf("%w: invalid organization domain %q", ErrInvalidInput, raw)
	}
	host := strings.TrimPrefix(strings.TrimSuffix(strings.ToLower(u.Hostname()), "."), "www.")
	if !isHostname(host) {
		return "", fmt.Errorf("%w: invalid organization domain %q", ErrInvalidInput, raw)
	}
	return host, nil
}

// isHostname reports whether s is at least two dot-separated labels of
// [a-z0-9-], each 1–63 chars, not starting or ending with a hyphen.
func isHostname(s string) bool {
	labels := strings.Split(s, ".")
	if len(s) > 253 || len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if l == "" || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
			return false
		}
		for _, c := range l {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
	}
	return true
}
