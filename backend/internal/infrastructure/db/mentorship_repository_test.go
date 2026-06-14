// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package db

import (
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

// compile-time check that MentorshipRepositoryImpl satisfies domain.MentorshipRepository.
var _ domain.MentorshipRepository = (*MentorshipRepositoryImpl)(nil)
