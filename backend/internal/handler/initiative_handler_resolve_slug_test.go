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
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

func TestResolveSlugToUID_Found_Returns200(t *testing.T) {
	initiativeID := "55555555-5555-5555-5555-555555555555"
	repo := &initiativeRepo{
		initiative: &models.Initiative{ID: initiativeID, Name: "Save The Bees"},
	}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/crowdfunding_initiatives/slug-to-uid/save-the-bees", nil)
	req = withURLParam(req, "slug", "save-the-bees")
	w := httptest.NewRecorder()
	h.ResolveSlugToUID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		UID string `json:"uid"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.UID != initiativeID {
		t.Errorf("expected uid %s, got %s", initiativeID, body.UID)
	}
}

func TestResolveSlugToUID_NotFound_Returns404(t *testing.T) {
	repo := &initiativeRepo{getErr: domain.ErrInitiativeNotFound}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/crowdfunding_initiatives/slug-to-uid/no-such-slug", nil)
	req = withURLParam(req, "slug", "no-such-slug")
	w := httptest.NewRecorder()
	h.ResolveSlugToUID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResolveSlugToUID_RepoError_Returns500(t *testing.T) {
	repo := &initiativeRepo{getErr: errors.New("boom")}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/crowdfunding_initiatives/slug-to-uid/save-the-bees", nil)
	req = withURLParam(req, "slug", "save-the-bees")
	w := httptest.NewRecorder()
	h.ResolveSlugToUID(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}
