// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"encoding/json"
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

func TestAuthServiceClientPrimaryEmail(t *testing.T) {
	stub := &authRequesterStub{reply: []byte(`{"success":true,"data":{"primary_email":"jdoe@example.com","alternate_emails":[]}}`)}

	email, err := NewAuthServiceClient(stub).PrimaryEmail(context.Background(), "jdoe")
	if err != nil {
		t.Fatalf("PrimaryEmail: %v", err)
	}
	if email != "jdoe@example.com" {
		t.Fatalf("email = %q", email)
	}
	var req struct {
		User struct {
			AuthToken string `json:"auth_token"`
		} `json:"user"`
	}
	if err := json.Unmarshal([]byte(stub.data), &req); err != nil || stub.subject != userEmailsReadSubject || req.User.AuthToken != "jdoe" {
		t.Fatalf("request = %q %q (%v)", stub.subject, stub.data, err)
	}
}

func TestAuthServiceClientPrimaryEmail_Empty(t *testing.T) {
	stub := &authRequesterStub{reply: []byte(`{"success":true,"data":{"primary_email":""}}`)}

	if _, err := NewAuthServiceClient(stub).PrimaryEmail(context.Background(), "jdoe"); err == nil {
		t.Fatal("want error for empty primary_email")
	}
}

func TestAuthServiceClientUsernameToSub(t *testing.T) {
	cases := []struct {
		name    string
		reply   string
		err     error
		want    string
		wantErr bool
	}{
		{"plain text subject", "auth0|jdoe\n", nil, "auth0|jdoe", false},
		{"error envelope", `{"success":false,"error":"user not found"}`, nil, "", true},
		{"not a subject", "jdoe", nil, "", true},
		{"empty", "", nil, "", true},
		{"request fails", "", errors.New("nats timeout"), "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stub := &authRequesterStub{reply: []byte(c.reply), err: c.err}
			got, err := NewAuthServiceClient(stub).UsernameToSub(context.Background(), "jdoe")
			if (err != nil) != c.wantErr || got != c.want {
				t.Fatalf("UsernameToSub = %q, %v; want %q (err %v)", got, err, c.want, c.wantErr)
			}
			if stub.subject != usernameToSubSubject || stub.data != "jdoe" {
				t.Errorf("request = %q %q", stub.subject, stub.data)
			}
		})
	}
}
