// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package fga

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// fakePublisherConn is a natsPublisher test double that records every
// published message and can be told to fail Publish or Flush independently.
type fakePublisherConn struct {
	published       []publishedMsg
	publishErr      error
	flushErr        error
	requireDeadline bool // simulate nats.go's real FlushWithContext deadline requirement
}

type publishedMsg struct {
	subject string
	msg     GenericFGAMessage
}

func (f *fakePublisherConn) Publish(subj string, data []byte) error {
	if f.publishErr != nil {
		return f.publishErr
	}
	var msg GenericFGAMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}
	f.published = append(f.published, publishedMsg{subject: subj, msg: msg})
	return nil
}

func (f *fakePublisherConn) FlushWithContext(ctx context.Context) error {
	if f.requireDeadline {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			return errors.New("nats: context requires a deadline")
		}
	}
	return f.flushErr
}

func TestPublisher_UpdateAccess(t *testing.T) {
	t.Run("owner only, personal attribution, unpublished", func(t *testing.T) {
		conn := &fakePublisherConn{}
		p := NewPublisher(nil)
		p.conn = conn

		err := p.UpdateAccess(context.Background(), InitiativeAccess{
			UID:           "init-1",
			OwnerUsername: "alice",
			Attribution:   models.Attribution{Type: models.AttributionPersonal},
			Published:     false,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(conn.published) != 1 {
			t.Fatalf("expected 1 published message, got %d", len(conn.published))
		}
		got := conn.published[0]
		if got.subject != UpdateAccessSubject {
			t.Errorf("subject = %q, want %q", got.subject, UpdateAccessSubject)
		}
		if got.msg.ObjectType != InitiativeObjectType || got.msg.Operation != "update_access" {
			t.Errorf("unexpected envelope: %+v", got.msg)
		}
		data, ok := got.msg.Data.(map[string]any)
		if !ok {
			t.Fatalf("data is %T, want map[string]any", got.msg.Data)
		}
		if data["public"] != false {
			t.Errorf("public = %v, want false", data["public"])
		}
		if _, hasRefs := data["references"]; hasRefs {
			t.Errorf("expected no references for personal attribution, got %v", data["references"])
		}
	})

	t.Run("project attribution and published sets references and public wildcard", func(t *testing.T) {
		conn := &fakePublisherConn{}
		p := NewPublisher(nil)
		p.conn = conn

		err := p.UpdateAccess(context.Background(), InitiativeAccess{
			UID:           "init-2",
			OwnerUsername: "bob",
			Attribution:   models.Attribution{Type: models.AttributionProject, EntityUID: "proj-uuid"},
			Published:     true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data := conn.published[0].msg.Data.(map[string]any)
		if data["public"] != true {
			t.Errorf("public = %v, want true", data["public"])
		}
		refs, ok := data["references"].(map[string]any)
		if !ok {
			t.Fatalf("references missing or wrong type: %+v", data["references"])
		}
		projectRefs, ok := refs["project"].([]any)
		if !ok || len(projectRefs) != 1 || projectRefs[0] != "proj-uuid" {
			t.Errorf("references.project = %+v, want [proj-uuid]", refs["project"])
		}
	})

	t.Run("organization attribution sets b2b_org reference", func(t *testing.T) {
		conn := &fakePublisherConn{}
		p := NewPublisher(nil)
		p.conn = conn

		err := p.UpdateAccess(context.Background(), InitiativeAccess{
			UID:           "init-3",
			OwnerUsername: "carol",
			Attribution:   models.Attribution{Type: models.AttributionOrganization, EntityUID: "0012M00002qnukOQAQ"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data := conn.published[0].msg.Data.(map[string]any)
		refs, ok := data["references"].(map[string]any)
		if !ok {
			t.Fatalf("references missing or wrong type: %+v", data["references"])
		}
		orgRefs, ok := refs["b2b_org"].([]any)
		if !ok || len(orgRefs) != 1 || orgRefs[0] != "0012M00002qnukOQAQ" {
			t.Errorf("references.b2b_org = %+v, want [0012M00002qnukOQAQ]", refs["b2b_org"])
		}
	})

	t.Run("missing uid is rejected", func(t *testing.T) {
		conn := &fakePublisherConn{}
		p := NewPublisher(nil)
		p.conn = conn
		if err := p.UpdateAccess(context.Background(), InitiativeAccess{OwnerUsername: "alice"}); err == nil {
			t.Fatal("expected error for missing uid")
		}
	})

	t.Run("missing owner username is rejected", func(t *testing.T) {
		conn := &fakePublisherConn{}
		p := NewPublisher(nil)
		p.conn = conn
		if err := p.UpdateAccess(context.Background(), InitiativeAccess{UID: "init-1"}); err == nil {
			t.Fatal("expected error for missing owner username")
		}
	})

	t.Run("nil publisher is a no-op", func(t *testing.T) {
		var p *Publisher
		if err := p.UpdateAccess(context.Background(), InitiativeAccess{UID: "init-1", OwnerUsername: "alice"}); err != nil {
			t.Fatalf("unexpected error from nil publisher: %v", err)
		}
	})

	t.Run("nil connection is a no-op", func(t *testing.T) {
		p := NewPublisher(nil)
		if err := p.UpdateAccess(context.Background(), InitiativeAccess{UID: "init-1", OwnerUsername: "alice"}); err != nil {
			t.Fatalf("unexpected error from nil connection: %v", err)
		}
	})

	t.Run("publish error is surfaced", func(t *testing.T) {
		conn := &fakePublisherConn{publishErr: errors.New("broker down")}
		p := NewPublisher(nil)
		p.conn = conn
		if err := p.UpdateAccess(context.Background(), InitiativeAccess{UID: "init-1", OwnerUsername: "alice"}); err == nil {
			t.Fatal("expected error to be surfaced")
		}
	})
}

func TestPublisher_DeleteAccess(t *testing.T) {
	t.Run("publishes and flushes", func(t *testing.T) {
		conn := &fakePublisherConn{}
		p := NewPublisher(nil)
		p.conn = conn

		if err := p.DeleteAccess(context.Background(), "init-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(conn.published) != 1 {
			t.Fatalf("expected 1 published message, got %d", len(conn.published))
		}
		got := conn.published[0]
		if got.subject != DeleteAccessSubject || got.msg.Operation != "delete_access" {
			t.Errorf("unexpected message: %+v", got)
		}
	})

	t.Run("flush error is surfaced", func(t *testing.T) {
		conn := &fakePublisherConn{flushErr: errors.New("flush timeout")}
		p := NewPublisher(nil)
		p.conn = conn
		if err := p.DeleteAccess(context.Background(), "init-1"); err == nil {
			t.Fatal("expected flush error to be surfaced")
		}
	})

	t.Run("missing uid is rejected", func(t *testing.T) {
		conn := &fakePublisherConn{}
		p := NewPublisher(nil)
		p.conn = conn
		if err := p.DeleteAccess(context.Background(), ""); err == nil {
			t.Fatal("expected error for missing uid")
		}
	})

	t.Run("nil publisher is a no-op", func(t *testing.T) {
		var p *Publisher
		if err := p.DeleteAccess(context.Background(), "init-1"); err != nil {
			t.Fatalf("unexpected error from nil publisher: %v", err)
		}
	})

	t.Run("deadline-less context gets a flush timeout applied", func(t *testing.T) {
		conn := &fakePublisherConn{requireDeadline: true}
		p := NewPublisher(nil)
		p.conn = conn
		// context.Background() has no deadline; the fake flush errors unless
		// DeleteAccess applies its own timeout before calling it.
		if err := p.DeleteAccess(context.Background(), "init-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
