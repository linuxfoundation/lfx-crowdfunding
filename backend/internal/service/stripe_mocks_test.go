// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"

	stripe "github.com/stripe/stripe-go/v85"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// configStripeClient is a configurable StripeClient for service unit tests.
// Any method whose function field is nil panics, making accidentally-called
// paths immediately visible.
type configStripeClient struct {
	onGetProduct          func(context.Context, string) (*models.StripeProduct, error)
	onCreateProduct       func(ctx context.Context, initiativeID, name string) (string, error)
	onDeleteProduct       func(context.Context, string) error
	onCreatePaymentIntent func(context.Context, models.PaymentIntentRequest) (*models.PaymentIntent, error)
	onCreateSubscription  func(context.Context, models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error)
	onCancelSubscription          func(context.Context, string) error
	onUpdatePaymentIntentMetadata func(context.Context, string, map[string]string) error
	onConstructWebhook    func([]byte, string, string) (stripe.Event, error)
	onCreateCustomer      func(context.Context, string, string) (string, error)
	onCreateSetupIntent   func(context.Context, string) (string, error)
	onAttachPaymentMethod func(context.Context, string, string) (*models.CardDetails, error)
	onGetPaymentMethod    func(context.Context, string) (*models.CardDetails, error)
	onDetachPaymentMethod func(context.Context, string) error
	onGetOrCreatePrice    func(context.Context, string, string, int64, string, string) (string, error)
}

func (c *configStripeClient) GetProduct(ctx context.Context, id string) (*models.StripeProduct, error) {
	if c.onGetProduct != nil {
		return c.onGetProduct(ctx, id)
	}
	panic("GetProduct not expected")
}
func (c *configStripeClient) CreateProduct(ctx context.Context, initiativeID, name string) (string, error) {
	if c.onCreateProduct != nil {
		return c.onCreateProduct(ctx, initiativeID, name)
	}
	panic("CreateProduct not expected")
}
func (c *configStripeClient) DeleteProduct(ctx context.Context, productID string) error {
	if c.onDeleteProduct != nil {
		return c.onDeleteProduct(ctx, productID)
	}
	panic("DeleteProduct not expected")
}
func (c *configStripeClient) CreatePaymentIntent(ctx context.Context, req models.PaymentIntentRequest) (*models.PaymentIntent, error) {
	if c.onCreatePaymentIntent != nil {
		return c.onCreatePaymentIntent(ctx, req)
	}
	panic("CreatePaymentIntent not expected")
}
func (c *configStripeClient) CreateSubscription(ctx context.Context, req models.StripeSubscriptionRequest) (*models.StripeSubscriptionResult, error) {
	if c.onCreateSubscription != nil {
		return c.onCreateSubscription(ctx, req)
	}
	panic("CreateSubscription not expected")
}
func (c *configStripeClient) CancelSubscription(ctx context.Context, id string) error {
	if c.onCancelSubscription != nil {
		return c.onCancelSubscription(ctx, id)
	}
	panic("CancelSubscription not expected")
}
func (c *configStripeClient) UpdatePaymentIntentMetadata(ctx context.Context, piID string, metadata map[string]string) error {
	if c.onUpdatePaymentIntentMetadata != nil {
		return c.onUpdatePaymentIntentMetadata(ctx, piID, metadata)
	}
	panic("UpdatePaymentIntentMetadata not expected")
}
func (c *configStripeClient) ConstructWebhookEvent(p []byte, sig, secret string) (stripe.Event, error) {
	if c.onConstructWebhook != nil {
		return c.onConstructWebhook(p, sig, secret)
	}
	panic("ConstructWebhookEvent not expected")
}
func (c *configStripeClient) CreateCustomer(ctx context.Context, userID, email string) (string, error) {
	if c.onCreateCustomer != nil {
		return c.onCreateCustomer(ctx, userID, email)
	}
	panic("CreateCustomer not expected")
}
func (c *configStripeClient) CreateSetupIntent(ctx context.Context, customerID string) (string, error) {
	if c.onCreateSetupIntent != nil {
		return c.onCreateSetupIntent(ctx, customerID)
	}
	panic("CreateSetupIntent not expected")
}
func (c *configStripeClient) AttachPaymentMethod(ctx context.Context, customerID, pmID string) (*models.CardDetails, error) {
	if c.onAttachPaymentMethod != nil {
		return c.onAttachPaymentMethod(ctx, customerID, pmID)
	}
	panic("AttachPaymentMethod not expected")
}
func (c *configStripeClient) GetPaymentMethod(ctx context.Context, pmID string) (*models.CardDetails, error) {
	if c.onGetPaymentMethod != nil {
		return c.onGetPaymentMethod(ctx, pmID)
	}
	panic("GetPaymentMethod not expected")
}
func (c *configStripeClient) DetachPaymentMethod(ctx context.Context, pmID string) error {
	if c.onDetachPaymentMethod != nil {
		return c.onDetachPaymentMethod(ctx, pmID)
	}
	panic("DetachPaymentMethod not expected")
}
func (c *configStripeClient) GetOrCreatePrice(ctx context.Context, productID, initiativeID string, amountCents int64, frequency string, idempotencyKey string) (string, error) {
	if c.onGetOrCreatePrice != nil {
		return c.onGetOrCreatePrice(ctx, productID, initiativeID, amountCents, frequency, idempotencyKey)
	}
	panic("GetOrCreatePrice not expected")
}

// testUserRepo is a configurable UserRepository. Methods with nil fields
// return zero values (no error) so only the fields under test need to be set.
type testUserRepo struct {
	onGetByUsername            func(context.Context, string) (*models.User, error)
	onGetByID                  func(context.Context, string) (*models.User, error)
	onUpsert                   func(context.Context, *models.User) (*models.User, error)
	onUpdateStripeInfo         func(context.Context, string, string, string) error
	onClearStripePaymentMethod func(context.Context, string) error
}

func (r *testUserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	if r.onGetByUsername != nil {
		return r.onGetByUsername(ctx, username)
	}
	return &models.User{ID: "00000000-0000-0000-0000-000000000001", Username: username, Email: username + "@test.example"}, nil
}
func (r *testUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	if r.onGetByID != nil {
		return r.onGetByID(ctx, id)
	}
	return &models.User{ID: id}, nil
}
func (r *testUserRepo) Upsert(ctx context.Context, user *models.User) (*models.User, error) {
	if r.onUpsert != nil {
		return r.onUpsert(ctx, user)
	}
	return user, nil
}
func (r *testUserRepo) UpdateStripeInfo(ctx context.Context, userID, customerID, pmID string) error {
	if r.onUpdateStripeInfo != nil {
		return r.onUpdateStripeInfo(ctx, userID, customerID, pmID)
	}
	return nil
}
func (r *testUserRepo) ClearStripePaymentMethod(ctx context.Context, userID string) error {
	if r.onClearStripePaymentMethod != nil {
		return r.onClearStripePaymentMethod(ctx, userID)
	}
	return nil
}

// testDonationRepo is a configurable DonationRepository.
type testDonationRepo struct {
	onCreate                  func(context.Context, *models.Donation) (*models.Donation, error)
	onUpdateByPaymentIntentID func(context.Context, string, string, string) error
}

func (r *testDonationRepo) GetByID(_ context.Context, _ string) (*models.Donation, error) {
	return nil, nil
}
func (r *testDonationRepo) ListByInitiative(_ context.Context, _ string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *testDonationRepo) ListByUser(_ context.Context, _ string, _ models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *testDonationRepo) Create(ctx context.Context, d *models.Donation) (*models.Donation, error) {
	if r.onCreate != nil {
		return r.onCreate(ctx, d)
	}
	return d, nil
}
func (r *testDonationRepo) UpdateByPaymentIntentID(ctx context.Context, piID, status, chargeID string) error {
	if r.onUpdateByPaymentIntentID != nil {
		return r.onUpdateByPaymentIntentID(ctx, piID, status, chargeID)
	}
	return nil
}

// testSubscriptionRepo is a configurable SubscriptionRepository.
type testSubscriptionRepo struct {
	onGetByID                       func(context.Context, string) (*models.Subscription, error)
	onGetActiveByUserAndInitiative  func(context.Context, string, string) (*models.Subscription, error)
	onCreate                        func(context.Context, *models.Subscription) (*models.Subscription, error)
	onUpdate                        func(context.Context, *models.Subscription) (*models.Subscription, error)
	onUpdateByStripeSubscriptionID  func(context.Context, string, string) error
}

func (r *testSubscriptionRepo) GetByID(ctx context.Context, id string) (*models.Subscription, error) {
	if r.onGetByID != nil {
		return r.onGetByID(ctx, id)
	}
	return nil, nil
}
func (r *testSubscriptionRepo) GetActiveByUserAndInitiative(ctx context.Context, userID, initiativeID string) (*models.Subscription, error) {
	if r.onGetActiveByUserAndInitiative != nil {
		return r.onGetActiveByUserAndInitiative(ctx, userID, initiativeID)
	}
	return nil, domain.ErrSubscriptionNotFound
}
func (r *testSubscriptionRepo) ListByInitiative(_ context.Context, _ string, _ models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *testSubscriptionRepo) ListByUser(_ context.Context, _ string, _ models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error) {
	return nil, nil, nil
}
func (r *testSubscriptionRepo) Create(ctx context.Context, s *models.Subscription) (*models.Subscription, error) {
	if r.onCreate != nil {
		return r.onCreate(ctx, s)
	}
	return s, nil
}
func (r *testSubscriptionRepo) Update(ctx context.Context, s *models.Subscription) (*models.Subscription, error) {
	if r.onUpdate != nil {
		return r.onUpdate(ctx, s)
	}
	return s, nil
}
func (r *testSubscriptionRepo) UpdateByStripeSubscriptionID(ctx context.Context, subID, status string) error {
	if r.onUpdateByStripeSubscriptionID != nil {
		return r.onUpdateByStripeSubscriptionID(ctx, subID, status)
	}
	return nil
}
