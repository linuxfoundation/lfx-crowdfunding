// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

//go:embed email_templates/*.tmpl
var emailTemplateFS embed.FS

var (
	htmlEmailTemplates = htmltemplate.Must(htmltemplate.ParseFS(emailTemplateFS, "email_templates/*.html.tmpl"))
	textEmailTemplates = texttemplate.Must(texttemplate.ParseFS(emailTemplateFS, "email_templates/*.txt.tmpl"))
)

const (
	templateInitiativeReview   = "initiative_review"
	templateInitiativeApproved = "initiative_approved"
	templateInitiativeDeclined = "initiative_declined"
	templateDonationReceipt    = "donation_receipt"
	templateDonationReceived   = "donation_received"
)

// reviewerGreeting addresses the staff review inboxes.
const reviewerGreeting = "LFX Crowdfunding team"

// ErrEmailNotConfigured is returned when no email sender is wired (FGA_NATS_URL unset).
// Callers log it at WARN and continue.
var ErrEmailNotConfigured = errors.New("email: sender not configured")

// ErrNoNotificationRecipients is returned when EMAIL_REVIEW_INBOX is empty or unset,
// so callers can log a warning rather than silently dropping the review alert.
var ErrNoNotificationRecipients = errors.New("email: no notification recipients configured")

// emailTemplateData is the union of fields the notification templates read.
type emailTemplateData struct {
	RecipientName  string
	InitiativeName string
	InitiativeURL  string
	ApproveURL     string
	DeclineURL     string
	SubmitterName  string
	SubmitterEmail string
	DonorName      string
	DonorEmail     string
	Amount         string
	Category       string
	OrgName        string
	Payment        string
	DonationType   string
	Year           int
}

// copySubjectPrefix starts a copy's subject, followed by the original recipient and "] ".
// Matches lfx-v2-mentorship so both archives read the same way.
const copySubjectPrefix = "[Sent to "

type emailService struct {
	sender             EmailSender    // nil when email is not configured
	frontendBase       string         // e.g. "https://crowdfunding.lfx.linuxfoundation.org"
	notificationEmails []string       // reviewer inboxes for new-submission alerts
	copyInbox          string         // archive inbox sent a copy of every email; empty disables copies
	copies             sync.WaitGroup // in-flight background copies, drained by WaitCopies
	logger             *slog.Logger
}

// NewEmailService returns a domain.EmailService that renders in-repo templates
// and sends them through sender. A nil sender makes every send return ErrEmailNotConfigured.
// When copyInbox is set, every delivered send (not one the non-prod allowlist blocked)
// is followed by a best-effort copy to it,
// sent in the background (see sendCopy). Call WaitCopies on shutdown to drain them.
func NewEmailService(sender EmailSender, frontendBase string, notificationEmails []string, copyInbox string, logger *slog.Logger) domain.EmailService {
	if logger == nil {
		logger = slog.Default()
	}
	return &emailService{
		sender:             sender,
		frontendBase:       strings.TrimRight(frontendBase, "/"),
		notificationEmails: notificationEmails,
		copyInbox:          copyInbox,
		logger:             logger,
	}
}

// sendEmail renders the named template pair and sends it to one recipient.
func (s *emailService) sendEmail(ctx context.Context, to, subject, name string, data emailTemplateData) error {
	if s.sender == nil {
		return ErrEmailNotConfigured
	}
	data.Year = time.Now().UTC().Year()
	var html, text bytes.Buffer
	if err := htmlEmailTemplates.ExecuteTemplate(&html, name+".html.tmpl", data); err != nil {
		return fmt.Errorf("render %s html: %w", name, err)
	}
	if err := textEmailTemplates.ExecuteTemplate(&text, name+".txt.tmpl", data); err != nil {
		return fmt.Errorf("render %s text: %w", name, err)
	}
	msg := EmailMessage{To: to, Subject: subject, HTML: html.String(), Text: text.String()}
	if err := s.sender.Send(ctx, msg); err != nil {
		if errors.Is(err, ErrRecipientBlocked) {
			// The non-prod allowlist suppressed it: finished for the caller, but nothing
			// was delivered, so there is nothing to archive (and the copy would carry
			// the blocked recipient's details).
			s.logger.DebugContext(ctx, "email recipient blocked by allowlist, skipping copy", "template", name)
			return nil
		}
		return err
	}
	s.sendCopy(ctx, msg, name)
	return nil
}

// sendCopy sends msg to the copy inbox, naming the original recipient in the subject.
// SES keeps no copies, so this is the only record of what was sent. Best effort: a
// failure is logged and never fails the original send, which was already accepted.
//
// The copy goes out in a background goroutine so it adds no latency to the caller
// (the Stripe webhook, the reviewer-notification loops). It uses a detached context
// so it survives the request; the sender's own emailSendTimeout bounds it.
func (s *emailService) sendCopy(ctx context.Context, msg EmailMessage, template string) {
	if s.copyInbox == "" || strings.EqualFold(msg.To, s.copyInbox) {
		return
	}
	cp := EmailMessage{To: s.copyInbox, Subject: copySubjectPrefix + msg.To + "] " + msg.Subject, HTML: msg.HTML, Text: msg.Text}
	detached := context.WithoutCancel(ctx)
	s.copies.Add(1)
	go func() {
		defer s.copies.Done()
		if err := s.sender.Send(detached, cp); err != nil && !errors.Is(err, ErrRecipientBlocked) {
			s.logger.WarnContext(detached, "failed to send email copy", "template", template, "error", err)
		}
	}()
}

// WaitCopies blocks until every in-flight background copy has finished, or ctx is
// done. Called on shutdown before the email transport is closed.
func (s *emailService) WaitCopies(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.copies.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// InitiativeURL composes a full frontend deep-link for an initiative slug.
func (s *emailService) InitiativeURL(slug string) string {
	return fmt.Sprintf("%s/initiatives/%s", s.frontendBase, url.PathEscape(slug))
}

// SendProjectApprovedEmail notifies the initiative owner that their initiative was approved,
// and copies the same email to the reviewer inboxes as the approvers' confirmation (legacy
// Mandrill behaviour). Every recipient is attempted; failures are joined.
func (s *emailService) SendProjectApprovedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error {
	data := emailTemplateData{
		RecipientName:  toName,
		InitiativeName: initiativeName,
		InitiativeURL:  initiativeURL,
	}
	var errs []error
	for _, recipient := range append([]string{toEmail}, s.notificationEmails...) {
		if err := s.sendEmail(ctx, recipient, "Your submission for LFX Crowdfunding has been approved.", templateInitiativeApproved, data); err != nil {
			errs = append(errs, fmt.Errorf("send approved email to %s: %w", recipient, err))
		}
	}
	return errors.Join(errs...)
}

// SendProjectDeclinedEmail notifies the initiative owner that their initiative was declined.
func (s *emailService) SendProjectDeclinedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error {
	return s.sendEmail(ctx, toEmail, "Your submission to LFX Crowdfunding was rejected.", templateInitiativeDeclined, emailTemplateData{
		RecipientName:  toName,
		InitiativeName: initiativeName,
		InitiativeURL:  initiativeURL,
	})
}

// SendDonationConfirmationEmail sends a donation receipt to the donor.
func (s *emailService) SendDonationConfirmationEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL, amountFormatted, category, orgName, payment, donationType string) error {
	return s.sendEmail(ctx, toEmail, "Your donation to "+initiativeName+" has been submitted.", templateDonationReceipt, emailTemplateData{
		RecipientName:  toName,
		InitiativeName: initiativeName,
		InitiativeURL:  initiativeURL,
		DonorName:      toName,
		DonorEmail:     toEmail,
		Amount:         amountFormatted,
		Category:       category,
		OrgName:        orgName,
		Payment:        payment,
		DonationType:   donationType,
	})
}

// SendDonationAdminNotificationEmail notifies the initiative owner of a new donation.
func (s *emailService) SendDonationAdminNotificationEmail(ctx context.Context, ownerEmail, ownerName, donorName, donorEmail, initiativeName, initiativeURL, amountFormatted, category, orgName, payment, donationType string) error {
	if ownerEmail == "" {
		return nil // owner email not available; skip silently
	}
	return s.sendEmail(ctx, ownerEmail, "A donation was submitted to "+initiativeName+".", templateDonationReceived, emailTemplateData{
		RecipientName:  ownerName,
		InitiativeName: initiativeName,
		InitiativeURL:  initiativeURL,
		DonorName:      donorName,
		DonorEmail:     donorEmail,
		Amount:         amountFormatted,
		Category:       category,
		OrgName:        orgName,
		Payment:        payment,
		DonationType:   donationType,
	})
}

// SendProjectForReviewEmail notifies all reviewer inboxes that a new initiative has been submitted.
func (s *emailService) SendProjectForReviewEmail(ctx context.Context, ownerName, ownerEmail, initiativeName, initiativeURL, approveURL, declineURL string) error {
	if len(s.notificationEmails) == 0 {
		return ErrNoNotificationRecipients
	}
	data := emailTemplateData{
		RecipientName:  reviewerGreeting,
		InitiativeName: initiativeName,
		InitiativeURL:  initiativeURL,
		ApproveURL:     approveURL,
		DeclineURL:     declineURL,
		SubmitterName:  ownerName,
		SubmitterEmail: ownerEmail,
	}
	for _, recipient := range s.notificationEmails {
		if err := s.sendEmail(ctx, recipient, "LFX Crowdfunding Project application is pending your review", templateInitiativeReview, data); err != nil {
			return err
		}
	}
	return nil
}
