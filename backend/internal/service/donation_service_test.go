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

func newDonationSvc(
	donRepo *testDonationRepo,
	initRepo *mockInitiativeRepo,
	userRepo *testUserRepo,
	stripe *configStripeClient,
) *DonationService {
	return NewDonationService(donRepo, initRepo, userRepo, stripe)
}

func acceptingInitiative() *mockInitiativeRepo {
	return &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", AcceptFunding: true}}
}

// --- input validation ---

func TestDonationService_Create_ZeroAmount(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 0})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDonationService_Create_NegativeAmount(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: -100})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestDonationService_Create_MissingPaymentMethod(t *testing.T) {
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 1000})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing PM, got %v", err)
	}
}

func TestDonationService_Create_InitiativeNotFound(t *testing.T) {
	notFound := errors.New("initiative not found")
	initRepo := &mockInitiativeRepo{err: notFound}
	svc := newDonationSvc(&testDonationRepo{}, initRepo, &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-missing", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 100, StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, notFound) {
		t.Errorf("expected initiative-not-found error, got %v", err)
	}
}

func TestDonationService_Create_InitiativeNotAccepting(t *testing.T) {
	initRepo := &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", AcceptFunding: false}}
	svc := newDonationSvc(&testDonationRepo{}, initRepo, &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 500, StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// --- happy path: new customer, immediate success (no 3DS) ---

func TestDonationService_Create_NewCustomerImmediateSuccess(t *testing.T) {
	customerCreated := false

	donRepo := &testDonationRepo{}
	userRepo := &testUserRepo{
		onGetByUserID: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{UserID: "u1", StripeCustomerID: ""}, nil
		},
	}
	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, userID, email string) (string, error) {
			customerCreated = true
			return "cus_new", nil
		},
		onCreatePaymentIntent: func(_ context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			if req.CustomerID != "cus_new" {
				t.Errorf("PaymentIntent CustomerID = %q, want cus_new", req.CustomerID)
			}
			if req.AmountCents != 2000 {
				t.Errorf("AmountCents = %d, want 2000", req.AmountCents)
			}
			return &models.PaymentIntent{
				ID:     "pi_test",
				Status: "succeeded",
			}, nil
		},
	}

	svc := newDonationSvc(donRepo, acceptingInitiative(), userRepo, stripe)
	don, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 2000, StripePaymentMethodID: "pm_abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if don.Status != "succeeded" {
		t.Errorf("Status = %q, want succeeded", don.Status)
	}
	if don.StripePaymentIntentID != "pi_test" {
		t.Errorf("StripePaymentIntentID = %q, want pi_test", don.StripePaymentIntentID)
	}
	if don.ClientSecret != "" {
		t.Errorf("ClientSecret should be empty for succeeded payment, got %q", don.ClientSecret)
	}
	if !customerCreated {
		t.Error("CreateCustomer was not called for new user")
	}
}

// --- 3DS required: existing customer, returns client_secret ---

func TestDonationService_Create_ExistingCustomer3DS(t *testing.T) {
	const existingCustomerID = "cus_existing"
	const wantSecret = "pi_test_secret_3ds"

	customerCreated := false

	userRepo := &testUserRepo{
		onGetByUserID: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{UserID: "u1", StripeCustomerID: existingCustomerID}, nil
		},
	}
	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
			customerCreated = true
			return "cus_unexpected", nil
		},
		onCreatePaymentIntent: func(_ context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
			if req.CustomerID != existingCustomerID {
				t.Errorf("CustomerID = %q, want %q", req.CustomerID, existingCustomerID)
			}
			return &models.PaymentIntent{
				ID:           "pi_3ds",
				Status:       "requires_action",
				ClientSecret: wantSecret,
			}, nil
		},
	}

	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), userRepo, stripe)
	don, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 5000, StripePaymentMethodID: "pm_eu"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if don.Status != "requires_action" {
		t.Errorf("Status = %q, want requires_action", don.Status)
	}
	if don.ClientSecret != wantSecret {
		t.Errorf("ClientSecret = %q, want %q", don.ClientSecret, wantSecret)
	}
	if customerCreated {
		t.Error("CreateCustomer must not be called when customer already exists")
	}
}

// --- stripe error propagation ---

func TestDonationService_Create_StripePaymentIntentError(t *testing.T) {
	stripeErr := errors.New("stripe error")

	svc := newDonationSvc(
		&testDonationRepo{},
		acceptingInitiative(),
		&testUserRepo{},
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				return "cus_1", nil
			},
			onCreatePaymentIntent: func(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
				return nil, stripeErr
			},
		},
	)

	_, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, stripeErr) {
		t.Errorf("error = %v, want to wrap %v", err, stripeErr)
	}
}

// --- DB error propagation ---

func TestDonationService_Create_UserRepoTransientError(t *testing.T) {
	dbErr := errors.New("connection reset")

	userRepo := &testUserRepo{
		onGetByUserID: func(_ context.Context, _ string) (*models.User, error) {
			return nil, dbErr
		},
	}
	svc := newDonationSvc(&testDonationRepo{}, acceptingInitiative(), userRepo, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, dbErr) {
		t.Errorf("error = %v, want to wrap %v", err, dbErr)
	}
}

func TestDonationService_Create_DBError(t *testing.T) {
	dbErr := errors.New("db write failed")

	donRepo := &testDonationRepo{
		onCreate: func(_ context.Context, _ *models.Donation) (*models.Donation, error) {
			return nil, dbErr
		},
	}
	svc := newDonationSvc(
		donRepo,
		acceptingInitiative(),
		&testUserRepo{},
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				return "cus_1", nil
			},
			onCreatePaymentIntent: func(_ context.Context, _ models.PaymentIntentRequest) (*models.PaymentIntent, error) {
				return &models.PaymentIntent{ID: "pi_1", Status: "succeeded"}, nil
			},
		},
	)

	_, err := svc.Create(context.Background(), "init-1", "u1", "u@example.com",
		models.DonationCreateInput{AmountCents: 1000, StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, dbErr) {
		t.Errorf("error = %v, want to wrap %v", err, dbErr)
	}
}
