// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// canManage is the single gate for every owner-scoped initiative path (update,
// delete, private reads, owner transactions, announcements). Creator-only today.
// How writer access is enforced (lfx-v2-crowdfunding#259) is still open; it may
// move to the Heimdall openfga_check at the edge and remove this check rather
// than extend it.
func canManage(_ context.Context, callerID string, initiative *models.Initiative) (bool, error) {
	return initiative.OwnerID == callerID, nil
}
