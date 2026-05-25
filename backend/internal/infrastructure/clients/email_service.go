// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"context"
	"fmt"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

type emailService struct {
	mandrill          MandrillClient
	frontendBase      string // e.g. "https://crowdfunding.lfx.linuxfoundation.org"
	notificationEmail string // reviewer inbox for new-submission alerts
}

// NewEmailService returns a domain.EmailService backed by Mandrill.
func NewEmailService(mandrill MandrillClient, frontendBase, notificationEmail string) domain.EmailService {
	return &emailService{
		mandrill:          mandrill,
		frontendBase:      strings.TrimRight(frontendBase, "/"),
		notificationEmail: notificationEmail,
	}
}

// InitiativeURL composes a full frontend deep-link for an initiative slug.
func (s *emailService) InitiativeURL(slug string) string {
	return fmt.Sprintf("%s/initiatives/%s", s.frontendBase, slug)
}

// SendProjectApprovedEmail notifies the initiative owner that their initiative was approved.
func (s *emailService) SendProjectApprovedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error {
	return s.mandrill.SendTemplate(ctx, MandrillTemplateApproved, toEmail, toName, map[string]string{
		"FNAME":            toName,
		"PROJECT_NAME":     initiativeName,
		"VIEW_PROJECT_URL": initiativeURL,
	})
}

// SendProjectDeclinedEmail notifies the initiative owner that their initiative was declined.
func (s *emailService) SendProjectDeclinedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error {
	return s.mandrill.SendTemplate(ctx, MandrillTemplateDeclined, toEmail, toName, map[string]string{
		"FNAME":        toName,
		"PROJECT_NAME": initiativeName,
		"VIEW_URL":     initiativeURL,
	})
}

// SendProjectForReviewEmail notifies the reviewer inbox that a new initiative has been submitted.
func (s *emailService) SendProjectForReviewEmail(ctx context.Context, ownerName, ownerEmail, initiativeName, initiativeURL, approveURL, declineURL string) error {
	if s.notificationEmail == "" {
		return fmt.Errorf("SendProjectForReviewEmail: MANDRILL_NOTIFICATION_EMAIL is not configured")
	}
	return s.mandrill.SendTemplate(
		ctx,
		MandrillTemplateSubmittedForReview,
		s.notificationEmail,
		"LFX Crowdfunding Reviewers",
		map[string]string{
			"SUBMISSION_NAME": initiativeName,
			"SUBMITTER_NAME":  ownerName,
			"SUBMITTER_EMAIL": ownerEmail,
			"VIEW_URL":        initiativeURL,
			"APPROVE_URL":     approveURL,
			"DECLINE_URL":     declineURL,
		},
	)
}
