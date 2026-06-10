// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

func newSubscriptionSvc(
	subRepo *testSubscriptionRepo,
	initRepo *mockInitiativeRepo,
	userRepo *testUserRepo,
	stripe *configStripeClient,
) *SubscriptionService {
	return NewSubscriptionService(subRepo, initRepo, userRepo, stripe)
}

// --- input validation ---

func TestSubscriptionService_Create_ZeroAmount(t *testing.T) {
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 0, Frequency: "monthly"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSubscriptionService_Create_MissingFrequency(t *testing.T) {
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 1000, Frequency: ""})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSubscriptionService_Create_MissingPaymentMethod(t *testing.T) {
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 1000, Frequency: "monthly", StripePaymentMethodID: ""})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSubscriptionService_Create_NoStripeProduct(t *testing.T) {
	initRepo := &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", AcceptFunding: true, StripeProductID: ""}}
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, initRepo, &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 1000, Frequency: "monthly", StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing StripeProductID, got %v", err)
	}
}

func TestSubscriptionService_Create_InitiativeNotAccepting(t *testing.T) {
	initRepo := &mockInitiativeRepo{initiative: &models.Initiative{ID: "init-1", AcceptFunding: false}}
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, initRepo, &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 500, Frequency: "monthly"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

// --- happy path: new customer, subscription activates immediately (no 3DS) ---

func TestSubscriptionService_Create_NewCustomerActive(t *testing.T) {
	customerCreated := false
	var savedSubID, savedPriceID string

	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "u1", Email: "u1@test.example", StripeCustomerID: ""}, nil
		},
	}
	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
			customerCreated = true
			return "cus_new", nil
		},
		onGetOrCreatePrice: func(_ context.Context, productID, _ string, amountCents int64, frequency string, _ string) (string, error) {
			if productID != "prod-test" {
				t.Errorf("GetOrCreatePrice productID = %q, want prod-test", productID)
			}
			if amountCents != 1000 {
				t.Errorf("GetOrCreatePrice amountCents = %d, want 1000", amountCents)
			}
			if frequency != "monthly" {
				t.Errorf("GetOrCreatePrice frequency = %q, want monthly", frequency)
			}
			return "price_new", nil
		},
		onCreateSubscription: func(_ context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
			if req.StripeCustomerID != "cus_new" {
				t.Errorf("CreateSubscription CustomerID = %q, want cus_new", req.StripeCustomerID)
			}
			if req.StripePriceID != "price_new" {
				t.Errorf("CreateSubscription PriceID = %q, want price_new", req.StripePriceID)
			}
			return &models.StripeSubscriptionResult{
				SubscriptionID: "sub_stripe",
				PriceID:        "price_new",
				Status:         "active",
			}, nil
		},
	}
	subRepo := &testSubscriptionRepo{
		onCreate: func(_ context.Context, s *models.Subscription) (*models.Subscription, error) {
			savedSubID = s.StripeSubscriptionID
			savedPriceID = s.StripePriceID
			return s, nil
		},
	}

	svc := newSubscriptionSvc(subRepo, acceptingInitiative(), userRepo, stripe)
	sub, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{
			AmountCents:           1000,
			Frequency:             "monthly",
			StripePaymentMethodID: "pm_test",
			IdempotencyKey:        "idem-key-sub",
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.Status != "active" {
		t.Errorf("Status = %q, want active", sub.Status)
	}
	if sub.ClientSecret != "" {
		t.Errorf("ClientSecret should be empty for active subscription, got %q", sub.ClientSecret)
	}
	if savedSubID != "sub_stripe" {
		t.Errorf("stored StripeSubscriptionID = %q, want sub_stripe", savedSubID)
	}
	if savedPriceID != "price_new" {
		t.Errorf("stored StripePriceID = %q, want price_new", savedPriceID)
	}
	if !customerCreated {
		t.Error("CreateCustomer was not called for new user")
	}
}

// --- 3DS on first invoice: existing customer, returns client_secret ---

func TestSubscriptionService_Create_ExistingCustomer3DS(t *testing.T) {
	const existingCus = "cus_existing"
	const wantSecret = "pi_first_invoice_secret"

	customerCreated := false

	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: "u1", Email: "u1@test.example", StripeCustomerID: existingCus}, nil
		},
	}
	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
			customerCreated = true
			return "cus_unexpected", nil
		},
		onGetOrCreatePrice: func(_ context.Context, _ string, _ string, _ int64, _ string, _ string) (string, error) {
			return "price_1", nil
		},
		onCreateSubscription: func(_ context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
			if req.StripeCustomerID != existingCus {
				t.Errorf("CustomerID = %q, want %q", req.StripeCustomerID, existingCus)
			}
			return &models.StripeSubscriptionResult{
				SubscriptionID: "sub_3ds",
				PriceID:        "price_1",
				Status:         "incomplete",
				ClientSecret:   wantSecret,
			}, nil
		},
	}

	svc := newSubscriptionSvc(&testSubscriptionRepo{}, acceptingInitiative(), userRepo, stripe)
	sub, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 2000, Frequency: "monthly", StripePaymentMethodID: "pm_test", IdempotencyKey: "idem-key-sub"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.Status != "incomplete" {
		t.Errorf("Status = %q, want incomplete", sub.Status)
	}
	if sub.ClientSecret != wantSecret {
		t.Errorf("ClientSecret = %q, want %q", sub.ClientSecret, wantSecret)
	}
	if customerCreated {
		t.Error("CreateCustomer must not be called when customer already exists")
	}
}

// --- stripe error propagation ---

func TestSubscriptionService_Create_StripeError(t *testing.T) {
	stripeErr := errors.New("stripe error")

	svc := newSubscriptionSvc(
		&testSubscriptionRepo{},
		acceptingInitiative(),
		&testUserRepo{},
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				return "cus_1", nil
			},
			onGetOrCreatePrice: func(_ context.Context, _ string, _ string, _ int64, _ string, _ string) (string, error) {
				return "price_1", nil
			},
			onCreateSubscription: func(_ context.Context, _ models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
				return nil, stripeErr
			},
		},
	)

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 500, Frequency: "monthly", StripePaymentMethodID: "pm_test", IdempotencyKey: "idem-key-sub"})
	if !errors.Is(err, stripeErr) {
		t.Errorf("error = %v, want to wrap %v", err, stripeErr)
	}
}

// --- Cancel ---

func TestSubscriptionService_Create_UserRepoTransientError(t *testing.T) {
	dbErr := errors.New("connection reset")

	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, dbErr
		},
	}
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, acceptingInitiative(), userRepo, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 1000, Frequency: "monthly", StripePaymentMethodID: "pm_test", IdempotencyKey: "idem-key-sub"})
	if !errors.Is(err, dbErr) {
		t.Errorf("error = %v, want to wrap %v", err, dbErr)
	}
}

// --- Cancel ---

func TestSubscriptionService_Create_MissingIdempotencyKey(t *testing.T) {
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	// All fields valid except IdempotencyKey — must fail before Stripe calls.
	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 1000, Frequency: "monthly", StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for missing idempotency key, got %v", err)
	}
}

func TestSubscriptionService_Create_InvalidFrequency(t *testing.T) {
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{AmountCents: 1000, Frequency: "biweekly", StripePaymentMethodID: "pm_test"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for unsupported frequency, got %v", err)
	}
}

func TestSubscriptionService_Cancel_WrongOwner(t *testing.T) {
	subRepo := &testSubscriptionRepo{
		onGetByID: func(_ context.Context, id string) (*models.Subscription, error) {
			return &models.Subscription{ID: id, UserID: "owner-user"}, nil
		},
	}
	svc := newSubscriptionSvc(subRepo, acceptingInitiative(), &testUserRepo{}, &configStripeClient{})

	err := svc.Cancel(context.Background(), "sub-1", "different-user")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestSubscriptionService_Cancel_Success(t *testing.T) {
	const subID = "sub-local-1"
	const stripeSubID = "sub_stripe_1"

	cancelCalled := false
	var updatedStatus string

	subRepo := &testSubscriptionRepo{
		onGetByID: func(_ context.Context, _ string) (*models.Subscription, error) {
			return &models.Subscription{
				ID:                   subID,
				UserID:               "00000000-0000-0000-0000-000000000001",
				StripeSubscriptionID: stripeSubID,
				Status:               "active",
			}, nil
		},
		onUpdate: func(_ context.Context, s *models.Subscription) (*models.Subscription, error) {
			updatedStatus = s.Status
			return s, nil
		},
	}

	svc := newSubscriptionSvc(
		subRepo,
		acceptingInitiative(),
		&testUserRepo{},
		&configStripeClient{
			onCancelSubscription: func(_ context.Context, id string) error {
				if id != stripeSubID {
					t.Errorf("CancelSubscription id = %q, want %q", id, stripeSubID)
				}
				cancelCalled = true
				return nil
			},
		},
	)

	if err := svc.Cancel(context.Background(), subID, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cancelCalled {
		t.Error("CancelSubscription was not called")
	}
	if updatedStatus != "canceled" {
		t.Errorf("updated status = %q, want canceled", updatedStatus)
	}
}

func TestSubscriptionService_Create_UserNotFound_DescriptiveError(t *testing.T) {
	// When the user has not yet synced their profile, GetByUsername returns
	// ErrUserNotFound. The service converts this to ErrProfileNotSynced with
	// a PATCH /v1/me hint so the API response is actionable (maps to 400).
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return nil, domain.ErrUserNotFound
		},
	}
	svc := newSubscriptionSvc(&testSubscriptionRepo{}, acceptingInitiative(), userRepo, &configStripeClient{})

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{
			AmountCents:           1000,
			Frequency:             "monthly",
			StripePaymentMethodID: "pm_test",
			IdempotencyKey:        "key-1",
		})

	if !errors.Is(err, domain.ErrProfileNotSynced) {
		t.Fatalf("expected ErrProfileNotSynced, got %v", err)
	}
	if !strings.Contains(err.Error(), "PATCH /v1/me") {
		t.Errorf("error should mention PATCH /v1/me, got: %v", err)
	}
}

func TestSubscriptionService_Create_EmptyEmail_RequiresProfileSync(t *testing.T) {
	// A legacy/migrated user row may exist without an email address.
	// Stripe requires a non-empty email, so the service must fail fast.
	userRepo := &testUserRepo{
		onGetByUsername: func(_ context.Context, _ string) (*models.User, error) {
			return &models.User{ID: "u-uuid", Username: "u1", Email: ""}, nil
		},
	}
	customerCreated := false
	svc := newSubscriptionSvc(
		&testSubscriptionRepo{},
		acceptingInitiative(),
		userRepo,
		&configStripeClient{
			onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
				customerCreated = true
				return "cus_new", nil
			},
		},
	)

	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{
			AmountCents:           1000,
			Frequency:             "monthly",
			StripePaymentMethodID: "pm_test",
			IdempotencyKey:        "key-2",
		})

	if !errors.Is(err, domain.ErrProfileNotSynced) {
		t.Fatalf("expected ErrProfileNotSynced for empty email, got %v", err)
	}
	if !strings.Contains(err.Error(), "PATCH /v1/me") {
		t.Errorf("error should mention PATCH /v1/me, got: %v", err)
	}
	if customerCreated {
		t.Error("CreateCustomer must not be called when user email is empty")
	}
}

// --- Stripe product auto-heal ---

// TestSubscriptionService_Create_StaleProductAutoHeals verifies that when
// GetOrCreatePrice returns a resource_missing error for the product param,
// the service creates a replacement Stripe product, persists the new ID, and
// retries GetOrCreatePrice transparently so the subscription succeeds.
func TestSubscriptionService_Create_StaleProductAutoHeals(t *testing.T) {
	const staleProductID = "prod_stale"
	const newProductID = "prod_new"
	const newPriceID = "price_healed"

	var persistedProductID string
	priceCallCount := 0

	// Initiative starts with the stale product ID.
	initRepo := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID: "init-1", Name: "My Initiative",
			AcceptFunding: true, StripeProductID: staleProductID,
		},
	}
	// Capture the UpdateStripeProductID call.
	initRepo.onUpdateStripeProductID = func(_ context.Context, _, id string) error {
		persistedProductID = id
		return nil
	}

	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
			return "cus_1", nil
		},
		onCreateProduct: func(_ context.Context, _, _ string) (string, error) {
			return newProductID, nil
		},
		onGetOrCreatePrice: func(_ context.Context, productID, _ string, _ int64, _ string, _ string) (string, error) {
			priceCallCount++
			if productID == staleProductID {
				// First call: simulate Stripe returning resource_missing for the product.
				return "", fmt.Errorf("stripe create price: resource_missing product")
			}
			// Second call (after auto-heal): succeed with the new product.
			if productID != newProductID {
				t.Errorf("retry call used productID=%q, want %q", productID, newProductID)
			}
			return newPriceID, nil
		},
		onCreateSubscription: func(_ context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
			if req.StripePriceID != newPriceID {
				t.Errorf("CreateSubscription PriceID = %q, want %q", req.StripePriceID, newPriceID)
			}
			return &models.StripeSubscriptionResult{
				SubscriptionID: "sub_healed",
				PriceID:        newPriceID,
				Status:         "active",
			}, nil
		},
	}

	svc := newSubscriptionSvc(&testSubscriptionRepo{}, initRepo, &testUserRepo{}, stripe)
	sub, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{
			AmountCents:           1000,
			Frequency:             "monthly",
			StripePaymentMethodID: "pm_test",
			IdempotencyKey:        "key-heal",
		})
	if err != nil {
		t.Fatalf("expected auto-heal to succeed, got error: %v", err)
	}
	if sub.Status != "active" {
		t.Errorf("Status = %q, want active", sub.Status)
	}
	if persistedProductID != newProductID {
		t.Errorf("UpdateStripeProductID called with %q, want %q", persistedProductID, newProductID)
	}
	if priceCallCount != 2 {
		t.Errorf("GetOrCreatePrice called %d time(s), want 2 (initial fail + retry)", priceCallCount)
	}
}

// TestSubscriptionService_Create_StaleProductHealPersistFails verifies that if
// UpdateStripeProductID fails after creating the new product, the error is
// propagated and the subscription is not created.
func TestSubscriptionService_Create_StaleProductHealPersistFails(t *testing.T) {
	persistErr := fmt.Errorf("db connection lost")

	initRepo := &mockInitiativeRepo{
		initiative: &models.Initiative{
			ID: "init-1", Name: "My Initiative",
			AcceptFunding: true, StripeProductID: "prod_stale",
		},
	}
	initRepo.onUpdateStripeProductID = func(_ context.Context, _, _ string) error {
		return persistErr
	}

	stripe := &configStripeClient{
		onCreateCustomer: func(_ context.Context, _, _ string) (string, error) {
			return "cus_1", nil
		},
		onCreateProduct: func(_ context.Context, _, _ string) (string, error) {
			return "prod_new", nil
		},
		onGetOrCreatePrice: func(_ context.Context, _ string, _ string, _ int64, _ string, _ string) (string, error) {
			return "", fmt.Errorf("stripe create price: resource_missing product")
		},
	}

	svc := newSubscriptionSvc(&testSubscriptionRepo{}, initRepo, &testUserRepo{}, stripe)
	_, err := svc.Create(context.Background(), "init-1", "u1",
		models.SubscriptionCreateInput{
			AmountCents:           1000,
			Frequency:             "monthly",
			StripePaymentMethodID: "pm_test",
			IdempotencyKey:        "key-heal-fail",
		})
	if !errors.Is(err, persistErr) {
		t.Errorf("expected persistErr to be wrapped, got: %v", err)
	}
}
