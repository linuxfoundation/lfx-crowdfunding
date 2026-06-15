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
)

// mentorshipSource is the interface Syncer needs from its data source.
// Defined at the point of consumption — both snowflake.Client and
// snowflake.FixtureSource satisfy this interface.
type mentorshipSource interface {
	FetchPrograms(ctx context.Context) ([]models.MentorshipProgram, error)
}

// syncResult carries the per-run counters logged on completion.
type syncResult struct {
	total    int
	upserted int
	errors   int
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
//  1. Fetch all programs from source (Snowflake or fixture).
//  2. For each program: normalise status, upsert initiative row, upsert beneficiaries.
//  3. Log per-program errors without halting the run.
//  4. Return per-run counters for summary logging.
func (s *Syncer) Run(ctx context.Context) (syncResult, error) {
	programs, err := s.source.FetchPrograms(ctx)
	if err != nil {
		return syncResult{}, fmt.Errorf("fetch programs: %w", err)
	}

	result := syncResult{total: len(programs)}

	for _, p := range programs {
		p.Status = strings.ToLower(p.Status)
		// Jobspring legacy value — normalise to match CF Postgres expected value.
		if p.Status == legacyStatusHide {
			p.Status = normalizedStatusHidden
		}

		if strings.TrimSpace(p.OwnerLFUsername) == "" {
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
	}

	return result, nil
}
