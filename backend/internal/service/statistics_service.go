// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service contains the orchestration layer for the initiatives domain.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"go.opentelemetry.io/otel"
)

var statisticsSvcTracer = otel.Tracer("statistics-service")

const (
	donorTypeOrganization = "organization"
	donorTypeIndividual   = "individual"
	unknownOrgName        = "Unknown Organization"
	anonymousName         = "Anonymous"
)

// StatisticsService provides platform-wide aggregate data.
type StatisticsService struct {
	repo         domain.StatisticsRepository
	ledgerClient clients.LedgerClient
}

// NewStatisticsService returns a StatisticsService.
func NewStatisticsService(repo domain.StatisticsRepository, ledgerClient clients.LedgerClient) *StatisticsService {
	return &StatisticsService{repo: repo, ledgerClient: ledgerClient}
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

// GetPlatformDetails returns category totals, donor split, and top sponsors
// by fetching from the Ledger service and enriching sponsor names/avatars from CF DB.
func (s *StatisticsService) GetPlatformDetails(ctx context.Context) (*models.PlatformDetails, error) {
	ctx, span := statisticsSvcTracer.Start(ctx, "StatisticsService.GetPlatformDetails")
	defer span.End()

	raw, err := s.ledgerClient.GetPlatformBalance(ctx, 20)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, domain.ErrUpstreamUnavailable) {
			return &models.PlatformDetails{
				Categories:       []models.CategoryTotal{},
				TopOrganizations: []models.SponsorEntry{},
				TopIndividuals:   []models.SponsorEntry{},
			}, nil
		}
		return nil, fmt.Errorf("get platform balance: %w", err)
	}

	// Collect IDs for enrichment, skipping empty strings that would fail uuid[] cast.
	orgIDs := make([]string, 0, len(raw.TopOrganizations))
	for _, o := range raw.TopOrganizations {
		if o.ID != "" {
			orgIDs = append(orgIDs, o.ID)
		}
	}
	userIDs := make([]string, 0, len(raw.TopIndividuals))
	for _, u := range raw.TopIndividuals {
		if u.ID != "" {
			userIDs = append(userIDs, u.ID)
		}
	}

	orgs, err := s.repo.GetOrganizationsByIDs(ctx, orgIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("enrich sponsor organizations: %w", err)
	}
	users, err := s.repo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("enrich sponsor users: %w", err)
	}

	out := &models.PlatformDetails{
		TotalRaisedCents:   raw.OrganizationsCents + raw.IndividualsCents,
		TotalSupporters:    raw.TotalSupporters,
		OrganizationsCents: raw.OrganizationsCents,
		IndividualsCents:   raw.IndividualsCents,
		Categories:         []models.CategoryTotal{},
		TopOrganizations:   []models.SponsorEntry{},
		TopIndividuals:     []models.SponsorEntry{},
	}
	for _, c := range raw.Categories {
		out.Categories = append(out.Categories, models.CategoryTotal{
			Name:       c.Name,
			TotalCents: c.TotalCents,
			Count:      c.Count,
		})
	}
	for _, o := range raw.TopOrganizations {
		entry := models.SponsorEntry{ID: o.ID, TotalCents: o.Total}
		if org, ok := orgs[o.ID]; ok {
			entry.Name = org.Name
			entry.AvatarURL = org.AvatarURL
		} else {
			entry.Name = unknownOrgName
		}
		out.TopOrganizations = append(out.TopOrganizations, entry)
	}
	for _, u := range raw.TopIndividuals {
		entry := models.SponsorEntry{ID: u.ID, TotalCents: u.Total}
		if user, ok := users[u.ID]; ok {
			entry.Name = user.Name
			entry.AvatarURL = user.AvatarURL
		} else {
			entry.Name = anonymousName
		}
		out.TopIndividuals = append(out.TopIndividuals, entry)
	}
	return out, nil
}

// GetPlatformMonthly returns monthly donation buckets for the last 12 months.
func (s *StatisticsService) GetPlatformMonthly(ctx context.Context) (*models.PlatformMonthly, error) {
	ctx, span := statisticsSvcTracer.Start(ctx, "StatisticsService.GetPlatformMonthly")
	defer span.End()

	raw, err := s.ledgerClient.GetPlatformMonthly(ctx, 12)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, domain.ErrUpstreamUnavailable) {
			return &models.PlatformMonthly{Buckets: []models.MonthlyBucket{}}, nil
		}
		return nil, fmt.Errorf("get platform monthly: %w", err)
	}

	out := &models.PlatformMonthly{Buckets: []models.MonthlyBucket{}}
	for _, b := range raw.Buckets {
		out.Buckets = append(out.Buckets, models.MonthlyBucket{
			Year:          b.Year,
			Month:         b.Month,
			TotalCents:    b.TotalCents,
			Supporters:    b.Supporters,
			NewSupporters: b.NewSupporters,
		})
	}
	return out, nil
}

// GetRecentDonations returns the most recent platform-wide donations enriched
// with donor names and avatars from the CF database.
func (s *StatisticsService) GetRecentDonations(ctx context.Context) (*models.RecentDonationsResponse, error) {
	ctx, span := statisticsSvcTracer.Start(ctx, "StatisticsService.GetRecentDonations")
	defer span.End()

	raw, err := s.ledgerClient.GetPlatformRecentDonations(ctx)
	if err != nil {
		span.RecordError(err)
		if errors.Is(err, domain.ErrUpstreamUnavailable) {
			return &models.RecentDonationsResponse{Data: []models.RecentDonation{}}, nil
		}
		return nil, fmt.Errorf("get platform recent donations: %w", err)
	}

	// Collect IDs for enrichment
	orgIDs := make([]string, 0, len(raw))
	userIDs := make([]string, 0, len(raw))
	projectIDs := make([]string, 0, len(raw))
	for _, d := range raw {
		if d.OrganizationID != "" {
			orgIDs = append(orgIDs, d.OrganizationID)
		} else if d.UserID != "" {
			userIDs = append(userIDs, d.UserID)
		}
		if d.ProjectID != "" {
			projectIDs = append(projectIDs, d.ProjectID)
		}
	}

	orgs, err := s.repo.GetOrganizationsByIDs(ctx, orgIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("enrich donor organizations: %w", err)
	}
	users, err := s.repo.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("enrich donor users: %w", err)
	}
	projectNames, err := s.repo.GetInitiativeNamesByIDs(ctx, projectIDs)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("enrich project names: %w", err)
	}

	donations := make([]models.RecentDonation, 0, len(raw))
	for _, d := range raw {
		entry := models.RecentDonation{
			TxnID:       d.TxnID,
			ProjectID:   d.ProjectID,
			ProjectName: projectNames[d.ProjectID],
			AmountCents: d.Amount,
			TxnDate:     d.TxnDate,
			Category:    d.TxnCategory,
		}
		if d.OrganizationID != "" {
			entry.DonorType = donorTypeOrganization
			if org, ok := orgs[d.OrganizationID]; ok {
				entry.DonorName = org.Name
				entry.DonorAvatarURL = org.AvatarURL
			} else {
				entry.DonorName = d.SubmitterName
			}
		} else {
			entry.DonorType = donorTypeIndividual
			if user, ok := users[d.UserID]; ok {
				entry.DonorName = user.Name
				entry.DonorAvatarURL = user.AvatarURL
			} else {
				entry.DonorName = d.SubmitterName
			}
		}
		if entry.DonorName == "" {
			entry.DonorName = anonymousName
		}
		donations = append(donations, entry)
	}
	return &models.RecentDonationsResponse{Data: donations}, nil
}
