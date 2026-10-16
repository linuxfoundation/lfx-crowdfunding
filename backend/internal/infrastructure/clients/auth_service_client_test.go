// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
)

type authRequesterStub struct {
	reply   []byte
	err     error
	subject string
	data    string
}

func (s *authRequesterStub) RequestWithContext(_ context.Context, subj string, data []byte) (*nats.Msg, error) {
	s.subject, s.data = subj, string(data)
	if s.err != nil {
		return nil, s.err
	}
	return &nats.Msg{Data: s.reply}, nil
}

func TestAuthServiceClientUserMetadata_Success(t *testing.T) {
	stub := &authRequesterStub{reply: []byte(`{"success":true,"data":{"name":"John Doe","given_name":"John","family_name":"Doe","picture":"https://x/p.jpg","city":"SF"}}`)}

	md, err := NewAuthServiceClient(stub).UserMetadata(context.Background(), "jdoe")
	if err != nil {
		t.Fatalf("UserMetadata: %v", err)
	}
	if stub.subject != userMetadataReadSubject || stub.data != "jdoe" {
		t.Fatalf("request = %q %q", stub.subject, stub.data)
	}
	want := UserMetadata{Name: "John Doe", GivenName: "John", FamilyName: "Doe", Picture: "https://x/p.jpg"}
	if *md != want {
		t.Fatalf("metadata = %+v, want %+v", *md, want)
	}
}

func TestAuthServiceClientUserMetadata_Errors(t *testing.T) {
	for name, stub := range map[string]*authRequesterStub{
		"error reply":     {reply: []byte(`{"success":false,"error":"user not found"}`)},
		"transport error": {err: errors.New("nats: timeout")},
		"no data":         {reply: []byte(`{"success":true}`)},
		"malformed reply": {reply: []byte(`not json`)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewAuthServiceClient(stub).UserMetadata(context.Background(), "ghost"); err == nil {
				t.Fatal("want error")
			}
		})
	}
}
