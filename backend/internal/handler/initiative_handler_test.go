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

// ── stubs ─────────────────────────────────────────────────────────────────────

// initiativeRepo is a configurable InitiativeRepository stub.
type initiativeRepo struct {
	initiative          *models.Initiative
	initiatives         []*models.Initiative
	meta                *models.PaginationMeta
	getErr              error
	listErr             error
	createErr           error
	updateErr           error
	deleteErr           error
	lastUpdated         *models.Initiative
	deletedID           string
	ownerEmail          string
	ownerEmailErr       error
	listPublishedResult []models.InitiativeSummary
	listPublishedErr    error
}

func (r *initiativeRepo) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return r.initiative, r.getErr
}
func (r *initiativeRepo) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return r.initiative, r.getErr
}
func (r *initiativeRepo) GetIDBySlug(_ context.Context, _ string) (string, error) {
	if r.getErr != nil {
		return "", r.getErr
	}
	if r.initiative != nil {
		return r.initiative.ID, nil
	}
	return "", domain.ErrInitiativeNotFound
}
func (r *initiativeRepo) ResolveSlug(_ context.Context, _ string) (string, error) {
	if r.getErr != nil {
		return "", r.getErr
	}
	if r.initiative != nil {
		return r.initiative.ID, nil
	}
	return "", domain.ErrInitiativeNotFound
}
func (r *initiativeRepo) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	if r.listErr != nil {
		return nil, nil, r.listErr
	}
	meta := r.meta
	if meta == nil {
		meta = &models.PaginationMeta{Total: len(r.initiatives), Limit: 20, Offset: 0}
	}
	return r.initiatives, meta, nil
}
func (r *initiativeRepo) Create(_ context.Context, i *models.Initiative, _ models.InitiativeCreateInput) (*models.Initiative, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	return i, nil
}
func (r *initiativeRepo) Update(_ context.Context, i *models.Initiative, _ models.InitiativeUpdateInput) (*models.Initiative, error) {
	r.lastUpdated = i
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	return i, nil
}
func (r *initiativeRepo) Delete(_ context.Context, id string) error {
	r.deletedID = id
	return r.deleteErr
}
func (r *initiativeRepo) UpdateStripeProductID(_ context.Context, _, _ string) error { return nil }
func (r *initiativeRepo) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return nil, nil
}
func (r *initiativeRepo) GetUsersByLegacyIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return nil, nil
}
func (r *initiativeRepo) GetOwnerInfoBySlug(_ context.Context, _ string) (models.OwnerInfo, error) {
	if r.ownerEmailErr != nil {
		return models.OwnerInfo{}, r.ownerEmailErr
	}
	return models.OwnerInfo{Email: r.ownerEmail}, nil
}
func (r *initiativeRepo) ListPublished(_ context.Context) ([]models.InitiativeSummary, error) {
	return r.listPublishedResult, r.listPublishedErr
}
func (r *initiativeRepo) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	return nil, nil
}

// initiativeUserRepo is a configurable UserRepository stub for initiative handler tests.
type initiativeUserRepo struct {
	user *models.User
	err  error
}

func (r *initiativeUserRepo) GetByUsername(_ context.Context, _ string) (*models.User, error) {
	return r.user, r.err
}
func (r *initiativeUserRepo) GetByID(_ context.Context, _ string) (*models.User, error) {
	return r.user, r.err
}
func (r *initiativeUserRepo) Upsert(_ context.Context, u *models.User) (*models.User, error) {
	return u, nil
}
func (r *initiativeUserRepo) UpdateStripeInfo(_ context.Context, _, _, _ string) error   { return nil }
func (r *initiativeUserRepo) ClearStripePaymentMethod(_ context.Context, _ string) error { return nil }

// ledgerWithTransactions returns a configurable LedgerClient stub.
type ledgerWithTransactions struct {
	apprLedgerClient
	list *models.TransactionList
	err  error
}

func (c *ledgerWithTransactions) GetTransactions(_ context.Context, _ clients.TransactionFilter) (*models.TransactionList, error) {
	return c.list, c.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newInitiativeHandler(repo *initiativeRepo, userRepo *initiativeUserRepo) *InitiativeHandler {
	svc := service.NewInitiativeService(repo, userRepo, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	return NewInitiativeHandler(svc, nil, slog.Default())
}

// withURLParam returns a copy of r with the named Chi URL parameter set.
func withURLParam(r *http.Request, key, value string) *http.Request {
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
}

// withPrincipal injects a principal into the request context.
func withPrincipal(r *http.Request, p *models.Principal) *http.Request {
	return r.WithContext(auth.ContextWithPrincipal(r.Context(), p))
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestList_ReturnsInitiatives(t *testing.T) {
	repo := &initiativeRepo{
		initiatives: []*models.Initiative{
			{ID: "init-1", Name: "Alpha", Status: models.StatusPublished},
			{ID: "init-2", Name: "Beta", Status: models.StatusPublished},
		},
		meta: &models.PaginationMeta{Total: 2, Limit: 20, Offset: 0},
	}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []*models.Initiative   `json:"data"`
		Meta *models.PaginationMeta `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 2 {
		t.Errorf("expected 2 initiatives, got %d", len(body.Data))
	}
	if body.Meta == nil || body.Meta.Total != 2 {
		t.Errorf("expected meta.total=2, got %v", body.Meta)
	}
}

func TestList_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	h := newInitiativeHandler(&initiativeRepo{}, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Data []any `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Data == nil {
		t.Error("data must be a JSON array, not null")
	}
}

func TestList_InvalidPagination_Returns400(t *testing.T) {
	h := newInitiativeHandler(&initiativeRepo{}, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives?limit=notanumber", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestGetByID_Published_Returns200(t *testing.T) {
	initiativeID := "11111111-1111-1111-1111-111111111111"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Name:   "My Project",
			Status: models.StatusPublished,
		},
	}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID, nil)
	req = withURLParam(req, "id", initiativeID)
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got models.Initiative
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != initiativeID {
		t.Errorf("expected ID %s, got %s", initiativeID, got.ID)
	}
}

func TestGetByID_NotPublished_NoApprover_Returns404(t *testing.T) {
	initiativeID := "22222222-2222-2222-2222-222222222222"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Name:   "Draft Project",
			Status: models.StatusSubmitted,
		},
	}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID, nil)
	req = withURLParam(req, "id", initiativeID)
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetByID_NotPublished_Approver_Returns200(t *testing.T) {
	initiativeID := "33333333-3333-3333-3333-333333333333"
	approver := "approver-user"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Name:   "Submitted Project",
			Status: models.StatusSubmitted,
		},
	}
	svc := service.NewInitiativeService(repo, &initiativeUserRepo{}, &apprLedgerClient{}, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, []string{approver}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID, nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{Username: approver})
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for approver, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetByID_NotFound_Returns404(t *testing.T) {
	repo := &initiativeRepo{getErr: domain.ErrInitiativeNotFound}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/no-such-slug", nil)
	req = withURLParam(req, "id", "no-such-slug")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetByID_ETagNotModified_Returns304(t *testing.T) {
	initiativeID := "44444444-4444-4444-4444-444444444444"
	initiative := &models.Initiative{
		ID:     initiativeID,
		Name:   "Stable Project",
		Status: models.StatusPublished,
	}
	repo := &initiativeRepo{initiative: initiative}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	// First request to obtain the ETag.
	req1 := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID, nil)
	req1 = withURLParam(req1, "id", initiativeID)
	w1 := httptest.NewRecorder()
	h.GetByID(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header in response")
	}

	// Second request with the ETag — must return 304.
	req2 := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID, nil)
	req2 = withURLParam(req2, "id", initiativeID)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	h.GetByID(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Errorf("expected 304, got %d", w2.Code)
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestCreate_NoPrincipal_Returns401(t *testing.T) {
	h := newInitiativeHandler(&initiativeRepo{}, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodPost, "/v1/initiatives",
		strings.NewReader(`{"name":"Test","initiative_type":"project"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreate_InvalidJSON_Returns400(t *testing.T) {
	h := newInitiativeHandler(&initiativeRepo{}, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodPost, "/v1/initiatives",
		strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	req = withPrincipal(req, &models.Principal{Username: "testuser"})
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_UserNotFound_Returns403(t *testing.T) {
	// Service translates ErrUserNotFound into ErrForbidden for Create.
	userRepo := &initiativeUserRepo{err: domain.ErrUserNotFound}
	h := newInitiativeHandler(&initiativeRepo{}, userRepo)

	req := httptest.NewRequest(http.MethodPost, "/v1/initiatives",
		strings.NewReader(`{"name":"Test","initiative_type":"project"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withPrincipal(req, &models.Principal{Username: "unknown"})
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCreate_MissingName_Returns400(t *testing.T) {
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: "user-1", Username: "testuser"},
	}
	h := newInitiativeHandler(&initiativeRepo{}, userRepo)

	req := httptest.NewRequest(http.MethodPost, "/v1/initiatives",
		strings.NewReader(`{"initiative_type":"project"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withPrincipal(req, &models.Principal{Username: "testuser"})
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_Success_Returns201(t *testing.T) {
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: "user-uuid-1", Username: "testuser", Email: "test@example.com"},
	}
	h := newInitiativeHandler(&initiativeRepo{}, userRepo)

	body := `{"name":"My Initiative","initiative_type":"project"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/initiatives", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withPrincipal(req, &models.Principal{Username: "testuser"})
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var got models.Initiative
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "My Initiative" {
		t.Errorf("expected name 'My Initiative', got %q", got.Name)
	}
	if got.Status != models.StatusSubmitted {
		t.Errorf("expected status submitted, got %q", got.Status)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestUpdate_NoPrincipal_Returns401(t *testing.T) {
	h := newInitiativeHandler(&initiativeRepo{}, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodPatch, "/v1/initiatives/some-id",
		strings.NewReader(`{"name":"New Name"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", "some-id")
	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUpdate_NotOwner_Returns403(t *testing.T) {
	initiativeID := "55555555-5555-5555-5555-555555555555"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:      initiativeID,
			OwnerID: "owner-uuid",
			Status:  models.StatusSubmitted,
		},
	}
	// Caller is a different user.
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: "other-user-uuid", Username: "other"},
	}
	h := newInitiativeHandler(repo, userRepo)

	req := httptest.NewRequest(http.MethodPatch, "/v1/initiatives/"+initiativeID,
		strings.NewReader(`{"name":"Hacked"}`))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{Username: "other"})
	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestUpdate_Success_Returns200(t *testing.T) {
	initiativeID := "66666666-6666-6666-6666-666666666666"
	ownerID := "owner-uuid-2"
	newName := "Updated Name"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:      initiativeID,
			Name:    "Old Name",
			OwnerID: ownerID,
			Status:  models.StatusSubmitted,
		},
	}
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: ownerID, Username: "owner"},
	}
	h := newInitiativeHandler(repo, userRepo)

	body := `{"name":"Updated Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/initiatives/"+initiativeID,
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{Username: "owner"})
	w := httptest.NewRecorder()
	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got models.Initiative
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != newName {
		t.Errorf("expected name %q, got %q", newName, got.Name)
	}
}

// ── GetTransactions ───────────────────────────────────────────────────────────

func TestGetTransactions_Published_Returns200(t *testing.T) {
	initiativeID := "77777777-7777-7777-7777-777777777777"
	ledger := &ledgerWithTransactions{
		list: &models.TransactionList{
			Data:       []models.Transaction{{ID: "txn-1", AmountCents: 1000}},
			TotalCount: 1,
		},
	}
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusPublished,
		},
	}
	svc := service.NewInitiativeService(repo, &initiativeUserRepo{}, ledger, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID+"/transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	w := httptest.NewRecorder()
	h.GetTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got models.TransactionList
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Data) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(got.Data))
	}
}

func TestGetTransactions_SubscriptionOnly_ForwardsFlag(t *testing.T) {
	// Verify that ?subscriptionOnly=true is parsed by the handler and reaches
	// the Ledger client as TransactionFilter.SubscriptionOnly = true.
	initiativeID := "77777778-7777-7777-7777-777777777777"
	capture := &filterCapturingLedger{
		list: &models.TransactionList{
			Data:       []models.Transaction{{ID: "txn-s", AmountCents: 500}},
			TotalCount: 1,
		},
	}
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusPublished,
		},
	}
	svc := service.NewInitiativeService(repo, &initiativeUserRepo{}, capture, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID+"/transactions?subscriptionOnly=true", nil)
	req = withURLParam(req, "id", initiativeID)
	w := httptest.NewRecorder()
	h.GetTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !capture.lastFilter.SubscriptionOnly {
		t.Error("TransactionFilter.SubscriptionOnly should be true when ?subscriptionOnly=true is sent")
	}
}

func TestGetTransactions_NotPublished_Returns404(t *testing.T) {
	initiativeID := "88888888-8888-8888-8888-888888888888"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:     initiativeID,
			Status: models.StatusSubmitted,
		},
	}
	h := newInitiativeHandler(repo, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/initiatives/"+initiativeID+"/transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	w := httptest.NewRecorder()
	h.GetTransactions(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── GetTransactionsForUser ────────────────────────────────────────────────────

func TestGetTransactionsForUser_NoPrincipal_Returns401(t *testing.T) {
	h := newInitiativeHandler(&initiativeRepo{}, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/some-id/transactions", nil)
	req = withURLParam(req, "id", "some-id")
	w := httptest.NewRecorder()
	h.GetTransactionsForUser(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetTransactionsForUser_NotOwner_Returns404(t *testing.T) {
	initiativeID := "99999999-9999-9999-9999-999999999999"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:      initiativeID,
			OwnerID: "real-owner-uuid",
			Status:  models.StatusSubmitted,
		},
	}
	// Caller resolves to a different user.
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: "other-uuid", Username: "other"},
	}
	h := newInitiativeHandler(repo, userRepo)

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{Username: "other"})
	w := httptest.NewRecorder()
	h.GetTransactionsForUser(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetTransactionsForUser_Owner_Returns200(t *testing.T) {
	initiativeID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	ownerID := "owner-uuid-txn"
	ledger := &ledgerWithTransactions{
		list: &models.TransactionList{
			Data:       []models.Transaction{{ID: "txn-2", AmountCents: 500}},
			TotalCount: 1,
		},
	}
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:      initiativeID,
			OwnerID: ownerID,
			Status:  models.StatusSubmitted,
		},
	}
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: ownerID, Username: "owner"},
	}
	svc := service.NewInitiativeService(repo, userRepo, ledger, &apprStripeClient{}, &apprEmailService{}, nil, slog.Default())
	h := NewInitiativeHandler(svc, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/v1/me/initiatives/"+initiativeID+"/transactions", nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{Username: "owner"})
	w := httptest.NewRecorder()
	h.GetTransactionsForUser(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Vary header must be set to prevent cross-user cache poisoning.
	if w.Header().Get("Vary") != "Authorization" {
		t.Errorf("expected Vary: Authorization header, got %q", w.Header().Get("Vary"))
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestDelete_NoPrincipal_Returns401(t *testing.T) {
	h := newInitiativeHandler(&initiativeRepo{}, &initiativeUserRepo{})

	req := httptest.NewRequest(http.MethodDelete, "/v1/initiatives/some-id", nil)
	req = withURLParam(req, "id", "some-id")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDelete_NotOwner_Returns403(t *testing.T) {
	initiativeID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:      initiativeID,
			OwnerID: "real-owner-uuid",
		},
	}
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: "other-uuid", Username: "other"},
	}
	h := newInitiativeHandler(repo, userRepo)

	req := httptest.NewRequest(http.MethodDelete, "/v1/initiatives/"+initiativeID, nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{Username: "other"})
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestDelete_NotFound_Returns404(t *testing.T) {
	repo := &initiativeRepo{getErr: domain.ErrInitiativeNotFound}
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: "user-1", Username: "testuser"},
	}
	h := newInitiativeHandler(repo, userRepo)

	req := httptest.NewRequest(http.MethodDelete, "/v1/initiatives/missing", nil)
	req = withURLParam(req, "id", "missing")
	req = withPrincipal(req, &models.Principal{Username: "testuser"})
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDelete_Success_Returns204(t *testing.T) {
	initiativeID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	ownerID := "owner-uuid-del"
	repo := &initiativeRepo{
		initiative: &models.Initiative{
			ID:      initiativeID,
			OwnerID: ownerID,
		},
	}
	userRepo := &initiativeUserRepo{
		user: &models.User{ID: ownerID, Username: "owner"},
	}
	h := newInitiativeHandler(repo, userRepo)

	req := httptest.NewRequest(http.MethodDelete, "/v1/initiatives/"+initiativeID, nil)
	req = withURLParam(req, "id", initiativeID)
	req = withPrincipal(req, &models.Principal{Username: "owner"})
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if repo.deletedID != initiativeID {
		t.Errorf("expected repo.Delete called with %s, got %s", initiativeID, repo.deletedID)
	}
}
