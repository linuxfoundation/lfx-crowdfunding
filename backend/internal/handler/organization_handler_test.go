// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

var errServiceFailure = errors.New("service failure")

// ── stubs ─────────────────────────────────────────────────────────────────────

type orgServiceStub struct {
	onListByOwner func(ctx context.Context, username string) ([]models.Organization, error)
	onCreate      func(ctx context.Context, username string, input models.OrganizationCreateInput) (*models.Organization, error)
	onUpdate      func(ctx context.Context, username, id string, input models.OrganizationUpdateInput) (*models.Organization, error)
	onDelete      func(ctx context.Context, username, id string) error
}

func (s *orgServiceStub) ListByOwner(ctx context.Context, username string) ([]models.Organization, error) {
	if s.onListByOwner != nil {
		return s.onListByOwner(ctx, username)
	}
	return nil, nil
}

func (s *orgServiceStub) Create(ctx context.Context, username string, input models.OrganizationCreateInput) (*models.Organization, error) {
	if s.onCreate != nil {
		return s.onCreate(ctx, username, input)
	}
	return nil, nil
}

func (s *orgServiceStub) Update(ctx context.Context, username, id string, input models.OrganizationUpdateInput) (*models.Organization, error) {
	if s.onUpdate != nil {
		return s.onUpdate(ctx, username, id, input)
	}
	return nil, nil
}

func (s *orgServiceStub) Delete(ctx context.Context, username, id string) error {
	if s.onDelete != nil {
		return s.onDelete(ctx, username, id)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newOrgHandler(svc *orgServiceStub) *OrganizationHandler {
	return NewOrganizationHandler(svc)
}

func orgReq(method, path, body string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

var testPrincipal = &models.Principal{Username: "testuser"}

// ── List tests ────────────────────────────────────────────────────────────────

func TestOrgList_NoPrincipal_Returns401(t *testing.T) {
	h := newOrgHandler(&orgServiceStub{})
	w := httptest.NewRecorder()
	h.List(w, orgReq(http.MethodGet, "/v1/me/organizations", "", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestOrgList_ServiceError_Returns500(t *testing.T) {
	svc := &orgServiceStub{
		onListByOwner: func(_ context.Context, _ string) ([]models.Organization, error) {
			return nil, errServiceFailure
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	h.List(w, orgReq(http.MethodGet, "/v1/me/organizations", "", testPrincipal))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestOrgList_Success_WrapsInDataField(t *testing.T) {
	now := time.Now()
	svc := &orgServiceStub{
		onListByOwner: func(_ context.Context, username string) ([]models.Organization, error) {
			if username != testPrincipal.Username {
				t.Errorf("unexpected username %q", username)
			}
			return []models.Organization{
				{ID: "org-1", Name: "Acme Corp", CreatedOn: now, UpdatedOn: now},
			}, nil
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	h.List(w, orgReq(http.MethodGet, "/v1/me/organizations", "", testPrincipal))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Data []models.Organization `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != "org-1" {
		t.Errorf("unexpected data: %+v", body.Data)
	}
}

func TestOrgList_NilFromService_ReturnsEmptyArray(t *testing.T) {
	svc := &orgServiceStub{
		onListByOwner: func(_ context.Context, _ string) ([]models.Organization, error) {
			return nil, nil
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	h.List(w, orgReq(http.MethodGet, "/v1/me/organizations", "", testPrincipal))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Data []models.Organization `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data == nil || len(body.Data) != 0 {
		t.Errorf("expected empty slice, got %+v", body.Data)
	}
}

// ── Create tests ──────────────────────────────────────────────────────────────

func TestOrgCreate_NoPrincipal_Returns401(t *testing.T) {
	h := newOrgHandler(&orgServiceStub{})
	w := httptest.NewRecorder()
	h.Create(w, orgReq(http.MethodPost, "/v1/me/organizations", `{"name":"Acme"}`, nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestOrgCreate_InvalidJSON_Returns400(t *testing.T) {
	h := newOrgHandler(&orgServiceStub{})
	w := httptest.NewRecorder()
	h.Create(w, orgReq(http.MethodPost, "/v1/me/organizations", `{bad json}`, testPrincipal))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgCreate_ServiceError_Returns500(t *testing.T) {
	svc := &orgServiceStub{
		onCreate: func(_ context.Context, _ string, _ models.OrganizationCreateInput) (*models.Organization, error) {
			return nil, errServiceFailure
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	h.Create(w, orgReq(http.MethodPost, "/v1/me/organizations", `{"name":"Acme"}`, testPrincipal))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestOrgCreate_Success_Returns201WithOrg(t *testing.T) {
	now := time.Now()
	created := &models.Organization{ID: "org-new", Name: "Acme Corp", CreatedOn: now, UpdatedOn: now}
	svc := &orgServiceStub{
		onCreate: func(_ context.Context, username string, input models.OrganizationCreateInput) (*models.Organization, error) {
			if username != testPrincipal.Username {
				t.Errorf("unexpected username %q", username)
			}
			if input.Name != "Acme Corp" {
				t.Errorf("unexpected name %q", input.Name)
			}
			return created, nil
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	h.Create(w, orgReq(http.MethodPost, "/v1/me/organizations", `{"name":"Acme Corp"}`, testPrincipal))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var org models.Organization
	if err := json.NewDecoder(w.Body).Decode(&org); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if org.ID != "org-new" {
		t.Errorf("expected org-new, got %q", org.ID)
	}
}

// ── Update tests ──────────────────────────────────────────────────────────────

func TestOrgUpdate_NoPrincipal_Returns401(t *testing.T) {
	h := newOrgHandler(&orgServiceStub{})
	w := httptest.NewRecorder()
	h.Update(w, orgReq(http.MethodPatch, "/v1/me/organizations/org-1", `{"name":"New"}`, nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestOrgUpdate_InvalidJSON_Returns400(t *testing.T) {
	h := newOrgHandler(&orgServiceStub{})
	// Chi URL param not set in unit test — the id="" branch fires first, but we
	// can also test invalid JSON by skipping URL param and relying on the JSON decode path.
	// Here we test explicitly via the JSON decode error path.
	w := httptest.NewRecorder()
	req := orgReq(http.MethodPatch, "/v1/me/organizations/org-1", `{bad}`, testPrincipal)
	// inject a non-empty id via a minimal chi context shim
	req = withURLParam(req, "id", "org-1")
	h.Update(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgUpdate_ServiceError_Returns500(t *testing.T) {
	svc := &orgServiceStub{
		onUpdate: func(_ context.Context, _, _ string, _ models.OrganizationUpdateInput) (*models.Organization, error) {
			return nil, errServiceFailure
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	req := orgReq(http.MethodPatch, "/v1/me/organizations/org-1", `{"name":"New"}`, testPrincipal)
	req = withURLParam(req, "id", "org-1")
	h.Update(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestOrgUpdate_Success_Returns200WithOrg(t *testing.T) {
	now := time.Now()
	updated := &models.Organization{ID: "org-1", Name: "New Name", CreatedOn: now, UpdatedOn: now}
	svc := &orgServiceStub{
		onUpdate: func(_ context.Context, username, id string, input models.OrganizationUpdateInput) (*models.Organization, error) {
			if username != testPrincipal.Username {
				t.Errorf("unexpected username %q", username)
			}
			if id != "org-1" {
				t.Errorf("unexpected id %q", id)
			}
			if input.Name != "New Name" {
				t.Errorf("unexpected name %q", input.Name)
			}
			return updated, nil
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	req := orgReq(http.MethodPatch, "/v1/me/organizations/org-1", `{"name":"New Name"}`, testPrincipal)
	req = withURLParam(req, "id", "org-1")
	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var org models.Organization
	if err := json.NewDecoder(w.Body).Decode(&org); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if org.ID != "org-1" || org.Name != "New Name" {
		t.Errorf("unexpected org: %+v", org)
	}
}

// ── Delete tests ──────────────────────────────────────────────────────────────

func TestOrgDelete_NoPrincipal_Returns401(t *testing.T) {
	h := newOrgHandler(&orgServiceStub{})
	w := httptest.NewRecorder()
	h.Delete(w, orgReq(http.MethodDelete, "/v1/me/organizations/org-1", "", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestOrgDelete_NotFound_Returns404(t *testing.T) {
	svc := &orgServiceStub{
		onDelete: func(_ context.Context, _, _ string) error {
			return domain.ErrOrganizationNotFound
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	req := orgReq(http.MethodDelete, "/v1/me/organizations/org-1", "", testPrincipal)
	req = withURLParam(req, "id", "org-1")
	h.Delete(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgDelete_Success_Returns204(t *testing.T) {
	deleted := false
	svc := &orgServiceStub{
		onDelete: func(_ context.Context, username, id string) error {
			if username != testPrincipal.Username {
				t.Errorf("unexpected username %q", username)
			}
			if id != "org-1" {
				t.Errorf("unexpected id %q", id)
			}
			deleted = true
			return nil
		},
	}
	h := newOrgHandler(svc)
	w := httptest.NewRecorder()
	req := orgReq(http.MethodDelete, "/v1/me/organizations/org-1", "", testPrincipal)
	req = withURLParam(req, "id", "org-1")
	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if !deleted {
		t.Error("expected service Delete to be called")
	}
}
