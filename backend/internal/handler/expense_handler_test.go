// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

type stubRSClient struct {
	// approvers / approversErr are returned by GetExpenseApprovers.
	approvers    []string
	approversErr error
	// err is returned by ProcessExpenseAction.
	err error

	// captures the last calls for assertion
	approversCalled  bool
	actionCalled     bool
	capturedAction   string
	capturedReportID string
	capturedActor    *models.Principal
}

func (s *stubRSClient) SyncPolicy(_ context.Context, _ *models.Initiative, _ *models.User) error {
	return nil
}

// Ensure stubRSClient satisfies the interface at compile time.
var _ clients.ReimbursementClient = (*stubRSClient)(nil)

func (s *stubRSClient) GetExpenseApprovers(_ context.Context, _ string) ([]string, error) {
	s.approversCalled = true
	return s.approvers, s.approversErr
}

func (s *stubRSClient) ProcessExpenseAction(_ context.Context, action, reportID string, actor *models.Principal) error {
	s.actionCalled = true
	s.capturedAction = action
	s.capturedReportID = reportID
	s.capturedActor = actor
	return s.err
}

// stubUserRepo implements only GetByUsername; the embedded interface satisfies
// the rest and panics if anything else is called.
type stubUserRepo struct {
	domain.UserRepository
	user *models.User
	err  error
}

func (s *stubUserRepo) GetByUsername(_ context.Context, _ string) (*models.User, error) {
	return s.user, s.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

const ownerEmail = "owner@example.org"

func expenseRouter(h *ExpenseHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/crowdfunding/expense/{action}/{reportId}", h.ProcessAction)
	return r
}

// expenseReq builds a request carrying the given principal (nil = none).
func expenseReq(action, reportID string, p *models.Principal) *http.Request {
	req := httptest.NewRequest(http.MethodPost,
		"/crowdfunding/expense/"+action+"/"+reportID, nil)
	if p != nil {
		req = req.WithContext(auth.ContextWithPrincipal(req.Context(), p))
	}
	return req
}

func owner() *models.Principal {
	return &models.Principal{Username: "owner", Email: ownerEmail}
}

func serve(stub *stubRSClient, repo domain.UserRepository, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	expenseRouter(NewExpenseHandler(stub, repo)).ServeHTTP(w, req)
	return w
}

// ── authorized ────────────────────────────────────────────────────────────────

func TestExpenseHandler_ProcessAction_Success(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}

	w := serve(stub, nil, expenseReq("approve", "R-001", owner()))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if stub.capturedAction != "approve" {
		t.Errorf("expected action=approve, got %q", stub.capturedAction)
	}
	if stub.capturedReportID != "R-001" {
		t.Errorf("expected reportID=R-001, got %q", stub.capturedReportID)
	}
	if stub.capturedActor == nil || stub.capturedActor.Email != ownerEmail || stub.capturedActor.Username != "owner" {
		t.Errorf("expected the caller to be forwarded as actor, got %+v", stub.capturedActor)
	}
}

func TestExpenseHandler_ProcessAction_RejectAction(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}

	w := serve(stub, nil, expenseReq("reject", "R-002", owner()))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if stub.capturedAction != "reject" {
		t.Errorf("expected action=reject, got %q", stub.capturedAction)
	}
}

func TestExpenseHandler_ProcessAction_EmailMatchIsCaseInsensitive(t *testing.T) {
	stub := &stubRSClient{approvers: []string{"  Owner@Example.ORG "}}

	w := serve(stub, nil, expenseReq("approve", "R-001", owner()))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestExpenseHandler_ProcessAction_TravelFundAdminAllowed(t *testing.T) {
	// The RS returns every recipient it emailed for the report; a Travel Fund
	// admin is one of them just like an initiative owner.
	stub := &stubRSClient{approvers: []string{"tf-admin@example.org"}}
	admin := &models.Principal{Username: "tfadmin", Email: "tf-admin@example.org"}

	w := serve(stub, nil, expenseReq("approve", "TF-1", admin))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── not authorized: the RS action must never be called ────────────────────────

func TestExpenseHandler_ProcessAction_NonApproverGetsNotFound(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}
	beneficiary := &models.Principal{Username: "bene", Email: "bene@example.org"}

	w := serve(stub, nil, expenseReq("approve", "R-001", beneficiary))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if stub.actionCalled {
		t.Error("RS action must not be called for a non-approver")
	}
}

func TestExpenseHandler_ProcessAction_UnknownAndUnauthorizedAreIndistinguishable(t *testing.T) {
	caller := &models.Principal{Username: "bene", Email: "bene@example.org"}

	notOwner := &stubRSClient{approvers: []string{ownerEmail}}
	unauthorized := serve(notOwner, nil, expenseReq("approve", "R-001", caller))

	missing := &stubRSClient{approversErr: domain.ErrExpenseReportNotFound}
	unknown := serve(missing, nil, expenseReq("approve", "R-404", caller))

	if unauthorized.Code != unknown.Code || unauthorized.Body.String() != unknown.Body.String() {
		t.Errorf("responses differ: %d %q vs %d %q",
			unauthorized.Code, unauthorized.Body.String(), unknown.Code, unknown.Body.String())
	}
	if missing.actionCalled || notOwner.actionCalled {
		t.Error("RS action must not be called")
	}
}

func TestExpenseHandler_ProcessAction_NoEmailIsForbidden(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}
	noEmail := &models.Principal{Username: "owner"}

	w := serve(stub, nil, expenseReq("approve", "R-001", noEmail))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	if stub.approversCalled || stub.actionCalled {
		t.Error("RS must not be contacted when the caller has no email")
	}
}

func TestExpenseHandler_ProcessAction_NoPrincipal(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}

	w := serve(stub, nil, expenseReq("approve", "R-001", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if stub.approversCalled || stub.actionCalled {
		t.Error("RS must not be contacted without a principal")
	}
}

// ── Heimdall tokens carry no email: resolved from the users table ─────────────

func heimdallPrincipal() *models.Principal {
	return &models.Principal{Username: "owner", IsHeimdallIssued: true}
}

func TestExpenseHandler_ProcessAction_HeimdallEmailFromUserRow(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}
	repo := &stubUserRepo{user: &models.User{Username: "owner", Email: ownerEmail}}

	w := serve(stub, repo, expenseReq("approve", "R-001", heimdallPrincipal()))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if stub.capturedActor == nil || stub.capturedActor.Email != ownerEmail {
		t.Errorf("expected resolved email on the actor, got %+v", stub.capturedActor)
	}
}

func TestExpenseHandler_ProcessAction_HeimdallUserNotFoundIsForbidden(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}
	repo := &stubUserRepo{err: domain.ErrUserNotFound}

	w := serve(stub, repo, expenseReq("approve", "R-001", heimdallPrincipal()))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
	if stub.approversCalled || stub.actionCalled {
		t.Error("RS must not be contacted when the caller's email can't be resolved")
	}
}

func TestExpenseHandler_ProcessAction_HeimdallUserWithoutEmailIsForbidden(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}
	repo := &stubUserRepo{user: &models.User{Username: "owner"}}

	w := serve(stub, repo, expenseReq("approve", "R-001", heimdallPrincipal()))

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestExpenseHandler_ProcessAction_HeimdallUserLookupError(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}
	repo := &stubUserRepo{err: errors.New("db down")}

	w := serve(stub, repo, expenseReq("approve", "R-001", heimdallPrincipal()))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	if stub.approversCalled || stub.actionCalled {
		t.Error("RS must not be contacted when the lookup fails")
	}
}

// ── request validation and upstream failures ──────────────────────────────────

func TestExpenseHandler_ProcessAction_InvalidAction(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}}

	w := serve(stub, nil, expenseReq("delete", "R-001", owner()))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if stub.approversCalled || stub.actionCalled {
		t.Error("expected no RS call")
	}
}

func TestExpenseHandler_ProcessAction_ReportNotFound(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}, err: domain.ErrExpenseReportNotFound}

	w := serve(stub, nil, expenseReq("approve", "missing-report", owner()))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestExpenseHandler_ProcessAction_UpstreamError(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}, err: errors.New("reimbursement service returned 500")}

	w := serve(stub, nil, expenseReq("approve", "R-003", owner()))

	// Unmapped errors fall through to the default 500 case in respond.go.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestExpenseHandler_ProcessAction_UpstreamUnavailable(t *testing.T) {
	stub := &stubRSClient{approvers: []string{ownerEmail}, err: domain.ErrUpstreamUnavailable}

	w := serve(stub, nil, expenseReq("approve", "R-003", owner()))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestExpenseHandler_ProcessAction_ApproversLookupUnavailable(t *testing.T) {
	stub := &stubRSClient{approversErr: domain.ErrUpstreamUnavailable}

	w := serve(stub, nil, expenseReq("approve", "R-003", owner()))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	if stub.actionCalled {
		t.Error("RS action must not be called when approvers can't be determined")
	}
}

func TestExpenseHandler_ProcessAction_NilClient(t *testing.T) {
	w := httptest.NewRecorder()
	expenseRouter(NewExpenseHandler(nil, nil)).ServeHTTP(w, expenseReq("approve", "R-004", owner()))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
