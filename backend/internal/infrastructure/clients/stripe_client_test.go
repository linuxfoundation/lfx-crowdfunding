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
