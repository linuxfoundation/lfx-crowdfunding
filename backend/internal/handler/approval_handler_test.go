// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	stripe "github.com/stripe/stripe-go/v82"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

// apprInitiativeRepo is a minimal InitiativeRepository stub for approval tests.
// GetByID and Update are configurable; all other methods are no-ops.
type apprInitiativeRepo struct {
	initiative  *models.Initiative
	getErr      error
	updateErr   error
	lastUpdated *models.Initiative
}

func (r *apprInitiativeRepo) GetByID(_ context.Context, _ string) (*models.Initiative, error) {
	return r.initiative, r.getErr
}
func (r *apprInitiativeRepo) Update(_ context.Context, i *models.Initiative) (*models.Initiative, error) {
	r.lastUpdated = i
	return i, r.updateErr
}
func (r *apprInitiativeRepo) GetBySlug(_ context.Context, _ string) (*models.Initiative, error) {
	return nil, nil
}
func (r *apprInitiativeRepo) GetIDBySlug(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (r *apprInitiativeRepo) List(_ context.Context, _ models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *apprInitiativeRepo) Create(_ context.Context, i *models.Initiative) (*models.Initiative, error) {
	return i, nil
}
func (r *apprInitiativeRepo) Delete(_ context.Context, _ string) error { return nil }
func (r *apprInitiativeRepo) GetUsersByIDs(_ context.Context, _ []string) (map[string]models.User, error) {
	return nil, nil
}
func (r *apprInitiativeRepo) GetOrganizationsByIDs(_ context.Context, _ []string) (map[string]models.Organization, error) {
	return nil, nil
}

// apprLedgerClient is a no-op LedgerClient stub.
type apprLedgerClient struct{}

func (c *apprLedgerClient) GetBalance(_ context.Context, _ string) (*clients.LedgerBalance, error) {
	return nil, nil
}
func (c *apprLedgerClient) GetAllBalances(_ context.Context) ([]models.LedgerRawBalance, error) {
	return nil, nil
}
func (c *apprLedgerClient) GetTransactions(_ context.Context, _ clients.TransactionFilter) (*models.TransactionList, error) {
	return nil, nil
}
func (c *apprLedgerClient) GetPlatformBalance(_ context.Context) (*clients.LedgerPlatformBalance, error) {
	return nil, nil
}
func (c *apprLedgerClient) GetPlatformMonthly(_ context.Context, _ int) (*clients.LedgerPlatformMonthly, error) {
	return nil, nil
}
func (c *apprLedgerClient) GetPlatformRecentDonations(_ context.Context) ([]clients.LedgerRecentDonation, error) {
	return nil, nil
}

// apprStripeClient is a no-op StripeClient stub.
type apprStripeClient struct{}

func (c *apprStripeClient) GetProduct(_ context.Context, _ string) (*models.StripeProduct, error) {
	return nil, nil
}
func (c *apprStripeClient) CreateProduct(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *apprStripeClient) DeleteProduct(_ context.Context, _ string) error { return nil }
func (c *apprStripeClient) CreatePaymentIntent(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	return nil, nil
}
func (c *apprStripeClient) CreateSubscription(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	return nil, nil
}
func (c *apprStripeClient) CancelSubscription(_ context.Context, _ string) error { return nil }
func (c *apprStripeClient) ConstructWebhookEvent(_ []byte, _, _ string) (stripe.Event, error) {
	return stripe.Event{}, nil
}
func (c *apprStripeClient) CreateCustomer(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (c *apprStripeClient) CreateSetupIntent(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (c *apprStripeClient) AttachPaymentMethod(_ context.Context, _, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *apprStripeClient) GetPaymentMethod(_ context.Context, _ string) (*models.CardDetails, error) {
	return nil, nil
}
func (c *apprStripeClient) DetachPaymentMethod(_ context.Context, _ string) error { return nil }
func (c *apprStripeClient) GetOrCreatePrice(_ context.Context, _ string, _ int64, _ string, _ string) (string, error) {
	return "", nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// newApprovalHandler builds an InitiativeHandler wired to the given repo and
// approvers list. Ledger and Stripe clients are no-op stubs.
func newApprovalHandler(repo *apprInitiativeRepo, approvers []string) *InitiativeHandler {
	svc := service.NewInitiativeService(repo, &apprLedgerClient{}, &apprStripeClient{}, slog.Default())
	return NewInitiativeHandler(svc, approvers)
}

// approvalRouter mounts only the approval route on a fresh Chi router.
func approvalRouter(h *InitiativeHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/v1/initiatives/{id}/process-approval/{status}", h.ProcessApproval)
	return r
}

// approvalReq builds a POST request for the approval endpoint, optionally
// injecting a principal into the request context.
func approvalReq(initiativeID, status string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodPost,
		"/v1/initiatives/"+initiativeID+"/process-approval/"+status, nil)
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

func decodeError(t *testing.T, body []byte) string {
	t.Helper()
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("failed to decode error body: %v", err)
	}
	return e.Error
}

func decodeInitiative(t *testing.T, body []byte) *models.Initiative {
	t.Helper()
	var i models.Initiative
	if err := json.Unmarshal(body, &i); err != nil {
		t.Fatalf("failed to decode initiative body: %v", err)
	}
	return &i
}

// ── tests ─────────────────────────────────────────────────────────────────────

const testInitiativeID = "11111111-1111-1111-1111-111111111111"

func TestApprovalHandler_NoPrincipal(t *testing.T) {
	h := newApprovalHandler(&apprInitiativeRepo{}, []string{"admin"})
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "approve", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestApprovalHandler_UsernameNotAllowed(t *testing.T) {
	h := newApprovalHandler(&apprInitiativeRepo{}, []string{"alice", "bob"})
	principal := &models.Principal{Username: "mallory"}
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "approve", principal))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	msg := decodeError(t, w.Body.Bytes())
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestApprovalHandler_ApproverMatchIsCaseInsensitive(t *testing.T) {
	repo := &apprInitiativeRepo{
		initiative: &models.Initiative{ID: testInitiativeID, Status: models.StatusSubmitted},
	}
	// Approver list uses title-case; token supplies lowercase.
	h := newApprovalHandler(repo, []string{"Admin"})
	principal := &models.Principal{Username: "admin"}
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "approve", principal))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (case-insensitive match failed)", w.Code)
	}
}

func TestApprovalHandler_InvalidAction(t *testing.T) {
	repo := &apprInitiativeRepo{
		initiative: &models.Initiative{ID: testInitiativeID, Status: models.StatusSubmitted},
	}
	h := newApprovalHandler(repo, []string{"admin"})
	principal := &models.Principal{Username: "admin"}
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "publish", principal))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestApprovalHandler_Approve(t *testing.T) {
	repo := &apprInitiativeRepo{
		initiative: &models.Initiative{ID: testInitiativeID, Status: models.StatusSubmitted},
	}
	h := newApprovalHandler(repo, []string{"admin"})
	principal := &models.Principal{Username: "admin"}
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "approve", principal))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	got := decodeInitiative(t, w.Body.Bytes())
	if got.Status != models.StatusPublished {
		t.Errorf("expected status %q, got %q", models.StatusPublished, got.Status)
	}
	if repo.lastUpdated == nil || repo.lastUpdated.Status != models.StatusPublished {
		t.Error("repo.Update not called with published status")
	}
}

func TestApprovalHandler_Decline(t *testing.T) {
	repo := &apprInitiativeRepo{
		initiative: &models.Initiative{ID: testInitiativeID, Status: models.StatusSubmitted},
	}
	h := newApprovalHandler(repo, []string{"reviewer"})
	principal := &models.Principal{Username: "reviewer"}
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "decline", principal))

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	got := decodeInitiative(t, w.Body.Bytes())
	if got.Status != models.StatusDeclined {
		t.Errorf("expected status %q, got %q", models.StatusDeclined, got.Status)
	}
}

func TestApprovalHandler_InitiativeNotFound(t *testing.T) {
	repo := &apprInitiativeRepo{getErr: domain.ErrInitiativeNotFound}
	h := newApprovalHandler(repo, []string{"admin"})
	principal := &models.Principal{Username: "admin"}
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "approve", principal))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestApprovalHandler_EmptyApproversList(t *testing.T) {
	// No approvers configured — every username must be rejected.
	repo := &apprInitiativeRepo{
		initiative: &models.Initiative{ID: testInitiativeID, Status: models.StatusSubmitted},
	}
	h := newApprovalHandler(repo, nil)
	principal := &models.Principal{Username: "admin"}
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "approve", principal))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestApprovalHandler_UpdateError(t *testing.T) {
	repo := &apprInitiativeRepo{
		initiative: &models.Initiative{ID: testInitiativeID, Status: models.StatusSubmitted},
		updateErr:  errors.New("db error"),
	}
	h := newApprovalHandler(repo, []string{"admin"})
	principal := &models.Principal{Username: "admin"}
	w := httptest.NewRecorder()
	approvalRouter(h).ServeHTTP(w, approvalReq(testInitiativeID, "approve", principal))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
