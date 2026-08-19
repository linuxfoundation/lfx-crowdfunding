// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"fmt"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"go.opentelemetry.io/otel"
)

var orgAffiliationTracer = otel.Tracer("organization-affiliation-service")

// OrganizationAffiliationService resolves the v2 platform organizations
// (member-service b2b_org) that the calling user can attribute an initiative
// to, for the fundraise-form affiliation picker (LFXV2-3322/3323).
type OrganizationAffiliationService struct {
	tokenExchange clients.TokenExchangeClient
	queryService  clients.QueryServiceClient
}

// NewOrganizationAffiliationService returns an OrganizationAffiliationService,
// or nil when either client is nil — callers treat a nil service as "org
// affiliation lookup disabled" (Auth0 token exchange / query-service config
// not set).
func NewOrganizationAffiliationService(tokenExchange clients.TokenExchangeClient, queryService clients.QueryServiceClient) *OrganizationAffiliationService {
	if tokenExchange == nil || queryService == nil {
		return nil
	}
	return &OrganizationAffiliationService{tokenExchange: tokenExchange, queryService: queryService}
}

// ListForUser exchanges the caller's own Crowdfunding-audience access token for
// a v2-audience token, then returns the b2b_org organizations the caller has a
// direct FGA relation to (writer or auditor). rawToken is the caller's raw
// bearer JWT (models.Principal.RawToken).
func (s *OrganizationAffiliationService) ListForUser(ctx context.Context, rawToken string) ([]clients.OrgCandidate, error) {
	ctx, span := orgAffiliationTracer.Start(ctx, "OrganizationAffiliationService.ListForUser")
	defer span.End()

	if rawToken == "" {
		return nil, fmt.Errorf("%w: no bearer token on request", domain.ErrUnauthorized)
	}

	v2Token, err := s.tokenExchange.Exchange(ctx, rawToken)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("%w: token exchange: %w", domain.ErrUpstreamUnavailable, err)
	}

	orgs, err := s.queryService.SearchOrganizationsForUser(ctx, v2Token)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("%w: query service: %w", domain.ErrUpstreamUnavailable, err)
	}
	return orgs, nil
}
