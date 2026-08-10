// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package fga

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// TestNATSResolver_Integration exercises NATSResolver against a real fga-sync
// deployment. Skipped unless FGA_NATS_URL is set (see AC: "resolver
// integration-tested against fga-sync in dev with 404/503 semantics
// verified"). Requires FGA_TEST_PROJECT_UID and FGA_TEST_USERNAME to name a
// real project the test principal is (or isn't) a writer on.
//
// Example, after port-forwarding fga-sync's NATS:
//
//	FGA_NATS_URL=nats://localhost:4222 \
//	FGA_TEST_PROJECT_UID=<uid> FGA_TEST_USERNAME=<username> \
//	go test ./internal/infrastructure/fga/ -run Integration -v
func TestNATSResolver_Integration(t *testing.T) {
	natsURL := os.Getenv("FGA_NATS_URL")
	if natsURL == "" {
		t.Skip("FGA_NATS_URL not set — skipping fga-sync integration test")
	}
	projectUID := os.Getenv("FGA_TEST_PROJECT_UID")
	username := os.Getenv("FGA_TEST_USERNAME")
	if projectUID == "" || username == "" {
		t.Skip("FGA_TEST_PROJECT_UID / FGA_TEST_USERNAME not set — skipping fga-sync integration test")
	}

	conn, err := nats.Connect(natsURL, nats.Timeout(5*time.Second))
	if err != nil {
		t.Fatalf("nats.Connect: %v", err)
	}
	defer conn.Close()
	resolver := NewNATSResolver(conn)

	// A definitive result (true or false, whichever it is) must come back
	// without error — this is the "404 semantics" leg: a real, reachable
	// fga-sync always returns a definitive verdict for a well-formed check.
	_, err = resolver.CanManage(context.Background(), models.AttributionProject, projectUID, username)
	if err != nil {
		t.Fatalf("CanManage against live fga-sync: %v", err)
	}

	// An unreachable fga-sync must surface as ErrUpstreamUnavailable (the
	// "503 semantics" leg), never a false/404.
	deadConn, err := nats.Connect("nats://127.0.0.1:1", nats.Timeout(1*time.Second), nats.NoReconnect())
	if err == nil {
		defer deadConn.Close()
		deadResolver := NewNATSResolver(deadConn)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err = deadResolver.CanManage(ctx, models.AttributionProject, projectUID, username)
		if err == nil {
			t.Fatal("expected an error from an unreachable fga-sync connection")
		}
		if !errors.Is(err, domain.ErrUpstreamUnavailable) {
			t.Fatalf("expected error to wrap ErrUpstreamUnavailable, got: %v", err)
		}
	}
}
