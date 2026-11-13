// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- mock ---

type mockSender struct {
	mu    sync.Mutex // copies are sent from background goroutines
	calls []EmailMessage
	errs  []error // returned in order; nil after exhausted (empty slice always returns nil)
}

func (m *mockSender) Send(_ context.Context, msg EmailMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	return NewEmailService(sender, "https://example.com", emails, "", nil).(*emailService)
}

func newSvcWithCopy(sender EmailSender, emails []string, copyInbox string) *emailService {
	return NewEmailService(sender, "https://example.com", emails, copyInbox, nil).(*emailService)
}

// waitCopies drains the background copies so tests can inspect every Send call.
func waitCopies(t *testing.T, svc *emailService) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := svc.WaitCopies(ctx); err != nil {
		t.Fatalf("background copies did not finish: %v", err)
	}
}

// splitCalls separates the originals (in send order) from the copies, keyed by subject.
// Copies run in the background, so their order relative to the originals is not fixed.
func splitCalls(calls []EmailMessage, copyInbox string) (originals []EmailMessage, copies map[string]EmailMessage) {
	copies = map[string]EmailMessage{}
	for _, c := range calls {
		if c.To == copyInbox {
			copies[c.Subject] = c
		} else {
			originals = append(originals, c)
		}
	}
	return originals, copies
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
	svc := NewEmailService(nil, "https://example.com", nil, "copies@example.org", nil)

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

func TestSendEmail_CopiesEachSendToCopyInbox(t *testing.T) {
	m := &mockSender{}
	svc := newSvcWithCopy(m, []string{"reviewer@example.com"}, "copies@example.org")

	if err := svc.SendProjectApprovedEmail(context.Background(), "owner@example.com", "Owner", "My Project", "http://url"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	waitCopies(t, svc)
	subject := "Your submission for LFX Crowdfunding has been approved."
	if len(m.calls) != 4 {
		t.Fatalf("expected 4 Send calls, got %d: %+v", len(m.calls), m.calls)
	}
	originals, copies := splitCalls(m.calls, "copies@example.org")
	if len(originals) != 2 || originals[0].To != "owner@example.com" || originals[1].To != "reviewer@example.com" {
		t.Fatalf("unexpected originals: %+v", originals)
	}
	for _, o := range originals {
		cp, ok := copies["[Sent to "+o.To+"] "+subject]
		if !ok {
			t.Errorf("no copy for %s; copies: %+v", o.To, copies)
			continue
		}
		if cp.HTML != o.HTML || cp.Text != o.Text {
			t.Errorf("copy body for %s differs from the original", o.To)
		}
	}
}

// blockingCopySender accepts originals at once and holds every copy until released.
type blockingCopySender struct {
	mockSender
	copyInbox string
	release   chan struct{}
}

func (b *blockingCopySender) Send(ctx context.Context, msg EmailMessage) error {
	if msg.To == b.copyInbox {
		<-b.release
	}
	return b.mockSender.Send(ctx, msg)
}

func TestSendEmail_StalledCopyDoesNotBlockSend(t *testing.T) {
	b := &blockingCopySender{copyInbox: "copies@example.org", release: make(chan struct{})}
	svc := newSvcWithCopy(b, nil, "copies@example.org")

	done := make(chan error, 1)
	go func() {
		done <- svc.SendDonationConfirmationEmail(context.Background(), "d@example.com", "Donor", "My Project", "http://url", "$50.00", "Travel", "", "Credit Card", "One-time")
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		close(b.release)
		t.Fatal("send waited on the stalled copy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := svc.WaitCopies(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WaitCopies should time out while the copy is stalled, got %v", err)
	}
	close(b.release)
	waitCopies(t, svc)
	if len(b.calls) != 2 || b.calls[1].To != "copies@example.org" {
		t.Fatalf("expected original then copy once released, got %+v", b.calls)
	}
}

func TestSendEmail_CopyFailureDoesNotFailSend(t *testing.T) {
	m := &mockSender{errs: []error{nil, errors.New("copy failed")}}
	svc := newSvcWithCopy(m, nil, "copies@example.org")

	if err := svc.SendProjectDeclinedEmail(context.Background(), "owner@example.com", "Owner", "My Project", "http://url"); err != nil {
		t.Fatalf("copy failure must not fail the send, got %v", err)
	}
	waitCopies(t, svc)
	if len(m.calls) != 2 {
		t.Fatalf("expected original + copy, got %d calls", len(m.calls))
	}
}

func TestSendEmail_NoCopyWhenOriginalFails(t *testing.T) {
	sendErr := errors.New("send failed")
	m := &mockSender{errs: []error{sendErr}}
	svc := newSvcWithCopy(m, nil, "copies@example.org")

	err := svc.SendDonationConfirmationEmail(context.Background(), "d@example.com", "Donor", "My Project", "http://url", "$50.00", "Travel", "", "Credit Card", "One-time")
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected sendErr, got %v", err)
	}
	waitCopies(t, svc)
	if len(m.calls) != 1 {
		t.Fatalf("expected no copy after a failed send, got %d calls", len(m.calls))
	}
}

func TestSendEmail_NoCopyToCopyInboxItself(t *testing.T) {
	m := &mockSender{}
	svc := newSvcWithCopy(m, []string{"Copies@Example.org"}, "copies@example.org")

	if err := svc.SendProjectForReviewEmail(context.Background(), "owner", "owner@example.com", "My Project", "http://url", "http://approve", "http://decline"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	waitCopies(t, svc)
	if len(m.calls) != 1 {
		t.Fatalf("expected no self-copy, got %d calls", len(m.calls))
	}
}

func TestSendEmail_NoCopyWhenUnset(t *testing.T) {
	m := &mockSender{}
	svc := newSvc(m, nil)

	if err := svc.SendProjectDeclinedEmail(context.Background(), "owner@example.com", "Owner", "My Project", "http://url"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	waitCopies(t, svc)
	if len(m.calls) != 1 {
		t.Fatalf("expected no copy when the copy inbox is unset, got %d calls", len(m.calls))
	}
}

func TestSendEmail_NoCopyWhenRecipientBlocked(t *testing.T) {
	m := &mockSender{errs: []error{ErrRecipientBlocked}}
	svc := newSvcWithCopy(m, nil, "copies@example.org")

	err := svc.SendDonationConfirmationEmail(context.Background(), "d@example.com", "Donor", "My Project", "http://url", "$50.00", "Travel", "", "Credit Card", "One-time")
	if err != nil {
		t.Fatalf("a blocked recipient must count as finished, got %v", err)
	}
	waitCopies(t, svc)
	if len(m.calls) != 1 {
		t.Fatalf("expected no copy for an allowlist-blocked original, got %d calls", len(m.calls))
	}
}

func TestSendEmail_BlockedRecipientInLoopDoesNotFail(t *testing.T) {
	m := &mockSender{errs: []error{ErrRecipientBlocked, nil}}
	svc := newSvcWithCopy(m, []string{"review@example.org"}, "copies@example.org")

	if err := svc.SendProjectApprovedEmail(context.Background(), "owner@example.com", "Owner", "My Project", "http://url"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	waitCopies(t, svc)
	originals, copies := splitCalls(m.calls, "copies@example.org")
	if len(originals) != 2 {
		t.Fatalf("expected both originals attempted, got %d", len(originals))
	}
	if len(copies) != 1 {
		t.Fatalf("expected a copy only for the delivered original, got %d", len(copies))
	}
	for subj := range copies {
		if !strings.Contains(subj, "review@example.org") {
			t.Errorf("copy should be for the delivered review inbox, got subject %q", subj)
		}
	}
}
