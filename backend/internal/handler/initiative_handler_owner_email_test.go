// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

// ── GetOwnerEmail ─────────────────────────────────────────────────────────────

func TestGetOwnerEmail_ReturnsEmail(t *testing.T) {
	repo := &initiativeRepo{ownerEmail: "owner@example.com"}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/my-slug/owner-email", nil)
	req = withURLParam(req, "slug", "my-slug")
	w := httptest.NewRecorder()
	h.GetOwnerEmail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := body["email"]; got != "owner@example.com" {
		t.Errorf("expected email %q, got %q", "owner@example.com", got)
	}
}

func TestGetOwnerEmail_SetsNoCacheHeaders(t *testing.T) {
	repo := &initiativeRepo{ownerEmail: "owner@example.com"}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/my-slug/owner-email", nil)
	req = withURLParam(req, "slug", "my-slug")
	w := httptest.NewRecorder()
	h.GetOwnerEmail(w, req)

	if got := w.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("expected Cache-Control %q, got %q", "private, no-store", got)
	}
	if got := w.Header().Get("Vary"); got != "Authorization" {
		t.Errorf("expected Vary %q, got %q", "Authorization", got)
	}
}

func TestGetOwnerEmail_NotFound_Returns404(t *testing.T) {
	repo := &initiativeRepo{ownerEmailErr: domain.ErrInitiativeNotFound}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/missing-slug/owner-email", nil)
	req = withURLParam(req, "slug", "missing-slug")
	w := httptest.NewRecorder()
	h.GetOwnerEmail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetOwnerEmail_UnexpectedError_Returns500(t *testing.T) {
	repo := &initiativeRepo{ownerEmailErr: errors.New("db timeout")}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/some-slug/owner-email", nil)
	req = withURLParam(req, "slug", "some-slug")
	w := httptest.NewRecorder()
	h.GetOwnerEmail(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetOwnerEmail_NullEmail_Returns400(t *testing.T) {
	repo := &initiativeRepo{ownerEmailErr: domain.ErrProfileNotSynced}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/some-slug/owner-email", nil)
	req = withURLParam(req, "slug", "some-slug")
	w := httptest.NewRecorder()
	h.GetOwnerEmail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
