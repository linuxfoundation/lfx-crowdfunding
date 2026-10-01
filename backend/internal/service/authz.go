// SPDX-License-Identifier: MIT

package service

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// canManage is the single gate for every owner-scoped initiative path (update,
// delete, private reads, owner transactions, announcements). Creator-only today;
// attribution-based writer access (lfx-v2-crowdfunding#259) plugs in here. The
// ctx and error return are for that resolver call.
func canManage(_ context.Context, callerID string, initiative *models.Initiative) (bool, error) {
	return initiative.OwnerID == callerID, nil
}
