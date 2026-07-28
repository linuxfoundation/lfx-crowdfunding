// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
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
	cachedJSON(w, r, stats)
}

// GetPlatformDetails handles GET /v1/statistics/platform
func (h *StatisticsHandler) GetPlatformDetails(w http.ResponseWriter, r *http.Request) {
	topLimit := 10
	if v := r.URL.Query().Get("top_limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			Error(w, fmt.Errorf("%w: top_limit must be a positive integer", domain.ErrInvalidInput))
			return
		}
		if n > 100 {
			n = 100
		}
		topLimit = n
	}
	details, err := h.svc.GetPlatformDetails(r.Context(), topLimit)
	if err != nil {
		Error(w, err)
		return
	}
	cachedJSON(w, r, details)
}

// GetPlatformMonthly handles GET /v1/statistics/monthly
func (h *StatisticsHandler) GetPlatformMonthly(w http.ResponseWriter, r *http.Request) {
	monthly, err := h.svc.GetPlatformMonthly(r.Context())
	if err != nil {
		Error(w, err)
		return
	}
	cachedJSON(w, r, monthly)
}

// GetOrgDonations handles GET /v1/statistics/org-donations
func (h *StatisticsHandler) GetOrgDonations(w http.ResponseWriter, r *http.Request) {
	donations, err := h.svc.GetOrgDonations(r.Context())
	if err != nil {
		Error(w, err)
		return
	}
	cachedJSON(w, r, donations)
}

// GetRecentDonations handles GET /v1/statistics/recent-donations
func (h *StatisticsHandler) GetRecentDonations(w http.ResponseWriter, r *http.Request) {
	donations, err := h.svc.GetRecentDonations(r.Context())
	if err != nil {
		Error(w, err)
		return
	}
	cachedJSON(w, r, donations)
}

// cachedJSON writes a JSON response with ETag and Cache-Control headers.
func cachedJSON(w http.ResponseWriter, r *http.Request, body any) {
	b, err := json.Marshal(body)
	if err != nil {
		Error(w, err)
		return
	}
	etag := etagOf(b)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
}
