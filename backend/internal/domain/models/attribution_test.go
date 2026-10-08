// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

import "testing"

func TestAttribution_Validate(t *testing.T) {
	const (
		validUUID   = "7cad5a8d-19d0-41a4-81a6-043453daf9ee"
		validSFID18 = "0012M00002qnukOQAQ"
		validSFID15 = "0012M00002qnukO"
	)

	tests := []struct {
		name    string
		attr    Attribution
		wantErr bool
	}{
		{"personal, no uid — valid", Attribution{Type: AttributionPersonal}, false},
		{"personal, with uid — invalid", Attribution{Type: AttributionPersonal, EntityUID: validUUID}, true},
		{"organization, valid 18-char sfid", Attribution{Type: AttributionOrganization, EntityUID: validSFID18}, false},
		{"organization, valid 15-char sfid", Attribution{Type: AttributionOrganization, EntityUID: validSFID15}, false},
		{"organization, missing uid", Attribution{Type: AttributionOrganization}, true},
		{"organization, uuid instead of sfid", Attribution{Type: AttributionOrganization, EntityUID: validUUID}, true},
		{"organization, malformed uid", Attribution{Type: AttributionOrganization, EntityUID: "not-an-sfid!"}, true},
		{"project, valid uid", Attribution{Type: AttributionProject, EntityUID: validUUID}, false},
		{"project, missing uid", Attribution{Type: AttributionProject}, true},
		{"project, malformed uid", Attribution{Type: AttributionProject, EntityUID: "not-a-uuid"}, true},
		{"project, sfid instead of uuid", Attribution{Type: AttributionProject, EntityUID: validSFID18}, true},
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

func TestAttribution_Validate_CanonicalizesProjectUID(t *testing.T) {
	attr := Attribution{Type: AttributionProject, EntityUID: "7CAD5A8D-19D0-41A4-81A6-043453DAF9EE"}
	if err := attr.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	want := "7cad5a8d-19d0-41a4-81a6-043453daf9ee"
	if attr.EntityUID != want {
		t.Fatalf("EntityUID = %q, want canonical %q", attr.EntityUID, want)
	}
}

func TestAttribution_Validate_CanonicalizesSFID(t *testing.T) {
	for in, want := range map[string]string{
		"0012M00002qnukO":    "0012M00002qnukOQAQ",
		"0012M00002qnukOQAQ": "0012M00002qnukOQAQ",
	} {
		a := Attribution{Type: AttributionOrganization, EntityUID: in}
		if err := a.Validate(); err != nil || a.EntityUID != want {
			t.Errorf("%s: got %q, err %v, want %q", in, a.EntityUID, err, want)
		}
	}
}

func TestAttribution_Canonical(t *testing.T) {
	tests := []struct{ in, want Attribution }{
		{Attribution{Type: AttributionOrganization, EntityUID: "0012M00002qnukO"}, Attribution{Type: AttributionOrganization, EntityUID: "0012M00002qnukOQAQ"}},
		{Attribution{Type: AttributionOrganization, EntityUID: "0012M00002qnukOQAQ"}, Attribution{Type: AttributionOrganization, EntityUID: "0012M00002qnukOQAQ"}},
		{Attribution{Type: AttributionProject, EntityUID: "7cad5a8d-19d0-41a"}, Attribution{Type: AttributionProject, EntityUID: "7cad5a8d-19d0-41a"}},
		{Attribution{Type: AttributionPersonal}, Attribution{Type: AttributionPersonal}},
	}
	for _, tt := range tests {
		if got := tt.in.Canonical(); got != tt.want {
			t.Errorf("Canonical(%+v) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}
