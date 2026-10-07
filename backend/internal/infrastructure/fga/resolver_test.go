// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package fga

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// fakeRequester is a natsRequester test double that returns a canned reply or
// error regardless of the request sent, letting each test control exactly
// what fga-sync "said".
type fakeRequester struct {
	reply []byte
	err   error
}

func (f *fakeRequester) RequestWithContext(_ context.Context, _ string, _ []byte) (*nats.Msg, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &nats.Msg{Data: f.reply}, nil
}

// blockingRequester never replies until its request context is done — it
// stands in for a subscriber that received the request but never sent a
// reply, the scenario the resolver's own timeout must bound.
type blockingRequester struct{}

func (blockingRequester) RequestWithContext(ctx context.Context, _ string, _ []byte) (*nats.Msg, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestNATSResolver_CanManage(t *testing.T) {
	const (
		project  = "7cad5a8d-19d0-41a4-81a6-043453daf9ee"
		username = "alice"
	)

	t.Run("writer true", func(t *testing.T) {
		r := &NATSResolver{conn: &fakeRequester{
			reply: []byte("project:" + project + "#writer@user:" + username + "\ttrue\n"),
		}}
		allowed, err := r.CanManage(context.Background(), models.AttributionProject, project, username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("expected allowed=true")
		}
	})

	t.Run("definitively not writer", func(t *testing.T) {
		r := &NATSResolver{conn: &fakeRequester{
			reply: []byte("project:" + project + "#writer@user:" + username + "\tfalse\n"),
		}}
		allowed, err := r.CanManage(context.Background(), models.AttributionProject, project, username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("expected allowed=false")
		}
	})

	t.Run("replies are unordered — correlated by request token, not position", func(t *testing.T) {
		// Simulate fga-sync returning an unrelated cached result first, per
		// fga-sync-contract.md: "Order is not guaranteed (cached results may
		// be returned first)". A resolver that trusted line position would
		// read the wrong answer here.
		reply := "committee:other-id#writer@user:bob\ttrue\n" +
			"project:" + project + "#writer@user:" + username + "\tfalse\n"
		r := &NATSResolver{conn: &fakeRequester{reply: []byte(reply)}}
		allowed, err := r.CanManage(context.Background(), models.AttributionProject, project, username)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("expected allowed=false — must not pick up the unrelated cached result")
		}
	})

	t.Run("missing result for the sent request is an error, not a false", func(t *testing.T) {
		r := &NATSResolver{conn: &fakeRequester{
			reply: []byte("project:some-other-id#writer@user:" + username + "\ttrue\n"),
		}}
		_, err := r.CanManage(context.Background(), models.AttributionProject, project, username)
		if err == nil {
			t.Fatal("expected an error when the reply has no result for the request")
		}
		if !errors.Is(err, domain.ErrUpstreamUnavailable) {
			t.Fatalf("expected error to wrap ErrUpstreamUnavailable, got: %v", err)
		}
	})

	t.Run("malformed reply line is an error", func(t *testing.T) {
		r := &NATSResolver{conn: &fakeRequester{reply: []byte("garbage, no tab or verdict\n")}}
		_, err := r.CanManage(context.Background(), models.AttributionProject, project, username)
		if !errors.Is(err, domain.ErrUpstreamUnavailable) {
			t.Fatalf("expected error to wrap ErrUpstreamUnavailable, got: %v", err)
		}
	})

	t.Run("transport error wraps ErrUpstreamUnavailable", func(t *testing.T) {
		r := &NATSResolver{conn: &fakeRequester{err: errors.New("connection refused")}}
		_, err := r.CanManage(context.Background(), models.AttributionProject, project, username)
		if !errors.Is(err, domain.ErrUpstreamUnavailable) {
			t.Fatalf("expected error to wrap ErrUpstreamUnavailable, got: %v", err)
		}
	})

	t.Run("personal attribution is rejected — no FGA call for personal initiatives", func(t *testing.T) {
		r := &NATSResolver{conn: &fakeRequester{err: errors.New("must not be called")}}
		_, err := r.CanManage(context.Background(), models.AttributionPersonal, project, username)
		if err == nil {
			t.Fatal("expected an error for personal attribution")
		}
	})

	t.Run("resolver timeout bounds a caller context with no deadline", func(t *testing.T) {
		// nats.Timeout on the connection only covers connection setup; a
		// caller context.Background() (no deadline) must still be bounded
		// by the resolver's own configured timeout, not block forever.
		r := &NATSResolver{conn: blockingRequester{}, timeout: 20 * time.Millisecond}
		_, err := r.CanManage(context.Background(), models.AttributionProject, project, username)
		if !errors.Is(err, domain.ErrUpstreamUnavailable) {
			t.Fatalf("expected error to wrap ErrUpstreamUnavailable, got: %v", err)
		}
	})
}

func TestParseAccessCheckReply(t *testing.T) {
	t.Run("empty lines are skipped", func(t *testing.T) {
		results, err := parseAccessCheckReply([]byte("project:a#writer@user:x\ttrue\n\nproject:b#writer@user:x\tfalse\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 || !results["project:a#writer@user:x"] || results["project:b#writer@user:x"] {
			t.Fatalf("unexpected results: %#v", results)
		}
	})
}

func TestNATSResolver_IsAffiliated(t *testing.T) {
	const (
		org      = "0014100000Te0yvAAB"
		username = "alice"
	)
	reply := func(body string) *NATSResolver {
		return &NATSResolver{conn: &fakeRequester{reply: []byte(body)}}
	}
	check := func(r *NATSResolver) (bool, error) {
		return r.IsAffiliated(context.Background(), models.AttributionOrganization, org, username)
	}

	for rel, want := range map[string]bool{"writer": true, "auditor": true, "owner": true, "key_contact": false} {
		got, err := check(reply(`{"results":["b2b_org:` + org + `#` + rel + `@user:alice"]}`))
		if err != nil || got != want {
			t.Errorf("relation %s: got (%v, %v), want %v", rel, got, err, want)
		}
	}
	// A grant on a different org (e.g. the parent) does not count.
	if got, err := check(reply(`{"results":["b2b_org:0014100000Other0AAA#writer@user:alice"]}`)); err != nil || got {
		t.Errorf("other entity: got (%v, %v), want false", got, err)
	}
	if got, err := check(reply(`{"results":[]}`)); err != nil || got {
		t.Errorf("empty: got (%v, %v), want false", got, err)
	}
	if _, err := check(reply(`{"error":"failed to read tuples"}`)); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("error reply: got %v, want ErrUpstreamUnavailable", err)
	}
	r := &NATSResolver{conn: &fakeRequester{err: errors.New("boom")}}
	if _, err := check(r); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("transport error: got %v, want ErrUpstreamUnavailable", err)
	}
}
