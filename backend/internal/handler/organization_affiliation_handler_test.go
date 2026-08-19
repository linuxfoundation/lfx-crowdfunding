// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

type orgAffiliationServiceStub struct {
	onListForUser func(ctx context.Context, rawToken string) ([]clients.OrgCandidate, error)
}

func (s *orgAffiliationServiceStub) ListForUser(ctx context.Context, rawToken string) ([]clients.OrgCandidate, error) {
	if s.onListForUser != nil {
		return s.onListForUser(ctx, rawToken)
	}
	return nil, nil
}

func TestOrgAffiliationList_NilService_Returns503(t *testing.T) {
	h := NewOrganizationAffiliationHandler(nil)
	w := httptest.NewRecorder()
	req := orgReq(http.MethodGet, "/v1/me/organization-affiliations", "", testPrincipal)
	h.List(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestOrgAffiliationList_NoPrincipal_Returns401(t *testing.T) {
	h := NewOrganizationAffiliationHandler(&orgAffiliationServiceStub{})
	w := httptest.NewRecorder()
	req := orgReq(http.MethodGet, "/v1/me/organization-affiliations", "", nil)
	h.List(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestOrgAffiliationList_ServiceError_Returns500(t *testing.T) {
	svc := &orgAffiliationServiceStub{
		onListForUser: func(context.Context, string) ([]clients.OrgCandidate, error) {
			return nil, errServiceFailure
		},
	}
	h := NewOrganizationAffiliationHandler(svc)
	w := httptest.NewRecorder()
	req := orgReq(http.MethodGet, "/v1/me/organization-affiliations", "", testPrincipal)
	h.List(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestOrgAffiliationList_200_returnsOrgs(t *testing.T) {
	principal := &models.Principal{Username: "testuser", RawToken: "cf-token"}
	var gotRawToken string
	svc := &orgAffiliationServiceStub{
		onListForUser: func(_ context.Context, rawToken string) ([]clients.OrgCandidate, error) {
			gotRawToken = rawToken
			return []clients.OrgCandidate{{UID: "u1", Name: "Acme", LogoURL: "https://x/acme.png"}}, nil
		},
	}
	h := NewOrganizationAffiliationHandler(svc)
	w := httptest.NewRecorder()
	req := orgReq(http.MethodGet, "/v1/me/organization-affiliations", "", principal)
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if gotRawToken != "cf-token" {
		t.Errorf("expected service called with principal's RawToken, got %q", gotRawToken)
	}

	var body struct {
		Data []clients.OrgCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].UID != "u1" {
		t.Errorf("unexpected response body: %+v", body.Data)
	}
}

func TestOrgAffiliationList_NilOrgs_returnsEmptyArray(t *testing.T) {
	svc := &orgAffiliationServiceStub{
		onListForUser: func(context.Context, string) ([]clients.OrgCandidate, error) {
			return nil, nil
		},
	}
	h := NewOrganizationAffiliationHandler(svc)
	w := httptest.NewRecorder()
	req := orgReq(http.MethodGet, "/v1/me/organization-affiliations", "", testPrincipal)
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Data []clients.OrgCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data == nil || len(body.Data) != 0 {
		t.Errorf("expected empty (non-nil) data array, got %+v", body.Data)
	}
}
