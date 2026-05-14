// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	stripe "github.com/stripe/stripe-go/v82"
)

// WebhookHandler handles inbound Stripe webhook events.
// Signature validation is ALWAYS performed before processing — never skip this check.
type WebhookHandler struct {
	stripeClient     clients.StripeClient
	webhookSecret    string
	logger           *slog.Logger
	ackUnimplemented bool // when true, reply 200 for known-but-unimplemented events
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(stripeClient clients.StripeClient, webhookSecret string, logger *slog.Logger, ackUnimplemented bool) *WebhookHandler {
	return &WebhookHandler{
		stripeClient:     stripeClient,
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
	h.logger.Info("stripe webhook event received",
		"type", event.Type,
		"id", event.ID,
	)
	var handled bool
	switch event.Type {
	case "payment_intent.succeeded":
		handled = h.handlePaymentIntentSucceeded(r, event)
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
		// Return 501 so Stripe keeps the event in its retry queue until
		// persistence is implemented.
		http.Error(w, "event handler not yet implemented", http.StatusNotImplemented)
	}
}

func (h *WebhookHandler) handlePaymentIntentSucceeded(r *http.Request, event stripe.Event) bool {
	h.logger.Info("payment_intent.succeeded", "event_id", event.ID)
	// TODO: update donation status in DB to "succeeded" using metadata.initiative_id
	return false
}

func (h *WebhookHandler) handleSubscriptionDeleted(r *http.Request, event stripe.Event) bool {
	h.logger.Info("customer.subscription.deleted", "event_id", event.ID)
	// TODO: mark subscription as cancelled in DB
	return false
}
