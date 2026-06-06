// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

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

// testUserInfoFetcher is a configurable UserInfoFetcher stub.
type testUserInfoFetcher struct {
	info *auth.UserInfo
	err  error
}

func (f *testUserInfoFetcher) FetchUserInfo(_ context.Context, _ string) (*auth.UserInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.info != nil {
		return f.info, nil
	}
	return &auth.UserInfo{}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// newSyncProfileRequest builds a PATCH /v1/me request with the principal in
// context and a fake Bearer token in the Authorization header.
func newSyncProfileRequest(principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodPatch, "/v1/me", nil)
	r.Header.Set("Authorization", "Bearer fake-test-token")
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
	fetcher := &testUserInfoFetcher{info: &auth.UserInfo{
		Sub:        "auth0|abc123",
		Username:   "jdoe",
		SSOEmail:   "jdoe@example.com",
		Name:       "John Doe",
		GivenName:  "John",
		FamilyName: "Doe",
		Picture:    "https://example.com/pic.jpg",
	}}
	h := NewUserHandler(repo, fetcher)

	principal := &models.Principal{
		UserID:   "auth0|abc123",
		Username: "jdoe",
		Scope:    auth.ScopeMe,
	}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// Verify profile fields came from UserInfo (not principal claims).
	got := repo.lastUpserted
	if got == nil {
		t.Fatal("Upsert was not called")
	}
	if got.Username != "jdoe" {
		t.Errorf("Username = %q, want %q", got.Username, "jdoe")
	}
	if got.LegacyUserID != principal.UserID {
		t.Errorf("LegacyUserID = %q, want %q", got.LegacyUserID, principal.UserID)
	}
	if got.Email != fetcher.info.SSOEmail {
		t.Errorf("Email = %q, want %q", got.Email, fetcher.info.SSOEmail)
	}
	if got.Name != fetcher.info.Name {
		t.Errorf("Name = %q, want %q", got.Name, fetcher.info.Name)
	}
	if got.GivenName != fetcher.info.GivenName {
		t.Errorf("GivenName = %q, want %q", got.GivenName, fetcher.info.GivenName)
	}
	if got.FamilyName != fetcher.info.FamilyName {
		t.Errorf("FamilyName = %q, want %q", got.FamilyName, fetcher.info.FamilyName)
	}
	if got.AvatarURL != fetcher.info.Picture {
		t.Errorf("AvatarURL = %q, want %q", got.AvatarURL, fetcher.info.Picture)
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
	h := NewUserHandler(repo, &testUserInfoFetcher{})

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
	h := NewUserHandler(repo, &testUserInfoFetcher{})

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

func TestSyncProfile_UserInfoUnknownError_Returns500(t *testing.T) {
	repo := &testUserRepo{}
	// A generic (non-sentinel) error from the fetcher should surface as 500.
	fetcher := &testUserInfoFetcher{err: errors.New("unexpected decode failure")}
	h := NewUserHandler(repo, fetcher)

	principal := &models.Principal{
		UserID:   "auth0|abc",
		Username: "jdoe",
		Scope:    auth.ScopeMe,
	}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if repo.lastUpserted != nil {
		t.Error("Upsert must not be called when UserInfo fetch fails")
	}
}

func TestSyncProfile_UserInfoTokenRejected_Returns401(t *testing.T) {
	// Auth0 UserInfo rejected the token (4xx) — handler must return 401 so the
	// client knows to re-authenticate rather than retrying.
	repo := &testUserRepo{}
	fetcher := &testUserInfoFetcher{err: fmt.Errorf("%w: HTTP 401", auth.ErrUserInfoTokenRejected)}
	h := NewUserHandler(repo, fetcher)

	principal := &models.Principal{UserID: "auth0|abc", Username: "jdoe", Scope: auth.ScopeMe}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestSyncProfile_UserInfoUnavailable_Returns503(t *testing.T) {
	// Auth0 UserInfo returned 5xx or was unreachable — handler must return 503.
	repo := &testUserRepo{}
	fetcher := &testUserInfoFetcher{err: fmt.Errorf("%w: HTTP 503", auth.ErrUserInfoUnavailable)}
	h := NewUserHandler(repo, fetcher)

	principal := &models.Principal{UserID: "auth0|abc", Username: "jdoe", Scope: auth.ScopeMe}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestSyncProfile_SubMismatch_Returns401(t *testing.T) {
	repo := &testUserRepo{}
	// UserInfo.Sub differs from principal.UserID — token substitution attempt.
	fetcher := &testUserInfoFetcher{info: &auth.UserInfo{
		Sub:      "auth0|different-user",
		Username: "otheruser",
		Email:    "other@example.com",
	}}
	h := NewUserHandler(repo, fetcher)

	principal := &models.Principal{
		UserID:   "auth0|abc123",
		Username: "jdoe",
		Scope:    auth.ScopeMe,
	}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if repo.lastUpserted != nil {
		t.Error("Upsert must not be called when sub does not match principal")
	}
}

func TestSyncProfile_EmptySub_Returns401(t *testing.T) {
	// A UserInfo response with a blank sub is malformed and must be rejected,
	// even if the rest of the claims look valid.
	repo := &testUserRepo{}
	fetcher := &testUserInfoFetcher{info: &auth.UserInfo{
		Sub:      "", // no sub in response
		Username: "jdoe",
		Email:    "jdoe@example.com",
	}}
	h := NewUserHandler(repo, fetcher)

	principal := &models.Principal{UserID: "auth0|abc123", Username: "jdoe", Scope: auth.ScopeMe}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if repo.lastUpserted != nil {
		t.Error("Upsert must not be called when sub is empty")
	}
}

func TestSyncProfile_MissingAuthHeader_Returns401(t *testing.T) {
	repo := &testUserRepo{}
	h := NewUserHandler(repo, &testUserInfoFetcher{})

	// Principal is present but the Authorization header was stripped (e.g. by
	// a proxy that only forwards the principal context, not the raw token).
	r := httptest.NewRequest(http.MethodPatch, "/v1/me", nil)
	// deliberately omit Authorization header
	principal := &models.Principal{UserID: "auth0|abc", Username: "jdoe", Scope: auth.ScopeMe}
	r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))

	w := httptest.NewRecorder()
	h.SyncProfile(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if repo.lastUpserted != nil {
		t.Error("Upsert must not be called when Authorization header is absent")
	}
}

func TestSyncProfile_RepoError_Returns500(t *testing.T) {
	repo := &testUserRepo{upsertErr: errors.New("db unavailable")}
	fetcher := &testUserInfoFetcher{info: &auth.UserInfo{Sub: "auth0|abc", Username: "jdoe", Email: "jdoe@example.com"}}
	h := NewUserHandler(repo, fetcher)

	principal := &models.Principal{
		UserID:   "auth0|abc",
		Username: "jdoe",
		Scope:    auth.ScopeMe,
	}

	w := httptest.NewRecorder()
	h.SyncProfile(w, newSyncProfileRequest(principal))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
