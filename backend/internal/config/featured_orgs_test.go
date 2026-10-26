// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package config

import "testing"

func TestMustParseFeaturedOrgs(t *testing.T) {
	if len(FeaturedOrgs()) == 0 {
		t.Fatal("embedded featured_orgs.json parsed to an empty list")
	}

	bad := map[string]string{
		"empty name":            `[{"domain": "acme.com", "name": " "}]`,
		"non-normalized domain": `[{"domain": "www.Acme.com", "name": "Acme"}]`,
	}
	for name, raw := range bad {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			mustParseFeaturedOrgs([]byte(raw))
		})
	}
}
