// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// txnLedgerClient is an apprLedgerClient that returns a non-nil empty transaction
// list, so the owner-transactions happy path can reach Ledger without panicking.
type txnLedgerClient struct{ apprLedgerClient }

func (c *txnLedgerClient) GetTransactions(_ context.Context, _ clients.TransactionFilter) (*models.TransactionList, error) {
	return &models.TransactionList{Data: []models.Transaction{}}, nil
}

// stubRepoForGetForUser implements domain.InitiativeRepository for GetForUser tests.
// GetBySlug returns the configured initiative regardless of status, mirroring the
// real repo (status gating lives in the handler, not the repo).
type stubRepoForGetForUser struct {
	initiative *models.Initiative
	err        error
}

func (r *stubRepoForGetForUser) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.initiative, nil
}
func (r *stubRepoForGetForUser) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.initiative, nil
}
func (r *stubRepoForGetForUser) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *stubRepoForGetForUser) GetIDBySlug(_ context.Context, _ string) (string, error) {
	return "", domain.ErrInitiativeNotFound
}
func (r *stubRepoForGetForUser) ResolveSlug(_ context.Context, _ string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if r.initiative == nil {
		return "", domain.ErrInitiativeNotFound
	}
	return r.initiative.ID, nil
}
func (r *stubRepoForGetForUser) Create(_ context.Context, _ *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	return nil, nil
}
func (r *stubRepoForGetForUser) Update(_ context.Context, _ *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	return nil, nil
}
func (r *stubRepoForGetForUser) Delete(_ context.Context, _ string) error { return nil }
func (r *stubRepoForGetForUser) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return nil, nil
}
func (r *stubRepoForGetForUser) GetUsersByLegacyIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return nil, nil
}
func (r *stubRepoForGetForUser) GetOwnerInfoBySlug(_ context.Context, _ string) (models.OwnerInfo, error) {
	return models.OwnerInfo{}, nil
}
func (r *stubRepoForGetForUser) ListPublished(_ context.Context) ([]models.InitiativeSummary, error) {
	return nil, nil
}
func (r *stubRepoForGetForUser) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	return nil, nil
}
func (r *stubRepoForGetForUser) UpdateStripeProductID(_ context.Context, _, _ string) error {
	return nil
}

// getForUserRouter mounts only the GetForUser route on a fresh Chi router so
// chi.URLParam("id") resolves the slug from the path.
func getForUserRouter(h *InitiativeHandler) chi.Router {
	r := chi.NewRouter()
	r.Get("/v1/me/initiatives/{id}", h.GetForUser)
	return r
}

func getForUserReq(slug string, principal *models.Principal) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+slug, nil)
	if principal != nil {
		req = req.WithContext(auth.ContextWithPrincipal(req.Context(), principal))
	}
	return req
}

func TestGetForUser_OwnerSeesOwnDraft(t *testing.T) {
	ownerID := "owner-uuid-1"
	username := "owner"

	userRepo := &stubUserRepoForListForUser{user: &models.User{ID: ownerID, Username: username}}
	repo := &stubRepoForGetForUser{
		initiative: &models.Initiative{ID: "init-1", Slug: "general-fund-4", OwnerID: ownerID, Status: models.StatusSubmitted},
	}
	svc := service.NewInitiativeService(repo, userRepo, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	w := httptest.NewRecorder()
	getForUserRouter(h).ServeHTTP(w, getForUserReq("general-fund-4", &models.Principal{Username: username}))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body models.Initiative
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID != "init-1" {
		t.Errorf("expected init-1, got %s", body.ID)
	}
}

func TestGetForUser_NonOwnerGets404(t *testing.T) {
	userRepo := &stubUserRepoForListForUser{user: &models.User{ID: "someone-else", Username: "intruder"}}
	repo := &stubRepoForGetForUser{
		initiative: &models.Initiative{ID: "init-1", Slug: "general-fund-4", OwnerID: "owner-uuid-1", Status: models.StatusSubmitted},
	}
	svc := service.NewInitiativeService(repo, userRepo, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	w := httptest.NewRecorder()
	getForUserRouter(h).ServeHTTP(w, getForUserReq("general-fund-4", &models.Principal{Username: "intruder"}))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetForUser_NoPrincipalReturns401(t *testing.T) {
	svc := service.NewInitiativeService(&stubRepoForGetForUser{}, &stubUserRepoForListForUser{}, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	w := httptest.NewRecorder()
	getForUserRouter(h).ServeHTTP(w, getForUserReq("general-fund-4", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetForUser_NotFoundReturns404(t *testing.T) {
	userRepo := &stubUserRepoForListForUser{user: &models.User{ID: "owner-uuid-1", Username: "owner"}}
	repo := &stubRepoForGetForUser{err: domain.ErrInitiativeNotFound}
	svc := service.NewInitiativeService(repo, userRepo, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	w := httptest.NewRecorder()
	getForUserRouter(h).ServeHTTP(w, getForUserReq("missing-slug", &models.Principal{Username: "owner"}))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

// getForUserTxnRouter mounts the owner transactions route on a fresh Chi router.
func getForUserTxnRouter(h *InitiativeHandler) chi.Router {
	r := chi.NewRouter()
	r.Get("/v1/me/initiatives/{id}/transactions", h.GetTransactionsForUser)
	return r
}

func getForUserTxnReq(slug string, principal *models.Principal) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+slug+"/transactions", nil)
	if principal != nil {
		req = req.WithContext(auth.ContextWithPrincipal(req.Context(), principal))
	}
	return req
}

func TestGetTransactionsForUser_OwnerSeesOwnDraft(t *testing.T) {
	ownerID := "owner-uuid-1"
	username := "owner"

	userRepo := &stubUserRepoForListForUser{user: &models.User{ID: ownerID, Username: username}}
	repo := &stubRepoForGetForUser{
		initiative: &models.Initiative{ID: "init-1", Slug: "general-fund-4", OwnerID: ownerID, Status: models.StatusSubmitted},
	}
	svc := service.NewInitiativeService(repo, userRepo, &txnLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	w := httptest.NewRecorder()
	getForUserTxnRouter(h).ServeHTTP(w, getForUserTxnReq("general-fund-4", &models.Principal{Username: username}))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Identity-scoped responses must never be shared-cacheable.
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "private") {
		t.Errorf("expected private Cache-Control on owner transactions, got %q", cc)
	}
	if v := w.Header().Get("Vary"); v != "Authorization" {
		t.Errorf("expected Vary: Authorization, got %q", v)
	}
}

func TestGetTransactionsForUser_NonOwnerGets404(t *testing.T) {
	userRepo := &stubUserRepoForListForUser{user: &models.User{ID: "someone-else", Username: "intruder"}}
	repo := &stubRepoForGetForUser{
		initiative: &models.Initiative{ID: "init-1", Slug: "general-fund-4", OwnerID: "owner-uuid-1", Status: models.StatusSubmitted},
	}
	svc := service.NewInitiativeService(repo, userRepo, &txnLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	w := httptest.NewRecorder()
	getForUserTxnRouter(h).ServeHTTP(w, getForUserTxnReq("general-fund-4", &models.Principal{Username: "intruder"}))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-owner, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetTransactionsForUser_NoPrincipalReturns401(t *testing.T) {
	svc := service.NewInitiativeService(&stubRepoForGetForUser{}, &stubUserRepoForListForUser{}, &txnLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	w := httptest.NewRecorder()
	getForUserTxnRouter(h).ServeHTTP(w, getForUserTxnReq("general-fund-4", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetTransactionsForUser_NotFoundReturns404(t *testing.T) {
	userRepo := &stubUserRepoForListForUser{user: &models.User{ID: "owner-uuid-1", Username: "owner"}}
	repo := &stubRepoForGetForUser{err: domain.ErrInitiativeNotFound}
	svc := service.NewInitiativeService(repo, userRepo, &txnLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	w := httptest.NewRecorder()
	getForUserTxnRouter(h).ServeHTTP(w, getForUserTxnReq("missing-slug", &models.Principal{Username: "owner"}))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
