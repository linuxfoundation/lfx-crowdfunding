// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/fga"
)

type fakeAccessRepo struct {
	rows []fga.InitiativeAccess
	err  error
}

func (f *fakeAccessRepo) ListForFGABackfill(_ context.Context) ([]fga.InitiativeAccess, error) {
	return f.rows, f.err
}

type fakeAccessPublisher struct {
	published []fga.InitiativeAccess
	failFor   map[string]error
}

func (f *fakeAccessPublisher) UpdateAccess(_ context.Context, access fga.InitiativeAccess) error {
	if err, ok := f.failFor[access.UID]; ok {
		return err
	}
	f.published = append(f.published, access)
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSyncer_Run(t *testing.T) {
	t.Run("republishes every row", func(t *testing.T) {
		repo := &fakeAccessRepo{rows: []fga.InitiativeAccess{
			{UID: "init-1", OwnerUsername: "alice"},
			{UID: "init-2", OwnerUsername: "bob"},
		}}
		pub := &fakeAccessPublisher{}
		s := newSyncer(repo, pub, discardLogger())

		result, err := s.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.total != 2 || result.published != 2 || result.failed != 0 {
			t.Fatalf("unexpected result: %+v", result)
		}
		if len(pub.published) != 2 {
			t.Fatalf("expected 2 published rows, got %d", len(pub.published))
		}
	})

	t.Run("one publish failure does not abort the run", func(t *testing.T) {
		repo := &fakeAccessRepo{rows: []fga.InitiativeAccess{
			{UID: "init-1", OwnerUsername: "alice"},
			{UID: "init-2", OwnerUsername: "bob"},
		}}
		pub := &fakeAccessPublisher{failFor: map[string]error{"init-1": errors.New("broker down")}}
		s := newSyncer(repo, pub, discardLogger())

		result, err := s.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.total != 2 || result.published != 1 || result.failed != 1 {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("no initiatives is a no-op", func(t *testing.T) {
		s := newSyncer(&fakeAccessRepo{}, &fakeAccessPublisher{}, discardLogger())
		result, err := s.Run(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.total != 0 {
			t.Fatalf("expected 0 total, got %d", result.total)
		}
	})

	t.Run("repo error propagates", func(t *testing.T) {
		s := newSyncer(&fakeAccessRepo{err: errors.New("db down")}, &fakeAccessPublisher{}, discardLogger())
		if _, err := s.Run(context.Background()); err == nil {
			t.Fatal("expected error")
		}
	})
}
