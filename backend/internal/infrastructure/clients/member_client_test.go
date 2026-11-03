// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestMemberClientLookupB2BOrgByWebsite(t *testing.T) {
	tests := []struct {
		name     string
		reply    string
		err      error
		wantSFID string
		wantErr  bool
	}{
		{name: "match", reply: `{"id":"0012M00002qnukOQAQ"}`, wantSFID: "0012M00002qnukOQAQ"},
		{name: "not found or ambiguous", reply: `{"error":"b2b org not found"}`},
		{name: "lookup failed", reply: `{"error":"b2b org lookup failed"}`, wantErr: true},
		{name: "transport error", err: nats.ErrNoResponders, wantErr: true},
		{name: "unrecognised reply", reply: `{}`, wantErr: true},
		{name: "not json", reply: `nope`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &requesterStub{reply: []byte(tt.reply), err: tt.err}
			sfid, err := NewMemberClient(stub).LookupB2BOrgByWebsite(context.Background(), "example.org")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if sfid != tt.wantSFID {
				t.Fatalf("sfid = %q, want %q", sfid, tt.wantSFID)
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
