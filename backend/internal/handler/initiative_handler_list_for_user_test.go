// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// stubInitiativeRepoForListForUser implements domain.InitiativeRepository for ListForUser tests.
// It captures the filter passed to List so tests can assert that OwnerID was resolved correctly.
type stubInitiativeRepoForListForUser struct {
	initiatives  []*models.Initiative
	meta         *models.PaginationMeta
	err          error
	capturedFilter models.InitiativeFilter
}

func (r *stubInitiativeRepoForListForUser) List(_ context.Context, f models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	r.capturedFilter = f
	if r.err != nil {
		return nil, nil, r.err
	}
	meta := r.meta
	if meta == nil {
		meta = &models.PaginationMeta{Total: len(r.initiatives), Limit: 20, Offset: 0}
	}
	return r.initiatives, meta, nil
}

func (r *stubInitiativeRepoForListForUser) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, domain.ErrInitiativeNotFound
}
func (r *stubInitiativeRepoForListForUser) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, domain.ErrInitiativeNotFound
}
func (r *stubInitiativeRepoForListForUser) GetIDBySlug(_ context.Context, _ string) (string, error) {
	return "", domain.ErrInitiativeNotFound
}
func (r *stubInitiativeRepoForListForUser) ResolveSlug(_ context.Context, _ string) (string, error) {
	return "", domain.ErrInitiativeNotFound
}
func (r *stubInitiativeRepoForListForUser) Create(_ context.Context, _ *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	return nil, nil
}
func (r *stubInitiativeRepoForListForUser) Update(_ context.Context, _ *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	return nil, nil
}
func (r *stubInitiativeRepoForListForUser) Delete(_ context.Context, _ string) error { return nil }
func (r *stubInitiativeRepoForListForUser) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return nil, nil
}
func (r *stubInitiativeRepoForListForUser) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	return nil, nil
}
func (r *stubInitiativeRepoForListForUser) UpdateStripeProductID(_ context.Context, _, _ string) error {
	return nil
}

// stubUserRepoForListForUser implements domain.UserRepository for ListForUser tests.
type stubUserRepoForListForUser struct {
	user *models.User
	err  error
}

func (r *stubUserRepoForListForUser) GetByUsername(_ context.Context, _ string) (*models.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.user, nil
}
func (r *stubUserRepoForListForUser) GetByID(_ context.Context, _ string) (*models.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *stubUserRepoForListForUser) Upsert(_ context.Context, u *models.User) (*models.User, error) {
	return u, nil
}
func (r *stubUserRepoForListForUser) UpdateStripeInfo(_ context.Context, _, _, _ string) error {
	return nil
}
func (r *stubUserRepoForListForUser) ClearStripePaymentMethod(_ context.Context, _ string) error {
	return nil
}

func TestListForUser_ReturnsOwnedInitiatives(t *testing.T) {
	ownerID := "user-uuid-123"
	username := "testuser"

	userRepo := &stubUserRepoForListForUser{
		user: &models.User{ID: ownerID, Username: username},
	}
	initiativeRepo := &stubInitiativeRepoForListForUser{
		initiatives: []*models.Initiative{
			{ID: "init-1", Name: "My Project", OwnerID: ownerID},
		},
	}

	svc := service.NewInitiativeService(initiativeRepo, userRepo, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives", nil)
	req = req.WithContext(auth.ContextWithPrincipal(req.Context(), &models.Principal{
		Username: username,
	}))
	w := httptest.NewRecorder()

	h.ListForUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Data []*models.Initiative   `json:"data"`
		Meta *models.PaginationMeta `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("expected 1 initiative, got %d", len(body.Data))
	}
	if body.Data[0].ID != "init-1" {
		t.Errorf("expected initiative ID init-1, got %s", body.Data[0].ID)
	}
	// Verify the service resolved username → UUID and set OwnerID on the filter
	// before calling the repo — if it didn't, any user's initiatives would be returned.
	if initiativeRepo.capturedFilter.OwnerID != ownerID {
		t.Errorf("expected repo called with OwnerID=%s, got %q", ownerID, initiativeRepo.capturedFilter.OwnerID)
	}
}

func TestListForUser_NoPrincipal_Returns401(t *testing.T) {
	svc := service.NewInitiativeService(&stubInitiativeRepoForListForUser{}, &stubUserRepoForListForUser{}, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives", nil)
	w := httptest.NewRecorder()

	h.ListForUser(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListForUser_UserNotFound_ReturnsEmptyList(t *testing.T) {
	userRepo := &stubUserRepoForListForUser{err: domain.ErrUserNotFound}
	svc := service.NewInitiativeService(&stubInitiativeRepoForListForUser{}, userRepo, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives?limit=10&offset=5", nil)
	req = req.WithContext(auth.ContextWithPrincipal(req.Context(), &models.Principal{
		Username: "unknown",
	}))
	w := httptest.NewRecorder()

	h.ListForUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []any                  `json:"data"`
		Meta *models.PaginationMeta `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 0 {
		t.Errorf("expected empty data, got %d items", len(body.Data))
	}
	// Meta must reflect normalized pagination — same values the DB repo would return.
	if body.Meta == nil {
		t.Fatal("expected meta in response, got nil")
	}
	if body.Meta.Limit != 10 {
		t.Errorf("expected meta.limit=10, got %d", body.Meta.Limit)
	}
	if body.Meta.Offset != 5 {
		t.Errorf("expected meta.offset=5, got %d", body.Meta.Offset)
	}
	if body.Meta.Total != 0 {
		t.Errorf("expected meta.total=0, got %d", body.Meta.Total)
	}
}
