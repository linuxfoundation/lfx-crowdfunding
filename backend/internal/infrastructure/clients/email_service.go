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
	"net/url"
	"strings"
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

type emailService struct {
	sender             EmailSender // nil when email is not configured
	frontendBase       string      // e.g. "https://crowdfunding.lfx.linuxfoundation.org"
	notificationEmails []string    // reviewer inboxes for new-submission alerts
}

// NewEmailService returns a domain.EmailService that renders in-repo templates
// and sends them through sender. A nil sender makes every send return ErrEmailNotConfigured.
func NewEmailService(sender EmailSender, frontendBase string, notificationEmails []string) domain.EmailService {
	return &emailService{
		sender:             sender,
		frontendBase:       strings.TrimRight(frontendBase, "/"),
		notificationEmails: notificationEmails,
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
	return s.sender.Send(ctx, EmailMessage{To: to, Subject: subject, HTML: html.String(), Text: text.String()})
}

// InitiativeURL composes a full frontend deep-link for an initiative slug.
func (s *emailService) InitiativeURL(slug string) string {
	return fmt.Sprintf("%s/initiatives/%s", s.frontendBase, url.PathEscape(slug))
}

// SendProjectApprovedEmail notifies the initiative owner that their initiative was approved.
func (s *emailService) SendProjectApprovedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error {
	return s.sendEmail(ctx, toEmail, initiativeName+" has been approved on LFX Crowdfunding", templateInitiativeApproved, emailTemplateData{
		RecipientName:  toName,
		InitiativeName: initiativeName,
		InitiativeURL:  initiativeURL,
	})
}

// SendProjectDeclinedEmail notifies the initiative owner that their initiative was declined.
func (s *emailService) SendProjectDeclinedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error {
	return s.sendEmail(ctx, toEmail, initiativeName+" was not approved on LFX Crowdfunding", templateInitiativeDeclined, emailTemplateData{
		RecipientName:  toName,
		InitiativeName: initiativeName,
		InitiativeURL:  initiativeURL,
	})
}

// SendDonationConfirmationEmail sends a donation receipt to the donor.
func (s *emailService) SendDonationConfirmationEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL, amountFormatted, category, orgName, payment, donationType string) error {
	return s.sendEmail(ctx, toEmail, "Your donation to "+initiativeName+" has been received", templateDonationReceipt, emailTemplateData{
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
	return s.sendEmail(ctx, ownerEmail, "A donation to "+initiativeName+" has been received", templateDonationReceived, emailTemplateData{
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
