// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/fga"
)

// canManage is the single gate for every owner-scoped initiative path (update,
// delete, private reads, owner transactions, announcements): the creator, or a
// writer on the attributed entity (lfx-v2-crowdfunding#259). Creator access is
// permanent and makes no FGA call; personal initiatives never consult the
// resolver. A nil resolver (FGA_NATS_URL unset) degrades to creator-only.
// A resolver outage surfaces as an error wrapping domain.ErrUpstreamUnavailable
// and must never be turned into a false result.
func canManage(ctx context.Context, resolver fga.EntityRoleResolver, caller *models.User, initiative *models.Initiative) (bool, error) {
	if initiative.OwnerID == caller.ID {
		return true, nil
	}
	a := initiative.Attribution
	if resolver == nil || (a.Type != models.AttributionOrganization && a.Type != models.AttributionProject) {
		return false, nil
	}
	return resolver.CanManage(ctx, a.Type, a.EntityUID, caller.Username)
}
