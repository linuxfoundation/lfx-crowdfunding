// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	stripe "github.com/stripe/stripe-go/v85"
)

// ledgerSourceType and ledgerTxnType are the fixed values sent to the Ledger
// service for all charges originating from this service.
const (
	ledgerSourceType = "stripe"
	ledgerTxnType    = "credit"
)

// WebhookHandler handles inbound Stripe webhook events.
// Signature validation is ALWAYS performed before processing — never skip this check.
type WebhookHandler struct {
	stripeClient     clients.StripeClient
	ledgerClient     clients.LedgerClient
	donationRepo     domain.DonationRepository
	subscriptionRepo domain.SubscriptionRepository
	emailService     domain.EmailService
	webhookSecret    string
	logger           *slog.Logger
	ackUnimplemented bool // when true, reply 200 for known-but-unimplemented events
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(
	stripeClient clients.StripeClient,
	ledgerClient clients.LedgerClient,
	donationRepo domain.DonationRepository,
	subscriptionRepo domain.SubscriptionRepository,
	emailService domain.EmailService,
	webhookSecret string,
	logger *slog.Logger,
	ackUnimplemented bool,
) *WebhookHandler {
	return &WebhookHandler{
		stripeClient:     stripeClient,
		ledgerClient:     ledgerClient,
		donationRepo:     donationRepo,
		subscriptionRepo: subscriptionRepo,
		emailService:     emailService,
		webhookSecret:    webhookSecret,
		logger:           logger,
		ackUnimplemented: ackUnimplemented,
	}
}

// Handle handles POST /v1/stripe/webhook
// The Stripe-Signature header MUST be validated before any event processing (OWASP).
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = 65536 // 64 KiB
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", "error", err)
		http.Error(w, "request body error", http.StatusBadRequest)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	if sigHeader == "" {
		h.logger.Warn("stripe webhook missing Stripe-Signature header — rejected")
		http.Error(w, "missing signature", http.StatusUnauthorized)
		return
	}

	event, err := h.stripeClient.ConstructWebhookEvent(payload, sigHeader, h.webhookSecret)
	if err != nil {
		h.logger.Warn("stripe webhook signature validation failed", "error", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	h.dispatch(r, event, w)
}

func (h *WebhookHandler) dispatch(r *http.Request, event stripe.Event, w http.ResponseWriter) {
	h.logger.Info("stripe webhook event received", "type", event.Type, "id", event.ID)

	var err error
	switch event.Type {
	case "payment_intent.succeeded":
		err = h.handlePaymentIntentSucceeded(r, event)
	case "payment_intent.payment_failed":
		err = h.handlePaymentIntentFailed(r, event)
	case "invoice.payment_succeeded":
		err = h.handleInvoicePaymentSucceeded(r, event)
	case "invoice.payment_failed":
		err = h.handleInvoicePaymentFailed(r, event)
	case "customer.subscription.updated":
		err = h.handleSubscriptionUpdated(r, event)
	case "customer.subscription.deleted":
		err = h.handleSubscriptionDeleted(r, event)
	default:
		h.logger.Info("unhandled stripe event type", "type", event.Type)
		w.WriteHeader(http.StatusOK) // unknown events are acknowledged immediately
		return
	}

	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Not-found errors are permanent — the record will never appear on retry.
	// Return 200 to stop Stripe's 72-hour retry loop. This also absorbs LFF-era
	// events whose Stripe IDs have no matching row in the CF database.
	if errors.Is(err, domain.ErrDonationNotFound) || errors.Is(err, domain.ErrSubscriptionNotFound) {
		h.logger.Warn("stripe webhook: no matching record, acknowledging to prevent retry",
			"type", event.Type, "id", event.ID, "error", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Transient error: log and return 500 so Stripe retries.
	// When ackUnimplemented=true (pre-production), suppress retries and ack instead.
	if h.ackUnimplemented {
		h.logger.Warn("suppressing webhook retry (ack_unimplemented=true)",
			"type", event.Type, "id", event.ID, "error", err)
		w.WriteHeader(http.StatusOK)
	} else {
		h.logger.Error("stripe webhook processing failed", "type", event.Type, "id", event.ID, "error", err)
		http.Error(w, "event processing failed", http.StatusInternalServerError)
	}
}

// handlePaymentIntentSucceeded marks the donation succeeded, records the charge ID,
// and posts the completed charge to the Ledger service for accounting and reporting.
func (h *WebhookHandler) handlePaymentIntentSucceeded(r *http.Request, event stripe.Event) error {
	if event.Data == nil {
		return fmt.Errorf("payment_intent.succeeded: event.Data is nil (event_id=%s)", event.ID)
	}
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		h.logger.Error("payment_intent.succeeded: unmarshal failed", "event_id", event.ID, "error", err)
		return fmt.Errorf("payment_intent.succeeded: unmarshal: %w", err)
	}
	chargeID := ""
	if pi.LatestCharge != nil {
		chargeID = pi.LatestCharge.ID
	}
	if err := h.donationRepo.UpdateByPaymentIntentID(r.Context(), pi.ID, models.DonationStatusSucceeded, chargeID); err != nil {
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			h.logger.Debug("payment_intent.succeeded: already processed, skipping", "pi_id", pi.ID)
			return nil
		}
		if !errors.Is(err, domain.ErrDonationNotFound) {
			h.logger.Error("payment_intent.succeeded: DB update failed", "pi_id", pi.ID, "error", err)
		}
		return fmt.Errorf("payment_intent.succeeded: db update: %w", err)
	}
	h.logger.Info("payment_intent.succeeded: donation updated", "pi_id", pi.ID)

	// Post the charge to the Ledger service.
	// Only v2 charges (initiated by this service) are posted — v1 charges from
	// LFF are handled by the Ledger-service's own webhook to avoid double-recording.
	if pi.Metadata["version"] != "v2" {
		h.logger.Debug("payment_intent.succeeded: not a v2 charge, skipping ledger post",
			"pi_id", pi.ID, "version", pi.Metadata["version"])
		return nil
	}
	initiativeID := pi.Metadata["initiative_id"]
	userID := pi.Metadata["user_id"]
	if initiativeID == "" || userID == "" {
		h.logger.Warn("payment_intent.succeeded: missing required metadata, skipping ledger post",
			"pi_id", pi.ID)
		return nil
	}
	customerID := ""
	if pi.Customer != nil {
		customerID = pi.Customer.ID
	}
	amount := int(pi.Amount)
	donorEmail := pi.Metadata["donor_email"]
	// Use the charge ID as the stable deduplication key for Ledger; fall back
	// to the PaymentIntent ID when latest_charge is not yet expanded.
	sourceTxnID := chargeID
	if sourceTxnID == "" {
		sourceTxnID = pi.ID
	}
	txn := clients.LedgerTransaction{
		ProjectID:       initiativeID,
		UserID:          userID,
		OrganizationID:  pi.Metadata["org_id"],
		AccountEmail:    donorEmail,
		SourceType:      ledgerSourceType,
		SourceTxnID:     sourceTxnID,
		SourceAccountID: customerID,
		TxnType:         ledgerTxnType,
		TxnCategory:     pi.Metadata["category"],
		Amount:          amount,
		TxnDate:         pi.Created,
	}
	// Ledger post is best-effort: the DB is already marked succeeded, so returning
	// an error would cause Stripe to retry — but the retry hits ErrAlreadyProcessed
	// and skips this path entirely. Log the failure and continue.
	if err := h.ledgerClient.PostTransaction(r.Context(), txn); err != nil {
		h.logger.Error("payment_intent.succeeded: failed to post transaction to ledger",
			"pi_id", pi.ID, "error", err)
	} else {
		h.logger.Info("payment_intent.succeeded: transaction posted to ledger",
			"pi_id", pi.ID, "initiative_id", initiativeID)
	}

	// Send donor confirmation and admin notification emails.
	// Email failures are logged but do not fail the webhook — the payment is
	// already recorded; a missing receipt is not a reason to retry the event.
	initiativeURL := ""
	if slug := pi.Metadata["initiative_slug"]; slug != "" {
		initiativeURL = h.emailService.InitiativeURL(slug)
	}
	amountFormatted := fmt.Sprintf("$%.2f", float64(amount)/100)
	donorName := pi.Metadata["donor_name"]
	if donorName == "" {
		donorName = donorEmail
	}
	initiativeName := pi.Metadata["initiative_name"]
	if initiativeName == "" {
		initiativeName = initiativeID
	}
	if donorEmail != "" {
		if emailErr := h.emailService.SendDonationConfirmationEmail(
			r.Context(), donorEmail, donorName, initiativeName, initiativeURL, amountFormatted,
		); emailErr != nil {
			h.logger.Warn("payment_intent.succeeded: donor confirmation email failed",
				"pi_id", pi.ID, "error", emailErr)
		}
	}
	if adminErr := h.emailService.SendDonationAdminNotificationEmail(
		r.Context(), pi.Metadata["owner_email"], donorName, donorEmail, initiativeName, initiativeURL, amountFormatted,
	); adminErr != nil {
		h.logger.Warn("payment_intent.succeeded: admin notification email failed",
			"pi_id", pi.ID, "error", adminErr)
	}
	return nil
}

// handlePaymentIntentFailed marks the donation failed (3DS challenge timed out or card declined).
func (h *WebhookHandler) handlePaymentIntentFailed(r *http.Request, event stripe.Event) error {
	if event.Data == nil {
		return fmt.Errorf("payment_intent.payment_failed: event.Data is nil (event_id=%s)", event.ID)
	}
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		h.logger.Error("payment_intent.payment_failed: unmarshal failed", "event_id", event.ID, "error", err)
		return fmt.Errorf("payment_intent.payment_failed: unmarshal: %w", err)
	}
	if err := h.donationRepo.UpdateByPaymentIntentID(r.Context(), pi.ID, models.DonationStatusFailed, ""); err != nil {
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			h.logger.Debug("payment_intent.payment_failed: already processed, skipping", "pi_id", pi.ID)
			return nil
		}
		if !errors.Is(err, domain.ErrDonationNotFound) {
			h.logger.Error("payment_intent.payment_failed: DB update failed", "pi_id", pi.ID, "error", err)
		}
		return fmt.Errorf("payment_intent.payment_failed: db update: %w", err)
	}
	h.logger.Info("payment_intent.payment_failed: donation updated", "pi_id", pi.ID)
	return nil
}

// handleInvoicePaymentSucceeded activates a subscription after its first (or renewal) invoice succeeds.
// For v2 subscriptions it also posts the charge to the Ledger service and sends donor + admin emails.
func (h *WebhookHandler) handleInvoicePaymentSucceeded(r *http.Request, event stripe.Event) error {
	if event.Data == nil {
		return fmt.Errorf("invoice.payment_succeeded: event.Data is nil (event_id=%s)", event.ID)
	}
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		h.logger.Error("invoice.payment_succeeded: unmarshal failed", "event_id", event.ID, "error", err)
		return fmt.Errorf("invoice.payment_succeeded: unmarshal: %w", err)
	}
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		return nil // not subscription-related; nothing to do
	}
	subID := inv.Parent.SubscriptionDetails.Subscription.ID
	// inv.Parent.SubscriptionDetails.Metadata is a snapshot of the subscription
	// metadata copied at invoice finalization — always present in the webhook payload.
	// Subscription.Metadata is NOT used: the Subscription object in an invoice webhook
	// is an unexpanded reference (only ID populated), so its Metadata map would be empty.
	subMeta := inv.Parent.SubscriptionDetails.Metadata
	if subMeta == nil {
		subMeta = map[string]string{}
	}
	if err := h.subscriptionRepo.UpdateByStripeSubscriptionID(r.Context(), subID, models.SubscriptionStatusActive); err != nil {
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			h.logger.Debug("invoice.payment_succeeded: already processed, skipping", "sub_id", subID)
			return nil
		}
		if !errors.Is(err, domain.ErrSubscriptionNotFound) {
			h.logger.Error("invoice.payment_succeeded: DB update failed",
				"sub_id", subID, "error", err)
		}
		return fmt.Errorf("invoice.payment_succeeded: db update: %w", err)
	}
	h.logger.Info("invoice.payment_succeeded: subscription activated", "sub_id", subID)

	// Only v2 charges (originated by this service) get Ledger + email treatment.
	// v1 LFF subscriptions carry no version metadata and are handled by the
	// Ledger-service's own webhook.
	if subMeta["version"] != "v2" {
		return nil
	}
	initiativeID := subMeta["initiative_id"]
	userID := subMeta["user_id"]
	if initiativeID == "" || userID == "" {
		h.logger.Warn("invoice.payment_succeeded: missing required subscription metadata, skipping ledger post",
			"sub_id", subID)
		return nil
	}
	customerID := ""
	if inv.Customer != nil {
		customerID = inv.Customer.ID
	}
	amount := int(inv.AmountPaid)
	// inv.CustomerEmail can be empty depending on how the Invoice was created.
	// Fall back to the donor_email stored in the subscription metadata snapshot.
	donorEmail := inv.CustomerEmail
	if donorEmail == "" {
		donorEmail = subMeta["donor_email"]
	}
	txn := clients.LedgerTransaction{
		ProjectID:       initiativeID,
		UserID:          userID,
		OrganizationID:  subMeta["org_id"],
		AccountEmail:    donorEmail,
		SourceType:      ledgerSourceType,
		SourceTxnID:     inv.ID, // invoice ID is the stable unique key for subscription payments
		SourceAccountID: customerID,
		TxnType:         ledgerTxnType,
		TxnCategory:     subMeta["category"],
		Amount:          amount,
		TxnDate:         inv.Created,
	}
	// Ledger post is best-effort: the DB is already marked active, so returning
	// an error would cause Stripe to retry — but the retry hits ErrAlreadyProcessed
	// and skips this path entirely. Log the failure and continue.
	if err := h.ledgerClient.PostTransaction(r.Context(), txn); err != nil {
		h.logger.Error("invoice.payment_succeeded: failed to post transaction to ledger",
			"sub_id", subID, "error", err)
	} else {
		h.logger.Info("invoice.payment_succeeded: transaction posted to ledger",
			"sub_id", subID, "initiative_id", initiativeID)
	}

	// Send emails — failures are logged but do not fail the webhook.
	initiativeURL := ""
	if slug := subMeta["initiative_slug"]; slug != "" {
		initiativeURL = h.emailService.InitiativeURL(slug)
	}
	amountFormatted := fmt.Sprintf("$%.2f", float64(amount)/100)
	donorName := subMeta["donor_name"]
	if donorName == "" {
		donorName = donorEmail
	}
	initiativeName := subMeta["initiative_name"]
	if initiativeName == "" {
		initiativeName = initiativeID
	}
	if donorEmail != "" {
		if emailErr := h.emailService.SendDonationConfirmationEmail(
			r.Context(), donorEmail, donorName, initiativeName, initiativeURL, amountFormatted,
		); emailErr != nil {
			h.logger.Warn("invoice.payment_succeeded: donor confirmation email failed",
				"sub_id", subID, "error", emailErr)
		}
	}
	if adminErr := h.emailService.SendDonationAdminNotificationEmail(
		r.Context(), subMeta["owner_email"], donorName, donorEmail, initiativeName, initiativeURL, amountFormatted,
	); adminErr != nil {
		h.logger.Warn("invoice.payment_succeeded: admin notification email failed",
			"sub_id", subID, "error", adminErr)
	}
	return nil
}

// handleInvoicePaymentFailed marks a subscription past_due when a renewal invoice fails.
// First-invoice failures (billing_reason=subscription_create) are skipped — the subscription
// is already "incomplete" in both Stripe and CF's DB; no update is needed.
func (h *WebhookHandler) handleInvoicePaymentFailed(r *http.Request, event stripe.Event) error {
	if event.Data == nil {
		return fmt.Errorf("invoice.payment_failed: event.Data is nil (event_id=%s)", event.ID)
	}
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		h.logger.Error("invoice.payment_failed: unmarshal failed", "event_id", event.ID, "error", err)
		return fmt.Errorf("invoice.payment_failed: unmarshal: %w", err)
	}
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		return nil
	}
	subID := inv.Parent.SubscriptionDetails.Subscription.ID
	// First-invoice failure: subscription is already "incomplete" in both Stripe and CF's DB.
	if inv.BillingReason == stripe.InvoiceBillingReasonSubscriptionCreate {
		h.logger.Info("invoice.payment_failed: first-invoice failure, subscription remains incomplete",
			"sub_id", subID)
		return nil
	}
	if err := h.subscriptionRepo.UpdateByStripeSubscriptionID(r.Context(), subID, models.SubscriptionStatusPastDue); err != nil {
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			h.logger.Debug("invoice.payment_failed: already processed, skipping", "sub_id", subID)
			return nil
		}
		if !errors.Is(err, domain.ErrSubscriptionNotFound) {
			h.logger.Error("invoice.payment_failed: DB update failed",
				"sub_id", subID, "error", err)
		}
		return fmt.Errorf("invoice.payment_failed: db update: %w", err)
	}
	h.logger.Info("invoice.payment_failed: subscription marked past_due", "sub_id", subID)
	return nil
}

// handleSubscriptionUpdated handles customer.subscription.updated events.
// It only acts when the update is a cancellation — identified by a non-zero CanceledAt
// and a terminal status (canceled or incomplete_expired). All other updates are ignored.
func (h *WebhookHandler) handleSubscriptionUpdated(r *http.Request, event stripe.Event) error {
	if event.Data == nil {
		return fmt.Errorf("customer.subscription.updated: event.Data is nil (event_id=%s)", event.ID)
	}
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		h.logger.Error("customer.subscription.updated: unmarshal failed", "event_id", event.ID, "error", err)
		return fmt.Errorf("customer.subscription.updated: unmarshal: %w", err)
	}
	// Only act on cancellations: canceled_at must be set and the status must be terminal.
	isCancellation := sub.CanceledAt != 0 &&
		(sub.Status == stripe.SubscriptionStatusCanceled || sub.Status == stripe.SubscriptionStatusIncompleteExpired)
	if !isCancellation {
		h.logger.Debug("customer.subscription.updated: not a cancellation, ignoring",
			"sub_id", sub.ID, "status", sub.Status)
		return nil
	}
	return h.handleSubscriptionCanceled(r, sub.ID)
}

// handleSubscriptionDeleted marks a subscription cancelled when Stripe deletes it
// (e.g. after too many failed invoices or an explicit cancellation via the Dashboard).
func (h *WebhookHandler) handleSubscriptionDeleted(r *http.Request, event stripe.Event) error {
	if event.Data == nil {
		return fmt.Errorf("customer.subscription.deleted: event.Data is nil (event_id=%s)", event.ID)
	}
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		h.logger.Error("customer.subscription.deleted: unmarshal failed", "event_id", event.ID, "error", err)
		return fmt.Errorf("customer.subscription.deleted: unmarshal: %w", err)
	}
	return h.handleSubscriptionCanceled(r, sub.ID)
}

// handleSubscriptionCanceled writes the canceled status to the database.
// It is the single exit point for both deleted and updated-to-canceled events.
func (h *WebhookHandler) handleSubscriptionCanceled(r *http.Request, subID string) error {
	if err := h.subscriptionRepo.UpdateByStripeSubscriptionID(r.Context(), subID, models.SubscriptionStatusCanceled); err != nil {
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			h.logger.Debug("subscription cancellation: already processed, skipping", "sub_id", subID)
			return nil
		}
		if !errors.Is(err, domain.ErrSubscriptionNotFound) {
			h.logger.Error("subscription cancellation: DB update failed", "sub_id", subID, "error", err)
		}
		return fmt.Errorf("subscription cancellation: db update: %w", err)
	}
	h.logger.Info("subscription cancelled", "sub_id", subID)
	return nil
}
