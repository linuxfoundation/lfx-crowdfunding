// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// --- CreateSetupIntent ---

func TestPaymentService_CreateSetupIntent_NewCustomer(t *testing.T) {
	// User has no stripe customer yet — one must be created and persisted.
	const wantSecret = "seti_abc_secret_xyz"
	const wantCustomerID = "cus_new"

	var createdCustomerID string
	var savedCustomerID string

	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "u1", StripeCustomerID: ""}, nil
			},
			onUpdateStripeInfo: func(_ context.Context, userUUID, customerID, _ string) error {
				if userUUID != "00000000-0000-0000-0000-000000000001" {
					t.Errorf("UpdateStripeInfo userUUID = %q, want 00000000-0000-0000-0000-000000000001", userUUID)
				}
				savedCustomerID = customerID
				return nil
			},
		},
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, userID, _ string) (string, error) {
				createdCustomerID = wantCustomerID
				return wantCustomerID, nil
			},
			onCreateSetupIntent: func(_ context.Context, customerID string) (string, error) {
				if customerID != wantCustomerID {
					t.Errorf("CreateSetupIntent got customerID %q, want %q", customerID, wantCustomerID)
				}
				return wantSecret, nil
			},
		},
	)

	result, err := svc.CreateSetupIntent(context.Background(), "u1", "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClientSecret != wantSecret {
		t.Errorf("ClientSecret = %q, want %q", result.ClientSecret, wantSecret)
	}
	if createdCustomerID != wantCustomerID {
		t.Error("CreateCustomer was not called")
	}
	if savedCustomerID != wantCustomerID {
		t.Error("UpdateStripeInfo was not called with the new customer ID")
	}
}

func TestPaymentService_CreateSetupIntent_ExistingCustomer(t *testing.T) {
	// User already has a stripe customer — CreateCustomer must NOT be called.
	const existingCustomerID = "cus_existing"
	customerCreated := false

	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{Username: "u1", StripeCustomerID: existingCustomerID}, nil
			},
		},
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				customerCreated = true
				return "cus_unexpected", nil
			},
			onCreateSetupIntent: func(_ context.Context, customerID string) (string, error) {
				if customerID != existingCustomerID {
					t.Errorf("CreateSetupIntent customerID = %q, want %q", customerID, existingCustomerID)
				}
				return "seti_existing_secret", nil
			},
		},
	)

	result, err := svc.CreateSetupIntent(context.Background(), "u1", "user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClientSecret == "" {
		t.Error("expected non-empty client_secret")
	}
	if customerCreated {
		t.Error("CreateCustomer must not be called when customer already exists")
	}
}

func TestPaymentService_CreateSetupIntent_StripeError(t *testing.T) {
	stripeErr := errors.New("stripe unavailable")

	svc := NewPaymentService(
		&testUserRepo{},
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				return "cus_1", nil
			},
			onCreateSetupIntent: func(_ context.Context, _ string) (string, error) {
				return "", stripeErr
			},
		},
	)

	_, err := svc.CreateSetupIntent(context.Background(), "u1", "user@example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, stripeErr) {
		t.Errorf("error = %v, want to wrap %v", err, stripeErr)
	}
}

// --- AttachPaymentMethod ---

func TestPaymentService_AttachPaymentMethod_EmptyID(t *testing.T) {
	svc := NewPaymentService(&testUserRepo{}, &configStripeClient{})

	_, err := svc.AttachPaymentMethod(context.Background(), "u1", "user@example.com", "")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPaymentService_AttachPaymentMethod_Success(t *testing.T) {
	wantCard := &models.CardDetails{
		PaymentMethodID: "pm_abc",
		LastFour:        "4242",
		Brand:           "visa",
		ExpiryMonth:     12,
		ExpiryYear:      2027,
	}

	var savedPMID string

	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{Username: "u1", StripeCustomerID: "cus_1"}, nil
			},
			onUpdateStripeInfo: func(_ context.Context, _, _, pmID string) error {
				savedPMID = pmID
				return nil
			},
		},
		&configStripeClient{
			onAttachPaymentMethod: func(_ context.Context, _, _ string) (*models.CardDetails, error) {
				return wantCard, nil
			},
		},
	)

	card, err := svc.AttachPaymentMethod(context.Background(), "u1", "user@example.com", "pm_abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.LastFour != wantCard.LastFour {
		t.Errorf("LastFour = %q, want %q", card.LastFour, wantCard.LastFour)
	}
	if savedPMID != "pm_abc" {
		t.Errorf("UpdateStripeInfo called with pmID %q, want pm_abc", savedPMID)
	}
}

func TestPaymentService_AttachPaymentMethod_StripeError(t *testing.T) {
	stripeErr := errors.New("card declined")

	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{Username: "u1", StripeCustomerID: "cus_1"}, nil
			},
		},
		&configStripeClient{
			onAttachPaymentMethod: func(_ context.Context, _, _ string) (*models.CardDetails, error) {
				return nil, stripeErr
			},
		},
	)

	_, err := svc.AttachPaymentMethod(context.Background(), "u1", "user@example.com", "pm_abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, stripeErr) {
		t.Errorf("error = %v, want to wrap %v", err, stripeErr)
	}
}

// --- GetPaymentAccount ---

func TestPaymentService_GetPaymentAccount_NoCard(t *testing.T) {
	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{Username: "u1", StripeDefaultPaymentMethod: ""}, nil
			},
		},
		&configStripeClient{},
	)

	_, err := svc.GetPaymentAccount(context.Background(), "u1")
	if !errors.Is(err, domain.ErrPaymentMethodNotFound) {
		t.Errorf("expected ErrPaymentMethodNotFound, got %v", err)
	}
}

func TestPaymentService_GetPaymentAccount_Success(t *testing.T) {
	wantCard := &models.CardDetails{LastFour: "1234", Brand: "mastercard"}

	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{Username: "u1", StripeDefaultPaymentMethod: "pm_saved"}, nil
			},
		},
		&configStripeClient{
			onGetPaymentMethod: func(_ context.Context, pmID string) (*models.CardDetails, error) {
				if pmID != "pm_saved" {
					t.Errorf("GetPaymentMethod pmID = %q, want pm_saved", pmID)
				}
				return wantCard, nil
			},
		},
	)

	card, err := svc.GetPaymentAccount(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.LastFour != wantCard.LastFour {
		t.Errorf("LastFour = %q, want %q", card.LastFour, wantCard.LastFour)
	}
}

// --- DeletePaymentMethod ---

func TestPaymentService_DeletePaymentMethod_NoCard(t *testing.T) {
	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{Username: "u1"}, nil
			},
		},
		&configStripeClient{},
	)

	err := svc.DeletePaymentMethod(context.Background(), "u1")
	if !errors.Is(err, domain.ErrPaymentMethodNotFound) {
		t.Errorf("expected ErrPaymentMethodNotFound, got %v", err)
	}
}

func TestPaymentService_DeletePaymentMethod_Success(t *testing.T) {
	detachCalled := false
	clearCalled := false

	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{Username: "u1", StripeDefaultPaymentMethod: "pm_saved"}, nil
			},
			onClearStripePaymentMethod: func(_ context.Context, _ string) error {
				clearCalled = true
				return nil
			},
		},
		&configStripeClient{
			onDetachPaymentMethod: func(_ context.Context, pmID string) error {
				if pmID != "pm_saved" {
					t.Errorf("DetachPaymentMethod pmID = %q, want pm_saved", pmID)
				}
				detachCalled = true
				return nil
			},
		},
	)

	if err := svc.DeletePaymentMethod(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !detachCalled {
		t.Error("DetachPaymentMethod was not called")
	}
	if !clearCalled {
		t.Error("ClearStripePaymentMethod was not called")
	}
}

func TestPaymentService_DeletePaymentMethod_StripeError(t *testing.T) {
	stripeErr := errors.New("stripe detach failed")

	svc := NewPaymentService(
		&testUserRepo{
			onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
				return &models.User{Username: "u1", StripeDefaultPaymentMethod: "pm_saved"}, nil
			},
		},
		&configStripeClient{
			onDetachPaymentMethod: func(_ context.Context, _ string) error {
				return stripeErr
			},
		},
	)

	err := svc.DeletePaymentMethod(context.Background(), "u1")
	if !errors.Is(err, stripeErr) {
		t.Errorf("error = %v, want to wrap %v", err, stripeErr)
	}
}
