// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

type testTokenExchangeClient struct {
	onExchange func(context.Context, string) (string, error)
}

func (c *testTokenExchangeClient) Exchange(ctx context.Context, subjectToken string) (string, error) {
	if c.onExchange != nil {
		return c.onExchange(ctx, subjectToken)
	}
	return "", nil
}

type testQueryServiceClient struct {
	onSearch func(context.Context, string) ([]clients.OrgCandidate, error)
}

func (c *testQueryServiceClient) SearchOrganizationsForUser(ctx context.Context, bearerToken string) ([]clients.OrgCandidate, error) {
	if c.onSearch != nil {
		return c.onSearch(ctx, bearerToken)
	}
	return nil, nil
}

func TestNewOrganizationAffiliationService_NilClient_returnsNil(t *testing.T) {
	te := &testTokenExchangeClient{}
	qs := &testQueryServiceClient{}
	if s := NewOrganizationAffiliationService(nil, qs); s != nil {
		t.Error("expected nil service when tokenExchange is nil")
	}
	if s := NewOrganizationAffiliationService(te, nil); s != nil {
		t.Error("expected nil service when queryService is nil")
	}
}

func TestOrganizationAffiliationService_ListForUser_NoRawToken_errorsUnauthorized(t *testing.T) {
	svc := NewOrganizationAffiliationService(&testTokenExchangeClient{}, &testQueryServiceClient{})
	_, err := svc.ListForUser(context.Background(), "")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestOrganizationAffiliationService_ListForUser_ExchangesThenQueries(t *testing.T) {
	var gotSubject, gotBearer string
	te := &testTokenExchangeClient{onExchange: func(_ context.Context, subjectToken string) (string, error) {
		gotSubject = subjectToken
		return "v2-token", nil
	}}
	qs := &testQueryServiceClient{onSearch: func(_ context.Context, bearerToken string) ([]clients.OrgCandidate, error) {
		gotBearer = bearerToken
		return []clients.OrgCandidate{{UID: "u1", Name: "Acme"}}, nil
	}}

	orgs, err := NewOrganizationAffiliationService(te, qs).ListForUser(context.Background(), "cf-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSubject != "cf-token" {
		t.Errorf("expected exchange called with cf-token, got %q", gotSubject)
	}
	if gotBearer != "v2-token" {
		t.Errorf("expected query-service called with exchanged token, got %q", gotBearer)
	}
	if len(orgs) != 1 || orgs[0].UID != "u1" {
		t.Errorf("unexpected orgs: %+v", orgs)
	}
}

func TestOrganizationAffiliationService_ListForUser_ExchangeFails_mapsToUpstreamUnavailable(t *testing.T) {
	te := &testTokenExchangeClient{onExchange: func(context.Context, string) (string, error) {
		return "", errors.New("boom")
	}}
	qs := &testQueryServiceClient{}

	_, err := NewOrganizationAffiliationService(te, qs).ListForUser(context.Background(), "cf-token")
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable, got: %v", err)
	}
}

func TestOrganizationAffiliationService_ListForUser_QueryFails_mapsToUpstreamUnavailable(t *testing.T) {
	te := &testTokenExchangeClient{onExchange: func(context.Context, string) (string, error) {
		return "v2-token", nil
	}}
	qs := &testQueryServiceClient{onSearch: func(context.Context, string) ([]clients.OrgCandidate, error) {
		return nil, errors.New("boom")
	}}

	_, err := NewOrganizationAffiliationService(te, qs).ListForUser(context.Background(), "cf-token")
	if !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable, got: %v", err)
	}
}
