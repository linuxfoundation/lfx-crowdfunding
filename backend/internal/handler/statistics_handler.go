// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// StatisticsHandler holds Chi handlers for the /v1/statistics resource.
type StatisticsHandler struct {
	svc *service.StatisticsService
}

// NewStatisticsHandler creates a StatisticsHandler.
func NewStatisticsHandler(svc *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{svc: svc}
}

// GetPlatform handles GET /v1/statistics
func (h *StatisticsHandler) GetPlatform(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetPlatformStatistics(r.Context())
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, stats)
}
