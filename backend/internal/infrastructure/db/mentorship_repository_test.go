// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// TestMentorshipRepository_implementsInterface is a compile-time check that
// MentorshipRepositoryImpl satisfies domain.MentorshipRepository.
func TestMentorshipRepository_implementsInterface(t *testing.T) {
	t.Helper()
	_ = func() {
		var _ interface {
			UpsertProgram(context.Context, models.MentorshipProgram) (string, error)
			UpsertBeneficiaries(context.Context, string, []models.MentorshipBeneficiary) error
			ListJobspringIDs(context.Context) ([]string, error)
		} = (*MentorshipRepositoryImpl)(nil)
	}
}
