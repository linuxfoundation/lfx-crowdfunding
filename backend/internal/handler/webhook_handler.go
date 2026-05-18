// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	stripe "github.com/stripe/stripe-go/v82"
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

	var handled bool
	switch event.Type {
	case "payment_intent.succeeded":
		handled = h.handlePaymentIntentSucceeded(r, event)
	case "payment_intent.payment_failed":
		handled = h.handlePaymentIntentFailed(r, event)
	case "invoice.payment_succeeded":
		handled = h.handleInvoicePaymentSucceeded(r, event)
	case "invoice.payment_failed":
		handled = h.handleInvoicePaymentFailed(r, event)
	case "customer.subscription.deleted":
		handled = h.handleSubscriptionDeleted(r, event)
	default:
		h.logger.Info("unhandled stripe event type", "type", event.Type)
		w.WriteHeader(http.StatusOK) // unknown events are acknowledged immediately
		return
	}

	if handled {
		w.WriteHeader(http.StatusOK)
	} else if h.ackUnimplemented {
		// STRIPE_WEBHOOK_ACK_UNIMPLEMENTED=true: silently ack so Stripe does not
		// retry. Use this in pre-production environments receiving real events
		// before DB persistence is implemented.
		h.logger.Warn("acknowledging unimplemented stripe event to suppress retries",
			"type", event.Type, "id", event.ID)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "event handler not yet implemented", http.StatusNotImplemented)
	}
}

// handlePaymentIntentSucceeded marks the donation succeeded and records the charge ID.
func (h *WebhookHandler) handlePaymentIntentSucceeded(r *http.Request, event stripe.Event) bool {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		h.logger.Error("payment_intent.succeeded: unmarshal failed", "event_id", event.ID, "error", err)
		return false
	}
	chargeID := ""
	if pi.LatestCharge != nil {
		chargeID = pi.LatestCharge.ID
	}
	if err := h.donationRepo.UpdateByPaymentIntentID(r.Context(), pi.ID, "succeeded", chargeID); err != nil {
		h.logger.Error("payment_intent.succeeded: DB update failed", "pi_id", pi.ID, "error", err)
		return false
	}
	h.logger.Info("payment_intent.succeeded: donation updated", "pi_id", pi.ID)
	return true
}

// handlePaymentIntentFailed marks the donation failed (3DS challenge timed out or card declined).
func (h *WebhookHandler) handlePaymentIntentFailed(r *http.Request, event stripe.Event) bool {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		h.logger.Error("payment_intent.payment_failed: unmarshal failed", "event_id", event.ID, "error", err)
		return false
	}
	if err := h.donationRepo.UpdateByPaymentIntentID(r.Context(), pi.ID, "failed", ""); err != nil {
		h.logger.Error("payment_intent.payment_failed: DB update failed", "pi_id", pi.ID, "error", err)
		return false
	}
	h.logger.Info("payment_intent.payment_failed: donation updated", "pi_id", pi.ID)
	return true
}

// handleInvoicePaymentSucceeded activates a subscription after its first (or renewal) invoice succeeds.
func (h *WebhookHandler) handleInvoicePaymentSucceeded(r *http.Request, event stripe.Event) bool {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		h.logger.Error("invoice.payment_succeeded: unmarshal failed", "event_id", event.ID, "error", err)
		return false
	}
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		return true // not subscription-related; nothing to do
	}
	subID := inv.Parent.SubscriptionDetails.Subscription.ID
	if err := h.subscriptionRepo.UpdateByStripeSubscriptionID(r.Context(), subID, "active"); err != nil {
		h.logger.Error("invoice.payment_succeeded: DB update failed",
			"sub_id", subID, "error", err)
		return false
	}
	h.logger.Info("invoice.payment_succeeded: subscription activated", "sub_id", subID)
	return true
}

// handleInvoicePaymentFailed marks a subscription past_due when renewal fails.
func (h *WebhookHandler) handleInvoicePaymentFailed(r *http.Request, event stripe.Event) bool {
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		h.logger.Error("invoice.payment_failed: unmarshal failed", "event_id", event.ID, "error", err)
		return false
	}
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		return true
	}
	subID := inv.Parent.SubscriptionDetails.Subscription.ID
	if err := h.subscriptionRepo.UpdateByStripeSubscriptionID(r.Context(), subID, "past_due"); err != nil {
		h.logger.Error("invoice.payment_failed: DB update failed",
			"sub_id", subID, "error", err)
		return false
	}
	h.logger.Info("invoice.payment_failed: subscription marked past_due", "sub_id", subID)
	return true
}

// handleSubscriptionDeleted marks a subscription cancelled when Stripe deletes it
// (e.g. after too many failed invoices or an explicit cancellation via the Dashboard).
func (h *WebhookHandler) handleSubscriptionDeleted(r *http.Request, event stripe.Event) bool {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		h.logger.Error("customer.subscription.deleted: unmarshal failed", "event_id", event.ID, "error", err)
		return false
	}
	if err := h.subscriptionRepo.UpdateByStripeSubscriptionID(r.Context(), sub.ID, "canceled"); err != nil {
		h.logger.Error("customer.subscription.deleted: DB update failed", "sub_id", sub.ID, "error", err)
		return false
	}
	h.logger.Info("customer.subscription.deleted: subscription cancelled", "sub_id", sub.ID)
	return true
}
