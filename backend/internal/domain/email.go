// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package domain defines shared domain errors and repository contracts.
package domain

import "context"

// EmailService defines outbound notification operations for the initiative domain.
type EmailService interface {
	// SendProjectApprovedEmail notifies the initiative owner that their
	// initiative has been approved and is now live.
	SendProjectApprovedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error

	// SendProjectDeclinedEmail notifies the initiative owner that their
	// initiative has been declined.
	SendProjectDeclinedEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL string) error

	// SendProjectForReviewEmail notifies the reviewer inbox that a new initiative
	// has been submitted and is awaiting approval.
	// ownerName and ownerEmail are the submitter's details.
	// initiativeName and initiativeURL identify the submission.
	// approveURL and declineURL are deep-links for the reviewer to act directly.
	SendProjectForReviewEmail(ctx context.Context, ownerName, ownerEmail, initiativeName, initiativeURL, approveURL, declineURL string) error

	// SendDonationConfirmationEmail notifies the donor that their donation was
	// successfully processed.
	// toEmail/toName are the donor's contact details.
	// initiativeName is the display name of the funded initiative.
	// amountFormatted is a pre-formatted currency string, e.g. "$50.00".
	SendDonationConfirmationEmail(ctx context.Context, toEmail, toName, initiativeName, initiativeURL, amountFormatted string) error

	// SendDonationAdminNotificationEmail notifies the initiative owner that a new
	// donation has been received on their initiative.
	// ownerEmail is the recipient — the owner/creator of the initiative.
	// donorName and donorEmail identify the donor for the email body.
	// initiativeName and amountFormatted describe the transaction.
	SendDonationAdminNotificationEmail(ctx context.Context, ownerEmail, donorName, donorEmail, initiativeName, initiativeURL, amountFormatted string) error

	// InitiativeURL composes a full frontend deep-link for an initiative slug.
	InitiativeURL(slug string) string
}
