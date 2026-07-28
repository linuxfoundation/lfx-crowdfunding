// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// testOrganizationRepo is a configurable OrganizationRepository.
type testOrganizationRepo struct {
	onGetByID     func(context.Context, string) (*models.Organization, error)
	onListByOwner func(context.Context, string) ([]models.Organization, error)
	onCreate      func(context.Context, string, models.OrganizationCreateInput) (*models.Organization, error)
	onUpdate      func(context.Context, string, string, models.OrganizationUpdateInput) (*models.Organization, error)
	onDelete      func(context.Context, string, string) error
}

func (r *testOrganizationRepo) GetByID(ctx context.Context, id string) (*models.Organization, error) {
	if r.onGetByID != nil {
		return r.onGetByID(ctx, id)
	}
	return nil, nil
}
func (r *testOrganizationRepo) ListByOwner(ctx context.Context, ownerID string) ([]models.Organization, error) {
	if r.onListByOwner != nil {
		return r.onListByOwner(ctx, ownerID)
	}
	return nil, nil
}
func (r *testOrganizationRepo) Create(ctx context.Context, ownerID string, input models.OrganizationCreateInput) (*models.Organization, error) {
	if r.onCreate != nil {
		return r.onCreate(ctx, ownerID, input)
	}
	return &models.Organization{}, nil
}
func (r *testOrganizationRepo) Update(ctx context.Context, id, ownerID string, input models.OrganizationUpdateInput) (*models.Organization, error) {
	if r.onUpdate != nil {
		return r.onUpdate(ctx, id, ownerID, input)
	}
	return &models.Organization{}, nil
}
func (r *testOrganizationRepo) Delete(ctx context.Context, id, ownerID string) error {
	if r.onDelete != nil {
		return r.onDelete(ctx, id, ownerID)
	}
	return nil
}

// --- ListByOwner ---

func TestOrganizationService_ListByOwner(t *testing.T) {
	repo := &testOrganizationRepo{
		onListByOwner: func(_ context.Context, ownerID string) ([]models.Organization, error) {
			if ownerID != "user-uuid-1" {
				t.Errorf("ListByOwner ownerID = %q, want user-uuid-1", ownerID)
			}
			return []models.Organization{{ID: "org-1", Name: "Acme"}}, nil
		},
	}
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, username string) (*models.User, error) {
			if username != "alice" {
				t.Errorf("GetByUsername = %q, want alice", username)
			}
			return &models.User{ID: "user-uuid-1", Username: "alice"}, nil
		},
	}

	svc := NewOrganizationService(repo, userRepo)
	orgs, err := svc.ListByOwner(context.Background(), "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orgs) != 1 || orgs[0].ID != "org-1" {
		t.Fatalf("unexpected orgs: %+v", orgs)
	}
}

func TestOrganizationService_ListByOwner_UserLookupError(t *testing.T) {
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}

	svc := NewOrganizationService(&testOrganizationRepo{}, userRepo)
	_, err := svc.ListByOwner(context.Background(), "ghost")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestOrganizationService_ListByOwner_RepoError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &testOrganizationRepo{
		onListByOwner: func(_ context.Context, _ string) ([]models.Organization, error) {
			return nil, repoErr
		},
	}

	svc := NewOrganizationService(repo, &testUserRepo{})
	_, err := svc.ListByOwner(context.Background(), "alice")
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

// --- Create ---

func TestOrganizationService_Create(t *testing.T) {
	repo := &testOrganizationRepo{
		onCreate: func(_ context.Context, ownerID string, input models.OrganizationCreateInput) (*models.Organization, error) {
			if ownerID != "user-uuid-1" {
				t.Errorf("Create ownerID = %q, want user-uuid-1", ownerID)
			}
			return &models.Organization{ID: "org-new", Name: input.Name}, nil
		},
	}
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "user-uuid-1"}, nil
		},
	}

	svc := NewOrganizationService(repo, userRepo)
	org, err := svc.Create(context.Background(), "alice", models.OrganizationCreateInput{Name: "New Org"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.ID != "org-new" || org.Name != "New Org" {
		t.Errorf("unexpected org: %+v", org)
	}
}

func TestOrganizationService_Create_EmptyName(t *testing.T) {
	svc := NewOrganizationService(&testOrganizationRepo{}, &testUserRepo{})
	_, err := svc.Create(context.Background(), "alice", models.OrganizationCreateInput{Name: ""})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestOrganizationService_Create_UserLookupError(t *testing.T) {
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}

	svc := NewOrganizationService(&testOrganizationRepo{}, userRepo)
	_, err := svc.Create(context.Background(), "ghost", models.OrganizationCreateInput{Name: "New Org"})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestOrganizationService_Create_RepoError(t *testing.T) {
	repoErr := errors.New("insert failed")
	repo := &testOrganizationRepo{
		onCreate: func(_ context.Context, _ string, _ models.OrganizationCreateInput) (*models.Organization, error) {
			return nil, repoErr
		},
	}

	svc := NewOrganizationService(repo, &testUserRepo{})
	_, err := svc.Create(context.Background(), "alice", models.OrganizationCreateInput{Name: "New Org"})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

// --- Update ---

func TestOrganizationService_Update(t *testing.T) {
	repo := &testOrganizationRepo{
		onUpdate: func(_ context.Context, id, ownerID string, input models.OrganizationUpdateInput) (*models.Organization, error) {
			if id != "org-1" {
				t.Errorf("Update id = %q, want org-1", id)
			}
			if ownerID != "user-uuid-1" {
				t.Errorf("Update ownerID = %q, want user-uuid-1", ownerID)
			}
			return &models.Organization{ID: id, Name: input.Name}, nil
		},
	}
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "user-uuid-1"}, nil
		},
	}

	svc := NewOrganizationService(repo, userRepo)
	org, err := svc.Update(context.Background(), "alice", "org-1", models.OrganizationUpdateInput{Name: "Renamed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.Name != "Renamed" {
		t.Errorf("org name = %q, want Renamed", org.Name)
	}
}

func TestOrganizationService_Update_EmptyName(t *testing.T) {
	svc := NewOrganizationService(&testOrganizationRepo{}, &testUserRepo{})
	_, err := svc.Update(context.Background(), "alice", "org-1", models.OrganizationUpdateInput{Name: ""})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestOrganizationService_Update_UserLookupError(t *testing.T) {
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}

	svc := NewOrganizationService(&testOrganizationRepo{}, userRepo)
	_, err := svc.Update(context.Background(), "ghost", "org-1", models.OrganizationUpdateInput{Name: "Renamed"})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestOrganizationService_Update_RepoError(t *testing.T) {
	repoErr := errors.New("update failed")
	repo := &testOrganizationRepo{
		onUpdate: func(_ context.Context, _, _ string, _ models.OrganizationUpdateInput) (*models.Organization, error) {
			return nil, repoErr
		},
	}

	svc := NewOrganizationService(repo, &testUserRepo{})
	_, err := svc.Update(context.Background(), "alice", "org-1", models.OrganizationUpdateInput{Name: "Renamed"})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

// --- Delete ---

func TestOrganizationService_Delete(t *testing.T) {
	var deletedID, deletedOwner string
	repo := &testOrganizationRepo{
		onDelete: func(_ context.Context, id, ownerID string) error {
			deletedID, deletedOwner = id, ownerID
			return nil
		},
	}
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "user-uuid-1"}, nil
		},
	}

	svc := NewOrganizationService(repo, userRepo)
	if err := svc.Delete(context.Background(), "alice", "org-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "org-1" || deletedOwner != "user-uuid-1" {
		t.Errorf("Delete called with id=%q owner=%q, want org-1/user-uuid-1", deletedID, deletedOwner)
	}
}

func TestOrganizationService_Delete_UserLookupError(t *testing.T) {
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}

	svc := NewOrganizationService(&testOrganizationRepo{}, userRepo)
	err := svc.Delete(context.Background(), "ghost", "org-1")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestOrganizationService_Delete_RepoError(t *testing.T) {
	repoErr := errors.New("delete failed")
	repo := &testOrganizationRepo{
		onDelete: func(_ context.Context, _, _ string) error {
			return repoErr
		},
	}

	svc := NewOrganizationService(repo, &testUserRepo{})
	err := svc.Delete(context.Background(), "alice", "org-1")
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}
