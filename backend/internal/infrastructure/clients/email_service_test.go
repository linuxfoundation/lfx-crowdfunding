// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// --- mock ---

type mockSender struct {
	calls []EmailMessage
	errs  []error // returned in order; nil after exhausted (empty slice always returns nil)
}

func (m *mockSender) Send(_ context.Context, msg EmailMessage) error {
	m.calls = append(m.calls, msg)
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

func newSvc(sender EmailSender, emails []string) *emailService {
	return NewEmailService(sender, "https://example.com", emails).(*emailService)
}

// --- tests ---

func TestInitiativeURL_EscapesSlug(t *testing.T) {
	svc := newSvc(&mockSender{}, nil)

	got := svc.InitiativeURL("x/process-approval/decline#")
	want := "https://example.com/initiatives/x%2Fprocess-approval%2Fdecline%23"
	if got != want {
		t.Errorf("InitiativeURL = %q, want %q", got, want)
	}
}

func TestSendProjectForReviewEmail_NoRecipients(t *testing.T) {
	m := &mockSender{}
	svc := newSvc(m, nil)

	err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline")
	if !errors.Is(err, ErrNoNotificationRecipients) {
		t.Fatalf("expected ErrNoNotificationRecipients, got %v", err)
	}
	if len(m.calls) != 0 {
		t.Fatalf("expected 0 Send calls, got %d", len(m.calls))
	}
}

func TestSendProjectForReviewEmail_SingleRecipient(t *testing.T) {
	m := &mockSender{}
	svc := newSvc(m, []string{"reviewer@example.com"})

	if err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.calls) != 1 {
		t.Fatalf("expected 1 Send call, got %d", len(m.calls))
	}
	if m.calls[0].To != "reviewer@example.com" {
		t.Errorf("expected recipient reviewer@example.com, got %q", m.calls[0].To)
	}
}

func TestSendProjectForReviewEmail_MultipleRecipients(t *testing.T) {
	m := &mockSender{}
	recipients := []string{"a@example.com", "b@example.com", "c@example.com"}
	svc := newSvc(m, recipients)

	if err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.calls) != 3 {
		t.Fatalf("expected 3 Send calls, got %d", len(m.calls))
	}
	for i, want := range recipients {
		if m.calls[i].To != want {
			t.Errorf("call %d: expected %q, got %q", i, want, m.calls[i].To)
		}
	}
}

func TestSendProjectForReviewEmail_ErrorAbortsEarly(t *testing.T) {
	sendErr := errors.New("send failed")
	m := &mockSender{errs: []error{nil, sendErr}}
	svc := newSvc(m, []string{"a@example.com", "b@example.com", "c@example.com"})

	err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline")
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected sendErr, got %v", err)
	}
	if len(m.calls) != 2 {
		t.Fatalf("expected 2 Send calls before abort, got %d", len(m.calls))
	}
}

func TestSendProjectApprovedEmail_CopiesReviewInboxes(t *testing.T) {
	sendErr := errors.New("send failed")
	m := &mockSender{errs: []error{nil, sendErr, nil}}
	svc := newSvc(m, []string{"a@example.com", "b@example.com"})

	err := svc.SendProjectApprovedEmail(context.Background(), "owner@example.com", "Owner", "My Project", "http://url")
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected sendErr, got %v", err)
	}
	want := []string{"owner@example.com", "a@example.com", "b@example.com"}
	if len(m.calls) != len(want) {
		t.Fatalf("expected %d Send calls (failure must not abort), got %d", len(want), len(m.calls))
	}
	for i, to := range want {
		if m.calls[i].To != to {
			t.Errorf("call %d: expected %q, got %q", i, to, m.calls[i].To)
		}
	}
}

func TestSendProjectApprovedEmail_NoInboxes(t *testing.T) {
	m := &mockSender{}
	svc := newSvc(m, nil)

	if err := svc.SendProjectApprovedEmail(context.Background(), "owner@example.com", "Owner", "My Project", "http://url"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.calls) != 1 || m.calls[0].To != "owner@example.com" {
		t.Fatalf("expected only the owner email, got %+v", m.calls)
	}
}

func TestSendEmail_NotConfigured(t *testing.T) {
	svc := NewEmailService(nil, "https://example.com", nil)

	err := svc.SendProjectApprovedEmail(context.Background(), "owner@example.com", "Owner", "My Project", "http://url")
	if !errors.Is(err, ErrEmailNotConfigured) {
		t.Fatalf("expected ErrEmailNotConfigured, got %v", err)
	}
}

// TestTemplates_RenderAndEscape renders every template and checks that
// user-controlled fields are HTML-escaped in the HTML body.
func TestTemplates_RenderAndEscape(t *testing.T) {
	ctx := context.Background()
	m := &mockSender{}
	svc := newSvc(m, []string{"reviewer@example.com"})
	name := `<script>x</script>`

	sends := []func() error{
		func() error {
			return svc.SendProjectForReviewEmail(ctx, "Owner", "owner@example.com", name, "http://url", "http://approve", "http://decline")
		},
		func() error { return svc.SendProjectApprovedEmail(ctx, "o@example.com", "Owner", name, "http://url") },
		func() error { return svc.SendProjectDeclinedEmail(ctx, "o@example.com", "Owner", name, "http://url") },
		func() error {
			return svc.SendDonationConfirmationEmail(ctx, "d@example.com", "Donor", name, "http://url", "$50.00", "Travel", "Acme", "Credit Card", "One-time")
		},
		func() error {
			return svc.SendDonationAdminNotificationEmail(ctx, "o@example.com", "Owner", "Donor", "d@example.com", name, "http://url", "$50.00", "Travel", "Acme", "Credit Card", "One-time")
		},
	}
	for i, send := range sends {
		if err := send(); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	// +1: the approved email is also copied to the reviewer inbox.
	if len(m.calls) != len(sends)+1 {
		t.Fatalf("expected %d sends, got %d", len(sends)+1, len(m.calls))
	}
	for i, msg := range m.calls {
		if msg.Subject == "" || msg.HTML == "" || msg.Text == "" {
			t.Errorf("send %d: empty subject, html, or text: %+v", i, msg)
		}
		if strings.Contains(msg.HTML, name) || !strings.Contains(msg.HTML, "&lt;script&gt;") {
			t.Errorf("send %d: initiative name not escaped in html", i)
		}
		if !strings.Contains(msg.Text, name) {
			t.Errorf("send %d: initiative name missing from text", i)
		}
		if !strings.Contains(msg.HTML, "http://url") {
			t.Errorf("send %d: initiative link missing from html", i)
		}
	}
}
