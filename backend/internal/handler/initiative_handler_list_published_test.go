// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// ── ListPublished ─────────────────────────────────────────────────────────────

func TestListPublished_ReturnsData(t *testing.T) {
	repo := &initiativeRepo{
		listPublishedResult: []models.InitiativeSummary{
			{ID: "id-1", Name: "Alpha Project"},
			{ID: "id-2", Name: "Beta Fund"},
		},
	}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/published-list", nil)
	w := httptest.NewRecorder()
	h.ListPublished(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Data []models.InitiativeSummary `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(body.Data))
	}
	if body.Data[0].ID != "id-1" || body.Data[0].Name != "Alpha Project" {
		t.Errorf("unexpected first item: %+v", body.Data[0])
	}
	if body.Data[1].ID != "id-2" || body.Data[1].Name != "Beta Fund" {
		t.Errorf("unexpected second item: %+v", body.Data[1])
	}
}

func TestListPublished_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	repo := &initiativeRepo{listPublishedResult: nil}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/published-list", nil)
	w := httptest.NewRecorder()
	h.ListPublished(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body struct {
		Data []models.InitiativeSummary `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data == nil {
		t.Error("expected non-nil data array, got null")
	}
	if len(body.Data) != 0 {
		t.Errorf("expected empty array, got %d items", len(body.Data))
	}
}

func TestListPublished_SetsNoCacheHeaders(t *testing.T) {
	repo := &initiativeRepo{}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/published-list", nil)
	w := httptest.NewRecorder()
	h.ListPublished(w, req)

	if got := w.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("expected Cache-Control %q, got %q", "private, no-store", got)
	}
	if got := w.Header().Get("Vary"); got != "Authorization" {
		t.Errorf("expected Vary %q, got %q", "Authorization", got)
	}
}

func TestListPublished_DBError_Returns500WithCacheHeaders(t *testing.T) {
	repo := &initiativeRepo{listPublishedErr: errors.New("db connection reset")}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/published-list", nil)
	w := httptest.NewRecorder()
	h.ListPublished(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	// Cache headers must be set even on error responses (set before service call).
	if got := w.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("expected Cache-Control %q on error response, got %q", "private, no-store", got)
	}
	if got := w.Header().Get("Vary"); got != "Authorization" {
		t.Errorf("expected Vary %q on error response, got %q", "Authorization", got)
	}
}

func TestListPublished_ResponseContainsOnlyIDAndName(t *testing.T) {
	repo := &initiativeRepo{
		listPublishedResult: []models.InitiativeSummary{
			{ID: "abc-123", Name: "My Initiative"},
		},
	}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/published-list", nil)
	w := httptest.NewRecorder()
	h.ListPublished(w, req)

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("decode outer: %v", err)
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw["data"], &items); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item")
	}
	if _, ok := items[0]["id"]; !ok {
		t.Error("expected 'id' field in response item")
	}
	if _, ok := items[0]["name"]; !ok {
		t.Error("expected 'name' field in response item")
	}
	// Ensure no extra fields leak (e.g. status, description, etc.)
	if len(items[0]) != 2 {
		t.Errorf("expected exactly 2 fields (id, name), got %d: %v", len(items[0]), items[0])
	}
}
