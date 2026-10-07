// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

func listByAttribution(t *testing.T, typ models.AttributionType, uid, query string) (*httptest.ResponseRecorder, *stubInitiativeRepoForListForUser) {
	t.Helper()
	repo := &stubInitiativeRepoForListForUser{}
	svc := service.NewInitiativeService(repo, &stubUserRepoForListForUser{}, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	r := chi.NewRouter()
	r.Get("/x/{uid}/initiatives", h.ListByAttribution(typ))
	req := httptest.NewRequest(http.MethodGet, "/x/"+uid+"/initiatives"+query, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, repo
}

func TestListByAttribution_ScopesFilterAndCanonicalizesProjectUID(t *testing.T) {
	w, repo := listByAttribution(t, models.AttributionProject, "6BA7B810-9DAD-11D1-80B4-00C04FD430C8", "?status=hidden")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	f := repo.capturedFilter
	if f.AttributedToType != models.AttributionProject || f.AttributedToUID != "6ba7b810-9dad-11d1-80b4-00c04fd430c8" {
		t.Errorf("filter not scoped/canonicalized: %+v", f)
	}
	if f.OwnerID != "" || len(f.Statuses) != 1 {
		t.Errorf("unexpected owner/status filter: %+v", f)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "private, no-store" {
		t.Errorf("Cache-Control = %q", cc)
	}
}

func TestListByAttribution_Org(t *testing.T) {
	w, repo := listByAttribution(t, models.AttributionOrganization, "0014100000Te0yvAAB", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if repo.capturedFilter.AttributedToType != models.AttributionOrganization || repo.capturedFilter.AttributedToUID != "0014100000Te0yvAAB" {
		t.Errorf("filter: %+v", repo.capturedFilter)
	}
}

func TestListByAttribution_MalformedUID_400(t *testing.T) {
	for _, tc := range []struct {
		typ models.AttributionType
		uid string
	}{
		{models.AttributionProject, "not-a-uuid"},
		{models.AttributionOrganization, "short"},
	} {
		w, repo := listByAttribution(t, tc.typ, tc.uid, "")
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s/%s: expected 400, got %d", tc.typ, tc.uid, w.Code)
		}
		if repo.capturedFilter.AttributedToUID != "" {
			t.Errorf("repo must not be queried on invalid uid")
		}
	}
}
