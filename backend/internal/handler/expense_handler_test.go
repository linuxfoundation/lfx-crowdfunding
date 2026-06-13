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
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// ── stub ──────────────────────────────────────────────────────────────────────

type stubRSClient struct {
	err error
	// captures the last call for assertion
	capturedAction   string
	capturedReportID string
}

func (s *stubRSClient) SyncPolicy(_ context.Context, _ *models.Initiative, _ *models.User) error {
	return nil
}

// Ensure stubRSClient satisfies the interface at compile time.
var _ clients.ReimbursementClient = (*stubRSClient)(nil)

func (s *stubRSClient) ProcessExpenseAction(_ context.Context, action, reportID string) error {
	s.capturedAction = action
	s.capturedReportID = reportID
	return s.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func expenseRouter(h *ExpenseHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/v1/expense/{action}/{reportId}", h.ProcessAction)
	return r
}

func expenseReq(action, reportID string) *http.Request {
	return httptest.NewRequest(http.MethodPost,
		"/v1/expense/"+action+"/"+reportID, nil)
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestExpenseHandler_ProcessAction_Success(t *testing.T) {
	stub := &stubRSClient{}
	h := NewExpenseHandler(stub)
	w := httptest.NewRecorder()

	expenseRouter(h).ServeHTTP(w, expenseReq("approve", "R-001"))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if stub.capturedAction != "approve" {
		t.Errorf("expected action=approve, got %q", stub.capturedAction)
	}
	if stub.capturedReportID != "R-001" {
		t.Errorf("expected reportID=R-001, got %q", stub.capturedReportID)
	}
}

func TestExpenseHandler_ProcessAction_RejectAction(t *testing.T) {
	stub := &stubRSClient{}
	h := NewExpenseHandler(stub)
	w := httptest.NewRecorder()

	expenseRouter(h).ServeHTTP(w, expenseReq("reject", "R-002"))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if stub.capturedAction != "reject" {
		t.Errorf("expected action=reject, got %q", stub.capturedAction)
	}
}

func TestExpenseHandler_ProcessAction_ReportNotFound(t *testing.T) {
	stub := &stubRSClient{err: domain.ErrExpenseReportNotFound}
	h := NewExpenseHandler(stub)
	w := httptest.NewRecorder()

	expenseRouter(h).ServeHTTP(w, expenseReq("approve", "missing-report"))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestExpenseHandler_ProcessAction_UpstreamError(t *testing.T) {
	stub := &stubRSClient{err: errors.New("reimbursement service returned 500")}
	h := NewExpenseHandler(stub)
	w := httptest.NewRecorder()

	expenseRouter(h).ServeHTTP(w, expenseReq("approve", "R-003"))

	// Unmapped errors fall through to the default 500 case in respond.go.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestExpenseHandler_ProcessAction_UpstreamUnavailable(t *testing.T) {
	stub := &stubRSClient{err: domain.ErrUpstreamUnavailable}
	h := NewExpenseHandler(stub)
	w := httptest.NewRecorder()

	expenseRouter(h).ServeHTTP(w, expenseReq("approve", "R-003"))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestExpenseHandler_ProcessAction_NilClient(t *testing.T) {
	h := NewExpenseHandler(nil)
	w := httptest.NewRecorder()

	expenseRouter(h).ServeHTTP(w, expenseReq("approve", "R-004"))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
