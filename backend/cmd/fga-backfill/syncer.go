// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/fga"
)

// accessRepo is the interface the Syncer needs from the initiative
// repository. Defined at the point of consumption per constitution
// Principle III.
type accessRepo interface {
	ListForFGABackfill(ctx context.Context) ([]fga.InitiativeAccess, error)
}

// accessPublisher is the interface the Syncer needs from fga.Publisher.
type accessPublisher interface {
	UpdateAccess(ctx context.Context, access fga.InitiativeAccess) error
}

// syncResult carries the per-run counters logged on completion.
type syncResult struct {
	total     int // initiatives found in CF DB
	published int // UpdateAccess calls that succeeded
	failed    int // UpdateAccess calls that returned an error
}

// Syncer orchestrates a single fga-backfill run.
type Syncer struct {
	repo      accessRepo
	publisher accessPublisher
	logger    *slog.Logger
}

func newSyncer(repo accessRepo, publisher accessPublisher, logger *slog.Logger) *Syncer {
	return &Syncer{repo: repo, publisher: publisher, logger: logger}
}

// Run republishes every initiative's current access state. A single row's
// publish failure is logged and counted, not fatal — the job keeps going so
// one bad row doesn't block the rest of the backfill, and it's safe to rerun.
func (s *Syncer) Run(ctx context.Context) (syncResult, error) {
	rows, err := s.repo.ListForFGABackfill(ctx)
	if err != nil {
		return syncResult{}, fmt.Errorf("list initiatives for fga backfill: %w", err)
	}

	result := syncResult{total: len(rows)}
	for _, access := range rows {
		if err := s.publisher.UpdateAccess(ctx, access); err != nil {
			s.logger.WarnContext(ctx, "fga-backfill: publish failed for initiative",
				"initiative_id", access.UID, "error", err)
			result.failed++
			continue
		}
		result.published++
	}
	return result, nil
}
