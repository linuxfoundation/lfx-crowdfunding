// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package domain defines shared domain errors and repository contracts.
package domain

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// InitiativeRepository defines persistence operations for initiatives.
type InitiativeRepository interface {
	GetByID(ctx context.Context, id string) (*models.Initiative, error)
	GetBySlug(ctx context.Context, slug string) (*models.Initiative, error)
	// GetIDBySlug returns only the UUID for a given slug.
	// Used by the transactions handler to resolve a slug without triggering Ledger enrichment.
	GetIDBySlug(ctx context.Context, slug string) (string, error)
	List(ctx context.Context, filter models.InitiativeFilter) ([]*models.Initiative, *models.PaginationMeta, error)
	Create(ctx context.Context, initiative *models.Initiative) (*models.Initiative, error)
	Update(ctx context.Context, initiative *models.Initiative) (*models.Initiative, error)
	Delete(ctx context.Context, id string) error

	// GetUsersByIDs returns a map of Auth0 user_id → User for the given IDs.
	// Missing IDs are absent from the map. Used to enrich Ledger transactions.
	GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]models.User, error)

	// GetOrganizationsByIDs returns a map of org UUID → Organization for the given IDs.
	// Missing IDs are absent from the map. Used to enrich Ledger transactions.
	GetOrganizationsByIDs(ctx context.Context, ids []string) (map[string]models.Organization, error)
}

// DonationRepository defines persistence operations for donations.
type DonationRepository interface {
	GetByID(ctx context.Context, id string) (*models.Donation, error)
	ListByInitiative(ctx context.Context, initiativeID string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error)
	ListByUser(ctx context.Context, userID string, filter models.DonationFilter) ([]models.Donation, *models.PaginationMeta, error)
	Create(ctx context.Context, donation *models.Donation) (*models.Donation, error)
	// UpdateByPaymentIntentID is called by the Stripe webhook to reconcile async 3DS results.
	UpdateByPaymentIntentID(ctx context.Context, piID, status, chargeID string) error
}

// SubscriptionRepository defines persistence operations for subscriptions.
type SubscriptionRepository interface {
	GetByID(ctx context.Context, id string) (*models.Subscription, error)
	ListByInitiative(ctx context.Context, initiativeID string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error)
	ListByUser(ctx context.Context, userID string, filter models.SubscriptionFilter) ([]models.Subscription, *models.PaginationMeta, error)
	Create(ctx context.Context, sub *models.Subscription) (*models.Subscription, error)
	Update(ctx context.Context, sub *models.Subscription) (*models.Subscription, error)
	// UpdateByStripeSubscriptionID is called by the Stripe webhook to advance subscription status.
	UpdateByStripeSubscriptionID(ctx context.Context, stripeSubID, status string) error
}

// OrganizationRepository defines persistence operations for organizations.
type OrganizationRepository interface {
	GetByID(ctx context.Context, id string) (*models.Organization, error)
	ListByOwner(ctx context.Context, ownerID string) ([]models.Organization, error)
}

// UserRepository defines persistence operations for users.
type UserRepository interface {
	GetByUserID(ctx context.Context, userID string) (*models.User, error)
	Upsert(ctx context.Context, user *models.User) (*models.User, error)
	// UpdateStripeInfo persists the Stripe Customer ID and default PaymentMethod
	// after the user completes the setup-intent / attach-payment-method flow.
	// An empty string for either field leaves the existing DB value unchanged.
	UpdateStripeInfo(ctx context.Context, userID, customerID, paymentMethodID string) error
	// ClearStripePaymentMethod sets stripe_default_payment_method to NULL.
	// Called when the user removes their saved card.
	ClearStripePaymentMethod(ctx context.Context, userID string) error
}

// StatisticsRepository defines persistence operations for platform-wide statistics.
type StatisticsRepository interface {
	GetPlatformStatistics(ctx context.Context) (*models.PlatformStatistics, error)

	// GetOrganizationsByIDs returns a map of org UUID → Organization for the given IDs.
	// Missing IDs are absent from the map. Used to enrich Ledger sponsor/donor data.
	GetOrganizationsByIDs(ctx context.Context, ids []string) (map[string]models.Organization, error)

	// GetUsersByIDs returns a map of Auth0 user_id → User for the given IDs.
	// Missing IDs are absent from the map. Used to enrich Ledger sponsor/donor data.
	GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]models.User, error)
}

// LedgerStatsRepository defines the persistence operations used exclusively by
// the ledger-stats-sync CronJob.  All methods are scoped to the batch read/write
// pattern of that job and should not be used by the HTTP API.
type LedgerStatsRepository interface {
	// ListActiveSyncIDs returns the UUIDs of all initiatives whose status is
	// not 'archived' or 'draft'.  These are the initiatives the CronJob must
	// attempt to sync on every run.
	ListActiveSyncIDs(ctx context.Context) ([]string, error)

	// BulkUpsertLedgerStats inserts or updates rows in initiative_ledger_stats.
	// Returns the number of rows successfully upserted.
	BulkUpsertLedgerStats(ctx context.Context, stats []models.LedgerStats) (int, error)

	// GetOrganizationsByIDs returns a map of org UUID → Organization for all
	// IDs in the slice.  Missing IDs are simply absent from the map.
	GetOrganizationsByIDs(ctx context.Context, ids []string) (map[string]models.Organization, error)

	// GetUsersByIDs returns a map of user_id (Auth0 subject) → User for all
	// IDs in the slice.  Missing IDs are simply absent from the map.
	GetUsersByIDs(ctx context.Context, userIDs []string) (map[string]models.User, error)
}
