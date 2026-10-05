// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

func TestCanManage_CreatorOnly(t *testing.T) {
	i := &models.Initiative{OwnerID: "u1"}
	for caller, want := range map[string]bool{"u1": true, "u2": false} {
		got, err := canManage(context.Background(), caller, i)
		if err != nil || got != want {
			t.Errorf("canManage(%q) = %v, %v; want %v, nil", caller, got, err, want)
		}
	}
}
