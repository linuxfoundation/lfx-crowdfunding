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

// ─── mocks ────────────────────────────────────────────────────────────────────

type mockAnnouncementRepo struct {
	announcements []models.Announcement
	created       *models.Announcement
	updated       *models.Announcement
	listErr       error
	createErr     error
	updateErr     error
	deleteErr     error
}

func (m *mockAnnouncementRepo) List(_ context.Context, _ string, _ models.AnnouncementFilter) ([]models.Announcement, *models.PaginationMeta, error) {
	if m.listErr != nil {
		return nil, nil, m.listErr
	}
	meta := &models.PaginationMeta{Total: len(m.announcements), Limit: 20, Offset: 0}
	return m.announcements, meta, nil
}
func (m *mockAnnouncementRepo) Create(_ context.Context, a *models.Announcement) (*models.Announcement, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	a.ID = "new-id"
	m.created = a
	return a, nil
}
func (m *mockAnnouncementRepo) Update(_ context.Context, id, _ string, input models.AnnouncementUpdateInput) (*models.Announcement, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	result := &models.Announcement{ID: id, Title: input.Title, Description: input.Description}
	m.updated = result
	return result, nil
}
func (m *mockAnnouncementRepo) Delete(_ context.Context, _, _ string) error {
	return m.deleteErr
}

// mockInitiativeRepoForAnn is a minimal mock satisfying domain.InitiativeRepository.
type mockInitiativeRepoForAnn struct {
	initiative *models.Initiative
	err        error
}

func (m *mockInitiativeRepoForAnn) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return m.initiative, m.err
}
func (m *mockInitiativeRepoForAnn) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return m.initiative, m.err
}
func (m *mockInitiativeRepoForAnn) GetIDBySlug(_ context.Context, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.initiative != nil {
		return m.initiative.ID, nil
	}
	return "", nil
}
func (m *mockInitiativeRepoForAnn) ResolveSlug(_ context.Context, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.initiative != nil {
		return m.initiative.ID, nil
	}
	return "", nil
}
func (m *mockInitiativeRepoForAnn) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (m *mockInitiativeRepoForAnn) Create(_ context.Context, i *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	return i, nil
}
func (m *mockInitiativeRepoForAnn) Update(_ context.Context, i *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	return i, nil
}
func (m *mockInitiativeRepoForAnn) Delete(_ context.Context, _ string) error { return nil }
func (m *mockInitiativeRepoForAnn) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return nil, nil
}
func (m *mockInitiativeRepoForAnn) GetUsersByLegacyIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return nil, nil
}
func (m *mockInitiativeRepoForAnn) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	return nil, nil
}
func (m *mockInitiativeRepoForAnn) GetOwnerInfoBySlug(_ context.Context, _ string) (models.OwnerInfo, error) {
	return models.OwnerInfo{}, nil
}
func (m *mockInitiativeRepoForAnn) ListPublished(_ context.Context) ([]models.InitiativeSummary, error) {
	return nil, nil
}
func (m *mockInitiativeRepoForAnn) UpdateStripeProductID(_ context.Context, _, _ string) error {
	return nil
}

// mockUserRepoForAnn is a minimal mock satisfying domain.UserRepository.
type mockUserRepoForAnn struct {
	user *models.User
	err  error
}

func (m *mockUserRepoForAnn) GetByUsername(_ context.Context, username string) (*models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.user != nil {
		return m.user, nil
	}
	return &models.User{ID: username, Username: username}, nil
}
func (m *mockUserRepoForAnn) GetByID(_ context.Context, id string) (*models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &models.User{ID: id}, nil
}
func (m *mockUserRepoForAnn) Upsert(_ context.Context, u *models.User) (*models.User, error) {
	return u, nil
}
func (m *mockUserRepoForAnn) UpdateStripeInfo(_ context.Context, _, _, _ string) error   { return nil }
func (m *mockUserRepoForAnn) ClearStripePaymentMethod(_ context.Context, _ string) error { return nil }

// ─── helpers ──────────────────────────────────────────────────────────────────

func publishedInitiative(ownerID string) *models.Initiative {
	return &models.Initiative{ID: "init-1", OwnerID: ownerID, Status: models.StatusPublished}
}

func newAnnouncementSvc(
	repo *mockAnnouncementRepo,
	initRepo *mockInitiativeRepoForAnn,
	userRepo *mockUserRepoForAnn,
) *AnnouncementService {
	return NewAnnouncementService(repo, initRepo, userRepo)
}

// ─── List ─────────────────────────────────────────────────────────────────────

func TestAnnouncementService_List_Success(t *testing.T) {
	want := []models.Announcement{{ID: "a1", Title: "Hello"}}
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{announcements: want},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative("owner-1")},
		&mockUserRepoForAnn{},
	)
	got, meta, err := svc.List(context.Background(), "init-1", models.AnnouncementFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a1" {
		t.Errorf("unexpected announcements: %v", got)
	}
	if meta == nil || meta.Total != 1 {
		t.Errorf("unexpected meta: %v", meta)
	}
}

func TestAnnouncementService_List_InitiativeNotFound(t *testing.T) {
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{},
		&mockInitiativeRepoForAnn{err: domain.ErrInitiativeNotFound},
		&mockUserRepoForAnn{},
	)
	_, _, err := svc.List(context.Background(), "bogus", models.AnnouncementFilter{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAnnouncementService_List_NonPublishedInitiativeHidden(t *testing.T) {
	for _, status := range []models.InitiativeStatus{
		models.StatusSubmitted, models.StatusPending, models.StatusDeclined, models.StatusHidden,
	} {
		t.Run(string(status), func(t *testing.T) {
			init := &models.Initiative{ID: "init-1", Status: status}
			svc := newAnnouncementSvc(
				&mockAnnouncementRepo{announcements: []models.Announcement{{ID: "a1"}}},
				&mockInitiativeRepoForAnn{initiative: init},
				&mockUserRepoForAnn{},
			)
			_, _, err := svc.List(context.Background(), "init-1", models.AnnouncementFilter{})
			if !errors.Is(err, domain.ErrInitiativeNotFound) {
				t.Errorf("status %q: expected ErrInitiativeNotFound, got %v", status, err)
			}
		})
	}
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestAnnouncementService_Create_Success(t *testing.T) {
	ownerID := "user-1"
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative(ownerID)},
		&mockUserRepoForAnn{user: &models.User{ID: ownerID, Username: "alice"}},
	)
	a, err := svc.Create(context.Background(), "init-1", "alice", models.AnnouncementCreateInput{
		Title:       "Hello",
		Description: "World",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Title != "Hello" {
		t.Errorf("unexpected title: %s", a.Title)
	}
}

func TestAnnouncementService_Create_EmptyTitle(t *testing.T) {
	svc := newAnnouncementSvc(&mockAnnouncementRepo{}, &mockInitiativeRepoForAnn{}, &mockUserRepoForAnn{})
	_, err := svc.Create(context.Background(), "init-1", "alice", models.AnnouncementCreateInput{
		Title:       "",
		Description: "World",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAnnouncementService_Create_TitleTooLong(t *testing.T) {
	svc := newAnnouncementSvc(&mockAnnouncementRepo{}, &mockInitiativeRepoForAnn{}, &mockUserRepoForAnn{})
	longTitle := string(make([]byte, 256))
	_, err := svc.Create(context.Background(), "init-1", "alice", models.AnnouncementCreateInput{
		Title:       longTitle,
		Description: "World",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAnnouncementService_Create_EmptyDescription(t *testing.T) {
	svc := newAnnouncementSvc(&mockAnnouncementRepo{}, &mockInitiativeRepoForAnn{}, &mockUserRepoForAnn{})
	_, err := svc.Create(context.Background(), "init-1", "alice", models.AnnouncementCreateInput{
		Title:       "Hello",
		Description: "",
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAnnouncementService_Create_NotOwner(t *testing.T) {
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative("owner-1")},
		// caller resolves to a different user ID
		&mockUserRepoForAnn{user: &models.User{ID: "other-user", Username: "bob"}},
	)
	_, err := svc.Create(context.Background(), "init-1", "bob", models.AnnouncementCreateInput{
		Title:       "Hello",
		Description: "World",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestAnnouncementService_Create_UnknownCaller(t *testing.T) {
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative("owner-1")},
		&mockUserRepoForAnn{err: domain.ErrUserNotFound},
	)
	_, err := svc.Create(context.Background(), "init-1", "ghost", models.AnnouncementCreateInput{
		Title:       "Hello",
		Description: "World",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// ─── Update ───────────────────────────────────────────────────────────────────

func TestAnnouncementService_Update_Success(t *testing.T) {
	ownerID := "user-1"
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative(ownerID)},
		&mockUserRepoForAnn{user: &models.User{ID: ownerID, Username: "alice"}},
	)
	got, err := svc.Update(context.Background(), "init-1", "ann-1", "alice", models.AnnouncementUpdateInput{
		Title:       "Updated",
		Description: "New body",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != "Updated" {
		t.Errorf("unexpected title: %s", got.Title)
	}
}

func TestAnnouncementService_Update_NotFound(t *testing.T) {
	ownerID := "user-1"
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{updateErr: domain.ErrAnnouncementNotFound},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative(ownerID)},
		&mockUserRepoForAnn{user: &models.User{ID: ownerID, Username: "alice"}},
	)
	_, err := svc.Update(context.Background(), "init-1", "missing", "alice", models.AnnouncementUpdateInput{
		Title:       "X",
		Description: "Y",
	})
	if !errors.Is(err, domain.ErrAnnouncementNotFound) {
		t.Errorf("expected ErrAnnouncementNotFound, got %v", err)
	}
}

func TestAnnouncementService_Update_ValidationErrors(t *testing.T) {
	svc := newAnnouncementSvc(&mockAnnouncementRepo{}, &mockInitiativeRepoForAnn{}, &mockUserRepoForAnn{})
	cases := []struct {
		name  string
		input models.AnnouncementUpdateInput
	}{
		{"empty title", models.AnnouncementUpdateInput{Title: "", Description: "ok"}},
		{"empty description", models.AnnouncementUpdateInput{Title: "ok", Description: ""}},
		{"title too long", models.AnnouncementUpdateInput{Title: string(make([]byte, 256)), Description: "ok"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Update(context.Background(), "init-1", "ann-1", "alice", tc.input)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

// ─── Delete ───────────────────────────────────────────────────────────────────

func TestAnnouncementService_Delete_Success(t *testing.T) {
	ownerID := "user-1"
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative(ownerID)},
		&mockUserRepoForAnn{user: &models.User{ID: ownerID, Username: "alice"}},
	)
	if err := svc.Delete(context.Background(), "init-1", "ann-1", "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnnouncementService_Delete_NotFound(t *testing.T) {
	ownerID := "user-1"
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{deleteErr: domain.ErrAnnouncementNotFound},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative(ownerID)},
		&mockUserRepoForAnn{user: &models.User{ID: ownerID, Username: "alice"}},
	)
	err := svc.Delete(context.Background(), "init-1", "missing", "alice")
	if !errors.Is(err, domain.ErrAnnouncementNotFound) {
		t.Errorf("expected ErrAnnouncementNotFound, got %v", err)
	}
}

func TestAnnouncementService_Delete_Forbidden(t *testing.T) {
	svc := newAnnouncementSvc(
		&mockAnnouncementRepo{},
		&mockInitiativeRepoForAnn{initiative: publishedInitiative("owner-1")},
		&mockUserRepoForAnn{user: &models.User{ID: "other", Username: "bob"}},
	)
	err := svc.Delete(context.Background(), "init-1", "ann-1", "bob")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}
