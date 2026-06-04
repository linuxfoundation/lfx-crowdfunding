// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// ── stub ─────────────────────────────────────────────────────────────────────

// testUserRepo is a configurable UserRepository stub for user handler tests.
type testUserRepo struct {
	upsertResult *models.User
	upsertErr    error
	lastUpserted *models.User
}

func (r *testUserRepo) GetByUsername(_ context.Context, _ string) (*models.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *testUserRepo) GetByID(_ context.Context, _ string) (*models.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *testUserRepo) Upsert(_ context.Context, u *models.User) (*models.User, error) {
	r.lastUpserted = u
	if r.upsertErr != nil {
		return nil, r.upsertErr
	}
	if r.upsertResult != nil {
		return r.upsertResult, nil
	}
	return u, nil
}
func (r *testUserRepo) UpdateStripeInfo(_ context.Context, _, _, _ string) error   { return nil }
func (r *testUserRepo) ClearStripePaymentMethod(_ context.Context, _ string) error { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

func newSyncProfileRequest(principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodPatch, "/v1/me", nil)
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestSyncProfile_Success(t *testing.T) {
	want := &models.User{
		ID:       "uuid-1",
		Username: "jdoe",
		Email:    "jdoe@example.com",
		Name:     "John Doe",
	}
	repo := &testUserRepo{upsertResult: want}
	h := NewUserHandler(repo)

	principal := &models.Principal{
		UserID:     "auth0|abc123",
		Username:   "jdoe",
		Email:      "jdoe@example.com",
		Name:       "John Doe",
		GivenName:  "John",
		FamilyName: "Doe",
		Picture:    "https://example.com/pic.jpg",
	}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// Verify all principal fields were forwarded to the repository.
	got := repo.lastUpserted
	if got == nil {
		t.Fatal("Upsert was not called")
	}
	if got.Username != principal.Username {
		t.Errorf("Username = %q, want %q", got.Username, principal.Username)
	}
	// LegacyUserID must NOT be populated from the JWT — it is reserved for
	// DynamoDB-migrated users and must remain empty for post-migration users.
	if got.LegacyUserID != "" {
		t.Errorf("LegacyUserID = %q, want empty", got.LegacyUserID)
	}
	if got.Email != principal.Email {
		t.Errorf("Email = %q, want %q", got.Email, principal.Email)
	}
	if got.Name != principal.Name {
		t.Errorf("Name = %q, want %q", got.Name, principal.Name)
	}
	if got.GivenName != principal.GivenName {
		t.Errorf("GivenName = %q, want %q", got.GivenName, principal.GivenName)
	}
	if got.FamilyName != principal.FamilyName {
		t.Errorf("FamilyName = %q, want %q", got.FamilyName, principal.FamilyName)
	}
	if got.AvatarURL != principal.Picture {
		t.Errorf("AvatarURL = %q, want %q", got.AvatarURL, principal.Picture)
	}

	// Response body should be the upserted user JSON.
	var body models.User
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID != want.ID {
		t.Errorf("response ID = %q, want %q", body.ID, want.ID)
	}
}

func TestSyncProfile_NoPrincipal_Returns401(t *testing.T) {
	repo := &testUserRepo{}
	h := NewUserHandler(repo)

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if repo.lastUpserted != nil {
		t.Error("Upsert must not be called when principal is absent")
	}
}

func TestSyncProfile_EmptyUsername_Returns401(t *testing.T) {
	repo := &testUserRepo{}
	h := NewUserHandler(repo)

	// A principal with no username (e.g. non-SSO token) must be rejected.
	principal := &models.Principal{UserID: "auth0|abc"}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if repo.lastUpserted != nil {
		t.Error("Upsert must not be called when username is empty")
	}
}

func TestSyncProfile_RepoError_Returns500(t *testing.T) {
	repo := &testUserRepo{upsertErr: errors.New("db unavailable")}
	h := NewUserHandler(repo)

	principal := &models.Principal{
		UserID:   "auth0|abc",
		Username: "jdoe",
	}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
