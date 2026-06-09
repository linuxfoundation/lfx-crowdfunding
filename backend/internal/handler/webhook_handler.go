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

// WebhookHandler handles inbound Stripe webhook events.
// Signature validation is ALWAYS performed before processing — never skip this check.
type WebhookHandler struct {
	stripeClient     clients.StripeClient
	donationRepo     domain.DonationRepository
	subscriptionRepo domain.SubscriptionRepository
	webhookSecret    string
	logger           *slog.Logger
	ackUnimplemented bool // when true, reply 200 for known-but-unimplemented events
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(
	stripeClient clients.StripeClient,
	donationRepo domain.DonationRepository,
	subscriptionRepo domain.SubscriptionRepository,
	webhookSecret string,
	logger *slog.Logger,
	ackUnimplemented bool,
) *WebhookHandler {
	return &WebhookHandler{
		stripeClient:     stripeClient,
		donationRepo:     donationRepo,
		subscriptionRepo: subscriptionRepo,
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

// handlePaymentIntentSucceeded marks the donation succeeded and records the charge ID.
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
		if !errors.Is(err, domain.ErrDonationNotFound) {
			h.logger.Error("payment_intent.succeeded: DB update failed", "pi_id", pi.ID, "error", err)
		}
		return fmt.Errorf("payment_intent.succeeded: db update: %w", err)
	}
	h.logger.Info("payment_intent.succeeded: donation updated", "pi_id", pi.ID)
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
		if !errors.Is(err, domain.ErrDonationNotFound) {
			h.logger.Error("payment_intent.payment_failed: DB update failed", "pi_id", pi.ID, "error", err)
		}
		return fmt.Errorf("payment_intent.payment_failed: db update: %w", err)
	}
	h.logger.Info("payment_intent.payment_failed: donation updated", "pi_id", pi.ID)
	return nil
}

// handleInvoicePaymentSucceeded activates a subscription after its first (or renewal) invoice succeeds.
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
	if err := h.subscriptionRepo.UpdateByStripeSubscriptionID(r.Context(), subID, models.SubscriptionStatusActive); err != nil {
		if !errors.Is(err, domain.ErrSubscriptionNotFound) {
			h.logger.Error("invoice.payment_succeeded: DB update failed",
				"sub_id", subID, "error", err)
		}
		return fmt.Errorf("invoice.payment_succeeded: db update: %w", err)
	}
	h.logger.Info("invoice.payment_succeeded: subscription activated", "sub_id", subID)
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
	// First-invoice failure: subscription is already "incomplete" in both Stripe and CF's DB.
	if inv.BillingReason == stripe.InvoiceBillingReasonSubscriptionCreate {
		h.logger.Info("invoice.payment_failed: first-invoice failure, subscription remains incomplete",
			"sub_id", inv.Parent.SubscriptionDetails.Subscription.ID)
		return nil
	}
	subID := inv.Parent.SubscriptionDetails.Subscription.ID
	if err := h.subscriptionRepo.UpdateByStripeSubscriptionID(r.Context(), subID, models.SubscriptionStatusPastDue); err != nil {
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
		if !errors.Is(err, domain.ErrSubscriptionNotFound) {
			h.logger.Error("subscription cancellation: DB update failed", "sub_id", subID, "error", err)
		}
		return fmt.Errorf("subscription cancellation: db update: %w", err)
	}
	h.logger.Info("subscription cancelled", "sub_id", subID)
	return nil
}
