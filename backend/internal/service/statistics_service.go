// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"fmt"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"go.opentelemetry.io/otel"
)

var statisticsSvcTracer = otel.Tracer("statistics-service")

// StatisticsService provides platform-wide aggregate data.
type StatisticsService struct {
	repo domain.StatisticsRepository
}

// NewStatisticsService returns a StatisticsService.
func NewStatisticsService(repo domain.StatisticsRepository) *StatisticsService {
	return &StatisticsService{repo: repo}
}

// GetPlatformStatistics returns platform-wide aggregates for the landing page.
func (s *StatisticsService) GetPlatformStatistics(ctx context.Context) (*models.PlatformStatistics, error) {
	ctx, span := statisticsSvcTracer.Start(ctx, "StatisticsService.GetPlatformStatistics")
	defer span.End()

	stats, err := s.repo.GetPlatformStatistics(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("get platform statistics: %w", err)
	}
	return stats, nil
}
