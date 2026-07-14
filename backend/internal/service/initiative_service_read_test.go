// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// listInitiativeRepo is a configurable InitiativeRepository for the read/list
// service methods. Unlike mockInitiativeRepo it lets each lookup be driven
// independently, which the ownership-scoped reads require.
type listInitiativeRepo struct {
	onGetByID       func(context.Context, string) (*models.Initiative, error)
	onGetBySlug     func(context.Context, string) (*models.Initiative, error)
	onGetIDBySlug   func(context.Context, string) (string, error)
	onResolveSlug   func(context.Context, string) (string, error)
	onList          func(context.Context, models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error)
	onListPublished func(context.Context) ([]models.InitiativeSummary, error)
	onDelete        func(context.Context, string) error
}

func (r *listInitiativeRepo) GetByID(ctx context.Context, id string) (*models.Initiative, error) {
	if r.onGetByID != nil {
		return r.onGetByID(ctx, id)
	}
	return nil, nil
}
func (r *listInitiativeRepo) GetBySlug(ctx context.Context, slug string) (*models.Initiative, error) {
	if r.onGetBySlug != nil {
		return r.onGetBySlug(ctx, slug)
	}
	return nil, nil
}
func (r *listInitiativeRepo) GetIDBySlug(ctx context.Context, slug string) (string, error) {
	if r.onGetIDBySlug != nil {
		return r.onGetIDBySlug(ctx, slug)
	}
	return "", nil
}
func (r *listInitiativeRepo) ResolveSlug(ctx context.Context, slug string) (string, error) {
	if r.onResolveSlug != nil {
		return r.onResolveSlug(ctx, slug)
	}
	return "", nil
}
func (r *listInitiativeRepo) List(ctx context.Context, filter models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	if r.onList != nil {
		return r.onList(ctx, filter)
	}
	return nil, nil, nil
}
func (r *listInitiativeRepo) Create(_ context.Context, i *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	return i, nil
}
func (r *listInitiativeRepo) Update(_ context.Context, i *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	return i, nil
}
func (r *listInitiativeRepo) Delete(ctx context.Context, id string) error {
	if r.onDelete != nil {
		return r.onDelete(ctx, id)
	}
	return nil
}
func (r *listInitiativeRepo) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return map[string]models.User{}, nil
}
func (r *listInitiativeRepo) GetUsersByLegacyIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return map[string]models.User{}, nil
}
func (r *listInitiativeRepo) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	return map[string]models.Organization{}, nil
}
func (r *listInitiativeRepo) GetOwnerInfoBySlug(_ context.Context, _ string) (models.OwnerInfo, error) {
	return models.OwnerInfo{}, nil
}
func (r *listInitiativeRepo) ListPublished(ctx context.Context) ([]models.InitiativeSummary, error) {
	if r.onListPublished != nil {
		return r.onListPublished(ctx)
	}
	return nil, nil
}
func (r *listInitiativeRepo) UpdateStripeProductID(_ context.Context, _, _ string) error {
	return nil
}

func newReadSvc(repo domain.InitiativeRepository, userRepo domain.UserRepository) *InitiativeService {
	return NewInitiativeService(repo, userRepo, &mockLedgerClient{}, &mockStripeClient{}, &mockEmailService{}, nil, slog.Default())
}

// --- CheckPublishedByID ---

func TestCheckPublishedByID_Published(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", Status: models.StatusPublished}, nil
		},
	}
	if err := newReadSvc(repo, &mockUserRepository{}).CheckPublishedByID(context.Background(), "init-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckPublishedByID_NotPublished(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", Status: models.StatusSubmitted}, nil
		},
	}
	err := newReadSvc(repo, &mockUserRepository{}).CheckPublishedByID(context.Background(), "init-1")
	if !errors.Is(err, domain.ErrInitiativeNotFound) {
		t.Errorf("expected ErrInitiativeNotFound, got %v", err)
	}
}

func TestCheckPublishedByID_RepoError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return nil, repoErr
		},
	}
	err := newReadSvc(repo, &mockUserRepository{}).CheckPublishedByID(context.Background(), "init-1")
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

// --- GetIDBySlug / ResolveSlug ---

func TestGetIDBySlug(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetIDBySlug: func(_ context.Context, slug string) (string, error) {
			if slug != "my-slug" {
				t.Errorf("slug = %q, want my-slug", slug)
			}
			return "init-1", nil
		},
	}
	id, err := newReadSvc(repo, &mockUserRepository{}).GetIDBySlug(context.Background(), "my-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "init-1" {
		t.Errorf("id = %q, want init-1", id)
	}
}

func TestGetIDBySlug_RepoError(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetIDBySlug: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not found")
		},
	}
	_, err := newReadSvc(repo, &mockUserRepository{}).GetIDBySlug(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveSlug(t *testing.T) {
	repo := &listInitiativeRepo{
		onResolveSlug: func(_ context.Context, _ string) (string, error) {
			return "init-2", nil
		},
	}
	id, err := newReadSvc(repo, &mockUserRepository{}).ResolveSlug(context.Background(), "draft-slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "init-2" {
		t.Errorf("id = %q, want init-2", id)
	}
}

func TestResolveSlug_RepoError(t *testing.T) {
	repo := &listInitiativeRepo{
		onResolveSlug: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not found")
		},
	}
	_, err := newReadSvc(repo, &mockUserRepository{}).ResolveSlug(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetBySlug ---

func TestGetBySlug(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetBySlug: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", Name: "Kubernetes"}, nil
		},
	}
	init, err := newReadSvc(repo, &mockUserRepository{}).GetBySlug(context.Background(), "kubernetes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if init.Name != "Kubernetes" {
		t.Errorf("name = %q, want Kubernetes", init.Name)
	}
}

func TestGetBySlug_RepoError(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetBySlug: func(_ context.Context, _ string) (*models.Initiative, error) {
			return nil, errors.New("not found")
		},
	}
	_, err := newReadSvc(repo, &mockUserRepository{}).GetBySlug(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ListPublished ---

func TestListPublished(t *testing.T) {
	repo := &listInitiativeRepo{
		onListPublished: func(_ context.Context) ([]models.InitiativeSummary, error) {
			return []models.InitiativeSummary{{ID: "init-1", Name: "K8s"}}, nil
		},
	}
	out, err := newReadSvc(repo, &mockUserRepository{}).ListPublished(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0].Name != "K8s" {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestListPublished_RepoError(t *testing.T) {
	repo := &listInitiativeRepo{
		onListPublished: func(_ context.Context) ([]models.InitiativeSummary, error) {
			return nil, errors.New("db down")
		},
	}
	_, err := newReadSvc(repo, &mockUserRepository{}).ListPublished(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- List ---

func TestList(t *testing.T) {
	repo := &listInitiativeRepo{
		onList: func(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
			return []*models.Initiative{{ID: "init-1"}}, &models.PaginationMeta{Total: 1}, nil
		},
	}
	out, meta, err := newReadSvc(repo, &mockUserRepository{}).List(context.Background(), models.InitiativeFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || meta.Total != 1 {
		t.Fatalf("unexpected result: out=%+v meta=%+v", out, meta)
	}
}

func TestList_RepoError(t *testing.T) {
	repo := &listInitiativeRepo{
		onList: func(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
			return nil, nil, errors.New("db down")
		},
	}
	_, _, err := newReadSvc(repo, &mockUserRepository{}).List(context.Background(), models.InitiativeFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ListForUser ---

func TestListForUser(t *testing.T) {
	var seenOwnerID string
	repo := &listInitiativeRepo{
		onList: func(_ context.Context, filter models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
			seenOwnerID = filter.OwnerID
			return []*models.Initiative{{ID: "init-1"}}, &models.PaginationMeta{Total: 1}, nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	out, _, err := newReadSvc(repo, userRepo).ListForUser(context.Background(), "alice", models.InitiativeFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 initiative, got %d", len(out))
	}
	if seenOwnerID != "owner-uuid" {
		t.Errorf("filter.OwnerID = %q, want owner-uuid", seenOwnerID)
	}
}

func TestListForUser_UnknownUserReturnsEmpty(t *testing.T) {
	userRepo := &mockUserRepository{err: domain.ErrUserNotFound}

	out, meta, err := newReadSvc(&listInitiativeRepo{}, userRepo).ListForUser(
		context.Background(), "ghost", models.InitiativeFilter{Limit: 30, Offset: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty list, got %d", len(out))
	}
	if meta.Limit != 30 || meta.Offset != 10 {
		t.Errorf("meta = %+v, want Limit 30 / Offset 10", meta)
	}
}

func TestListForUser_UserLookupError(t *testing.T) {
	userRepo := &mockUserRepository{err: errors.New("db down")}
	_, _, err := newReadSvc(&listInitiativeRepo{}, userRepo).ListForUser(
		context.Background(), "alice", models.InitiativeFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetForUser ---

func TestGetForUser_BySlug_Owned(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetBySlug: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", OwnerID: "owner-uuid", Name: "Draft"}, nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	init, err := newReadSvc(repo, userRepo).GetForUser(context.Background(), "draft-slug", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if init.Name != "Draft" {
		t.Errorf("name = %q, want Draft", init.Name)
	}
}

func TestGetForUser_ByUUID_Owned(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, gotID string) (*models.Initiative, error) {
			if gotID != id {
				t.Errorf("GetByID id = %q, want %q", gotID, id)
			}
			return &models.Initiative{ID: id, OwnerID: "owner-uuid"}, nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	init, err := newReadSvc(repo, userRepo).GetForUser(context.Background(), id, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if init.ID != id {
		t.Errorf("id = %q, want %q", init.ID, id)
	}
}

func TestGetForUser_NotOwnedReturnsNotFound(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetBySlug: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", OwnerID: "someone-else"}, nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	_, err := newReadSvc(repo, userRepo).GetForUser(context.Background(), "draft-slug", "alice")
	if !errors.Is(err, domain.ErrInitiativeNotFound) {
		t.Errorf("expected ErrInitiativeNotFound, got %v", err)
	}
}

func TestGetForUser_UnknownCaller(t *testing.T) {
	userRepo := &mockUserRepository{err: domain.ErrUserNotFound}
	_, err := newReadSvc(&listInitiativeRepo{}, userRepo).GetForUser(context.Background(), "slug", "ghost")
	if !errors.Is(err, domain.ErrInitiativeNotFound) {
		t.Errorf("expected ErrInitiativeNotFound, got %v", err)
	}
}

func TestGetForUser_RepoError(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetBySlug: func(_ context.Context, _ string) (*models.Initiative, error) {
			return nil, errors.New("db down")
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}
	_, err := newReadSvc(repo, userRepo).GetForUser(context.Background(), "slug", "alice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ResolveOwnedInitiativeID ---

func TestResolveOwnedInitiativeID_BySlug_Owned(t *testing.T) {
	repo := &listInitiativeRepo{
		onResolveSlug: func(_ context.Context, _ string) (string, error) {
			return "init-1", nil
		},
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", OwnerID: "owner-uuid"}, nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	id, err := newReadSvc(repo, userRepo).ResolveOwnedInitiativeID(context.Background(), "draft-slug", "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "init-1" {
		t.Errorf("id = %q, want init-1", id)
	}
}

func TestResolveOwnedInitiativeID_ByUUID_Owned(t *testing.T) {
	const uuid = "22222222-2222-2222-2222-222222222222"
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: uuid, OwnerID: "owner-uuid"}, nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	id, err := newReadSvc(repo, userRepo).ResolveOwnedInitiativeID(context.Background(), uuid, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != uuid {
		t.Errorf("id = %q, want %q", id, uuid)
	}
}

func TestResolveOwnedInitiativeID_NotOwned(t *testing.T) {
	repo := &listInitiativeRepo{
		onResolveSlug: func(_ context.Context, _ string) (string, error) {
			return "init-1", nil
		},
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", OwnerID: "someone-else"}, nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	_, err := newReadSvc(repo, userRepo).ResolveOwnedInitiativeID(context.Background(), "draft-slug", "alice")
	if !errors.Is(err, domain.ErrInitiativeNotFound) {
		t.Errorf("expected ErrInitiativeNotFound, got %v", err)
	}
}

func TestResolveOwnedInitiativeID_UnknownCaller(t *testing.T) {
	userRepo := &mockUserRepository{err: domain.ErrUserNotFound}
	_, err := newReadSvc(&listInitiativeRepo{}, userRepo).ResolveOwnedInitiativeID(context.Background(), "slug", "ghost")
	if !errors.Is(err, domain.ErrInitiativeNotFound) {
		t.Errorf("expected ErrInitiativeNotFound, got %v", err)
	}
}

func TestResolveOwnedInitiativeID_SlugResolveError(t *testing.T) {
	repo := &listInitiativeRepo{
		onResolveSlug: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("not found")
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}
	_, err := newReadSvc(repo, userRepo).ResolveOwnedInitiativeID(context.Background(), "slug", "alice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Delete ---

func TestInitiativeService_Delete_Owned(t *testing.T) {
	deleted := false
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", OwnerID: "owner-uuid"}, nil
		},
		onDelete: func(_ context.Context, _ string) error {
			deleted = true
			return nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	if err := newReadSvc(repo, userRepo).Delete(context.Background(), "init-1", "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected repo.Delete to be called")
	}
}

func TestInitiativeService_Delete_NotOwned(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", OwnerID: "someone-else"}, nil
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	err := newReadSvc(repo, userRepo).Delete(context.Background(), "init-1", "alice")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestInitiativeService_Delete_UnknownCaller(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", OwnerID: "owner-uuid"}, nil
		},
	}
	userRepo := &mockUserRepository{err: domain.ErrUserNotFound}

	err := newReadSvc(repo, userRepo).Delete(context.Background(), "init-1", "ghost")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestInitiativeService_Delete_GetError(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return nil, errors.New("db down")
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	err := newReadSvc(repo, userRepo).Delete(context.Background(), "init-1", "alice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestInitiativeService_Delete_RepoDeleteError(t *testing.T) {
	repo := &listInitiativeRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Initiative, error) {
			return &models.Initiative{ID: "init-1", OwnerID: "owner-uuid"}, nil
		},
		onDelete: func(_ context.Context, _ string) error {
			return errors.New("delete failed")
		},
	}
	userRepo := &mockUserRepository{user: &models.User{ID: "owner-uuid"}}

	err := newReadSvc(repo, userRepo).Delete(context.Background(), "init-1", "alice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
