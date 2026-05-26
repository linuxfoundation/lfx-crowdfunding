// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePaginationParams_ValidLimitOffset(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?limit=20&offset=40", nil)
	w := httptest.NewRecorder()

	limit, offset, ok := parsePaginationParams(w, r)

	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if limit != 20 {
		t.Errorf("limit = %d, want 20", limit)
	}
	if offset != 40 {
		t.Errorf("offset = %d, want 40", offset)
	}
}

func TestParsePaginationParams_AbsentParams_ZeroDefaults(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	limit, offset, ok := parsePaginationParams(w, r)

	if !ok {
		t.Fatal("expected ok=true for absent params")
	}
	if limit != 0 || offset != 0 {
		t.Errorf("expected 0,0 for absent params, got %d,%d", limit, offset)
	}
}

func TestParsePaginationParams_InvalidLimit_Returns400(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?limit=abc", nil)
	w := httptest.NewRecorder()

	_, _, ok := parsePaginationParams(w, r)

	if ok {
		t.Fatal("expected ok=false for non-integer limit")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestParsePaginationParams_InvalidOffset_Returns400(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?offset=xyz", nil)
	w := httptest.NewRecorder()

	_, _, ok := parsePaginationParams(w, r)

	if ok {
		t.Fatal("expected ok=false for non-integer offset")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestParsePaginationParams_LimitFloatRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/?limit=1.5", nil)
	w := httptest.NewRecorder()

	_, _, ok := parsePaginationParams(w, r)

	if ok {
		t.Fatal("expected ok=false for float limit")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestParsePaginationParams_NegativeValues_Accepted(t *testing.T) {
	// Negative values are syntactically valid integers; clamping is the handler's job.
	r := httptest.NewRequest(http.MethodGet, "/?limit=-1&offset=-5", nil)
	w := httptest.NewRecorder()

	limit, offset, ok := parsePaginationParams(w, r)

	if !ok {
		t.Fatal("expected ok=true for negative integers")
	}
	if limit != -1 || offset != -5 {
		t.Errorf("expected -1,-5, got %d,%d", limit, offset)
	}
}
