// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBuildChargeMetadata_TruncatesToStripeLimit(t *testing.T) {
	long := strings.Repeat("é", stripeMetadataValueMax+50) // multi-byte: must not split a rune
	m := buildChargeMetadata("id", "slug", long, "u", "donor", "", "", "", "cat", "", "", "")

	got := m["initiative_name"]
	if n := utf8.RuneCountInString(got); n != stripeMetadataValueMax {
		t.Errorf("initiative_name has %d chars, want %d", n, stripeMetadataValueMax)
	}
	if !utf8.ValidString(got) {
		t.Error("initiative_name is not valid UTF-8 after truncation")
	}
	if m["donor_name"] != "donor" {
		t.Errorf("short value changed: %q", m["donor_name"])
	}
}

func TestBuildChargeMetadata_OrgClaim(t *testing.T) {
	m := buildChargeMetadata("id", "slug", "name", "u", "", "", "", "", "", "Acme", "acme.com", "")
	if m["org_name"] != "Acme" || m["org_domain"] != "acme.com" {
		t.Errorf("org metadata = %q / %q, want Acme / acme.com", m["org_name"], m["org_domain"])
	}
	if _, ok := m["org_id"]; ok {
		t.Error("org_id must not be written for org claims")
	}

	m = buildChargeMetadata("id", "slug", "name", "u", "", "", "", "", "", "", "", "")
	if _, ok := m["org_name"]; ok {
		t.Error("org_name must be omitted for individual donations")
	}
	if _, ok := m["org_domain"]; ok {
		t.Error("org_domain must be omitted for individual donations")
	}
}
