// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

func TestMemberClientLookupB2BOrgByWebsite(t *testing.T) {
	tests := []struct {
		name      string
		reply     string
		err       error
		wantMatch models.B2BOrgMatch
		wantErr   bool
	}{
		{name: "match", reply: `{"id":"0012M00002qnukOQAQ"}`, wantMatch: models.B2BOrgMatch{SFID: "0012M00002qnukOQAQ"}},
		{
			name:      "match with logo",
			reply:     `{"id":"0012M00002qnukOQAQ","logo_url":"https://cdn.example/logo.png"}`,
			wantMatch: models.B2BOrgMatch{SFID: "0012M00002qnukOQAQ", LogoURL: "https://cdn.example/logo.png"},
		},
		{name: "not found or ambiguous", reply: `{"error":"b2b org not found"}`},
		{name: "lookup failed", reply: `{"error":"b2b org lookup failed"}`, wantErr: true},
		{name: "transport error", err: nats.ErrNoResponders, wantErr: true},
		{name: "unrecognised reply", reply: `{}`, wantErr: true},
		{name: "not json", reply: `nope`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &requesterStub{reply: []byte(tt.reply), err: tt.err}
			match, err := NewMemberClient(stub).LookupB2BOrgByWebsite(context.Background(), "example.org")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if match != tt.wantMatch {
				t.Fatalf("match = %+v, want %+v", match, tt.wantMatch)
			}
			if stub.got.Subject != b2bOrgLookupByWebsiteSubject {
				t.Fatalf("subject = %q", stub.got.Subject)
			}
			if got, want := string(stub.got.Data), `{"name":"","website":"example.org"}`; got != want {
				t.Fatalf("request = %s, want %s", got, want)
			}
		})
	}
}

func TestMemberClientGetB2BOrgLogo(t *testing.T) {
	tests := []struct {
		name      string
		reply     string
		err       error
		wantLogo  string
		wantFound bool
		wantErr   bool
	}{
		{name: "logo", reply: `{"id":"0012M00002qnukOQAQ","logo_url":"https://cdn.example/logo.png"}`, wantLogo: "https://cdn.example/logo.png", wantFound: true},
		{name: "no logo", reply: `{"id":"0012M00002qnukOQAQ"}`, wantFound: true},
		{name: "org gone", reply: `{"error":"b2b org not found"}`},
		{name: "lookup failed", reply: `{"error":"b2b org lookup failed"}`, wantErr: true},
		{name: "transport error", err: nats.ErrNoResponders, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &requesterStub{reply: []byte(tt.reply), err: tt.err}
			logo, found, err := NewMemberClient(stub).GetB2BOrgLogo(context.Background(), "0012M00002qnukOQAQ")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if logo != tt.wantLogo || found != tt.wantFound {
				t.Fatalf("got (%q, %v), want (%q, %v)", logo, found, tt.wantLogo, tt.wantFound)
			}
			if stub.got.Subject != b2bOrgLookupSubject {
				t.Fatalf("subject = %q", stub.got.Subject)
			}
			if got, want := string(stub.got.Data), `{"id":"0012M00002qnukOQAQ"}`; got != want {
				t.Fatalf("request = %s, want %s", got, want)
			}
		})
	}
}
