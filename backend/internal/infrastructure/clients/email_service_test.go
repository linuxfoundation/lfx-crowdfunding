// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

// --- mock ---

type mockMandrill struct {
	calls []mandrillCall
	errs  []error // returned in order; nil after exhausted (empty slice always returns nil)
}

type mandrillCall struct {
	toEmail string
}

func (m *mockMandrill) SendTemplate(_ context.Context, _ MandrillTemplateName, toEmail, _ string, _ map[string]string) error {
	m.calls = append(m.calls, mandrillCall{toEmail: toEmail})
	idx := len(m.calls) - 1
	if idx >= len(m.errs) {
		idx = len(m.errs) - 1
	}
	if idx < 0 {
		return nil
	}
	return m.errs[idx]
}

// --- helpers ---

func newSvc(mandrill MandrillClient, emails []string) *emailService {
	return NewEmailService(mandrill, "https://example.com", emails, false, slog.Default()).(*emailService)
}

// --- tests ---

func TestSendProjectForReviewEmail_NoRecipients(t *testing.T) {
	m := &mockMandrill{}
	svc := newSvc(m, nil)

	err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline")
	if !errors.Is(err, ErrNoNotificationRecipients) {
		t.Fatalf("expected ErrNoNotificationRecipients, got %v", err)
	}
	if len(m.calls) != 0 {
		t.Fatalf("expected 0 SendTemplate calls, got %d", len(m.calls))
	}
}

func TestSendProjectForReviewEmail_SingleRecipient(t *testing.T) {
	m := &mockMandrill{}
	svc := newSvc(m, []string{"reviewer@example.com"})

	if err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.calls) != 1 {
		t.Fatalf("expected 1 SendTemplate call, got %d", len(m.calls))
	}
	if m.calls[0].toEmail != "reviewer@example.com" {
		t.Errorf("expected recipient reviewer@example.com, got %q", m.calls[0].toEmail)
	}
}

func TestSendProjectForReviewEmail_MultipleRecipients(t *testing.T) {
	m := &mockMandrill{}
	recipients := []string{"a@example.com", "b@example.com", "c@example.com"}
	svc := newSvc(m, recipients)

	if err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.calls) != 3 {
		t.Fatalf("expected 3 SendTemplate calls, got %d", len(m.calls))
	}
	for i, want := range recipients {
		if m.calls[i].toEmail != want {
			t.Errorf("call %d: expected %q, got %q", i, want, m.calls[i].toEmail)
		}
	}
}

func TestEmailDryRun_SuppressesSend(t *testing.T) {
	m := &mockMandrill{}
	svc := NewEmailService(m, "https://example.com", []string{"reviewer@example.com"}, true, slog.Default()).(*emailService)

	if err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.calls) != 0 {
		t.Fatalf("dry-run: expected 0 Mandrill calls, got %d", len(m.calls))
	}
}

func TestSendProjectForReviewEmail_ErrorAbortsEarly(t *testing.T) {
	sendErr := errors.New("send failed")
	m := &mockMandrill{errs: []error{nil, sendErr}}
	svc := newSvc(m, []string{"a@example.com", "b@example.com", "c@example.com"})

	err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline")
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected sendErr, got %v", err)
	}
	if len(m.calls) != 2 {
		t.Fatalf("expected 2 SendTemplate calls before abort, got %d", len(m.calls))
	}
}
