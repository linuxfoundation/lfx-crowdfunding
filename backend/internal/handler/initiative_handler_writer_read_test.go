// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/fga"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// writerResolver is a fixed-answer EntityRoleResolver.
type writerResolver struct {
	ok  bool
	err error
}

func (r writerResolver) CanManage(_ context.Context, _ models.AttributionType, _, _ string) (bool, error) {
	return r.ok, r.err
}

const writerInitiativeID = "44444444-4444-4444-4444-444444444444"

// writerHandler builds a handler over a non-published initiative attributed to an
// org, with the caller being a non-creator whose writer access is res.
func writerHandler(status models.InitiativeStatus, res fga.EntityRoleResolver) *InitiativeHandler {
	repo := &stubRepoForGetForUser{initiative: &models.Initiative{
		ID: writerInitiativeID, Slug: "org-fund", OwnerID: "creator", Status: status,
		Attribution:   models.Attribution{Type: models.AttributionOrganization, EntityUID: "org-1"},
		Beneficiaries: []models.Beneficiary{{ID: "b1", Email: "ben@example.com"}},
	}}
	userRepo := &stubUserRepoForListForUser{user: &models.User{ID: "writer-id", Username: "writer"}}
	svc := service.NewInitiativeService(repo, userRepo, &txnLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	svc.SetEntityRoleResolver(res)
	return NewInitiativeHandler(svc, nil, slog.Default())
}

func writerGet(h func(http.ResponseWriter, *http.Request), suffix string) *httptest.ResponseRecorder {
	return writerGetQuery(h, suffix, "")
}

func writerGetQuery(h func(http.ResponseWriter, *http.Request), suffix, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/crowdfunding/initiatives/"+writerInitiativeID+suffix+query, nil)
	req = withURLParam(req, "id", writerInitiativeID)
	req = withPrincipal(req, &models.Principal{Username: "writer"})
	w := httptest.NewRecorder()
	h(w, req)
	return w
}

const manage = "?view=manage"

func TestGetByID_Manage_WriterSeesDraftWithContacts(t *testing.T) {
	w := writerGetQuery(writerHandler(models.StatusSubmitted, writerResolver{ok: true}).GetByID, "", manage)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", got)
	}
	if got := w.Header().Get("Vary"); got != "Authorization" {
		t.Errorf("Vary = %q, want Authorization", got)
	}
	if !strings.Contains(w.Body.String(), "ben@example.com") {
		t.Errorf("writer should see beneficiary contacts, body: %s", w.Body.String())
	}
}

func TestGetByID_Manage_NonWriterIs404_EvenWhenPublished(t *testing.T) {
	for _, st := range []models.InitiativeStatus{models.StatusSubmitted, models.StatusPublished} {
		w := writerGetQuery(writerHandler(st, writerResolver{}).GetByID, "", manage)
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", st, w.Code)
		}
	}
}

func TestGetByID_Manage_OutageIs503(t *testing.T) {
	outage := writerResolver{err: fmt.Errorf("nats: %w", domain.ErrUpstreamUnavailable)}
	w := writerGetQuery(writerHandler(models.StatusSubmitted, outage).GetByID, "", manage)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (never a false 404), got %d", w.Code)
	}
}

func TestGetByID_Manage_NoPrincipalIs401(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/crowdfunding/initiatives/"+writerInitiativeID+manage, nil)
	req = withURLParam(req, "id", writerInitiativeID)
	w := httptest.NewRecorder()
	writerHandler(models.StatusPublished, writerResolver{ok: true}).GetByID(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// countingResolver fails the test if consulted.
type countingResolver struct{ calls int }

func (r *countingResolver) CanManage(_ context.Context, _ models.AttributionType, _, _ string) (bool, error) {
	r.calls++
	return true, nil
}

func TestGetByID_Plain_NeverCallsResolver(t *testing.T) {
	res := &countingResolver{}
	h := writerHandler(models.StatusPublished, res)
	w := writerGet(h.GetByID, "")
	if w.Code != http.StatusOK || res.calls != 0 {
		t.Fatalf("code=%d resolver calls=%d, want 200 and 0", w.Code, res.calls)
	}
	if strings.Contains(w.Body.String(), "ben@example.com") {
		t.Error("plain read must be the public projection")
	}
	if w = writerGet(writerHandler(models.StatusSubmitted, writerResolver{ok: true}).GetByID, ""); w.Code != http.StatusNotFound {
		t.Errorf("plain read of a draft by a writer: expected 404, got %d", w.Code)
	}
}

func TestGetTransactions_Writer(t *testing.T) {
	w := writerGet(writerHandler(models.StatusSubmitted, writerResolver{ok: true}).GetTransactions, "/transactions")
	if w.Code != http.StatusOK {
		t.Fatalf("writer: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "private, max-age=60" {
		t.Errorf("Cache-Control = %q, want private, max-age=60", got)
	}

	w = writerGet(writerHandler(models.StatusSubmitted, writerResolver{}).GetTransactions, "/transactions")
	if w.Code != http.StatusNotFound {
		t.Errorf("non-writer: expected 404, got %d", w.Code)
	}

	outage := writerResolver{err: fmt.Errorf("nats: %w", domain.ErrUpstreamUnavailable)}
	w = writerGet(writerHandler(models.StatusSubmitted, outage).GetTransactions, "/transactions")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("outage: expected 503, got %d", w.Code)
	}
}
