// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

const (
	// legacyStatusHide is the Jobspring legacy status value that maps to hidden.
	legacyStatusHide       = "hide"
	normalizedStatusHidden = "hidden"
	// statusRejected is a Jobspring program_status with no CF equivalent; it maps to declined.
	statusRejected = "rejected"
)

// mentorshipSource is the interface Syncer needs from its data source.
// Defined at the point of consumption — both snowflake.Client and
// snowflake.FixtureSource satisfy this interface.
type mentorshipSource interface {
	FetchPrograms(ctx context.Context) ([]models.MentorshipProgram, error)
}

// syncResult carries the per-run counters logged on completion.
type syncResult struct {
	total      int
	upserted   int
	backfilled int // subset of upserted: outside the window but missing from CF
	unchanged  int // outside the window and already in CF
	errors     int
}

// Syncer orchestrates a single mentorship-sync run.
type Syncer struct {
	repo   domain.MentorshipRepository
	source mentorshipSource
	logger *slog.Logger
}

// newSyncer returns a configured Syncer ready to call Run.
func newSyncer(repo domain.MentorshipRepository, source mentorshipSource, logger *slog.Logger) *Syncer {
	return &Syncer{repo: repo, source: source, logger: logger}
}

// Run executes the full sync algorithm:
//
//  1. Fetch programs from source (Snowflake or fixture) and the IDs already in CF.
//  2. Skip programs outside the update window that already exist in CF.
//  3. For each remaining program: normalise status, upsert initiative row, upsert beneficiaries.
//  4. Log per-program errors without halting the run.
//  5. Return per-run counters for summary logging.
func (s *Syncer) Run(ctx context.Context) (syncResult, error) {
	programs, err := s.source.FetchPrograms(ctx)
	if err != nil {
		return syncResult{}, fmt.Errorf("fetch programs: %w", err)
	}

	existingIDs, err := s.repo.ListKnownProgramIDs(ctx)
	if err != nil {
		return syncResult{}, fmt.Errorf("list existing programs: %w", err)
	}
	existing := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existing[id] = struct{}{}
	}

	result := syncResult{total: len(programs)}

	for _, p := range programs {
		backfill := false
		if !p.RecentlyUpdated {
			if _, ok := existing[p.JobspringProjectID]; ok {
				result.unchanged++
				continue
			}
			backfill = true
		}

		p.Status = strings.ToLower(p.Status)
		// Map Jobspring values with no CF equivalent onto CF statuses.
		switch p.Status {
		case legacyStatusHide:
			p.Status = normalizedStatusHidden
		case statusRejected:
			p.Status = string(models.StatusDeclined)
		}
		if !models.InitiativeStatus(p.Status).IsValid() {
			s.logger.ErrorContext(ctx, "skipping program: invalid status",
				"jobspring_project_id", p.JobspringProjectID,
				"status", p.Status,
			)
			result.errors++
			continue
		}

		p.OwnerLFUsername = strings.TrimSpace(p.OwnerLFUsername)
		if p.OwnerLFUsername == "" {
			s.logger.ErrorContext(ctx, "skipping program: missing owner_lf_username",
				"jobspring_project_id", p.JobspringProjectID,
			)
			result.errors++
			continue
		}

		if _, err := s.repo.UpsertProgram(ctx, p); err != nil {
			s.logger.ErrorContext(ctx, "upsert program failed",
				"jobspring_project_id", p.JobspringProjectID,
				"error", err,
			)
			result.errors++
			continue
		}

		result.upserted++
		if backfill {
			result.backfilled++
			s.logger.InfoContext(ctx, "backfilled program missing from CF",
				"jobspring_project_id", p.JobspringProjectID,
			)
		}
	}

	return result, nil
}
