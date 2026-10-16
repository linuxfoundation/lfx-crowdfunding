// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	emailapi "github.com/linuxfoundation/lfx-v2-email-service/pkg/api"
	"github.com/nats-io/nats.go"
)

type requesterStub struct {
	reply []byte
	err   error
	got   *nats.Msg
}

func (s *requesterStub) RequestMsgWithContext(_ context.Context, msg *nats.Msg) (*nats.Msg, error) {
	s.got = msg
	if s.err != nil {
		return nil, s.err
	}
	return &nats.Msg{Data: s.reply}, nil
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestEmailClientSend_Success(t *testing.T) {
	stub := &requesterStub{reply: mustJSON(t, emailapi.SendEmailResponse{EmailID: "e-1"})}
	c := NewEmailClient(stub, "LFX Crowdfunding")

	if err := c.Send(context.Background(), EmailMessage{To: "a@linuxfoundation.org", Subject: "S", HTML: "<p>H</p>", Text: "H"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if stub.got.Subject != emailapi.SendEmailSubject {
		t.Fatalf("subject = %q", stub.got.Subject)
	}
	var sent emailapi.SendEmailRequest
	if err := json.Unmarshal(stub.got.Data, &sent); err != nil {
		t.Fatal(err)
	}
	want := emailapi.SendEmailRequest{To: "a@linuxfoundation.org", Subject: "S", HTML: "<p>H</p>", Text: "H", FromDisplayName: "LFX Crowdfunding"}
	if sent != want {
		t.Fatalf("request = %+v, want %+v", sent, want)
	}
}

func TestEmailClientSend_BlockedRecipientCountsAsSent(t *testing.T) {
	stub := &requesterStub{reply: []byte(`{"email_id":"","group_id":""}`)}

	if err := NewEmailClient(stub, "").Send(context.Background(), EmailMessage{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestEmailClientSend_Rejected(t *testing.T) {
	stub := &requesterStub{reply: mustJSON(t, emailapi.SendEmailErrorResponse{Error: "to is required"})}

	if err := NewEmailClient(stub, "").Send(context.Background(), EmailMessage{}); !errors.Is(err, ErrEmailRejected) {
		t.Fatalf("err = %v, want ErrEmailRejected", err)
	}
}

func TestEmailClientSend_TransportError(t *testing.T) {
	stub := &requesterStub{err: nats.ErrNoResponders}

	if err := NewEmailClient(stub, "").Send(context.Background(), EmailMessage{}); !errors.Is(err, nats.ErrNoResponders) {
		t.Fatalf("err = %v, want ErrNoResponders", err)
	}
}

func TestEmailClientSend_UnrecognisedReply(t *testing.T) {
	stub := &requesterStub{reply: []byte(`{}`)}

	if err := NewEmailClient(stub, "").Send(context.Background(), EmailMessage{}); err == nil {
		t.Fatal("expected error for reply without email_id")
	}
}
