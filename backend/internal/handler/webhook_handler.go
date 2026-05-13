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
	stripeClient  clients.StripeClient
	webhookSecret string
	logger        *slog.Logger
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(stripeClient clients.StripeClient, webhookSecret string, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		stripeClient:  stripeClient,
		webhookSecret: webhookSecret,
		logger:        logger,
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

	h.dispatch(r, event)
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) dispatch(r *http.Request, event stripe.Event) {
	h.logger.Info("stripe webhook event received",
		"type", event.Type,
		"id", event.ID,
	)
	switch event.Type {
	case "payment_intent.succeeded":
		h.handlePaymentIntentSucceeded(r, event)
	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(r, event)
	default:
		h.logger.Info("unhandled stripe event type", "type", event.Type)
	}
}

func (h *WebhookHandler) handlePaymentIntentSucceeded(r *http.Request, event stripe.Event) {
	h.logger.Info("payment_intent.succeeded", "event_id", event.ID)
	// TODO: update donation status in DB to "succeeded" using metadata.initiative_id
}

func (h *WebhookHandler) handleSubscriptionDeleted(r *http.Request, event stripe.Event) {
	h.logger.Info("customer.subscription.deleted", "event_id", event.ID)
	// TODO: mark subscription as cancelled in DB
}
