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
//	1. Fetch all programs from source (Snowflake or fixture).
//	2. For each program: normalise status, upsert initiative row, upsert beneficiaries.
//	3. Log per-program errors without halting the run.
//	4. Return per-run counters for summary logging.
func (s *Syncer) Run(ctx context.Context) (syncResult, error) {
	programs, err := s.source.FetchPrograms(ctx)
	if err != nil {
		return syncResult{}, fmt.Errorf("fetch programs: %w", err)
	}

	result := syncResult{total: len(programs)}

	for _, p := range programs {
		p.Status = strings.ToLower(p.Status)
		// Jobspring legacy value — normalise to match CF Postgres expected value.
		if p.Status == "hide" {
			p.Status = "hidden"
		}

		initiativeID, err := s.repo.UpsertProgram(ctx, p)
		if err != nil {
			s.logger.ErrorContext(ctx, "upsert program failed",
				"jobspring_project_id", p.JobspringProjectID,
				"error", err,
			)
			result.errors++
			continue
		}

		// Skip beneficiary upsert when the source did not provide beneficiary
		// data (nil slice). This avoids silently deleting existing rows when
		// the Snowflake query does not yet fetch SELECTED_MENTEES.
		if p.Beneficiaries != nil {
			if err := s.repo.UpsertBeneficiaries(ctx, initiativeID, p.Beneficiaries); err != nil {
				s.logger.ErrorContext(ctx, "upsert beneficiaries failed",
					"initiative_id", initiativeID,
					"jobspring_project_id", p.JobspringProjectID,
					"error", err,
				)
				result.errors++
				continue
			}
		}

		result.upserted++
	}

	return result, nil
}
