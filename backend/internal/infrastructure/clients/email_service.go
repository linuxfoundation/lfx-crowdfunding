// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

type emailService struct {
	mandrill           MandrillClient
	frontendBase       string   // e.g. "https://crowdfunding.lfx.linuxfoundation.org"
	notificationEmails []string // reviewer inboxes for new-submission alerts
}

// NewEmailService returns a domain.EmailService backed by Mandrill.
func NewEmailService(mandrill MandrillClient, frontendBase string, notificationEmails []string) domain.EmailService {
	return &emailService{
		mandrill:           mandrill,
		frontendBase:       strings.TrimRight(frontendBase, "/"),
		notificationEmails: notificationEmails,
	}
}

// emailRequest is the internal send descriptor used by sendEmail.
type emailRequest struct {
	Recipient          string
	RecipientName      string
	TemplateName       MandrillTemplateName
	TemplateParameters map[string]string
}

// sendEmail is the single generic send method that all specific email methods delegate to.
func (s *emailService) sendEmail(ctx context.Context, req emailRequest) error {
	return s.mandrill.SendTemplate(ctx, req.TemplateName, req.Recipient, req.RecipientName, req.TemplateParameters)
}

// InitiativeURL composes a full frontend deep-link for an initiative slug.
func (s *emailService) InitiativeURL(slug string) string {
	return fmt.Sprintf("%s/initiatives/%s", s.frontendBase, slug)
}

// SendProjectApprovedEmail notifies the initiative owner that their initiative was approved.
func (s *emailService) SendProjectApprovedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error {
	return s.sendEmail(ctx, emailRequest{
		Recipient:     toEmail,
		RecipientName: toName,
		TemplateName:  MandrillTemplateApproved,
		TemplateParameters: map[string]string{
			"FNAME":            toName,
			"PROJECT_NAME":     initiativeName,
			"VIEW_PROJECT_URL": initiativeURL,
		},
	})
}

// SendProjectDeclinedEmail notifies the initiative owner that their initiative was declined.
func (s *emailService) SendProjectDeclinedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error {
	return s.sendEmail(ctx, emailRequest{
		Recipient:     toEmail,
		RecipientName: toName,
		TemplateName:  MandrillTemplateDeclined,
		TemplateParameters: map[string]string{
			"FNAME":        toName,
			"PROJECT_NAME": initiativeName,
			"VIEW_URL":     initiativeURL,
		},
	})
}

// SendDonationConfirmationEmail sends a donation receipt to the donor.
func (s *emailService) SendDonationConfirmationEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL, amountFormatted string) error {
	return s.sendEmail(ctx, emailRequest{
		Recipient:     toEmail,
		RecipientName: toName,
		TemplateName:  MandrillTemplateDonationConfirmation,
		TemplateParameters: map[string]string{
			"FNAME":            toName,
			"PROJECT_NAME":     initiativeName,
			"VIEW_PROJECT_URL": initiativeURL,
			"AMOUNT":           amountFormatted,
		},
	})
}

// SendDonationAdminNotificationEmail notifies the initiative owner of a new donation.
func (s *emailService) SendDonationAdminNotificationEmail(ctx context.Context, ownerEmail, donorName, donorEmail, initiativeName, initiativeURL, amountFormatted string) error {
	if ownerEmail == "" {
		return nil // owner email not available; skip silently
	}
	return s.sendEmail(ctx, emailRequest{
		Recipient:     ownerEmail,
		RecipientName: "",
		TemplateName:  MandrillTemplateDonationAdminNotification,
		TemplateParameters: map[string]string{
			"DONOR_NAME":       donorName,
			"DONOR_EMAIL":      donorEmail,
			"PROJECT_NAME":     initiativeName,
			"VIEW_PROJECT_URL": initiativeURL,
			"AMOUNT":           amountFormatted,
		},
	})
}

// ErrNoNotificationRecipients is returned when MANDRILL_NOTIFICATION_EMAIL is empty or unset,
// so callers can log a warning rather than silently dropping the review alert.
var ErrNoNotificationRecipients = errors.New("email: no notification recipients configured")

// SendProjectForReviewEmail notifies all reviewer inboxes that a new initiative has been submitted.
func (s *emailService) SendProjectForReviewEmail(ctx context.Context, ownerName, ownerEmail, initiativeName, initiativeURL, approveURL, declineURL string) error {
	if len(s.notificationEmails) == 0 {
		return ErrNoNotificationRecipients
	}
	params := map[string]string{
		"SUBMISSION_NAME": initiativeName,
		"SUBMITTER_NAME":  ownerName,
		"SUBMITTER_EMAIL": ownerEmail,
		"VIEW_URL":        initiativeURL,
		"APPROVE_URL":     approveURL,
		"DECLINE_URL":     declineURL,
	}
	for _, recipient := range s.notificationEmails {
		if err := s.sendEmail(ctx, emailRequest{
			Recipient:          recipient,
			RecipientName:      "LFX Crowdfunding Reviewers",
			TemplateName:       MandrillTemplateSubmittedForReview,
			TemplateParameters: params,
		}); err != nil {
			return err
		}
	}
	return nil
}
