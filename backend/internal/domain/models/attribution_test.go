// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

import "testing"

func TestAttribution_Validate(t *testing.T) {
	const validUUID = "7cad5a8d-19d0-41a4-81a6-043453daf9ee"

	tests := []struct {
		name    string
		attr    Attribution
		wantErr bool
	}{
		{"personal, no uid — valid", Attribution{Type: AttributionPersonal}, false},
		{"personal, with uid — invalid", Attribution{Type: AttributionPersonal, EntityUID: validUUID}, true},
		{"organization, valid uid", Attribution{Type: AttributionOrganization, EntityUID: validUUID}, false},
		{"organization, missing uid", Attribution{Type: AttributionOrganization}, true},
		{"organization, malformed uid", Attribution{Type: AttributionOrganization, EntityUID: "not-a-uuid"}, true},
		{"project, valid uid", Attribution{Type: AttributionProject, EntityUID: validUUID}, false},
		{"project, missing uid", Attribution{Type: AttributionProject}, true},
		{"project, malformed uid", Attribution{Type: AttributionProject, EntityUID: "not-a-uuid"}, true},
		{"unknown type", Attribution{Type: "bogus"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.attr.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
