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

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// ── stub ──────────────────────────────────────────────────────────────────────

type announcementServiceStub struct {
	onList   func(ctx context.Context, initiativeID string, filter models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error)
	onCreate func(ctx context.Context, initiativeID, callerUsername string, input models.AnnouncementCreateInput) (*models.Announcement, error)
	onUpdate func(ctx context.Context, initiativeID, announcementID, callerUsername string, input models.AnnouncementUpdateInput) (*models.Announcement, error)
	onDelete func(ctx context.Context, initiativeID, announcementID, callerUsername string) error
}

func (s *announcementServiceStub) List(ctx context.Context, initiativeID string, filter models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error) {
	if s.onList != nil {
		return s.onList(ctx, initiativeID, filter)
	}
	return []models.Announcement{}, &models.PaginationMeta{}, nil
}

func (s *announcementServiceStub) Create(ctx context.Context, initiativeID, callerUsername string, input models.AnnouncementCreateInput) (*models.Announcement, error) {
	if s.onCreate != nil {
		return s.onCreate(ctx, initiativeID, callerUsername, input)
	}
	return &models.Announcement{}, nil
}

func (s *announcementServiceStub) Update(ctx context.Context, initiativeID, announcementID, callerUsername string, input models.AnnouncementUpdateInput) (*models.Announcement, error) {
	if s.onUpdate != nil {
		return s.onUpdate(ctx, initiativeID, announcementID, callerUsername, input)
	}
	return &models.Announcement{}, nil
}

func (s *announcementServiceStub) Delete(ctx context.Context, initiativeID, announcementID, callerUsername string) error {
	if s.onDelete != nil {
		return s.onDelete(ctx, initiativeID, announcementID, callerUsername)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

var announcementPrincipal = &models.Principal{Username: "initiative-owner"}

func newAnnouncementHandler(svc *announcementServiceStub) *AnnouncementHandler {
	return NewAnnouncementHandler(svc)
}

// announcementReq builds a test request with optional Chi URL params and principal.
func announcementReq(method, path, body string, params map[string]string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	ctx := r.Context()
	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	}
	if principal != nil {
		ctx = auth.ContextWithPrincipal(ctx, principal)
	}
	return r.WithContext(ctx)
}

func decodeAnnouncementBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestAnnouncementList_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &announcementServiceStub{
		onList: func(_ context.Context, initiativeID string, _ models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error) {
			if initiativeID != "init-123" {
				t.Errorf("unexpected initiativeID %q", initiativeID)
			}
			return []models.Announcement{
				{ID: "ann-1", InitiativeID: "init-123", CreatedBy: "owner", Title: "Hello", Description: "<p>World</p>", CreatedOn: now, UpdatedOn: now},
			}, &models.PaginationMeta{Total: 1, Limit: 20, Offset: 0}, nil
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.List(w, announcementReq(http.MethodGet, "/v1/initiatives/init-123/announcements", "", map[string]string{"id": "init-123"}, nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := decodeAnnouncementBody(t, w)
	data, ok := body["data"].([]any)
	if !ok || len(data) != 1 {
		t.Errorf("expected 1 item in data, got %v", body["data"])
	}
}

func TestAnnouncementList_InitiativeNotFound_Returns404(t *testing.T) {
	svc := &announcementServiceStub{
		onList: func(_ context.Context, _ string, _ models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error) {
			return nil, nil, domain.ErrInitiativeNotFound
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.List(w, announcementReq(http.MethodGet, "/v1/initiatives/missing/announcements", "", map[string]string{"id": "missing"}, nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAnnouncementList_ServiceError_Returns500(t *testing.T) {
	svc := &announcementServiceStub{
		onList: func(_ context.Context, _ string, _ models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error) {
			return nil, nil, errors.New("db unavailable")
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.List(w, announcementReq(http.MethodGet, "/v1/initiatives/init-123/announcements", "", map[string]string{"id": "init-123"}, nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestAnnouncementCreate_NoPrincipal_Returns401(t *testing.T) {
	h := newAnnouncementHandler(&announcementServiceStub{})
	w := httptest.NewRecorder()
	h.Create(w, announcementReq(http.MethodPost, "/v1/me/initiatives/init-123/announcements", `{"title":"T","description":"D"}`, map[string]string{"id": "init-123"}, nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAnnouncementCreate_InvalidJSON_Returns400(t *testing.T) {
	h := newAnnouncementHandler(&announcementServiceStub{})
	w := httptest.NewRecorder()
	h.Create(w, announcementReq(http.MethodPost, "/v1/me/initiatives/init-123/announcements", `not-json`, map[string]string{"id": "init-123"}, announcementPrincipal))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAnnouncementCreate_Forbidden_Returns403(t *testing.T) {
	svc := &announcementServiceStub{
		onCreate: func(_ context.Context, _, _ string, _ models.AnnouncementCreateInput) (*models.Announcement, error) {
			return nil, domain.ErrForbidden
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.Create(w, announcementReq(http.MethodPost, "/v1/me/initiatives/init-123/announcements", `{"title":"T","description":"D"}`, map[string]string{"id": "init-123"}, announcementPrincipal))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAnnouncementCreate_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	var capturedInput models.AnnouncementCreateInput
	svc := &announcementServiceStub{
		onCreate: func(_ context.Context, initiativeID, username string, input models.AnnouncementCreateInput) (*models.Announcement, error) {
			capturedInput = input
			return &models.Announcement{
				ID: "ann-new", InitiativeID: initiativeID, CreatedBy: username,
				Title: input.Title, Description: input.Description,
				CreatedOn: now, UpdatedOn: now,
			}, nil
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	body := `{"title":"Spring Update","description":"<p>Details here</p>"}`
	h.Create(w, announcementReq(http.MethodPost, "/v1/me/initiatives/init-123/announcements", body, map[string]string{"id": "init-123"}, announcementPrincipal))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if capturedInput.Title != "Spring Update" {
		t.Errorf("expected title 'Spring Update', got %q", capturedInput.Title)
	}
	if capturedInput.Description != "<p>Details here</p>" {
		t.Errorf("expected HTML description, got %q", capturedInput.Description)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestAnnouncementUpdate_NoPrincipal_Returns401(t *testing.T) {
	h := newAnnouncementHandler(&announcementServiceStub{})
	w := httptest.NewRecorder()
	h.Update(w, announcementReq(http.MethodPut, "/v1/me/initiatives/init-123/announcements/ann-1", `{"title":"T","description":"D"}`,
		map[string]string{"id": "init-123", "announcementId": "ann-1"}, nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAnnouncementUpdate_Forbidden_Returns403(t *testing.T) {
	svc := &announcementServiceStub{
		onUpdate: func(_ context.Context, _, _, _ string, _ models.AnnouncementUpdateInput) (*models.Announcement, error) {
			return nil, domain.ErrForbidden
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.Update(w, announcementReq(http.MethodPut, "/v1/me/initiatives/init-123/announcements/ann-1",
		`{"title":"T","description":"D"}`,
		map[string]string{"id": "init-123", "announcementId": "ann-1"}, announcementPrincipal))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAnnouncementUpdate_NotFound_Returns404(t *testing.T) {
	svc := &announcementServiceStub{
		onUpdate: func(_ context.Context, _, _, _ string, _ models.AnnouncementUpdateInput) (*models.Announcement, error) {
			return nil, domain.ErrAnnouncementNotFound
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.Update(w, announcementReq(http.MethodPut, "/v1/me/initiatives/init-123/announcements/missing",
		`{"title":"T","description":"D"}`,
		map[string]string{"id": "init-123", "announcementId": "missing"}, announcementPrincipal))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAnnouncementUpdate_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	svc := &announcementServiceStub{
		onUpdate: func(_ context.Context, initiativeID, announcementID, _ string, input models.AnnouncementUpdateInput) (*models.Announcement, error) {
			return &models.Announcement{
				ID: announcementID, InitiativeID: initiativeID,
				Title: input.Title, Description: input.Description,
				CreatedOn: now, UpdatedOn: now,
			}, nil
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.Update(w, announcementReq(http.MethodPut, "/v1/me/initiatives/init-123/announcements/ann-1",
		`{"title":"Updated","description":"<b>new</b>"}`,
		map[string]string{"id": "init-123", "announcementId": "ann-1"}, announcementPrincipal))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestAnnouncementDelete_NoPrincipal_Returns401(t *testing.T) {
	h := newAnnouncementHandler(&announcementServiceStub{})
	w := httptest.NewRecorder()
	h.Delete(w, announcementReq(http.MethodDelete, "/v1/me/initiatives/init-123/announcements/ann-1", "",
		map[string]string{"id": "init-123", "announcementId": "ann-1"}, nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAnnouncementDelete_NotFound_Returns404(t *testing.T) {
	svc := &announcementServiceStub{
		onDelete: func(_ context.Context, _, _, _ string) error {
			return domain.ErrAnnouncementNotFound
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.Delete(w, announcementReq(http.MethodDelete, "/v1/me/initiatives/init-123/announcements/missing", "",
		map[string]string{"id": "init-123", "announcementId": "missing"}, announcementPrincipal))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAnnouncementDelete_Forbidden_Returns403(t *testing.T) {
	svc := &announcementServiceStub{
		onDelete: func(_ context.Context, _, _, _ string) error {
			return domain.ErrForbidden
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.Delete(w, announcementReq(http.MethodDelete, "/v1/me/initiatives/init-123/announcements/ann-1", "",
		map[string]string{"id": "init-123", "announcementId": "ann-1"}, announcementPrincipal))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAnnouncementDelete_Success(t *testing.T) {
	svc := &announcementServiceStub{
		onDelete: func(_ context.Context, initiativeID, announcementID, username string) error {
			if initiativeID != "init-123" || announcementID != "ann-1" || username != announcementPrincipal.Username {
				t.Errorf("unexpected args: %q %q %q", initiativeID, announcementID, username)
			}
			return nil
		},
	}
	h := newAnnouncementHandler(svc)
	w := httptest.NewRecorder()
	h.Delete(w, announcementReq(http.MethodDelete, "/v1/me/initiatives/init-123/announcements/ann-1", "",
		map[string]string{"id": "init-123", "announcementId": "ann-1"}, announcementPrincipal))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
