// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
)

// PaymentHandler holds Chi handlers for the /v1/me payment-account resource.
type PaymentHandler struct {
	svc *service.PaymentService
}

// NewPaymentHandler creates a PaymentHandler.
func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// CreateSetupIntent handles POST /v1/me/setup-intent
//
// Creates a Stripe SetupIntent for the authenticated user and returns its
// client_secret. The frontend passes this to the Stripe.js Payment Element
// to collect and 3DS-authenticate a card before attaching it.
func (h *PaymentHandler) CreateSetupIntent(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	result, err := h.svc.CreateSetupIntent(r.Context(), principal.UserID, principal.Email)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusCreated, result)
}

// AttachPaymentMethod handles POST /v1/me/payment-method
//
// After the frontend confirms the SetupIntent, it sends the resulting pm_xxx
// here to attach it to the Stripe customer and persist it as the default.
func (h *PaymentHandler) AttachPaymentMethod(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	var body struct {
		PaymentMethodID string `json:"payment_method_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PaymentMethodID == "" {
		Error(w, domain.ErrInvalidInput)
		return
	}

	card, err := h.svc.AttachPaymentMethod(r.Context(), principal.UserID, principal.Email, body.PaymentMethodID)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, card)
}

// GetPaymentAccount handles GET /v1/me/payment-account
//
// Returns the last4, brand, and expiry of the user's saved card.
func (h *PaymentHandler) GetPaymentAccount(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	card, err := h.svc.GetPaymentAccount(r.Context(), principal.UserID)
	if err != nil {
		Error(w, err)
		return
	}
	JSON(w, http.StatusOK, card)
}

// DeletePaymentMethod handles DELETE /v1/me/payment-method
//
// Detaches the user's saved card from Stripe and clears the reference in the DB.
func (h *PaymentHandler) DeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	if err := h.svc.DeletePaymentMethod(r.Context(), principal.UserID); err != nil {
		Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
